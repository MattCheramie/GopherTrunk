// Package composer bridges the trunking engine's CallStart events to
// the per-call demod chain that turns IQ samples on a freshly-tuned
// Voice device into 16-bit PCM the recorder can write.
//
// One Composer subscribes to events.KindCallStart / events.KindCallEnd
// from the bus. On a CallStart it looks up the Voice device by serial,
// opens its IQ stream, and starts a goroutine that runs an FM
// passthrough chain (LPF → decimate → quadrature FM demod → optional
// post-demod de-emphasis → optional Kaiser audio LPF → optional
// audio AGC → optional polyphase resample (or coarse decimate) →
// int16 PCM → recorder.WritePCM). The
// chain also calls Engine.Touch on a one-second cadence so the
// engine's silent-call watchdog doesn't kill the call
// mid-conversation.
//
// DMR voice grants run a dedicated chain (see dmr_voice.go): IQ →
// DMR receiver → voice superframe decoder → on-air AMBE frames
// appended to the recorder's .raw sidecar. P25 Phase 1 / Phase 2 have
// their own chains; TETRA follows the traffic channel and lays down a
// raw full-slot sidecar (see tetra_voice.go — TCH/S FEC + ACELP are
// follow-ups). The remaining digital protocols (NXDN, dPMR, YSF,
// D-STAR, EDACS ProVoice) have no composer chain yet — their grants
// are logged and bypassed.
package composer

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/autotune"
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/equalizer"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/cryptocap"
)

// EqualizerConfig opts an adaptive blind equalizer (CMA) into the
// post-decimation, pre-FM-demod stage of the per-call analog chain.
// The win is simulcast-distortion mitigation: when multiple
// transmitters cover the same frequency at slightly different arrival
// delays, CMA drives the complex baseband back toward a constant
// modulus and unblurs the FM signal. Defaults are conservative — eight
// taps and a small step size — chosen so the equalizer behaves close
// to a pass-through on a clean signal and converges within a few
// hundred samples on a degraded one.
//
// FM voice has constant envelope on air, so the LMS variant (which
// needs a known training symbol stream) isn't useful here; the LMS
// type is exported from internal/dsp/equalizer for protocol decoders
// (P25 C4FM with a known FSW preamble) that need a directed update.
type EqualizerConfig struct {
	Enabled  bool
	Taps     int     // default 8
	StepSize float32 // default 1e-4
}

// IQSource is the subset of sdr.Device the composer needs. Decoupling
// keeps the package free of a hard import on internal/sdr and makes
// testing trivial with an in-memory channel.
//
// SampleRateHz reports the chunk rate this source delivers. The
// composer reads it once when a call starts so per-source rates
// (a wideband-derived virtual voice tuner emits 48 kHz while a
// physical SDR emits the daemon-wide 2.4 MS/s) drive the right
// decimation factor. Sources that return 0 fall back to the
// daemon-wide rate the Composer was constructed with.
type IQSource interface {
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
	SampleRateHz() uint32
}

// exactRateSource is an optional capability an IQSource may implement to
// report its delivered rate with the fractional part preserved. A
// wideband-derived virtual voice tuner's DDC resampler does not always
// land exactly on 48 kHz; the integer SampleRateHz rounds that away, but a
// symbol-recovery loop clocked at the rounded rate drifts off the true
// symbol phase and periodically slips, surfacing as voice spikes/glitches.
// Sources that know their exact rate implement this so the digital voice
// chains clock their receivers at the true rate. Physical SDRs (exact
// integer rates) need not implement it — the composer falls back to
// SampleRateHz. Mirrors the control-channel fix in widebandt2 (issue #550).
type exactRateSource interface {
	SampleRateExactHz() float64
}

// dmoColourSource is the optional capability a voice IQ source implements when
// it can report the DM traffic colour code the control-side TETRA DMO pipeline
// has recovered (ccdecoder.CCVoiceSource delegates to the decoder's atomic).
// A DMO grant structurally fires before the pipeline finishes recovering the
// colour (grant at 4 qualified DNBs, recovery needs 20), so the grant's colour
// hint is almost always 0; the running voice chain polls this instead, adopting
// the pipeline's answer the moment it lands rather than brute-forcing 64
// colours on its own IQ goroutine. A dedicated voice SDR has no CC decoder and
// simply lacks the capability.
type dmoColourSource interface {
	TETRADMOColour() (colour uint32, known bool)
}

// Devices resolves a Voice-role IQ source by its serial. The daemon
// supplies a wrapper around sdr.Pool; tests use a map.
type Devices interface {
	FindBySerial(serial string) IQSource
}

// SquelchState is the optional live squelch feed the conventional
// scanner exposes (issue #1090). The analog FM chain polls it per IQ
// chunk: while the scanner reports the channel squelch-closed (the
// hangtime tail — carrier gone, countdown running) the chain freezes
// the audio AGC and mutes its PCM output, instead of demodulating pure
// receiver noise and AGC-boosting it into the recording as a loud
// multi-second squelch crash.
type SquelchState interface {
	// SquelchOpen reports the current squelch decision for deviceSerial.
	// ok=false means the provider has no live decision for this serial
	// (not a conventional-scanner dwell); the chain must then stay
	// ungated — analog-trunk voice channels (Motorola/LTR/MPT 1327)
	// share runFMChain but have no scanner-side squelch.
	SquelchOpen(deviceSerial string) (open, ok bool)
}

// PCMSink is the subset of voice.Recorder we touch. Recorder.WritePCM
// matches this signature exactly.
type PCMSink interface {
	WritePCM(deviceSerial string, samples []int16) error
}

