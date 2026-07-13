// Package widebandt2 monitors several trunked / conventional carriers
// with a single SDR dongle. The dongle is pinned to a configured centre
// frequency; an internal/dsp/tuner.Bank extracts one narrow-band IQ
// stream per channel inside the dongle's IQ band; each stream feeds a
// protocol-specific receiver and per-channel state machine that
// publishes cc.locked / grant events on the bus.
//
// Supported channel state machines:
//
//   - DMR Tier II conventional (config protocol "dmr-tier2"). Each
//     channel is a per-repeater carrier; the state machine
//     (radio/dmr/tier2) emits a grant on every Voice LC Header burst.
//
//   - DMR Tier III trunked control channel (config protocol "dmr").
//     The channel frequency must match one of the system's
//     control_channels; the state machine (radio/dmr/tier3) emits
//     grants from the CSBK chain.
//
//   - P25 Phase 1 trunked control channel (config protocol "p25").
//     The channel frequency must match one of the system's
//     control_channels; the state machine (radio/p25/phase1) emits
//     grants from the TSBK chain. C4FM vs CQPSK / LSM is selected
//     via the system's p25_phase1_demod_mode key.
//
//   - P25 Phase 2 trunked control channel (config protocol
//     "p25-phase2"). The channel frequency must match one of the
//     system's control_channels; the state machine
//     (radio/p25/phase2) consumes superframes assembled by the
//     receiver and emits grants from the MAC PDU chain.
//
// Published grants flow through the trunking engine's existing
// voice-pool allocator, which binds each to either a physical
// role: voice dongle or a virtual voice tap (a per-grant DDC on this
// same wideband IQ stream — see internal/sdr/wbvoice, enabled per
// dongle via voice_taps). So a wideband dongle can decode its own
// voice grants without a separate voice SDR, and falls back to a
// physical voice SDR when the grant lands outside its IQ window or
// every tap is busy. DMR Tier III grants reference channels by LCN,
// so the system needs a dmr_band_plan to resolve LCN → frequency;
// without it those grants are dropped (decode.error stage=no-bandplan).
//
// One Engine owns one wideband SDR. Multiple wideband dongles each
// get their own Engine; they run independently and share only the
// events.Bus and the metrics surface.
package widebandt2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25phase2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// narrowbandRateHz is the per-tap sample rate the bank decimates to.
// 48 kHz matches the rate the DMR receiver's matched filter +
// Mueller-Müller clock loop are sized for (≈10 samples/symbol at
// 4800 baud), the same rate the existing single-channel ccdecoder
// down-converter targets.
const narrowbandRateHz = 48_000.0

// Per-protocol receiver constants mirror the per-pipeline settings the
// single-frequency CC decoder uses, so wideband-channelized streams
// behave identically to the dedicated-dongle path.
//
// DMR (ETSI TS 102 361-1 §6.3): peak deviation 1944 Hz at symbol ±3.
// T2's harder symbol distribution (mean transition magnitude 1.27 vs
// T3's 0.90) needs a more conservative loop gain.
//
// P25 Phase 1 (TIA-102.BAAA-A): nominal peak deviation 1800 Hz at
// symbol ±3. Only consulted on the C4FM path; the CQPSK / LSM path
// is amplitude-invariant after the matched filter.
//
// P25 Phase 2 (TIA-102.BBAC): H-DQPSK at 6000 symbols/s. The
// pipeline factory's tuned Gardner gain (0.005) matches PR #154's
// observation that π/4-DQPSK family signals slip differently than
// C4FM at the receiver's default gain.
const (
	dmrDeviationHz       = 1944.0
	dmrClockGainTier2    = 0.015 // matches newDMRTier2Pipeline in ccdecoder
	dmrClockGainTier3    = 0.025 // matches newDMRTier3Pipeline in ccdecoder
	p25Phase1DeviationHz = 1800.0
	p25Phase2GardnerGain = 0.005
)

// guardFrac is the fraction of the IQ band the tuner reserves at
// each edge as a guard against alias roll-off. Mirrors the value
// the config validator uses to reject out-of-band channels.
const guardFrac = 0.05

// sampleRateAdvisoryFactor is how far above the plan's minimum required rate the
// capture rate must sit before New advises lowering it. 1.5× leaves comfortable
// headroom (front-end roll-off, retune slack) before flagging genuine waste.
const sampleRateAdvisoryFactor = 1.5

// strategyAutoThreshold is the channel-count cutoff for the "auto"
// tuner strategy: at or below it, DDCBank wins on simplicity;
// above it, ChannelizerBank's shared wide-band filter pays off.
const strategyAutoThreshold = 6

// channelizerBinWidthHz is the target per-bin width the polyphase
// channelizer aims for. 16 bins across a 2.4 MS/s IQ band gives 150 kHz
// per bin — wide enough that 12.5 kHz DMR repeater spacing maps to
// distinct bins, narrow enough that the shared anti-alias filter is sharp.
// channelizerBinsFor keeps this width roughly constant as the sample rate
// rises, so a 10 MS/s band gets ~64 bins instead of 16 (625 kHz) bins that
// collapse adjacent carriers together (issue #764).
const channelizerBinWidthHz = 150_000.0

// channelizerBinsFor returns the channelizer bin count for an IQ sample
// rate: the power of two that lands closest to channelizerBinWidthHz per
// bin, floored at 16 so low rates behave exactly as before.
func channelizerBinsFor(sampleRateHz uint32) int {
	bins := 16
	for float64(sampleRateHz)/float64(bins*2) >= channelizerBinWidthHz {
		bins *= 2
	}
	return bins
}

// channelizerTapsPerBranch and channelizerKaiserBeta match the
// channelizer package's own test defaults — wideband-clean
// passband, ~70 dB peak sidelobe.
const channelizerTapsPerBranch = 16
const channelizerKaiserBeta = 9.0

// channelizerCleanResidualFrac is the |residual|/binRate above which a tap sits
// far enough toward its channelizer bin edge to lose noticeable SNR. A
// critically-sampled bin is flat to ~0.45·binRate then rolls off to −6 dB at
// the edge (0.5). New warns when a polyphase plan crowds taps past this — a
// dense, irregular plan (e.g. 70 DMR repeaters on a 12.5 kHz grid that never
// aligns to the bin centres) inevitably does. The channelizer is still used
// because it is the only bank that stays real-time at this tap count (a per-tap
// DDC benches ~6x heavier — see BenchmarkDense71* in internal/dsp/tuner); the
// warning just makes the trade-off visible.
const channelizerCleanResidualFrac = 0.40

