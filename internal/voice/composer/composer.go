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
// appended to the recorder's .raw sidecar. Other digital protocols
// (P25, NXDN, ...) have no composer chain yet — their grants are
// logged and bypassed.
package composer

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/equalizer"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
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
type IQSource interface {
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
}

// Devices resolves a Voice-role IQ source by its serial. The daemon
// supplies a wrapper around sdr.Pool; tests use a map.
type Devices interface {
	FindBySerial(serial string) IQSource
}

// PCMSink is the subset of voice.Recorder we touch. Recorder.WritePCM
// matches this signature exactly.
type PCMSink interface {
	WritePCM(deviceSerial string, samples []int16) error
}

// EngineHooks exposes the Touch / EndCall calls the chain uses to
// keep the engine in sync with what the chain hears. Stubbing this
// interface lets tests assert Touch fires on a real cadence.
type EngineHooks interface {
	Touch(deviceSerial string)
	EndCall(deviceSerial string, reason trunking.EndReason) bool
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

	iqHz       uint32
	pcmHz      uint32
	bw         uint32
	touchEvery time.Duration
	eqCfg      EqualizerConfig
	deemphCfg  DeEmphasisConfig
	lpfCfg     AudioLPFConfig
	agcCfg     AudioAGCConfig
	resampCfg  AudioResamplerConfig

	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu     sync.Mutex
	chains map[string]*chain
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
		bus:        opts.Bus,
		dev:        opts.Devices,
		sink:       opts.Sink,
		engine:     opts.Engine,
		log:        log,
		iqHz:       opts.IQSampleRate,
		pcmHz:      opts.PCMSampleRate,
		bw:         opts.VoiceBandwidthHz,
		touchEvery: opts.TouchInterval,
		eqCfg:      opts.Equalizer,
		deemphCfg:  opts.DeEmphasis,
		lpfCfg:     opts.AudioLPF,
		agcCfg:     opts.AudioAGC,
		resampCfg:  opts.AudioResampler,
		chains:     make(map[string]*chain),
		runDone:    make(chan struct{}),
	}
	c.sub = opts.Bus.Subscribe()
	return c, nil
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

func (c *Composer) handleStart(parent context.Context, cs trunking.CallStart) {
	proto := cs.Grant.Protocol
	isFM := proto == "" || proto == "fm" || proto == "analog"
	isDMRVoice := proto == "dmr-tier2" || proto == "dmr-tier3"
	isP25P2Voice := proto == "p25-phase2"
	if !isFM && !isDMRVoice && !isP25P2Voice {
		// Other digital protocols (P25 Phase 1, NXDN, ...) have no
		// composer voice chain yet — their voice bursts are not decoded.
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

	chainCtx, cancel := context.WithCancel(parent)
	iqCh, err := src.StreamIQ(chainCtx)
	if err != nil {
		cancel()
		c.log.Warn("composer: StreamIQ failed", "serial", cs.DeviceSerial, "err", err)
		return
	}
	ch := &chain{cancel: cancel, done: make(chan struct{})}
	c.mu.Lock()
	c.chains[cs.DeviceSerial] = ch
	c.mu.Unlock()

	switch {
	case isDMRVoice:
		go c.runDMRVoiceChain(chainCtx, cs.DeviceSerial, iqCh, ch.done)
	case isP25P2Voice:
		go c.runP25Phase2VoiceChain(chainCtx, cs.DeviceSerial, iqCh, ch.done)
	default:
		go c.runFMChain(chainCtx, cs.DeviceSerial, iqCh, ch.done)
	}
}

func (c *Composer) handleEnd(ce trunking.CallEnd) {
	c.mu.Lock()
	ch := c.chains[ce.DeviceSerial]
	delete(c.chains, ce.DeviceSerial)
	c.mu.Unlock()
	if ch == nil {
		return
	}
	ch.cancel()
	<-ch.done
}

func (c *Composer) cancelAll() {
	c.mu.Lock()
	chains := c.chains
	c.chains = make(map[string]*chain)
	c.mu.Unlock()
	for _, ch := range chains {
		ch.cancel()
		<-ch.done
	}
}

// runFMChain consumes IQ for one call. The chain is intentionally
// straightforward: LPF the IQ to voice bandwidth, naive-decimate to
// roughly 48 kHz, quadrature-FM-demod, naive-decimate again to the
// recorder's PCM rate, and convert to int16. A higher-fidelity version
// (proper polyphase resamplers, de-emphasis, post-demod LPF) is a
// follow-up; this is honest passthrough quality good enough to verify
// the wiring end-to-end and to land the operator-visible plumbing.
func (c *Composer) runFMChain(ctx context.Context, serial string, iqCh <-chan []complex64, done chan<- struct{}) {
	defer close(done)

	// Front-end LPF: cutoff = bw / iqHz (normalized 0..0.5).
	cutoff := float64(c.bw) / float64(c.iqHz)
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	taps := filter.LowpassKaiser(81, cutoff, 8.6)
	lpf := filter.NewFIR(taps)

	// Naive decimation factors. They aren't exact; resampling-quality
	// audio lands with the polyphase resampler in a follow-up.
	const intermediateHz = 48_000
	decim1 := int(c.iqHz) / intermediateHz
	if decim1 < 1 {
		decim1 = 1
	}
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
	intermediateHzf := float64(c.iqHz) / float64(decim1)
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

	for {
		select {
		case <-ctx.Done():
			emitTail()
			return
		case <-touchTicker.C:
			if c.engine != nil {
				c.engine.Touch(serial)
			}
		case iq, ok := <-iqCh:
			if !ok {
				emitTail()
				return
			}
			filtered := lpf.Process(nil, iq)
			decimated := decimateComplex(filtered, decim1)
			if eq != nil {
				equalized := make([]complex64, len(decimated))
				for i, x := range decimated {
					y, _ := eq.Process(x)
					equalized[i] = y
				}
				decimated = equalized
			}
			audio := fm.Process(nil, decimated)
			if deemph != nil {
				audio = deemph.Process(audio, audio)
			}
			if audioLPF != nil {
				audio = audioLPF.Process(audio, audio)
			}
			if agc != nil {
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
			if c.sink != nil && len(pcm) > 0 {
				_ = c.sink.WritePCM(serial, pcm)
				lastSample = pcm[len(pcm)-1]
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