// EngineHooks exposes the Touch / EndCall / UpdateSignal calls the chain
// uses to keep the engine in sync with what the chain hears. Stubbing this
// interface lets tests assert Touch fires on a real cadence.
type EngineHooks interface {
	Touch(deviceSerial string)
	EndCall(deviceSerial string, reason trunking.EndReason) bool
	// UpdateSignal stamps the call's measured received channel power
	// (dBFS) onto the bound ActiveCall so the engine carries it into
	// CallEnd. Called once at end-of-call, before EndCall.
	UpdateSignal(deviceSerial string, dbfs float64)
	// UpdateDemod stamps the call's measured demod quality — RMS EVM (%) and
	// estimated SNR (dB) — onto the bound ActiveCall so the engine carries it
	// into CallEnd. Called once at end-of-call, before EndCall, and only for a
	// call whose chain fed the receiver soft/symbol taps (P25 Phase 1).
	UpdateDemod(deviceSerial string, evmPct, snrDB float64)
}

// Options configure a Composer.
type Options struct {
	Bus     *events.Bus
	Devices Devices
	Sink    PCMSink     // typically *voice.Recorder
	Engine  EngineHooks // typically *trunking.Engine
	Log     *slog.Logger
	// IQSampleRate is the per-second sample rate the SDR pool delivers
	// (typically 2.4e6). Required.
	IQSampleRate uint32
	// PCMSampleRate is the rate the recorder expects (default 8000).
	PCMSampleRate uint32
	// VoiceBandwidthHz is the cutoff of the front-end LPF (default
	// 12_500 — wide enough for analog FM voice with some margin).
	VoiceBandwidthHz uint32
	// TouchInterval is how often the chain pings Engine.Touch while
	// audio is flowing (default 1 s).
	TouchInterval time.Duration
	// VoiceIQDebug enables per-call voice-channel IQ debug captures (the
	// diagnostic-container workflow, see voice_iq_debug.go). Off by
	// default; zero cost when disabled.
	VoiceIQDebug VoiceIQDebugConfig
	// VoiceHangtime is the universal end-of-transmission window applied
	// to every voice chain: once voice has been decoding, the chain ends
	// the call this long after the last decoded voice frame (rather than
	// waiting out the engine's much longer call-timeout watchdog).
	// Default 3.5 s.
	VoiceHangtime time.Duration
	// SplitPerTransmission selects the recording-boundary mode for every
	// voice chain. True (default) rolls the recording to a new file at
	// each end-of-transmission boundary (one file per over). False
	// ("conversation") keeps consecutive same-talkgroup overs in one
	// file, splitting only on a talkgroup change or VoiceHangtime idle.
	SplitPerTransmission bool
	// Equalizer optionally enables a CMA blind equalizer between the
	// front-end LPF and the FM demod. Off by default; flip Enabled
	// to true and tune Taps / StepSize per site.
	Equalizer EqualizerConfig
	// DeEmphasis configures the post-demod single-pole IIR that
	// recovers the pre-emphasized treble curve broadcast FM
	// transmitters apply for SNR. Off by default — set Enabled and
	// pick TimeConstant (75µs in NA, 50µs in EU). Filter runs at the
	// intermediate ~48 kHz rate, before the second decimation.
	DeEmphasis DeEmphasisConfig
	// AudioLPF configures a Kaiser-windowed FIR low-pass on the real
	// audio after de-emphasis and before the decimation to PCM. The
	// point is two-fold: band-limit voice to roughly 3.4 kHz (telephony
	// quality, kills hiss + sub-carriers), and act as the
	// anti-aliasing filter for the second decimation. Off by default;
	// callers tune CutoffHz (typical 3400) and Taps (default 81).
	AudioLPF AudioLPFConfig
	// AudioAGC configures a real-valued envelope-follower-based AGC
	// applied after the audio LPF (so the envelope follower sees a
	// clean band-limited signal). The point is to level out the
	// loudness difference between weak and strong transmitters on
	// the same talkgroup so recordings don't whiplash. Off by
	// default; analog FM systems opt in via daemon config.
	AudioAGC AudioAGCConfig
	// AudioResampler swaps the naive integer decimation that hands
	// audio to the recorder for a polyphase L/M resampler with
	// proper anti-aliasing built into the prototype filter. The
	// resampler is sized from the intermediate-rate / PCM-rate ratio
	// the chain already computes, so the caller only opts in
	// (Enabled) and optionally tunes TapsPerBranch / Beta. Off by
	// default; the existing AudioLPF + naive decimation produces
	// equivalent audio when the two rates are integer multiples.
	AudioResampler AudioResamplerConfig
	// Autotune, when non-nil, is the shared per-dongle carrier-error
	// registry (sdr.autotune). The P25 Phase 1 voice chain pre-rotates its
	// IQ by the voice dongle's running-average correction so the receiver's
	// AFC starts near lock, and folds the residual offset back into the
	// average at end-of-call. Nil disables autotune on the voice path at
	// zero cost. See internal/autotune.
	Autotune *autotune.Registry
	// CryptoSink, when non-nil, opts into the cryptolab crypto-frame bridge:
	// the P25 Phase 1 voice chain hands it each encrypted superframe's
	// Message Indicator + encrypted voice frames for offline keystream-reuse
	// analysis. Nil (default) disables the capture at zero cost — no frame
	// extraction runs for it. See internal/voice/cryptocap.
	CryptoSink cryptocap.Sink
	// Squelch, when non-nil, gates the analog FM chain's audio on the
	// conventional scanner's live squelch decision (issue #1090). The
	// daemon builds the composer before the scanner, so it wires this
	// via SetSquelchState instead; the option exists for callers (and
	// tests) that have the provider up front. Nil leaves every chain
	// ungated — identical to the pre-gate behavior.
	Squelch SquelchState
}

// DeEmphasisConfig holds runtime knobs for the post-FM-demod
// de-emphasis filter.
type DeEmphasisConfig struct {
	Enabled      bool
	TimeConstant time.Duration // typically 75µs (NA) or 50µs (EU)
}