// ChannelConfig binds one repeater frequency to the trunking system
// it belongs to. The Engine creates one DMR state machine per entry,
// dispatched by the referenced system's protocol (dmr-tier2 →
// tier2.ConventionalChannel; dmr → tier3.ControlChannel).
type ChannelConfig struct {
	FrequencyHz uint32
	SystemName  string
}

// IQPowerObserver is the minimal metrics surface the engine uses to
// publish its per-channel window-averaged dBFS gauge plus the whole-capture
// power / clip diagnostics (issue #749). internal/metrics.Metrics satisfies
// it; nil disables the gauges while leaving the throttled WARNs and the DEBUG
// decode-activity summary in place. Declared locally (rather than imported
// from ccdecoder) so this scanner package doesn't depend on another.
type IQPowerObserver interface {
	RecordIQPowerDbFS(system string, dbfs float64)
	ClearIQPowerDbFS(system string)
	// Whole-capture diagnostics, labelled by dongle serial. The per-tap
	// gauge above can't surface front-end overload (a strong site clipping
	// the shared ADC raises the floor for every tap) — these can.
	RecordWidebandInputPowerDbFS(serial string, dbfs float64)
	RecordWidebandInputClipRatio(serial string, ratio float64)
	ClearWidebandInput(serial string)
}

// Options bundles the inputs Engine needs at construction time.
type Options struct {
	Log *slog.Logger
	Bus *events.Bus

	// Metrics, if non-nil, receives each channel's window-averaged dBFS
	// gauge (labelled by "system @ <freqMHz>" so per-channel readings
	// don't collide) plus the whole-capture power / clip-ratio gauges
	// (labelled by Serial). Nil-safe: the throttled WARNs and the DEBUG
	// decode-activity summary still fire without it.
	Metrics IQPowerObserver

	// Serial labels this dongle's whole-capture power/clip gauges
	// (issue #749). Empty is tolerated (labelled "unknown").
	Serial string

	// Device is the wideband SDR. It is set to CenterFreqHz, then
	// streamed continuously until Run's context cancels.
	Device sdr.Device
	// SampleRateHz is the dongle's IQ sample rate.
	SampleRateHz uint32
	// CenterFreqHz is the wide-band centre frequency.
	CenterFreqHz uint32

	// TunerStrategy picks the Bank implementation. "" / "auto"
	// auto-selects by channel count; "ddc" forces DDCBank;
	// "polyphase" forces ChannelizerBank.
	TunerStrategy string

	// Channels is the list of per-repeater carriers to monitor.
	// All FrequencyHz values must fall inside the dongle's IQ
	// band (CenterFreqHz ± SampleRateHz/2 minus guardFrac).
	Channels []ChannelConfig

	// Systems is the trunking-system table. The Engine looks up
	// each ChannelConfig.SystemName here to decide whether the
	// channel is a Tier II conventional carrier (protocol
	// "dmr-tier2") or a Tier III control channel (protocol "dmr").
	// Required when Channels is non-empty.
	Systems []trunking.System

	// Now overrides the wall clock the per-channel state machines
	// use for Grant.At timestamps. nil ⇒ time.Now.
	Now func() time.Time
}

// Engine owns the per-dongle IQ pump goroutine and the per-channel
// receiver + state-machine fan-out.
type Engine struct {
	log         *slog.Logger
	bus         *events.Bus
	device      sdr.Device
	centerHz    uint32
	sampleRate  uint32
	bank        tuner.Bank
	channels    []*engineChannel
	strategyTag string

	// metrics is the optional per-channel dBFS gauge sink (nil-safe).
	metrics IQPowerObserver
	// serial labels the whole-capture power/clip gauges (issue #749).
	serial string
	// now is the clock the diagnostics window uses; injectable for tests.
	now func() time.Time
	// lastDiagAt is the wall-clock time the per-channel diagnostics were
	// last flushed. Owned by the Run pump goroutine.
	lastDiagAt time.Time
	// widebandPwr accumulates the raw wideband input power (before
	// channelization) across one diagnostics window, drained alongside the
	// per-channel powers. It distinguishes a dead front end (wideband input
	// also low ⇒ antenna/gain/cable) from a misplaced channel (wideband input
	// healthy but this channel low ⇒ carrier outside the captured passband or
	// the wrong frequency) when a low-power WARN fires. Owned by the Run pump.
	widebandPwr iqpower.Accumulator
	// wbClipped / wbClipSamples count rail-pinned wideband input samples over
	// the same window (issue #749). A strong site clipping the shared ADC is
	// the front-end-overload failure mode the per-tap power gauge can't see —
	// it raises the noise floor for every tap. Owned by the Run pump.
	wbClipped     int
	wbClipSamples int
	// wbClipLogAt throttles the overload WARN. Owned by the Run pump.
	wbClipLogAt time.Time

	// suspended, when set, makes the Run pump drain its IQ stream without
	// processing it: no channelization, no per-channel decode, no grants, no
	// diagnostics. It exists so a live hunt that borrows this engine's SDR can
	// retune it away without the engine decoding the off-frequency IQ — which
	// otherwise floods the log with "channel iq power very low" WARNs and emits
	// spurious grants whose voice taps capture no audio (empty TG folders).
	// Read on the pump goroutine; set via Suspend/Resume from another goroutine.
	suspended atomic.Bool
}

// Serial returns the dongle serial this engine decodes on (the value passed as
// Options.Serial). It lets a coordinator — e.g. the daemon's hunt acquirer —
// match a running engine to the SDR a hunt is about to borrow.
func (e *Engine) Serial() string { return e.serial }

// Suspend pauses decoding: the Run pump keeps draining the IQ stream (so the
// SDR/broker never blocks) but processes nothing until Resume. Idempotent and
// safe to call from any goroutine. Used while a hunt borrows this engine's SDR.
func (e *Engine) Suspend() {
	if e.suspended.Swap(true) {
		return
	}
	e.log.Info("widebandt2: decode suspended (SDR borrowed by hunt)", "serial", e.serial)
}

// Resume re-enables decoding after a Suspend. Because the borrower retuned the
// shared SDR, Resume first reprograms it back to this engine's centre frequency
// so the next chunk is on-channel again; it then clears the suspend flag.
// Idempotent and safe to call from any goroutine.
func (e *Engine) Resume() {
	if !e.suspended.Load() {
		return
	}
	if err := e.device.SetCenterFreq(e.centerHz); err != nil {
		// Log and clear anyway: leaving the engine suspended on a retune error
		// would strand it decoding nothing, which is strictly worse.
		e.log.Warn("widebandt2: resume retune failed; resuming anyway",
			"serial", e.serial, "center_freq_hz", e.centerHz, "err", err)
	}
	e.suspended.Store(false)
	e.log.Info("widebandt2: decode resumed (SDR returned by hunt)", "serial", e.serial)
}