// AudioLPFConfig holds runtime knobs for the post-demod audio
// low-pass. CutoffHz is in Hz (relative to the intermediate rate the
// FM demod emits). Taps controls the FIR length; longer = sharper
// transition at the cost of latency. Both fall back to sane defaults
// when zero.
type AudioLPFConfig struct {
	Enabled  bool
	CutoffHz uint32
	Taps     int
}

// AudioResamplerConfig holds runtime knobs for the polyphase audio
// resampler. TapsPerBranch (default 16) controls the prototype
// filter's per-branch length; Beta (default 8.6) is the Kaiser
// window shape parameter — higher β = steeper transition, more
// stopband rejection, longer impulse.
type AudioResamplerConfig struct {
	Enabled       bool
	TapsPerBranch int
	Beta          float64
}

// AudioAGCConfig holds runtime knobs for the post-demod audio AGC.
// Reference, Attack, Release, and MaxGain default to sane voice
// values when zero.
type AudioAGCConfig struct {
	Enabled   bool
	Reference float32       // target |output| (default 0.3)
	Attack    time.Duration // ramp-up time constant (default 5 ms)
	Release   time.Duration // ramp-down time constant (default 200 ms)
	MaxGain   float32       // ceiling on adaptive gain (default 64.0)
}

// Composer is the long-lived event-driven bridge.
type Composer struct {
	bus    *events.Bus
	dev    Devices
	sink   PCMSink
	engine EngineHooks
	log    *slog.Logger

	iqHz         uint32
	voiceIQDebug VoiceIQDebugConfig
	pcmHz        uint32
	bw           uint32
	touchEvery   time.Duration
	hangtime     time.Duration
	splitTx      bool
	eqCfg        EqualizerConfig
	deemphCfg    DeEmphasisConfig
	lpfCfg       AudioLPFConfig
	agcCfg       AudioAGCConfig
	resampCfg    AudioResamplerConfig
	autotune     *autotune.Registry
	cryptoSink   cryptocap.Sink
	// squelch is the optional conventional-scanner squelch feed for the
	// FM chain (issue #1090). Guarded by mu: the daemon sets it after
	// construction (SetSquelchState) and each chain reads it once at
	// start.
	squelch SquelchState

	// drainCoord, when the sink supports it (the recorder does), lets the
	// composer tell the recorder that a call's voice chain has fully drained
	// its buffered audio, so the recorder finalizes the recording only AFTER
	// the transmission-tail frames have been written. Cached once at New so the
	// hot teardown path does no type-assert. nil for sinks that don't support
	// coordination (test stubs, analog-only callers) — the recorder then
	// finalizes on CallEnd as before.
	drainCoord drainCoordinator

	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu     sync.Mutex
	chains map[string]*chain
	// tetraDemuxes holds one shared voice demultiplexer per same-carrier TETRA
	// control carrier (keyed by sameCarrierSource.CarrierKey). It decodes the
	// carrier once and routes each burst to the single call that owns its AACH
	// usage marker, so concurrent same-carrier calls decode independently without
	// the per-call pre-anchor / hangtime-reuse cross-slot leaks. See tetra_voice.go.
	tetraDemuxes map[string]*tetraSlotDemux
}

// sameCarrierSource is the optional capability a voice IQ source implements when
// it is a shared same-carrier tap: several such taps on one control carrier
// deliver the same post-DDC stream, so they must share ONE TETRA voice demux
// rather than each running its own receiver. CarrierKey groups the taps that
// belong to the same physical carrier. Implemented by ccdecoder.CCVoiceSource.
type sameCarrierSource interface {
	CarrierKey() string
}

type chain struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// New validates options and constructs a Composer. The bus subscription
// is created at construction time so callers can publish events before
// Run starts without losing them.
func New(opts Options) (*Composer, error) {
	if opts.Bus == nil {
		return nil, errors.New("composer: events.Bus is required")
	}
	if opts.Devices == nil {
		return nil, errors.New("composer: Devices is required")
	}
	if opts.IQSampleRate == 0 {
		return nil, errors.New("composer: IQSampleRate is required")
	}
	if opts.PCMSampleRate == 0 {
		opts.PCMSampleRate = 8000
	}
	if opts.VoiceBandwidthHz == 0 {
		opts.VoiceBandwidthHz = 12_500
	}
	if opts.TouchInterval <= 0 {
		opts.TouchInterval = time.Second
	}
	if opts.VoiceHangtime <= 0 {
		opts.VoiceHangtime = 3500 * time.Millisecond
	}
	if opts.Equalizer.Enabled {
		if opts.Equalizer.Taps <= 0 {
			opts.Equalizer.Taps = 8
		}
		if opts.Equalizer.StepSize <= 0 {
			opts.Equalizer.StepSize = 1e-4
		}
	}
	if opts.DeEmphasis.Enabled && opts.DeEmphasis.TimeConstant <= 0 {
		opts.DeEmphasis.TimeConstant = filter.DeEmphasis75us
	}
	if opts.AudioLPF.Enabled {
		if opts.AudioLPF.CutoffHz == 0 {
			opts.AudioLPF.CutoffHz = 3_400
		}
		if opts.AudioLPF.Taps <= 0 {
			opts.AudioLPF.Taps = 81
		}
	}
	// AudioAGC defaults are applied inside dsp.NewAudioAGC, so the
	// composer doesn't need to materialize them here.
	if opts.AudioResampler.Enabled {
		if opts.AudioResampler.TapsPerBranch <= 0 {
			opts.AudioResampler.TapsPerBranch = 16
		}
		if opts.AudioResampler.Beta <= 0 {
			opts.AudioResampler.Beta = 8.6
		}
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	c := &Composer{
		bus:          opts.Bus,
		dev:          opts.Devices,
		sink:         opts.Sink,
		engine:       opts.Engine,
		log:          log,
		iqHz:         opts.IQSampleRate,
		voiceIQDebug: opts.VoiceIQDebug,
		pcmHz:        opts.PCMSampleRate,
		bw:           opts.VoiceBandwidthHz,
		touchEvery:   opts.TouchInterval,
		hangtime:     opts.VoiceHangtime,
		splitTx:      opts.SplitPerTransmission,
		eqCfg:        opts.Equalizer,
		deemphCfg:    opts.DeEmphasis,
		lpfCfg:       opts.AudioLPF,
		agcCfg:       opts.AudioAGC,
		resampCfg:    opts.AudioResampler,
		autotune:     opts.Autotune,
		cryptoSink:   opts.CryptoSink,
		squelch:      opts.Squelch,
		chains:       make(map[string]*chain),
		tetraDemuxes: make(map[string]*tetraSlotDemux),
		runDone:      make(chan struct{}),
	}
	// Enable drain coordination when the sink (the recorder) supports it: the
	// composer will signal NotifyDrainComplete once each call's chain has
	// drained, and the recorder defers finalize until that signal (or a safety
	// timeout) so the transmission tail is not dropped by the finalize race.
	if dc, ok := c.sink.(drainCoordinator); ok && dc != nil {
		c.drainCoord = dc
		dc.EnableDrainCoordination()
	}
	c.sub = opts.Bus.Subscribe()
	return c, nil
}

// SetSquelchState wires the conventional scanner's live squelch
// decision into the analog FM chain after construction (issue #1090) —
// the daemon builds the composer before the scanner, so it cannot pass
// the provider through Options. Each chain reads the provider once at
// start, so calls made before the chain's CallStart take effect;
// in the daemon both happen during single-threaded construction,
// before Run delivers any event. A nil provider (or one reporting
// ok=false for a serial) leaves chains ungated.
func (c *Composer) SetSquelchState(sq SquelchState) {
	c.mu.Lock()
	c.squelch = sq
	c.mu.Unlock()
}

// Run drains CallStart / CallEnd events until ctx cancels, spawning /
// reaping per-call demod goroutines. Every active chain is cancelled
// on context cancel so Close drains cleanly.
func (c *Composer) Run(ctx context.Context) error {
	defer close(c.runDone)
	for {
		select {
		case <-ctx.Done():
			c.cancelAll()
			return ctx.Err()
		case ev, ok := <-c.sub.C:
			if !ok {
				c.cancelAll()
				return nil
			}
			switch ev.Kind {
			case events.KindCallStart:
				if cs, ok := ev.Payload.(trunking.CallStart); ok {
					c.handleStart(ctx, cs)
				}
			case events.KindCallEnd:
				if ce, ok := ev.Payload.(trunking.CallEnd); ok {
					c.handleEnd(ce)
				}
			}
		}
	}
}

// Close releases the bus subscription and waits for Run to drain. It
// also cancels every active chain. Idempotent.
func (c *Composer) Close() error {
	c.closeOnce.Do(func() {
		c.sub.Close()
		select {
		case <-c.runDone:
		case <-time.After(time.Second):
		}
		c.cancelAll()
	})
	return nil
}

// ActiveChains returns the device serials with running chains. Test
// helper; takes the lock so it's race-free.
func (c *Composer) ActiveChains() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.chains))
	for k := range c.chains {
		out = append(out, k)
	}
	return out
}

// voiceKind classifies a grant's protocol into the composer voice chain that
// decodes it. It is computed once per call in handleStart, replacing the fan of
// per-protocol booleans (isFM/isDMRVoice/…) that used to be recomputed for the
// bypass check, the same-carrier TETRA special-case, and the dispatch switch.
type voiceKind int

const (
	// voiceKindUnsupported has no composer voice chain yet (dPMR, YSF, D-STAR,
	// EDACS ProVoice) — its voice bursts are not decoded into PCM here.
	voiceKindUnsupported voiceKind = iota
	// voiceKindFM is analog narrowband FM: conventional channels ("", fm,
	// fm-conv, analog) and the analog-trunk voice channels (Motorola/LTR/
	// MPT 1327, and EDACS unless it is digital ProVoice), which all decode
	// through runFMChain.
	voiceKindFM
	voiceKindDMR      // dmr-tier1/2/3
	voiceKindP25P1    // p25
	voiceKindP25P2    // p25-phase2
	voiceKindTETRA    // tetra
	voiceKindTETRADMO // tetra-dmo (Direct Mode)
	voiceKindNXDN     // nxdn
)

// classifyVoiceKind maps a grant to its voiceKind. Digital protocols are matched
// first; everything analog (incl. the analog-trunk protocols, and EDACS when it
// is not ProVoice) is voiceKindFM; anything else is voiceKindUnsupported.
func classifyVoiceKind(cs trunking.CallStart) voiceKind {
	switch cs.Grant.Protocol {
	case "dmr-tier1", "dmr-tier2", "dmr-tier3":
		return voiceKindDMR
	case "p25":
		return voiceKindP25P1
	case "p25-phase2":
		return voiceKindP25P2
	case "tetra":
		return voiceKindTETRA
	case "tetra-dmo":
		return voiceKindTETRADMO
	case "nxdn":
		return voiceKindNXDN
	}
	proto := cs.Grant.Protocol
	isAnalogTrunk := proto == "motorola" || proto == "ltr" || proto == "mpt1327" ||
		(proto == "edacs" && !cs.Grant.ProVoice)
	if proto == "" || proto == "fm" || proto == "fm-conv" || proto == "analog" || isAnalogTrunk {
		return voiceKindFM
	}
	return voiceKindUnsupported
}