// iqClipThreshold is the normalized magnitude at or above which an I or Q
// component counts as ADC rail-clipping; mirrors ccdecoder's value.
const iqClipThreshold = 0.98

// inputClipWarnRatio is the per-window clipped fraction above which the engine
// warns that the shared front end is overloaded (issue #749). Mirrors
// ccdecoder's iqClipWarnRatio.
const inputClipWarnRatio = 0.002

// foldInputClip counts rail-pinned samples in one wideband chunk.
func (e *Engine) foldInputClip(chunk []complex64) {
	for _, c := range chunk {
		r, i := real(c), imag(c)
		if r >= iqClipThreshold || r <= -iqClipThreshold || i >= iqClipThreshold || i <= -iqClipThreshold {
			e.wbClipped++
		}
	}
	e.wbClipSamples += len(chunk)
}

// channelProcessor is the per-channel dibit consumer. DMR Tier II's
// ConventionalChannel, DMR Tier III's ControlChannel, P25 Phase 1's
// ControlChannel and P25 Phase 2's ControlChannel all expose
// Process(dibits []uint8, baseIdx int) int with the same semantics, so
// the engine treats them uniformly. (Phase 2's ControlChannel also
// exposes IngestSuperframe; the build function wires the receiver's
// DibitSink through a SuperframeDecoder before calling Process so the
// engine still only sees a dibit-shaped processor.)
type channelProcessor interface {
	Process(dibits []uint8, baseIdx int) int
}

// narrowbandReceiver consumes the per-channel narrowband IQ stream.
// Every protocol's receiver (DMR, P25 Phase 1, P25 Phase 2) implements
// Process([]complex64) — the engine doesn't care which one it holds,
// only that the receiver's DibitSink is already wired to the channel's
// channelProcessor at construction.
type narrowbandReceiver interface {
	Process(iq []complex64)
}

type engineChannel struct {
	freqHz    uint32
	sysName   string
	protoTag  string // e.g. "dmr-tier2", "dmr-tier3", "p25-phase1", "p25-phase2"
	processor channelProcessor
	receiver  narrowbandReceiver
	// dmrTier3 is the typed Tier III control channel for "dmr-tier3"
	// channels (nil otherwise). Exposed via DMRTier3ControlChannel so the
	// DMR LCN autoconfig learner can hot-swap a learned band-plan resolver.
	dmrTier3 *tier3.ControlChannel

	// pwr accumulates this channel's narrowband IQ power across one
	// diagnostics window. Folded in the tap sink, drained by
	// maybeLogDiagnostics — both on the single pump goroutine.
	pwr iqpower.Accumulator
	// tier2Cnt is the typed Tier II channel whose decode-activity counters
	// feed the diagnostics summary (nil for non-dmr-tier2 channels).
	tier2Cnt *tier2.ConventionalChannel
	// lastCnt is the previous window's counter snapshot, so the summary
	// can report per-window deltas instead of lifetime totals.
	lastCnt tier2.Counters
	// pwLowLogAt throttles the "iq power very low" WARN for this channel.
	pwLowLogAt time.Time

	// strongNoSyncWindows counts consecutive diagnostics windows in which
	// this channel carried a strong signal yet produced zero sync/FEC — the
	// signature of an uncorrected tuner frequency offset (issue #836). It
	// resets the moment any sync is seen or the signal drops.
	strongNoSyncWindows int
	// noSyncLogAt throttles the "strong signal but no sync" hint WARN.
	noSyncLogAt time.Time

	// decoded returns this channel's lifetime decoded-frame count from the
	// underlying protocol state machine — Tier II FECPass, Tier III CSBKs,
	// Phase 1 TSBKs, Phase 2 MAC PDUs. maybeLogDiagnostics diffs it against
	// lastDecoded each window to gate the power-log event on real decode
	// activity (delta > 0). Set per protocol in buildChannel.
	decoded     func() uint64
	lastDecoded uint64
}