func (c *Composer) handleStart(parent context.Context, cs trunking.CallStart) {
	proto := cs.Grant.Protocol
	kind := classifyVoiceKind(cs)
	if kind == voiceKindUnsupported {
		c.log.Info("composer: digital protocol not yet decoded; chain bypassed",
			"device", cs.DeviceSerial, "protocol", proto,
			"group", cs.Grant.GroupID)
		return
	}
	src := c.dev.FindBySerial(cs.DeviceSerial)
	if src == nil {
		c.log.Warn("composer: no device for serial", "serial", cs.DeviceSerial)
		return
	}
	c.mu.Lock()
	if existing := c.chains[cs.DeviceSerial]; existing != nil {
		// Engine should have ended the prior call first; defensive.
		existing.cancel()
		<-existing.done
		delete(c.chains, cs.DeviceSerial)
	}
	c.mu.Unlock()

	// Same-carrier TETRA: route through the shared per-carrier voice demux instead
	// of opening a per-call IQ subscription + receiver. One demux decodes the
	// carrier and delivers each burst only to the call that currently owns that
	// burst's AACH usage marker, eliminating the per-call pre-anchor accept-all and
	// hangtime marker-reuse leaks. Non-same-carrier TETRA (a dedicated retuned voice
	// SDR — one call per tap) and every other protocol fall through to the per-call
	// StreamIQ path below.
	if kind == voiceKindTETRA {
		if scs, ok := src.(sameCarrierSource); ok {
			if key := scs.CarrierKey(); key != "" {
				c.followTETRASameCarrier(parent, src, key, cs)
				return
			}
		}
	}

	chainCtx, cancel := context.WithCancel(parent)
	iqCh, err := src.StreamIQ(chainCtx)
	if err != nil {
		cancel()
		c.log.Warn("composer: StreamIQ failed", "serial", cs.DeviceSerial, "err", err)
		return
	}
	// Per-source rate lets a virtual voice tuner (wideband-derived,
	// emits 48 kHz IQ) coexist with physical SDRs (daemon-wide
	// rate, typically 2.4 MS/s) in the same composer — the chain
	// resolves its decimator off this value instead of a fixed
	// daemon-wide constant. Sources that don't yet know their
	// rate (return 0) fall back to the daemon-wide setting.
	rateHz := src.SampleRateHz()
	if rateHz == 0 {
		rateHz = c.iqHz
	}
	// Prefer the source's exact (fractional) rate when it exposes one, so
	// the digital voice chains clock their symbol-recovery loop at the true
	// rate rather than a rounded nominal — a nominal-rate clock drifts and
	// slips, which the listener hears as spikes/glitches (issue #550 parity
	// for the voice path). Physical SDRs don't implement this and fall back
	// to the exact integer rate above.
	rateHzF := float64(rateHz)
	if exact, ok := src.(exactRateSource); ok {
		if r := exact.SampleRateExactHz(); r > 0 {
			rateHzF = r
		}
	}
	// Diagnostic-container voice capture: tee the exact IQ stream this
	// call's chain will decode into a per-call file (voice_iq_debug.go).
	// Lossless toward the chain; disk side is best-effort.
	if c.voiceIQDebug.Enabled {
		iqCh = c.teeVoiceIQ(chainCtx, iqCh, cs, rateHzF)
	}
	ch := &chain{cancel: cancel, done: make(chan struct{})}
	c.mu.Lock()
	c.chains[cs.DeviceSerial] = ch
	c.mu.Unlock()

	switch kind {
	case voiceKindDMR:
		if cs.Grant.Encrypted {
			// Surface encryption clearly: the .raw sidecar will hold
			// encrypted AMBE+2 frames and the WAV will be unintelligible
			// until in-process descramble lands (docs/dmr-encryption.md).
			c.log.Info("composer: DMR voice call is encrypted; .raw sidecar holds encrypted AMBE+2 frames, in-process decryption not yet available",
				"device", cs.DeviceSerial, "system", cs.Grant.System,
				"group", cs.Grant.GroupID)
		}
		go c.runDMRVoiceChain(chainCtx, cs.DeviceSerial, cs.Grant.System, iqCh, rateHzF, cs.Grant.GroupID, cs.Grant.DMRInterleavedVoice, ch.done)
	case voiceKindP25P2:
		macCfg := p25p2.MACDecodeConfig{
			Trellis:      p25p2.TrellisMode(cs.Grant.P25Phase2Decode.Trellis),
			RS:           p25p2.RSMode(cs.Grant.P25Phase2Decode.RS),
			Interleave:   p25p2.InterleaveMode(cs.Grant.P25Phase2Decode.Interleave),
			Scrambler:    p25p2.ScramblerMode(cs.Grant.P25Phase2Decode.Scrambler),
			Seed:         cs.Grant.P25Phase2Decode.Seed,
			SoftDecision: cs.Grant.P25Phase2Decode.SoftDecision,
			Equalizer:    cs.Grant.P25Phase2Decode.Equalizer,
			DCBlock:      cs.Grant.P25Phase2Decode.DCBlock,
		}
		go c.runP25Phase2VoiceChain(chainCtx, cs.DeviceSerial, cs.Grant.System, macCfg, iqCh, rateHzF, ch.done)
	case voiceKindP25P1:
		go c.runP25Phase1VoiceChain(chainCtx, cs.DeviceSerial, cs.Grant.System, iqCh, rateHzF, cs.Grant.P25Phase1DemodMode, cs.Grant.GroupID, cs.Grant.CallID, cs.Grant.PatchedGroups, ch.done)
	case voiceKindTETRA:
		go c.runTETRAVoiceChain(chainCtx, cs.DeviceSerial, iqCh, rateHzF, cs.Grant.GroupID, cs.Grant.Timeslot, cs.Grant.TETRAColourExt, cs.Grant.TETRAUsageMarker, cs.Grant.TETRATrafficLMS, ch.done)
	case voiceKindTETRADMO:
		var liveColour func() (uint32, bool)
		if dcs, ok := src.(dmoColourSource); ok {
			liveColour = dcs.TETRADMOColour
		}
		go c.runTETRADMOVoiceChain(chainCtx, cs.DeviceSerial, iqCh, rateHzF, cs.Grant.TETRAColourExt, cs.Grant.TETRADMOBaseMNI, liveColour, ch.done)
	case voiceKindNXDN:
		go c.runNXDNVoiceChain(chainCtx, cs.DeviceSerial, cs.Grant.System, iqCh, rateHzF, cs.Grant.GroupID, ch.done)
	default:
		// Analog FM has no symbol clock to drift, so the rounded integer
		// rate is fine; keep its uint32 signature unchanged.
		go c.runFMChain(chainCtx, cs.DeviceSerial, iqCh, uint32(math.Round(rateHzF)), ch.done)
	}
}

func (c *Composer) handleEnd(ce trunking.CallEnd) {
	c.mu.Lock()
	ch := c.chains[ce.DeviceSerial]
	delete(c.chains, ce.DeviceSerial)
	c.mu.Unlock()
	if ch != nil {
		ch.cancel()
		// Block until the chain goroutine has fully drained and written every
		// tail frame (drainTETRAIQ / the same-carrier owner worker for TETRA,
		// the in-line teardown for DMR/P25/FM). Only after this returns are all
		// of the call's frames guaranteed written to the recorder.
		<-ch.done
	}
	// Tell the recorder the drain is complete so it can finalize the recording.
	// Fired for EVERY CallEnd — including calls with no chain — so a
	// drain-coordinated recorder never blocks waiting for a signal that will
	// not come. Ordering (this vs the recorder's own CallEnd handling) is
	// arbitrary; the recorder finalizes once both have arrived.
	if c.drainCoord != nil {
		c.drainCoord.NotifyDrainComplete(ce.DeviceSerial)
	}
}

func (c *Composer) cancelAll() {
	c.mu.Lock()
	chains := c.chains
	c.chains = make(map[string]*chain)
	demuxes := c.tetraDemuxes
	c.tetraDemuxes = make(map[string]*tetraSlotDemux)
	c.mu.Unlock()
	// Cancel the per-call chains first so their owners unregister from the
	// demuxes, then tear the demuxes down.
	for _, ch := range chains {
		ch.cancel()
		<-ch.done
	}
	for _, d := range demuxes {
		d.cancel()
		<-d.done
	}
}

// followTETRASameCarrier binds one same-carrier TETRA call to the carrier's
// shared voice demux: it ensures the demux exists, then spawns a thin chain
// goroutine that registers this call as the owner of its granted AACH usage
// marker and unregisters on call end. The chain does no IQ work of its own — the
// demux delivers its marker's decoded speech frames.
func (c *Composer) followTETRASameCarrier(parent context.Context, src IQSource, key string, cs trunking.CallStart) {
	d := c.ensureTETRADemux(parent, key, src, cs.Grant.TETRAColourExt, cs.Grant.TETRATrafficLMS)
	if d == nil {
		c.log.Warn("composer: could not start TETRA voice demux", "serial", cs.DeviceSerial, "key", key)
		return
	}
	chainCtx, cancel := context.WithCancel(parent)
	ch := &chain{cancel: cancel, done: make(chan struct{})}
	c.mu.Lock()
	c.chains[cs.DeviceSerial] = ch
	c.mu.Unlock()
	go c.runTETRASameCarrierChain(chainCtx, d, cs.DeviceSerial, cs.Grant.GroupID, cs.Grant.Timeslot, cs.Grant.TETRAUsageMarker, ch.done)
}

// ensureTETRADemux returns the shared voice demux for a control carrier, creating
// and starting it on first use. The demux lives for the composer's lifetime (its
// SB anchor + AACH state stay warm across calls, so a new call never re-anchors
// from scratch) until Close/cancelAll or the control decoder's IQ stream closes.
// Only ever called from the single Run goroutine, so the create is race-free.
func (c *Composer) ensureTETRADemux(parent context.Context, key string, src IQSource, colourExt uint32, trafficLMS bool) *tetraSlotDemux {
	c.mu.Lock()
	if d := c.tetraDemuxes[key]; d != nil {
		c.mu.Unlock()
		return d
	}
	d := &tetraSlotDemux{
		c:          c,
		key:        key,
		colour:     colourExt,
		trafficLMS: trafficLMS,
		owners:     make(map[uint8]*tetraSlotOwner),
		done:       make(chan struct{}),
	}
	c.tetraDemuxes[key] = d
	c.mu.Unlock()

	demuxCtx, cancel := context.WithCancel(parent)
	d.cancel = cancel
	iqCh, err := src.StreamIQ(demuxCtx)
	if err != nil {
		cancel()
		c.mu.Lock()
		delete(c.tetraDemuxes, key)
		c.mu.Unlock()
		close(d.done)
		c.log.Warn("composer: TETRA voice demux StreamIQ failed", "key", key, "err", err)
		return nil
	}
	rateHzF := float64(src.SampleRateHz())
	if rateHzF == 0 {
		rateHzF = float64(c.iqHz)
	}
	if exact, ok := src.(exactRateSource); ok {
		if r := exact.SampleRateExactHz(); r > 0 {
			rateHzF = r
		}
	}
	go d.run(demuxCtx, iqCh, rateHzF)
	return d
}

// removeTETRADemux drops a demux from the registry once its IQ stream has ended
// (self-called from the demux goroutine), so the next same-carrier grant builds a
// fresh one (e.g. after a control-SDR reacquire swaps the decoder).
func (c *Composer) removeTETRADemux(key string, d *tetraSlotDemux) {
	c.mu.Lock()
	if c.tetraDemuxes[key] == d {
		delete(c.tetraDemuxes, key)
	}
	c.mu.Unlock()
}