// New constructs an Engine. The device is not opened or streamed
// here; that happens in Run. Returns an error if construction
// inputs are invalid (offset out of band, colliding channelizer
// bins, etc.).
func New(opts Options) (*Engine, error) {
	if opts.Device == nil {
		return nil, errors.New("widebandt2: Device is required")
	}
	if opts.Bus == nil {
		return nil, errors.New("widebandt2: Bus is required")
	}
	if opts.SampleRateHz == 0 {
		return nil, errors.New("widebandt2: SampleRateHz is required")
	}
	if opts.CenterFreqHz == 0 {
		return nil, errors.New("widebandt2: CenterFreqHz is required")
	}
	if len(opts.Channels) == 0 {
		return nil, errors.New("widebandt2: at least one channel is required")
	}
	if len(opts.Systems) == 0 {
		return nil, errors.New("widebandt2: Systems table is required (used to resolve T2 vs T3 per channel)")
	}
	systemsByName := make(map[string]trunking.System, len(opts.Systems))
	for _, s := range opts.Systems {
		systemsByName[s.Name] = s
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	// Pre-compute every channel's offset from the dongle centre so the
	// strategy picker can see the actual plan (how close taps sit to bin
	// edges, how wide the set spans) and so the DDC fallback can size its
	// shared decimator to keep the widest tap in band.
	offsets := make([]float64, len(opts.Channels))
	var maxAbsOffset float64
	for i, ch := range opts.Channels {
		offsets[i] = float64(ch.FrequencyHz) - float64(opts.CenterFreqHz)
		if a := math.Abs(offsets[i]); a > maxAbsOffset {
			maxAbsOffset = a
		}
	}

	// Advise when the capture is materially oversampled for the plan. The DSP
	// and network cost scale with the capture rate, but the only rate the plan
	// needs is the one whose usable half-band (rate·(0.5−guard)) covers the
	// widest channel offset: minRate = maxAbsOffset/(0.5−guard). Running well
	// above that buys nothing and is a leading cause of the host failing to keep
	// up (overruns → dropped samples → glitchy audio). Reported config: 71 DMR
	// carriers spanning ±1.95 MHz captured at 10 MS/s, ~2.4× the ~4.3 MS/s the
	// span needs. maxAbsOffset==0 (a single centred channel) has no meaningful
	// minimum, so skip it.
	if maxAbsOffset > 0 {
		minRateHz := maxAbsOffset / (0.5 - guardFrac)
		if float64(opts.SampleRateHz) >= minRateHz*sampleRateAdvisoryFactor {
			log.Warn("widebandt2: capture is oversampled for the channel plan — the carriers span less than half the captured band, so a lower sdr.sample_rate would cut DSP + network load (and overrun pressure) for no loss of coverage",
				"serial", opts.Serial,
				"sample_rate_hz", opts.SampleRateHz,
				"channel_span_hz", uint64(2*maxAbsOffset),
				"min_sample_rate_hz", uint64(math.Ceil(minRateHz)),
			)
		}
	}

	strategy, tag := pickStrategy(opts.TunerStrategy, len(opts.Channels))
	var bank tuner.Bank
	switch strategy {
	case "ddc":
		// Span-aware: a 70-channel DMR plan spread across ±1.9 MHz of a
		// 10 MS/s capture exceeds the default ±1.1 MHz reduced-band floor,
		// which would reject the outer taps. Sizing the shared decimator to
		// maxAbsOffset keeps every tap in band (issue: 70-DMR stress test).
		bank = tuner.NewDDCBankForSpan(float64(opts.SampleRateHz), narrowbandRateHz, guardFrac, maxAbsOffset)
	case "polyphase":
		bank = tuner.NewChannelizerBank(float64(opts.SampleRateHz), narrowbandRateHz, guardFrac,
			channelizerBinsFor(opts.SampleRateHz), channelizerTapsPerBranch, channelizerKaiserBeta)
	default:
		return nil, fmt.Errorf("widebandt2: unknown tuner strategy %q", strategy)
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}
	engine := &Engine{
		log:         log,
		bus:         opts.Bus,
		device:      opts.Device,
		centerHz:    opts.CenterFreqHz,
		sampleRate:  opts.SampleRateHz,
		bank:        bank,
		strategyTag: tag,
		metrics:     opts.Metrics,
		serial:      opts.Serial,
		now:         now,
	}

	for i, ch := range opts.Channels {
		sys, ok := systemsByName[ch.SystemName]
		if !ok {
			return nil, fmt.Errorf("widebandt2: channel freq=%d references unknown system %q",
				ch.FrequencyHz, ch.SystemName)
		}
		offset := offsets[i]
		ec, err := buildChannel(sys, ch, bank.OutputRateHz(), opts.Bus, log, opts.Now)
		if err != nil {
			return nil, err
		}
		sink := func(ec *engineChannel) tuner.SinkFunc {
			return func(out []complex64) {
				if len(out) == 0 {
					return
				}
				ec.pwr.Add(out)
				ec.receiver.Process(out)
			}
		}(ec)
		if err := bank.AddTap(offset, sink); err != nil {
			return nil, fmt.Errorf("widebandt2: AddTap freq=%d offset=%.0f Hz: %w",
				ch.FrequencyHz, offset, err)
		}
		engine.channels = append(engine.channels, ec)
	}

	// Dense, irregular plans crowd some taps onto channelizer bin edges, where
	// the critically-sampled bin rolls off and those channels decode at reduced
	// SNR. The channelizer is still the right bank (a per-tap DDC at this tap
	// count is ~6x heavier and would not stay real-time), so this is a
	// heads-up, not a switch: a channel that won't lock may simply be sitting on
	// a bin edge — give it its own dongle, or thin the plan, if it matters.
	if cb, ok := bank.(*tuner.ChannelizerBank); ok {
		if frac := cb.MaxResidualFrac(); frac > channelizerCleanResidualFrac {
			log.Warn("widebandt2: dense plan crowds some taps onto channelizer bin edges — those channels decode at reduced SNR (inherent to packing this many carriers on one wideband tuner)",
				"serial", opts.Serial, "worst_residual_frac", fmt.Sprintf("%.2f", frac),
				"clean_threshold", channelizerCleanResidualFrac)
		}
		// Fan the per-tap fine-tune loop out across CPU cores. A dense plan's
		// 71 NCO+resampler+receiver chains are the bulk of the per-chunk cost
		// and are independent (each tap owns its state, reads the shared bins
		// read-only), so parallelizing them is what lets the host keep up with
		// the live pump at high sample rates instead of back-pressuring the SDR
		// into overruns. No-op below parallelTapThreshold taps. Run's defer
		// closes the pool on teardown.
		cb.EnableWorkers(tuner.DefaultTapWorkers())
	}
	return engine, nil
}

// buildChannel instantiates the right per-protocol receiver and state
// machine for a wideband channel. Trunked control-channel protocols
// (DMR Tier III, P25 Phase 1, P25 Phase 2) require the channel
// frequency to match one of the system's declared control_channels;
// DMR Tier II is conventional and accepts any per-repeater carrier.
//
// The returned engineChannel has its receiver's DibitSink already wired
// to the per-channel state machine — the caller just pumps IQ into
// receiver.Process and the bus picks up cc.locked / grant events as
// the protocol's framer locks on.
func buildChannel(sys trunking.System, ch ChannelConfig, outRateHz float64, bus *events.Bus, log *slog.Logger, now func() time.Time) (*engineChannel, error) {
	freqHz := ch.FrequencyHz
	switch sys.Protocol {
	case trunking.ProtocolDMR:
		if err := requireControlChannel(sys, freqHz, "dmr / Tier III"); err != nil {
			return nil, err
		}
		cc := tier3.New(tier3.Options{
			Bus:         bus,
			Log:         log.With("system", sys.Name, "freq_hz", freqHz, "tier", 3),
			SystemName:  sys.Name,
			FrequencyHz: freqHz,
			// The LCN→downlink resolver comes from the system's
			// dmr_band_plan. Without it every T3 voice grant drops with
			// decode.error stage=no-bandplan; the daemon already warns
			// at load time. See internal/radio/dmr/tier3.ResolverFromPlan.
			Resolver:         tier3.ResolverFromPlan(sys.DMRBandPlan),
			Now:              now,
			InterleavedVoice: sys.DMRInterleavedVoice,
		})
		rx := dmrrx.New(dmrrx.Options{
			SampleRateHz: outRateHz,
			DibitSink:    dmr.DibitSink(func(d []uint8, b int) { cc.Process(d, b) }),
			DeviationHz:  dmrDeviationHz,
			ClockGain:    dmrClockGainTier3,
		})
		return &engineChannel{freqHz: freqHz, sysName: sys.Name, protoTag: "dmr-tier3", processor: cc, receiver: rx, dmrTier3: cc,
			decoded: func() uint64 { return cc.DecodedFrames() }}, nil

	case trunking.ProtocolDMRTier2:
		cc := tier2.New(tier2.Options{
			Bus:         bus,
			Log:         log.With("system", sys.Name, "freq_hz", freqHz, "tier", 2),
			SystemName:  sys.Name,
			FrequencyHz: freqHz,
			Now:         now,
		})
		rx := dmrrx.New(dmrrx.Options{
			SampleRateHz: outRateHz,
			DibitSink:    dmr.DibitSink(func(d []uint8, b int) { cc.Process(d, b) }),
			DeviationHz:  dmrDeviationHz,
			ClockGain:    dmrClockGainTier2,
		})
		return &engineChannel{freqHz: freqHz, sysName: sys.Name, protoTag: "dmr-tier2", processor: cc, receiver: rx, tier2Cnt: cc,
			decoded: func() uint64 { return cc.Counters().FECPass }}, nil

	case trunking.ProtocolP25:
		if err := requireControlChannel(sys, freqHz, "p25 / Phase 1"); err != nil {
			return nil, err
		}
		demodMode, ok := p25phase1rx.ParseDemodMode(sys.P25Phase1DemodMode)
		if !ok {
			log.Warn("widebandt2: unrecognised p25_phase1_demod_mode; falling back to c4fm",
				"system", sys.Name, "value", sys.P25Phase1DemodMode)
		}
		// C4FM has only two physical rotations (identity, polarity
		// flip); the CQPSK / LSM path has the full four-fold QPSK
		// ambiguity. Mirrors the ccdecoder pipeline.
		rotations := p25phase1.RotationsAll
		if demodMode == p25phase1rx.DemodC4FM {
			rotations = p25phase1.RotationsC4FM
		}
		var bandPlan *p25phase1.BandPlan
		if len(sys.P25BandPlan) > 0 {
			bandPlan = &p25phase1.BandPlan{}
			for _, e := range sys.P25BandPlan {
				bandPlan.Apply(p25phase1.IdentifierUpdate{
					ChannelID:   e.ChannelID,
					BaseHz:      e.BaseHz,
					SpacingHz:   e.SpacingHz,
					TxOffsetHz:  e.TxOffsetHz,
					BandwidthHz: e.BandwidthHz,
				})
			}
		}
		// A Phase 1 CC can issue Phase 2 TDMA voice grants (hybrid
		// systems such as Victorian MMR — issue #882). The voice
		// composer's Phase 2 chain needs the per-system FEC config to
		// decode those grants' MAC PDUs; without it every grant carries
		// TrellisOff / ScramblerOff / zero seed and MAC decode fails.
		// Parse and forward the same knobs the ccdecoder pipeline does.
		p2Trellis, p2RS, p2Interleave, p2Scrambler := parseP25Phase2FECModes(sys, log)
		cc := p25phase1.New(p25phase1.Options{
			Bus:                 bus,
			Log:                 log.With("system", sys.Name, "freq_hz", freqHz, "phase", 1),
			SystemName:          sys.Name,
			FrequencyHz:         freqHz,
			BandPlan:            bandPlan,
			Rotations:           rotations,
			P25Phase2Trellis:    p2Trellis,
			P25Phase2RS:         p2RS,
			P25Phase2Interleave: p2Interleave,
			P25Phase2Scrambler:  p2Scrambler,
		})
		rx := p25phase1rx.New(p25phase1rx.Options{
			SampleRateHz: outRateHz,
			DeviationHz:  p25Phase1DeviationHz,
			DemodMode:    demodMode,
			DibitSink: p25phase1.DibitSink(func(d []uint8, b int) {
				cc.Process(d, b)
			}),
		})
		return &engineChannel{freqHz: freqHz, sysName: sys.Name, protoTag: "p25-phase1", processor: cc, receiver: rx,
			decoded: func() uint64 { return uint64(cc.Stats().TSBKDecoded) }}, nil

	case trunking.ProtocolP25Phase2:
		if err := requireControlChannel(sys, freqHz, "p25-phase2 / Phase 2"); err != nil {
			return nil, err
		}
		cc := p25phase2.New(p25phase2.Options{
			Bus:         bus,
			Log:         log.With("system", sys.Name, "freq_hz", freqHz, "phase", 2),
			SystemName:  sys.Name,
			FrequencyHz: freqHz,
		})
		applyP25Phase2Modes(cc, sys, log)
		clockMode, ok := p25phase2rx.ParseClockMode(sys.P25Phase2ClockMode)
		if !ok {
			log.Warn("widebandt2: unrecognised p25_phase2_clock_mode; falling back to gardner",
				"system", sys.Name, "value", sys.P25Phase2ClockMode)
		}
		sfDec := p25phase2.NewSuperframeDecoder()
		rx := p25phase2rx.New(p25phase2rx.Options{
			SampleRateHz: outRateHz,
			DibitSink: p25phase2.DibitSink(func(d []uint8, b int) {
				for _, sf := range sfDec.Process(d, b) {
					cc.IngestSuperframe(sf)
				}
			}),
			ClockMode:   clockMode,
			GardnerGain: p25Phase2GardnerGain,
		})
		return &engineChannel{freqHz: freqHz, sysName: sys.Name, protoTag: "p25-phase2", processor: cc, receiver: rx,
			decoded: func() uint64 { return cc.DecodedFrames() }}, nil

	default:
		return nil, fmt.Errorf(
			"widebandt2: system %q has protocol %q; wideband supports dmr-tier2, dmr, p25, and p25-phase2",
			sys.Name, sys.Protocol.String())
	}
}

// requireControlChannel rejects a wideband channel that doesn't sit on
// one of the system's declared control_channels. Used by every trunked
// protocol case in buildChannel — the protocol's state machine only
// makes sense on a CC frequency, and the config validator already
// enforces the same rule at load time.
func requireControlChannel(sys trunking.System, freqHz uint32, label string) error {
	for _, cc := range sys.ControlChannels {
		if cc == freqHz {
			return nil
		}
	}
	return fmt.Errorf("widebandt2: channel freq=%d on system %q (protocol %s) "+
		"must match one of the system's control_channels %v",
		freqHz, sys.Name, label, sys.ControlChannels)
}

// parseP25Phase2FECModes parses the per-system Phase 2 FEC knobs
// (trellis / RS / interleave / scrambler) into the uint8 codes the
// p25phase1.Options block carries, mirroring ccdecoder's
// newP25Phase1Pipeline. A Phase 1 CC on a hybrid system uses these to
// stamp its Phase 2 TDMA voice grants so the voice composer can decode
// their MAC PDUs (issue #882). Unrecognised values warn then fall back
// to the same defaults the ccdecoder path uses; an empty per-system
// value keeps the default (Trellis=on, everything else=off).
func parseP25Phase2FECModes(sys trunking.System, log *slog.Logger) (trellis, rs, interleave, scrambler uint8) {
	p2Trellis, ok := p25phase2.ParseTrellisMode(sys.P25Phase2TrellisMode)
	if !ok {
		log.Warn("widebandt2: unrecognised p25_phase2_trellis_mode; falling back to on",
			"system", sys.Name, "value", sys.P25Phase2TrellisMode)
	}
	p2RS, ok := p25phase2.ParseRSMode(sys.P25Phase2RSMode)
	if !ok {
		log.Warn("widebandt2: unrecognised p25_phase2_rs_mode; falling back to off",
			"system", sys.Name, "value", sys.P25Phase2RSMode)
	}
	p2Interleave, ok := p25phase2.ParseInterleaveMode(sys.P25Phase2InterleaveMode)
	if !ok {
		log.Warn("widebandt2: unrecognised p25_phase2_interleave_mode; falling back to off",
			"system", sys.Name, "value", sys.P25Phase2InterleaveMode)
	}
	p2Scrambler, ok := p25phase2.ParseScramblerMode(sys.P25Phase2ScramblerMode)
	if !ok {
		log.Warn("widebandt2: unrecognised p25_phase2_scrambler_mode; falling back to on",
			"system", sys.Name, "value", sys.P25Phase2ScramblerMode)
	}
	return uint8(p2Trellis), uint8(p2RS), uint8(p2Interleave), uint8(p2Scrambler)
}

// applyP25Phase2Modes mirrors newP25Phase2Pipeline's per-system mode
// wiring (trellis / RS / interleave / scrambler) so a wideband Phase 2
// CC tap decodes traffic identically to the dedicated ccdecoder path.
func applyP25Phase2Modes(cc *p25phase2.ControlChannel, sys trunking.System, log *slog.Logger) {
	trellisMode, ok := p25phase2.ParseTrellisMode(sys.P25Phase2TrellisMode)
	if !ok {
		log.Warn("widebandt2: unrecognised p25_phase2_trellis_mode; falling back to on",
			"system", sys.Name, "value", sys.P25Phase2TrellisMode)
	}
	cc.SetTrellisMode(trellisMode)
	rsMode, rsOK := p25phase2.ParseRSMode(sys.P25Phase2RSMode)
	if !rsOK {
		log.Warn("widebandt2: unrecognised p25_phase2_rs_mode; falling back to off",
			"system", sys.Name, "value", sys.P25Phase2RSMode)
	}
	cc.SetRSMode(rsMode)
	interleaveMode, ilOK := p25phase2.ParseInterleaveMode(sys.P25Phase2InterleaveMode)
	if !ilOK {
		log.Warn("widebandt2: unrecognised p25_phase2_interleave_mode; falling back to off",
			"system", sys.Name, "value", sys.P25Phase2InterleaveMode)
	}
	cc.SetInterleaveMode(interleaveMode)
	scramblerMode, scrOK := p25phase2.ParseScramblerMode(sys.P25Phase2ScramblerMode)
	if !scrOK {
		log.Warn("widebandt2: unrecognised p25_phase2_scrambler_mode; falling back to on",
			"system", sys.Name, "value", sys.P25Phase2ScramblerMode)
	}
	if scramblerMode == p25phase2.ScramblerProbe && rsMode != p25phase2.RSOn {
		log.Warn("widebandt2: p25_phase2_scrambler_mode=probe requires p25_phase2_rs_mode=on; descrambler will degrade to offset 0",
			"system", sys.Name)
	}
	cc.SetScramblerMode(scramblerMode)
	cc.SetScramblerSeed(framing.PN44SeedFromIdentity(
		sys.WACN, sys.SystemID, uint16(sys.Site),
	))
}

// Channels returns the per-channel frequencies the engine is
// monitoring. Used by callers (the daemon, tests) for logging and
// status snapshots.
func (e *Engine) Channels() []uint32 {
	out := make([]uint32, 0, len(e.channels))
	for _, c := range e.channels {
		out = append(out, c.freqHz)
	}
	return out
}

// ChannelProtocolTags returns a frequency → protocol-tag map for the
// channels this engine drives ("dmr-tier2", "dmr-tier3", "p25-phase1",
// or "p25-phase2"). Used by the daemon's startup log and by tests to
// verify the dispatcher picked the right state machine per channel.
func (e *Engine) ChannelProtocolTags() map[uint32]string {
	out := make(map[uint32]string, len(e.channels))
	for _, c := range e.channels {
		out[c.freqHz] = c.protoTag
	}
	return out
}

// Strategy reports which tuner strategy the engine resolved at
// construction ("ddc" or "polyphase"). Used for diagnostics /
// startup logging.
func (e *Engine) Strategy() string { return e.strategyTag }

// DMRTier3ControlChannel returns the Tier III control channel monitoring
// the given frequency, or nil if no such "dmr-tier3" channel exists. The
// daemon uses it to bind a dmrlcn.Learner's resolver hot-swap to the
// running control channel.
func (e *Engine) DMRTier3ControlChannel(freqHz uint32) *tier3.ControlChannel {
	for _, c := range e.channels {
		if c.freqHz == freqHz && c.dmrTier3 != nil {
			return c.dmrTier3
		}
	}
	return nil
}

// Run programs the dongle's centre frequency, opens its IQ stream,
// and pumps chunks through the tuner bank until ctx cancels. Within a
// chunk the per-tap fine-tune chains may run concurrently across worker
// goroutines (see ChannelizerBank.EnableWorkers), but bank.Process
// barriers before returning, so successive chunks — and the post-Process
// diagnostics below — never overlap a running tap and each receiver still
// sees its chunks in order.
// ErrIQStreamClosed is returned by Run when the SDR's IQ channel closed
// unexpectedly — the USB reaper died (a physical disconnect, or the macOS
// stall watchdog aborting a silently-frozen Airspy endpoint; see
// internal/sdr/rtlsdr/usb.ErrStreamStalled) — rather than the engine being
// asked to stop via ctx cancellation. The daemon's runWidebandWithRetry
// supervisor treats it as a signal to reacquire the dongle and restart,
// mirroring the control-channel decoder's retry loop (issue #345). A clean
// ctx-cancel shutdown still returns nil.
var ErrIQStreamClosed = errors.New("widebandt2: IQ stream closed unexpectedly")

// streamDeadCauser is the optional extension a driver (airspy, rtlsdr-purego,
// or the iqtap broker that wraps them) implements to report the concrete USB
// error that killed its last bulk-IN stream.
type streamDeadCauser interface{ StreamDeadCause() error }

// streamClosedErr returns ErrIQStreamClosed, wrapping the device's concrete
// stream-death cause when it exposes one, so the daemon's "IQ stream died" log
// names e.g. "usb: bulk-IN stream stalled ..." instead of a bare message. The
// result still satisfies errors.Is(err, ErrIQStreamClosed), so the retry loop's
// classification is unchanged.
func streamClosedErr(dev sdr.Device) error {
	if c, ok := dev.(streamDeadCauser); ok {
		if cause := c.StreamDeadCause(); cause != nil {
			return fmt.Errorf("%w: %w", ErrIQStreamClosed, cause)
		}
	}
	return ErrIQStreamClosed
}

func (e *Engine) Run(ctx context.Context) error {
	if err := e.device.SetCenterFreq(e.centerHz); err != nil {
		return fmt.Errorf("widebandt2: SetCenterFreq %d Hz: %w", e.centerHz, err)
	}
	e.log.Info("widebandt2: starting",
		"center_freq_hz", e.centerHz,
		"sample_rate_hz", e.sampleRate,
		"channels", len(e.channels),
		"strategy", e.strategyTag,
	)
	stream, err := e.device.StreamIQ(ctx)
	if err != nil {
		return fmt.Errorf("widebandt2: StreamIQ: %w", err)
	}
	// Drop this dongle's whole-capture series on teardown so stale values
	// don't linger in Prometheus after the engine stops (issue #749).
	if e.metrics != nil {
		defer e.metrics.ClearWidebandInput(e.serial)
	}
	// Stop the bank's tap worker pool (if any) on teardown so its goroutines
	// don't outlive the engine when an SDR is retuned/recreated.
	if c, ok := e.bank.(interface{ Close() }); ok {
		defer c.Close()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, ok := <-stream:
			if !ok {
				// The driver closed the IQ channel. A concurrent
				// ctx-cancel means a clean shutdown; otherwise the
				// stream died under us (USB reaper death or the stall
				// watchdog aborting a frozen endpoint) and the daemon
				// supervisor should reacquire the dongle and restart.
				if ctx.Err() != nil {
					return nil
				}
				return streamClosedErr(e.device)
			}
			if e.suspended.Load() {
				// SDR borrowed by a hunt and retuned: drop the off-frequency
				// chunk (keeping the stream drained) without decoding it.
				continue
			}
			e.widebandPwr.Add(chunk)
			e.foldInputClip(chunk)
			e.bank.Process(chunk)
			e.maybeLogDiagnostics(e.now())
		}
	}
}