// runFMChain consumes IQ for one call. The chain is intentionally
// straightforward: LPF the IQ to voice bandwidth, naive-decimate to
// roughly 48 kHz, quadrature-FM-demod, naive-decimate again to the
// recorder's PCM rate, and convert to int16. A higher-fidelity version
// (proper polyphase resamplers, de-emphasis, post-demod LPF) is a
// follow-up; this is honest passthrough quality good enough to verify
// the wiring end-to-end and to land the operator-visible plumbing.
func (c *Composer) runFMChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-fm:"+serial, nil)

	// Shared boundary controller: Touch heartbeat + hangtime end-of-call,
	// uniform with the digital chains. Analog FM has no in-band talkgroup
	// (grantTG 0 → gating disabled) and emits PCM continuously, so in
	// practice the engine's grant lifecycle / watchdog bounds the call;
	// the tracker keeps the engine's LastHeardAt fresh.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	// Front-end decimator: an 81-tap anti-alias FIR that convolves ONLY at
	// the output positions, replacing the old full-rate FIR + every-Nth-
	// sample decimation that filtered every input sample and discarded most
	// of the result (~194M MACs/sec at 2.4 MS/s). Same filter and same kept
	// samples, so the demodulated audio is byte-for-byte unchanged. Unlike
	// the digital chains, FM band-limits even when not decimating, so
	// filterAtUnity is true.
	const intermediateHz = 48_000
	fe := newDecimatingFIR(float64(iqHz), intermediateHz, float64(c.bw), true)

	// Second-stage decimation to PCM. Still naive; resampling-quality audio
	// lands with the opt-in polyphase resampler below.
	decim2 := intermediateHz / int(c.pcmHz)
	if decim2 < 1 {
		decim2 = 1
	}

	fm := demod.NewFM()

	// Optional CMA blind equalizer for simulcast-distortion mitigation.
	// Sits between the front-end LPF (decimated) and the FM demod so it
	// operates at the intermediate rate (~48 kHz) rather than 2.4 MS/s.
	// R^2 = 1 because FM has unit-magnitude carrier on air.
	var eq *equalizer.CMA
	if c.eqCfg.Enabled {
		eq = equalizer.NewCMA(c.eqCfg.Taps, c.eqCfg.StepSize, 1.0)
	}

	// Optional post-demod de-emphasis. The transmitter pre-emphasized
	// treble for SNR; without the matching low-pass the recording
	// sounds harsh. Filter runs on the real audio at the intermediate
	// rate (~48 kHz) before the second naive decimation to PCM.
	intermediateHzf := fe.OutRateHz()
	var deemph *filter.DeEmphasis
	if c.deemphCfg.Enabled {
		deemph = filter.NewDeEmphasis(c.deemphCfg.TimeConstant, intermediateHzf)
	}

	// Optional post-demod audio LPF. Two jobs: band-limit voice to
	// ~3.4 kHz (telephony grade, kills hiss + sub-carriers like the
	// 19 kHz pilot tone on broadcast FM if any leaks through) and
	// act as the anti-aliasing filter for the decimation that
	// follows. Cutoff is normalized against the intermediate rate.
	var audioLPF *filter.RealFIR
	if c.lpfCfg.Enabled {
		fc := float64(c.lpfCfg.CutoffHz) / intermediateHzf
		if fc >= 0.5 {
			fc = 0.45
		}
		audioLPF = filter.NewRealFIR(filter.LowpassKaiser(c.lpfCfg.Taps, fc, 8.6))
	}

	// Optional audio AGC. Sits after the LPF so the envelope
	// follower sees a clean band-limited signal — pre-emphasis is
	// already undone, hiss + sub-carriers already trimmed — which
	// keeps the level estimate stable and prevents the AGC from
	// chasing high-frequency garbage. Operates at the intermediate
	// rate so attack/release time constants line up with what the
	// caller configured.
	var agc *dsp.AudioAGC
	if c.agcCfg.Enabled {
		agc = dsp.NewAudioAGC(dsp.AudioAGCConfig{
			Reference:  c.agcCfg.Reference,
			Attack:     c.agcCfg.Attack,
			Release:    c.agcCfg.Release,
			MaxGain:    c.agcCfg.MaxGain,
			SampleRate: intermediateHzf,
		})
	}

	// Optional polyphase audio resampler. Replaces the naive
	// decimateAndConvert audio decimation with an L/M polyphase
	// resampler whose prototype filter doubles as the anti-aliasing
	// LPF. L and M come from the intermediate-rate / PCM-rate ratio
	// the chain already computed (decim2 = M with L = 1 for the
	// integer-multiple case), so callers only opt in.
	var resamp *dsp.RealResampler
	if c.resampCfg.Enabled {
		resamp = dsp.NewRealResampler(1, decim2, c.resampCfg.TapsPerBranch, c.resampCfg.Beta)
	}

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	// pcmWrites counts PCM batches successfully handed to the sink —
	// i.e. real audio delivery. The touch ticker (below) only refreshes
	// the engine's LastHeardAt when this counter has advanced since
	// the previous tick. Without this gate a stalled IQ source still
	// kept the call alive forever via an unconditional 1 s heartbeat
	// (issue #356).
	var pcmWrites atomic.Uint64

	// lastSample tracks the most recent PCM sample written so we
	// can ramp it down to zero when the chain ends. Without this
	// the audio sink hears an abrupt cut from carrier-active audio
	// to silence — a 'click' that's the analog-scanner equivalent
	// of a squelch tail. A 10 ms linear fade-out covers most call
	// ends inaudibly.
	var lastSample int16
	emitTail := func() {
		if c.sink == nil {
			return
		}
		// 10 ms at the PCM rate; integer division is fine here
		// because the cadence is forgiving.
		n := int(c.pcmHz / 100)
		if n < 8 {
			n = 8
		}
		tail := make([]int16, n)
		startF := float32(lastSample)
		for i := range n {
			ramp := 1.0 - float32(i)/float32(n)
			tail[i] = int16(startF * ramp)
		}
		_ = c.sink.WritePCM(serial, tail)
	}

	// Reusable equalizer scratch — the equalized slice is fully consumed by
	// fm.Process within the same iteration, so a single grow-and-reslice
	// buffer per chain avoids a per-chunk allocation for the whole call
	// (issue #492 footprint reduction).
	var eqScratch []complex64

	// Squelch-tail gate (issue #1090). While the conventional scanner
	// reports the channel squelch-closed (hangtime countdown running,
	// only receiver noise on the air), the chain keeps running — the
	// decimators/filters stay warm and the PCM timeline stays continuous
	// for the recorder and live stream — but the audio AGC is frozen
	// (adapting on demodulated noise is what rode the gain up to the
	// loud audible tail) and the written PCM is replaced by a short
	// fade-out into silence. muteFade/muteStep carry the fade across
	// chunk boundaries: PCM chunks here are ~a dozen samples, well
	// under the ~10 ms ramp.
	c.mu.Lock()
	squelch := c.squelch
	c.mu.Unlock()
	var muteFade, muteStep float32
	squelchWasOpen := true

	for {
		select {
		case <-ctx.Done():
			emitTail()
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the shared boundary
			// tracker (bt.run).
		case iq, ok := <-iqCh:
			if !ok {
				emitTail()
				return
			}
			squelchOpen := true
			if squelch != nil {
				if open, ok := squelch.SquelchOpen(serial); ok {
					squelchOpen = open
				}
			}
			bt.observe(iq)
			decimated := fe.Process(nil, iq)
			if eq != nil {
				if cap(eqScratch) < len(decimated) {
					eqScratch = make([]complex64, len(decimated))
				}
				eqScratch = eqScratch[:len(decimated)]
				for i, x := range decimated {
					y, _ := eq.Process(x)
					eqScratch[i] = y
				}
				decimated = eqScratch
			}
			audio := fm.Process(nil, decimated)
			if deemph != nil {
				audio = deemph.Process(audio, audio)
			}
			if audioLPF != nil {
				audio = audioLPF.Process(audio, audio)
			}
			// Freeze the AGC while squelch-closed: its output is about
			// to be muted anyway, and letting the envelope follower
			// adapt on demodulated noise poisons the level estimate for
			// the next over inside the hangtime window.
			if agc != nil && squelchOpen {
				audio = agc.Process(audio, audio)
			}
			var pcm []int16
			if resamp != nil {
				// Polyphase rate-conversion already emits at the
				// PCM rate; convert in place without further
				// decimation.
				pcm = convertToPCM(resamp.Process(nil, audio))
			} else {
				pcm = decimateAndConvert(audio, decim2)
			}
			if !squelchOpen {
				if squelchWasOpen {
					// Entering the tail: fade from the last written
					// sample to zero over ~10 ms — an abrupt mute
					// clicks just like the chain-end cut emitTail
					// covers.
					muteFade = float32(lastSample)
					n := int(c.pcmHz / 100)
					if n < 8 {
						n = 8
					}
					muteStep = muteFade / float32(n)
				}
				for i := range pcm {
					muteFade -= muteStep
					if muteFade*muteStep <= 0 {
						// Reached (or crossed) zero: hold silence.
						muteFade, muteStep = 0, 0
					}
					pcm[i] = int16(muteFade)
				}
			}
			squelchWasOpen = squelchOpen
			if c.sink != nil && len(pcm) > 0 {
				// Plain WritePCM (no CallID fence) is correct here: an
				// analog FM chain keys on a stable per-channel physical
				// device serial, never the reused wb:<serial>:tap-N pool
				// the wideband engine drives (and which it restricts to
				// digital protocols). With one chain per serial, torn down
				// on CallEnd before the next CallStart, there is no
				// cross-call serial reuse — so the WritePCMForCall fence the
				// digital tap path needs would be guarding a condition that
				// can't occur on this path.
				_ = c.sink.WritePCM(serial, pcm)
				lastSample = pcm[len(pcm)-1]
				pcmWrites.Add(1)
				bt.onVoice(0)
			}
		}
	}
}