// lowPowerWarnInterval throttles the per-channel "iq power very low"
// WARN so a genuinely dead stream logs once every few seconds instead
// of every window. Matches the ccdecoder cadence (decoder.go).
const lowPowerWarnInterval = 5 * time.Second

// noSyncHintDbFS is the per-channel IQ power (dBFS) at or above which a
// channel is treated as carrying a real signal for the "strong signal but
// no sync" diagnostic. Sits ~10 dB above the low-power floor
// (iqpower.LowPowerThresholdDbFS = -55) so ordinary noise never trips it,
// while a keyed handheld — the issue #836 capture peaked near -16 dBFS —
// clears it comfortably.
const noSyncHintDbFS = -45.0

// strongNoSyncWindowsNeeded is how many consecutive diagnostics windows
// (~1 s each) a channel must show strong power with zero sync AND zero FEC
// before the "strong signal but no sync" hint fires. Requiring a sustained
// run keeps a brief adjacent-carrier splatter or a one-window glitch from
// tripping it; a real mistuned transmission holds for seconds.
const strongNoSyncWindowsNeeded = 3

// maybeLogDiagnostics flushes per-channel signal-power and decode-activity
// diagnostics once per iqpower.Window. It runs inline on the Run pump
// goroutine — the same goroutine that fed the tap sinks this window — so
// it reads the accumulators and Tier II counters without a lock and in
// chunk order. Cost is a wall-clock compare per chunk until the window
// elapses; the DEBUG activity summary is gated by slog.Enabled so it
// allocates nothing when DEBUG is off. The low-power WARN fires regardless
// of log level — that operator feedback is the whole point (a single
// wideband DMR channel that silently decodes nothing, issue: field report
// VE5XAM).
func (e *Engine) maybeLogDiagnostics(now time.Time) {
	if e.lastDiagAt.IsZero() {
		e.lastDiagAt = now
		return
	}
	if now.Sub(e.lastDiagAt) < iqpower.Window {
		return
	}
	e.lastDiagAt = now

	// Drain the wideband input power once per window. When a per-channel WARN
	// fires, a healthy wideband input localises the fault for the operator: the
	// front end is fine, so the dead channel is outside the captured passband
	// (CenterFreqHz ± SampleRateHz/2) or tuned to the wrong frequency — not an
	// antenna/gain/cable problem.
	wbDbFS, wbSamples := e.widebandPwr.MeanDbFSAndReset()
	wbHealthy := wbSamples > 0 && wbDbFS >= iqpower.LowPowerThresholdDbFS

	// Whole-capture clip ratio over the same window (issue #749). A non-zero
	// value means a strong site is pinning the shared ADC rail — front-end
	// overload, which raises the noise floor and buries the weaker taps. This
	// is the "only one site decodes" failure mode the per-tap power gauge
	// can't surface on its own.
	var wbClip float64
	if e.wbClipSamples > 0 {
		wbClip = float64(e.wbClipped) / float64(e.wbClipSamples)
	}
	wbOverloaded := wbClip > inputClipWarnRatio
	e.wbClipped, e.wbClipSamples = 0, 0
	if e.metrics != nil && wbSamples > 0 {
		e.metrics.RecordWidebandInputPowerDbFS(e.serial, wbDbFS)
		e.metrics.RecordWidebandInputClipRatio(e.serial, wbClip)
	}
	if wbOverloaded && now.Sub(e.wbClipLogAt) >= lowPowerWarnInterval {
		// The fix is less RF, not more: reduce gain or add attenuation.
		e.log.Warn("widebandt2: wideband front end overloaded — IQ pinned to the ADC rail; a strong site is clipping the shared ADC and burying weaker taps. Reduce gain or add attenuation (do NOT raise gain). issue #749",
			"serial", e.serial, "clip_ratio", wbClip, "wideband_input_dbfs", wbDbFS)
		e.wbClipLogAt = now
	}

	debug := e.log.Enabled(context.Background(), slog.LevelDebug)
	for _, ec := range e.channels {
		if ec.pwr.Samples() == 0 {
			continue
		}
		dbfs, _ := ec.pwr.MeanDbFSAndReset()
		if e.metrics != nil {
			e.metrics.RecordIQPowerDbFS(ec.powerLabel(), dbfs)
		}

		// Decode-activity-gated power event. When the channel decoded at
		// least one frame this window, publish a per-window signal-level
		// snapshot for the optional power log. Gating on the decode delta
		// keeps idle / off-band channels out of the log entirely; the
		// LowPower flag lets the subscriber default to "decoding but weak"
		// windows only. Publish is non-blocking (bus drops to slow subs),
		// so this never stalls the pump goroutine.
		if ec.decoded != nil {
			total := ec.decoded()
			if delta := total - ec.lastDecoded; delta > 0 {
				ec.lastDecoded = total
				e.bus.Publish(events.Event{
					Kind:      events.KindChannelPower,
					Timestamp: now,
					Payload: events.ChannelPower{
						System:    ec.sysName,
						Protocol:  ec.protoTag,
						FreqHz:    ec.freqHz,
						PowerDbFS: dbfs,
						Decoded:   delta,
						LowPower:  dbfs < iqpower.LowPowerThresholdDbFS,
						At:        now,
					},
				})
			}
		}

		if dbfs < iqpower.LowPowerThresholdDbFS {
			if now.Sub(ec.pwLowLogAt) >= lowPowerWarnInterval {
				hint := "check antenna, gain, USB cable"
				switch {
				case wbOverloaded:
					// All taps share this front end, so don't blame the cable:
					// a stronger site is clipping the shared ADC and burying
					// this one (issue #749). Reduce gain, don't raise it.
					hint = "wideband front end is overloaded (clipping) — a stronger co-channel is burying this site; reduce gain or add attenuation, do NOT raise gain"
				case wbHealthy:
					// The shared capture is healthy and other taps may be
					// decoding, yet this one sits at the floor. Most often the
					// carrier is off-passband / mistuned; but on a multi-tap
					// device a single fixed gain can also under-amplify a weaker
					// co-tenant site below the ADC floor — try 'gain: auto' so
					// the front end runs hot enough for it to clear (issue #749).
					hint = "wideband input is healthy — this carrier is likely outside the captured passband or tuned to the wrong frequency; if other taps on this dongle decode, try 'gain: auto' (AGC) so a weaker co-tenant site can clear the shared ADC floor"
				}
				e.log.Warn("widebandt2: channel iq power very low — "+hint,
					"freq_hz", ec.freqHz, "system", ec.sysName, "proto", ec.protoTag,
					"dbfs", dbfs, "wideband_input_dbfs", wbDbFS)
				ec.pwLowLogAt = now
			}
		} else if debug {
			e.log.Debug("widebandt2: channel iq power",
				"freq_hz", ec.freqHz, "system", ec.sysName, "proto", ec.protoTag, "dbfs", dbfs)
		}

		// Per-window decode-activity deltas (Tier II conventional / Tier I).
		// Lets an operator tell no-signal / signal-but-no-sync /
		// sync-but-FEC-fail / decoding apart at a glance, and drives the
		// "strong signal but no sync" tuner-offset hint below. The deltas are
		// computed whenever the typed counters exist (not just under DEBUG) so
		// the hint works at the default log level.
		if ec.tier2Cnt != nil {
			c := ec.tier2Cnt.Counters()
			syncDelta := c.SyncHits - ec.lastCnt.SyncHits
			fecPassDelta := c.FECPass - ec.lastCnt.FECPass
			if debug {
				e.log.Debug("widebandt2: channel decode activity",
					"freq_hz", ec.freqHz, "system", ec.sysName,
					"sync_hits", syncDelta,
					"bursts", c.Bursts-ec.lastCnt.Bursts,
					"fec_pass", fecPassDelta,
					"fec_fail", c.FECFail-ec.lastCnt.FECFail,
					"locks_total", c.Locks)
			}
			ec.lastCnt = c

			// Strong-signal-but-no-sync hint (issue #836). A real transmission
			// that never produces a single burst-sync match is the classic
			// signature of an uncorrected tuner frequency error: the narrowband
			// 4-level demod tolerates only a tiny carrier offset (~±75 Hz),
			// while SDR++ still yields FM audio because audio is offset-immune.
			// Fire only after a sustained run so a brief splatter doesn't trip
			// it, and reset the instant any sync appears or the signal drops.
			if dbfs >= noSyncHintDbFS && syncDelta == 0 && fecPassDelta == 0 {
				ec.strongNoSyncWindows++
				if ec.strongNoSyncWindows >= strongNoSyncWindowsNeeded &&
					now.Sub(ec.noSyncLogAt) >= lowPowerWarnInterval {
					e.log.Warn("widebandt2: strong in-channel signal but no sync — the classic symptom of an uncorrected tuner frequency error on a narrowband carrier. Measure and set sdr.ppm for this dongle; even ~100 Hz of offset stops a DMR/P25/NXDN decode while SDR++ still produces FM audio. Also verify the carrier frequency and that this protocol/mode is supported. issue #836",
						"freq_hz", ec.freqHz, "system", ec.sysName, "proto", ec.protoTag,
						"dbfs", dbfs)
					ec.noSyncLogAt = now
				}
			} else {
				ec.strongNoSyncWindows = 0
			}
		}
	}
}

// powerLabel is the metrics label for this channel's dBFS gauge:
// "<system> @ <freqMHz>" so multiple channels sharing one system name
// don't collide on the system-keyed gauge.
func (ec *engineChannel) powerLabel() string {
	return fmt.Sprintf("%s @ %.4f MHz", ec.sysName, float64(ec.freqHz)/1e6)
}

// pickStrategy resolves the user-facing strategy choice into the internal
// bank kind. "" / "auto" auto-selects by channel count: a small fleet favours
// DDCBank (linear, no bin-alignment constraint); a larger fleet favours the
// shared polyphase channelizer, whose amortised wide-band filter is the only
// thing that stays real-time at high tap counts. A dense 71-DMR plan benches
// ~6x cheaper on the channelizer than on a per-tap DDC (one shared FFT vs 71
// reduced-rate resamplers — BenchmarkDense71* in internal/dsp/tuner), so auto
// keeps high counts on the channelizer even
// when the plan crowds taps onto bin edges — New warns about the resulting
// edge roll-off rather than trading real-time headroom for it. Explicit
// "ddc"/"polyphase" are honoured verbatim.
func pickStrategy(requested string, channelCount int) (kind, tag string) {
	switch requested {
	case "ddc":
		return "ddc", "ddc"
	case "polyphase":
		return "polyphase", "polyphase"
	case "", "auto":
		if channelCount <= strategyAutoThreshold {
			return "ddc", "auto(ddc)"
		}
		return "polyphase", "auto(polyphase)"
	default:
		return requested, requested
	}
}