func decimateComplex(in []complex64, factor int) []complex64 {
	if factor <= 1 {
		return in
	}
	out := make([]complex64, 0, len(in)/factor+1)
	for i := 0; i < len(in); i += factor {
		out = append(out, in[i])
	}
	return out
}

// decimateAndConvert decimates a real audio stream and converts to
// 16-bit signed PCM. The FM demodulator emits radians/sample in
// roughly [-π, +π]; we scale by ~10 000 to fill the int16 range for
// reasonable deviation, then clamp.
func decimateAndConvert(in []float32, factor int) []int16 {
	if factor < 1 {
		factor = 1
	}
	out := make([]int16, 0, len(in)/factor+1)
	for i := 0; i < len(in); i += factor {
		v := float64(in[i]) * 10_000
		if v > math.MaxInt16 {
			v = math.MaxInt16
		}
		if v < math.MinInt16 {
			v = math.MinInt16
		}
		out = append(out, int16(v))
	}
	return out
}

// convertToPCM converts a float32 audio stream that's already at the
// PCM sample rate (i.e. handed back from RealResampler) to 16-bit
// signed PCM with the same scale + clamp decimateAndConvert uses.
func convertToPCM(in []float32) []int16 {
	out := make([]int16, len(in))
	for i, x := range in {
		v := float64(x) * 10_000
		if v > math.MaxInt16 {
			v = math.MaxInt16
		}
		if v < math.MinInt16 {
			v = math.MinInt16
		}
		out[i] = int16(v)
	}
	return out
}
