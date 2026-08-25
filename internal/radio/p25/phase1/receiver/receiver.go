// Package receiver wires the IQ → dibit chain that feeds either the
// P25 Phase 1 LDU assembler (voice path) or the control-channel state
// machine (CC path) — or both at once. It composes primitives that
// already live in internal/dsp + internal/radio/p25/phase1.
//
// Two demod paths are selectable via Options.DemodMode (see modes.go):
//
//	DemodC4FM (default):
//	  IQ
//	    → FM discriminator (internal/dsp/demod.FM)
//	    → spec P25 C4FM matched filter (internal/dsp/demod.C4FM
//	      with demod.P25C4FMRxTaps — raised-cosine, not RRC)
//	    → coarse AFC: residual carrier-offset removal
//	      (internal/dsp/demod.CoarseAFC)
//	    → Mueller-Müller symbol clock recovery (sync.MuellerMuller)
//	    → 4-level slicer → C4FM symbol → 0..3 dibit
//	      (internal/dsp/demod.C4FM, phase1.SymbolToDibit)
//
//	DemodCQPSK (LSM / simulcast):
//	  IQ
//	    → complex RRC matched filter (demod.PiOver4DQPSK, rotation=π/4)
//	    → AGC: amplitude normalisation (internal/dsp.AGC)
//	    → Gardner timing recovery on complex IQ (sync.Gardner)
//	    → CMA blind equalizer: simulcast-multipath ISI removal
//	      (internal/dsp/equalizer.CMA)
//	    → differential QPSK quadrant decode
//	    → LSM dibit remap → canonical 0..3 dibit
//
// In both cases the dibit stream the receiver emits matches the
// TIA-102.BAAA convention, so downstream code (FSW detect, NID parse,
// TSBK trellis) is demod-agnostic. CQPSK demod is the path operators
// on simulcast P25 sites need — issue #275 surfaced because the
// FM-discriminator path produces near-random dibits on an LSM signal.
//
// Either sink may be nil; at least one must be set. The receiver is
// stateful and not safe for concurrent Process calls. Instantiate one
// per tuned frequency / per call chain. All primitives it composes
// own their own internal history, so chunk boundaries do not corrupt
// the stream.
package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// defaultGardnerGain is the Gardner step the CQPSK path uses when the
// caller leaves Options.GardnerGain at zero.
//
// The Gardner loop applies its correction as a phase step in *samples*
// (mu += sps + gain·err), so the effective per-symbol loop gain is
// gain/sps and the right value depends on the samples-per-symbol the path
// runs at. The original 0.03 here was the loop's generic default; at this
// path's 48 kHz channel rate (10 sps) that left the loop ~5× over-gained,
// so it overshot the timing null instead of settling into it and only
// "locked" when the input happened to already be symbol-aligned (sample
// phase 0). Synthetic fixtures start aligned and decoded; real captures,
// whose symbol clock has no relationship to sample 0, almost never
// acquired the control channel (issue #492).
//
// 0.005 is the same step the sibling π/4-DQPSK Gardner paths already run
// in production — the P25 Phase 2 and TETRA control-channel pipelines
// both tuned down to it after hitting this exact over-correction ("the
// standard gain over-corrects on clean signals and slips", see
// internal/scanner/ccdecoder/pipelines.go). The Phase 1 CQPSK path was
// the one π/4-DQPSK consumer left on the generic default. At 10 sps this
// puts the per-symbol gain back in the loop's stable pull-in range, so it
// acquires from any sub-symbol phase.
const defaultGardnerGain = 0.005

// maxAFCOffsetHz caps the DDA's integrator. Sized at ~25 kHz so a
// 420 MHz / 50 ppm RTL-SDR (~21 kHz worst case) clears it
// comfortably; the clamp engages only on adversarial input or a
// runaway loop. Issue #402.
const maxAFCOffsetHz = 25_000.0

// ddaHandoffSymbols is the number of accepted DDA updates the
// receiver waits for before freezing CoarseAFC and routing the
// estimate through the DDA alone. ~256 symbols (~53 ms at 4800 baud)
// is enough for the slicer to stabilise once the CC has locked and
// the AGC has seeded, without giving CoarseAFC time to wander far
// onto a sustained data-mean during the bootstrap window. Issue
// #402.
const ddaHandoffSymbols = 256

// ddaWarmupSymbols is the number of symbols CoarseAFC is allowed to
// converge for before the DDA starts integrating decisions. Sized at
// 8× the CoarseAFC time constant (8 × 64 = 512 symbols, ~107 ms at
// 4800 baud) — well past CoarseAFC's settling. Without this gate the
// DDA learns from samples that are still biased by an un-converged
// CoarseAFC: at carrier offsets ≥ ~1 kHz the slicer mis-decides
// inner symbols as outer (the bias pushes a +0.0785 inner past the
// 0.157 outer threshold), the wrong-decision residual lands inside
// the DDA's gate, and the contaminated estimate breaks lock once it
// folds into the post-handoff loop. Issue #402.
//
// Once warmup completes the receiver freezes CoarseAFC (subtract-only)
// for the whole learning window so the eye is stationary while the DDA
// integrates: the handoff then folds in exactly the CoarseAFC value the
// DDA learned against, with no "instantaneous − average" error. The
// earlier code kept CoarseAFC adapting through the learning window, so
// the fold-in captured a wandering data-mean value and the DDA settled
// onto a stable-but-wrong offset that broke lock (issue #402 regression
// after the first DDA cut).
const ddaWarmupSymbols = 512

// Handoff is committed only when the eye also looks genuinely open, not
// just when enough within-gate updates have accrued: a uniformly-biased
// eye still produces within-gate ("accepted") residuals, so the count
// alone can't tell a real lock from a biased false lock. Issue #402.
const (
	// ddaHandoffMinAcceptRate is the minimum fraction of learning-window
	// symbols that must land inside the DDA gate before handoff. Catches
	// a grossly-broken eye (many out-of-gate decisions); the residual-
	// mean test catches the subtler within-gate bias.
	ddaHandoffMinAcceptRate = 0.66

	// ddaMaxDriftHz bounds how far the DDA's estimate may wander from the
	// value it carried at handoff before the receiver reverts to
	// CoarseAFC-alone. A locked transmitter's residual drift is a tuner-
	// ppm affair (tens to low hundreds of Hz); a DDA that walks kilohertz
	// away from a gate-verified handoff is tracking something it
	// shouldn't, so falling back to the open-loop tracker can only help.
	// This is the reversibility guarantee: the post-handoff error is
	// bounded by the handoff gate plus this drift, so the DDA can never
	// strand the receiver below the pre-DDA behaviour. Issue #402.
	ddaMaxDriftHz = 4000.0
)

// P25 Phase 1 on-air parameters.
const (
	// SymbolRate is the channel symbol rate. Each symbol is one
	// dibit (2 bits) on the wire.
	SymbolRate = 4800.0
	// RolloffAlpha is the recommended RRC roll-off for P25 Phase 1
	// (TIA-102.BAEA). 0.2 lines up with both the transmit pulse-
	// shaping and the receiver matched filter.
	RolloffAlpha = 0.2
	// PulseSpanSymbols is the half-span of the RRC pulse on each
	// side of the symbol time. 8 symbols (4+4) is the standard
	// receiver-side compromise between truncation noise and CPU
	// cost; reduce to 6 for low-power targets at the price of ~1 dB
	// SNR penalty in heavy multipath.
	PulseSpanSymbols = 8
)

// Options configures a Receiver. Zero-valued fields fall back to
// sensible P25 Phase 1 defaults so the typical caller can write
//
//	r := receiver.New(receiver.Options{
//	    SampleRateHz: 48_000,
//	    Sink: func(ldu []byte) { ... },
//	})
//
// At least one of Sink (voice / LDU path) and DibitSink (control-
// channel path) must be set; both may be set to drive both paths
// from one IQ chain.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization (e.g. the polyphase channelizer's per-channel
	// output rate). Required; must be ≥ 2 * SymbolRate.
	SampleRateHz float64
	// Sink receives complete 1728-bit LDU buffers ready for
	// phase1.ExtractVoiceFrames. Optional when DibitSink is set.
	Sink phase1.LDUSink
	// DibitSink receives the raw dibit stream before LDU framing.
	// Wire it into phase1.ControlChannel.Process to drive the CC
	// state machine. Optional when Sink is set.
	DibitSink phase1.DibitSink
	// Tolerance is forwarded to the LDU assembler — the maximum
	// dibit-position mismatch allowed when matching the 24-dibit
	// FrameSyncWord. <0 uses the assembler's default of 4.
	Tolerance int
	// PulseSpanSymbols overrides the RRC half-span. <=0 uses
	// PulseSpanSymbols.
	PulseSpanSymbols int
	// Alpha overrides the RRC roll-off. <=0 uses RolloffAlpha.
	Alpha float64
	// ClockGain is the Mueller-Müller loop gain. <=0 uses 0.05,
	// which is appropriate for clean signals; raise for noisy /
	// drifting transmitters.
	ClockGain float64
	// DeviationHz is the peak frequency deviation of the C4FM
	// signal at symbol ±3 (1800 Hz on P25 Phase 1 per
	// TIA-102.BAAA). Used to calibrate the slicer thresholds
	// against the FM-discriminator output level (which lives in
	// rad/sample, so the slicer scale is 2π · DeviationHz /
	// SampleRateHz at symbol ±3). <=0 falls back to slicer
	// thresholds tuned for the legacy "FM-output-already-
	// normalised-to-±1" assumption, which only fires for
	// synthesized fixtures that pre-scale their signal levels.
	DeviationHz float64
	// DemodMode selects the symbol recovery path. Zero value is
	// DemodC4FM (the legacy FM-discriminator → Mueller-Müller
	// path). DemodCQPSK routes IQ through the LSM / linear-CQPSK
	// chain (complex RRC → Gardner → π/4-DQPSK + LSM dibit remap)
	// — for P25 sites that transmit a linear/LSM waveform rather
	// than C4FM (not implied by simulcast; see #935). See modes.go.
	DemodMode DemodMode
	// GardnerGain overrides the Gardner loop step used by the
	// CQPSK path. <=0 uses defaultGardnerGain. Ignored when
	// DemodMode == DemodC4FM (the C4FM path uses Mueller-Müller).
	GardnerGain float64
	// EnableDecisionDirectedAFC opts the C4FM path into the
	// decision-directed AFC (DDA) layered on top of CoarseAFC.
	// Default false: the receiver runs CoarseAFC-alone, the
	// behaviour that predates the DDA. The DDA is currently
	// experimental and OFF by default because it can stably
	// false-lock — on the issue #402 capture it held a wrong offset
	// and broke control-channel lock that CoarseAFC-alone recovered.
	// The receiver and the control channel are feed-forward siblings
	// with no FSW / CC-lock feedback into the demod, so nothing
	// internal catches a biased-but-self-consistent eye, and the DDA
	// commits a handoff to it. Leave this off until the eye-skew root
	// cause is pinned and the handoff is gated on a real lock signal.
	// Ignored when DemodMode == DemodCQPSK (no AFC stage). Issue #402.
	EnableDecisionDirectedAFC bool
	// EnableAdaptiveC4FMSlicer opts the C4FM path into the adaptive
	// 4-level slicer (issue #402) that tracks the observed per-symbol
	// eye instead of slicing at fixed nominal thresholds. Default false:
	// the receiver uses the fixed-threshold slicer. The adaptive slicer
	// is OFF by default because on the issue #402 capture (a closed,
	// asymmetric +3 eye whose outer population is spread low) every
	// adaptive threshold that rose above the fixed nominal decoded worse,
	// and the fixed slicer was the best performer (the asymmetry is an
	// RF-domain effect — see the I/Q-imbalance investigation — not
	// something the slicer can fix). Kept available behind this flag for
	// sites with a genuinely DC-shifted or compressed (not stretched) eye
	// and for A/B experimentation. Ignored when DeviationHz <= 0 or when
	// DemodMode == DemodCQPSK.
	EnableAdaptiveC4FMSlicer bool
	// EnableDCBlock inserts a first-order complex DC-removal high-pass at
	// the very top of Process, ahead of the FM discriminator, to strip the
	// static zero-IF DC spur that sits on an on-channel-tuned voice signal
	// (the HackRF's centre spike is large enough to collapse P25 voice
	// decode to zero — see dcblock.go). Off by default and intended for the
	// VOICE receivers only; never the control-channel DDC path. Safe for
	// C4FM: the ~1 Hz corner is decades below the ±600/±1800 Hz C4FM tones.
	EnableDCBlock bool
	// SoftSink, when non-nil, receives the per-symbol soft samples
	// produced by the matched filter + symbol-clock recovery, just
	// before slicing. The C4FM path emits the FM-discriminator +
	// matched-filtered + MM-clock-recovered samples (rad/sample,
	// typically ±2π·DeviationHz/SampleRateHz at the outer-symbol
	// centres). The CQPSK path emits the real part of the rotated
	// DQPSK quadrant decision per symbol. Nil by default — meant for
	// offline diagnostics (issue #275: surfacing the matched-filter
	// output level distribution when the slicer collapses to outer
	// symbols only).
	SoftSink func(softSamples []float32)
	// SymbolSink, when non-nil, receives the per-symbol *complex*
	// constellation points sampled at the symbol-decision instants — the
	// true symbol-domain constellation (post matched-filter, timing
	// recovery and carrier recovery, pre differential-decode). Only the
	// CQPSK / linear path produces these: c.symbols, the post-Costas
	// π/4-DQPSK points, which on a clean signal cluster at the four
	// ±45°/±135° constellation positions and degrade to a smeared X as
	// the eye closes. The C4FM path leaves it uncalled — its symbol
	// domain is the real 4-level soft waveform, surfaced via SoftSink
	// (a complex constellation there is just the 4 levels on the real
	// axis). When called it is aligned index-for-index with the dibit
	// batch fired on the same Process call. Nil by default — meant for
	// the web "Constellation" scope's symbol-domain view.
	SymbolSink func(symbols []complex64)
	// EyeSink, when non-nil, receives the *oversampled* matched-filter
	// output (sps samples per symbol) on the C4FM path, scaled to the
	// same soft units as SoftSink, together with the integer samples-
	// per-symbol. Folding this stream over the symbol period yields the
	// 4-level eye diagram (OP25's "datascope") — the open-eye view that
	// reveals symbol-timing and SNR health a 1-sample-per-symbol soft
	// track cannot. Only the C4FM path produces it; the CQPSK path leaves
	// it uncalled (its quality view is the complex constellation). Nil by
	// default — meant for the web eye-diagram scope. The slice is reused
	// across calls; the callee must copy what it needs synchronously.
	EyeSink func(oversampled []float32, sps int)
	// BitLLRSink, when non-nil, receives per-bit log-likelihood ratios for
	// each dibit batch on the C4FM path (p25_phase1_soft_decision): TWO
	// values per dibit — llrs[2i] for dibits[i]'s MSB (the 4-level soft
	// symbol's signed distance from the sign axis), llrs[2i+1] for its LSB
	// (its distance from the inner/outer magnitude threshold) — in the
	// framing convention (LLR > 0 ⇒ bit 0, magnitude = reliability). Fired
	// immediately BEFORE the DibitSink batch it aligns with, so a consumer
	// can stash the LLRs and pair them on the Process call (the
	// ControlChannel.StashSoft contract). The LLRs are derived against the
	// fixed slicer thresholds regardless of EnableAdaptiveC4FMSlicer. The
	// CQPSK path never fires it (no real 4-level soft track). Nil by
	// default; the slice is reused across calls — copy synchronously.
	BitLLRSink func(llrs []float32, baseIdx int)
}

// Receiver is the composed IQ → dibit → LDU pipeline. Process is the
// only hot path; instantiate once per call chain and reuse.
type Receiver struct {
	demodMode DemodMode

	// C4FM path: FM discriminator → real RRC matched filter →
	// coarse-AFC carrier-offset removal → Mueller-Müller →
	// symbol-AGC → 4-level slicer. Allocated only when demodMode ==
	// DemodC4FM. dda runs alongside afc once the slicer is producing
	// trustworthy decisions (issue #402); see the handoff logic in
	// Process for the bootstrap-then-refine choreography.
	fm               *demod.FM
	mf               *demod.C4FM
	dcBlk            *dcBlock                  // optional pre-discriminator complex DC-removal (voice), nil = off
	dcbuf            []complex64               // scratch for dcBlk.process
	slicer           *demod.AdaptiveC4FMSlicer // adaptive 4-level slicer; nil on the legacy pre-scaled-fixture path
	afc              *demod.CoarseAFC
	dda              *demod.DecisionDirectedAFC
	clock            *sync.MuellerMuller
	agc              demod.C4FMSymbolAGC
	ddaNominal       [4]float32 // post-AGC nominal value for sliced ±1, ±3
	ddaResidMeanGate float64    // max |AcceptedResidualMean| for handoff (slicerScale units)
	ddaMaxDrift      float64    // max |DDA − handoff estimate| before re-arm (rad/sample)
	ddaLearning      bool       // true once the DDA is integrating: CoarseAFC frozen (subtract-only)
	ddaActive        bool       // true after handoff: afc frozen, dda carries the estimate
	ddaValidUpdates  int        // accepted-update count this learning attempt; crossing ddaHandoffSymbols arms the handoff
	ddaTotalUpdates  int        // total DDA updates this learning attempt; with ddaValidUpdates gives the accept-rate
	ddaWarmupDoneAt  int        // c4fmSymbolsTotal at which learning may (re)start; bumped on a watchdog re-arm
	afcAtHandoff     float64    // total AFC estimate the DDA carried at handoff; the watchdog bounds drift from this
	ddaRearms        int        // number of watchdog re-arms (diagnostic)
	c4fmSymbolsTotal int        // symbols processed; the DDA waits ddaWarmupSymbols of these before learning

	// CQPSK / LSM path: complex RRC + Gardner + DQPSK quadrant
	// decode + LSM dibit remap. Allocated only when demodMode ==
	// DemodCQPSK.
	cq *cqpskDemod

	assembler  *phase1.LDUAssembler
	dibitSink  phase1.DibitSink
	softSink   func([]float32)
	symbolSink func([]complex64)
	eyeSink    func([]float32, int)
	bitLLRSink func([]float32, int)
	// llrThreshold is the fixed inner/outer slicer boundary
	// (2·slicerScale/3) the BitLLRSink derivation measures against;
	// llrBuf is its reused scratch.
	llrThreshold float32
	llrBuf       []float32
	eyeSPS     int       // integer samples/symbol for the eye fold (C4FM)
	eyeBuf     []float32 // scratch for the AGC-scaled oversampled eye output
	dibitBase  int

	// Reusable scratch slices so Process doesn't allocate per call
	// on the C4FM path.
	disc    []float32
	matched []float32
	symbols []float32
	sliced  []int8
	dibits  []uint8
}

// New constructs a Receiver from opts. Panics if SampleRateHz is
// unset, neither Sink nor DibitSink is set, or the resulting
// samples-per-symbol is below 2 (the Mueller-Müller loop's minimum).
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.Sink == nil && opts.DibitSink == nil {
		panic("receiver: Sink or DibitSink is required")
	}
	sps := opts.SampleRateHz / SymbolRate
	if sps < 2 {
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (9600 Hz)")
	}
	span := opts.PulseSpanSymbols
	if span <= 0 {
		span = PulseSpanSymbols
	}
	alpha := opts.Alpha
	if alpha <= 0 {
		alpha = RolloffAlpha
	}
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}

	// The C4FM slicer's thresholds are proportional to the physical
	// signal level: with the FM discriminator in rad/sample, an outer
	// (±3) symbol peaks at 2π · DeviationHz / SampleRateHz. The spec
	// C4FM receive filter (demod.P25C4FMRxTaps) is normalised so the
	// matched-filtered symbol centres land exactly there, so that value
	// is the slicer scale. Callers that don't supply DeviationHz fall
	// back to the legacy scale 1.0 (pre-scaled-fixture path).
	slicerScale := 1.0
	if opts.DeviationHz > 0 {
		slicerScale = 2.0 * math.Pi * opts.DeviationHz / opts.SampleRateHz
	}

	r := &Receiver{
		demodMode:  opts.DemodMode,
		dibitSink:  opts.DibitSink,
		softSink:   opts.SoftSink,
		symbolSink: opts.SymbolSink,
		eyeSink:    opts.EyeSink,
		bitLLRSink: opts.BitLLRSink,
		// The fixed inner/outer decision boundary demod.C4FM.Slice uses;
		// the sign-axis boundary is 0. Anchors the BitLLRSink derivation.
		llrThreshold: float32(2 * slicerScale / 3),
	}
	// The DC blocker runs at the very top of Process, before the demod-mode
	// split, so it must be initialised for BOTH the C4FM and CQPSK paths — the
	// voice composer enables it for every P25 Phase 1 voice chain regardless of
	// mode (an LSM/simulcast site resolves to DemodCQPSK). Keep it out of the
	// mode-specific switch below.
	if opts.EnableDCBlock {
		r.dcBlk = &dcBlock{}
	}
	switch opts.DemodMode {
	case DemodCQPSK:
		r.cq = newCQPSKDemod(opts.SampleRateHz, int(sps+0.5), span, alpha, opts.GardnerGain)
	default:
		// Integer samples/symbol for the eye-diagram fold. sps is
		// SampleRateHz/SymbolRate (10 at the 48 kHz channel rate); the
		// New guard above already ensured sps >= 2.
		r.eyeSPS = int(sps + 0.5)
		r.fm = demod.NewFM()
		// P25 Phase 1 C4FM is not a root-raised-cosine matched-pair
		// system: the transmitter shapes with a raised-cosine cascaded
		// with an inverse-sinc, so the matched filter is the spec C4FM
		// receive filter (a sinc), not an RRC — issue #275.
		r.mf = demod.NewC4FMP25(opts.SampleRateHz, slicerScale)
		// Adaptive 4-level slicer (issue #402), OPT-IN via
		// EnableAdaptiveC4FMSlicer. It tracks the observed per-symbol eye
		// and slices at variance-aware, inward-capped boundaries, so a
		// DC-shifted or compressed eye that the fixed C4FM.Slice mis-decides
		// can decode correctly. Off by default: on the MMR Site 9 capture
		// (a closed, +3-stretched eye whose outer population is spread low)
		// the fixed slicer outperformed every adaptive variant, and the
		// asymmetry is an RF-domain effect, not a slicing problem. Also
		// requires DeviationHz>0 (the legacy pre-scaled-fixture path keeps
		// the fixed slicer so fixtures stay byte-for-byte unchanged).
		if opts.EnableAdaptiveC4FMSlicer && opts.DeviationHz > 0 {
			r.slicer = demod.NewAdaptiveC4FMSlicer(slicerScale)
		}
		r.afc = demod.NewCoarseAFC(sps)
		r.clock = sync.NewMuellerMuller(sps, gain)
		// Symbol-AGC bridges the level mismatch between the spec-
		// faithful matched filter on a real P25 transmission and the
		// 4-level slicer's fixed thresholds: on real RTL-SDR captures
		// the matched-filter outer-symbol centres land at
		// sps × 2π·deviation/sampleRate radians (the filter has a
		// DC gain of sps so it integrates a full symbol's
		// FM-discriminator output), which is ~sps× larger than the
		// slicerScale the synthetic-modulator harness produces.
		// Without AGC, every real sample exceeds the slicer
		// threshold and the 4-level slicer collapses to ±3 only —
		// the dibit stream becomes {1, 3} only, NIDs fail BCH at
		// errs=10..11 regardless of alignment, and the control
		// channel never locks (issue #275 Phase B). The AGC
		// normalises the running mean|x| to the slicer's expected
		// threshold (slicerScale × 2/3 — what mean|x| equals on a
		// balanced inner/outer stream), so the slicer behaves
		// correctly across the full range of real and synthetic
		// signal levels. Time constant is ~256 symbols so the loop
		// rides out short bursts of skewed content (a long FSW
		// preamble is all-outer; an idle TDU run leans inner) but
		// follows real signal-level changes (channel fades, AGC
		// settling on the front-end) within a frame's worth of
		// symbols.
		r.agc = demod.C4FMSymbolAGC{
			Target: float32(slicerScale * 2.0 / 3.0),
			Rate:   1.0 / 256.0,
		}
		// Decision-directed AFC: issue #402. Layered on top of
		// CoarseAFC. After warmup the receiver folds afc.Offset()
		// into dda.dc, sets afc.dc to zero, and switches afc to
		// subtract-only — so the matched-filter buffer stops
		// seeing two independent integrators fight each other and
		// the DDA carries any further drift on its decision-fed
		// loop, immune to the symbol-distribution mean that drives
		// the open-loop tracker off carrier on a sustained
		// unbalanced control-channel stream.
		//
		// maxOffsetHz=25 kHz is just above the RTL-SDR worst-case
		// tuner accuracy at 420 MHz (50 ppm × 420 MHz ≈ 21 kHz);
		// the clamp is a safety net, not a normal-operation limit.
		// It's scaled by sps because the CoarseAFC tracks (and the
		// DDA inherits) values in matched-filter output units,
		// which carry a DC gain of ~sps relative to the
		// FM-discriminator input — same scale factor that drives
		// the existing AGC's target=slicerScale·2/3 calibration.
		//
		// Allocated only when the caller opts in: with r.dda nil the
		// Process loop skips every DDA branch (all guarded on
		// r.dda != nil / ddaActive / ddaLearning) and runs
		// CoarseAFC-alone. Off by default — see Options.
		// EnableDecisionDirectedAFC. Issue #402.
		if opts.EnableDecisionDirectedAFC && slicerScale > 0 {
			r.dda = demod.NewDecisionDirectedAFC(maxAFCOffsetHz*sps, opts.SampleRateHz, slicerScale)
			// Nominal post-AGC values for the four slicer
			// decisions. Indexed by ((sliced+3)/2) — maps
			// −3,−1,+1,+3 to 0,1,2,3.
			r.ddaNominal = [4]float32{
				float32(-slicerScale),     // sliced = -3
				float32(-slicerScale / 3), // sliced = -1
				float32(+slicerScale / 3), // sliced = +1
				float32(+slicerScale),     // sliced = +3
			}
			// Refuse a handoff when the mean accepted residual sits
			// further than an eighth of the slicer scale off zero —
			// a quarter of the gate (slicerScale/3). A converged,
			// correctly-decided eye sits well inside this; a
			// uniformly-biased false lock does not. Issue #402.
			r.ddaResidMeanGate = slicerScale / 8.0
			// Drift bound in matched-filter output units (same sps-gain
			// scale as the CoarseAFC/DDA estimates — see the clamp).
			r.ddaMaxDrift = 2.0 * math.Pi * ddaMaxDriftHz * sps / opts.SampleRateHz
			r.ddaWarmupDoneAt = ddaWarmupSymbols
		}
	}
	if opts.Sink != nil {
		r.assembler = phase1.NewLDUAssembler(opts.Sink, opts.Tolerance)
	}
	return r
}

// Process pushes one chunk of complex64 IQ samples through the chain.
// Zero or more LDUs may be emitted to the LDU sink during the call,
// and zero or more dibit batches may be emitted to the DibitSink,
// matching the standard "data-driven, callback per complete unit"
// pattern the rest of the radio packages use.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	// Strip the static zero-IF DC spur (e.g. the HackRF centre spike) before
	// anything else. A constant IQ bias corrupts the FM discriminator's phase
	// differences and collapses C4FM voice decode; the ~1 Hz high-pass leaves
	// the modulation untouched. Voice-receiver opt-in; nil on the CC path.
	if r.dcBlk != nil {
		r.dcbuf = r.dcBlk.process(r.dcbuf, iq)
		iq = r.dcbuf
	}
	if r.demodMode == DemodCQPSK {
		// cq.process returns its internal dibit buffer; we hand
		// it directly to the sinks below — both consume
		// synchronously before the next Process call.
		r.dibits = r.cq.process(iq)
		// Surface the complex symbol-decision points for the
		// constellation scope. cq.symbols is the post-Costas
		// π/4-DQPSK stream the dibits were decoded from, so it is
		// aligned index-for-index with r.dibits (DQPSK.Decode emits
		// one dibit per symbol). Fired before the dibit sinks so a
		// consumer that pairs the two sees them on the same batch.
		if r.symbolSink != nil && len(r.cq.symbols) > 0 {
			r.symbolSink(r.cq.symbols)
		}
	} else {
		r.disc = r.fm.Process(r.disc, iq)
		r.matched = r.mf.MatchedFilter(r.matched, r.disc)
		// Freeze CoarseAFC the moment the DDA is eligible to learn
		// (warmup complete, AGC seeded). Once frozen the eye is
		// stationary for the whole learning window, so the value
		// folded into the DDA at handoff is exactly the one the DDA
		// learned against — no wandering-data-mean error. Latched
		// here, before the CoarseAFC step, so the freeze takes effect
		// from the first learning batch. Issue #402.
		if r.dda != nil && !r.ddaActive && !r.ddaLearning &&
			r.agc.Seeded() && r.agc.Target > 0 && r.c4fmSymbolsTotal >= r.ddaWarmupDoneAt {
			r.ddaLearning = true
		}
		// Coarse AFC: track and subtract the residual carrier-offset
		// DC bias before the symbol clock + slicer see it, so a real
		// tuner's frequency error doesn't shift the 4-level eye off
		// the slicer's fixed thresholds (issue #275). While the DDA is
		// learning or has handed off (issue #402), CoarseAFC is frozen
		// — Subtract keeps removing its held value while the DDA
		// carries any further drift via its Apply call below.
		if r.ddaActive || r.ddaLearning {
			r.afc.Subtract(r.matched)
		} else {
			r.afc.Process(r.matched)
		}
		if r.dda != nil {
			r.dda.Apply(r.matched)
		}
		r.symbols = r.clock.Process(r.symbols, r.matched)
		if len(r.symbols) == 0 {
			return
		}
		// agcLevel is the mean gain level this batch was actually
		// scaled against (process mutates level per sample), so the
		// DDA's un-normalisation matches the gain its residuals were
		// formed under rather than the next sample's gain. Issue #402.
		agcLevel := r.agc.Process(r.symbols)
		if r.softSink != nil {
			r.softSink(r.symbols)
		}
		// Eye-diagram tap: emit the oversampled matched-filter output
		// (which the clock loop read but did not consume), scaled by the
		// batch's AGC gain so its rails line up with the soft levels the
		// slicer decides on. Folding sps samples per symbol reconstructs
		// the 4-level eye. r.matched is still the post-AFC/DDA oversampled
		// buffer here — MuellerMuller.Process reads it read-only.
		if r.eyeSink != nil && r.eyeSPS > 0 && len(r.matched) > 0 {
			g := float32(1)
			if r.agc.Target > 0 && agcLevel > 0 {
				g = r.agc.Target / float32(agcLevel)
			}
			if cap(r.eyeBuf) < len(r.matched) {
				r.eyeBuf = make([]float32, len(r.matched))
			} else {
				r.eyeBuf = r.eyeBuf[:len(r.matched)]
			}
			for i, x := range r.matched {
				r.eyeBuf[i] = x * g
			}
			r.eyeSink(r.eyeBuf, r.eyeSPS)
		}
		// Slice to the 4-level alphabet. The adaptive slicer (when
		// allocated — the calibrated path) tracks the observed eye and
		// slices at its midpoints; otherwise fall back to the matched
		// filter's fixed-threshold slicer. Issue #402.
		if r.slicer != nil {
			r.sliced = r.slicer.SliceMany(r.sliced, r.symbols)
		} else {
			r.sliced = r.mf.SliceMany(r.sliced, r.symbols)
		}
		if cap(r.dibits) < len(r.sliced) {
			r.dibits = make([]uint8, len(r.sliced))
		} else {
			r.dibits = r.dibits[:len(r.sliced)]
		}
		for i, sym := range r.sliced {
			r.dibits[i] = phase1.SymbolToDibit(sym)
		}
		// Decision-directed AFC update, handoff, and watchdog. Gates
		// for learning:
		//
		//   - c4fmSymbolsTotal ≥ ddaWarmupDoneAt: CoarseAFC has had
		//     time to converge; before that, the slicer mis-decides
		//     inner symbols at any offset above ~1 kHz and the DDA
		//     would learn from those wrong-decision residuals. The
		//     threshold is bumped past a watchdog re-arm so CoarseAFC
		//     gets to re-converge before the DDA tries again.
		//   - agc.seeded: the un-normalisation factor (level/target)
		//     is valid; without that the DDA folds gain noise in.
		//   - r.dda non-nil: skipped on the CQPSK / legacy-fixture
		//     paths the receiver doesn't allocate it on.
		r.c4fmSymbolsTotal += len(r.sliced)
		if r.dda != nil && r.c4fmSymbolsTotal >= r.ddaWarmupDoneAt && r.agc.Seeded() && r.agc.Target > 0 && agcLevel > 0 {
			agcUnscale := float32(agcLevel) / r.agc.Target
			for i, sym := range r.sliced {
				idx := (sym + 3) / 2 // -3,-1,+1,+3 → 0,1,2,3
				if r.dda.Update(r.symbols[i], r.ddaNominal[idx], agcUnscale) {
					r.ddaValidUpdates++
				}
				r.ddaTotalUpdates++
			}
			if !r.ddaActive {
				// Commit the handoff only when decisions are both
				// plentiful (count + accept-rate) and unbiased
				// (mean accepted residual near zero). The CoarseAFC
				// value is folded in exactly, since it has been
				// frozen for the whole learning window.
				if r.ddaHandoffReady() {
					r.dda.AddOffset(r.afc.Offset())
					r.afc.SetOffset(0)
					r.afcAtHandoff = r.dda.Offset()
					r.ddaActive = true
				}
			} else if math.Abs(r.dda.Offset()-r.afcAtHandoff) > r.ddaMaxDrift {
				// Post-handoff watchdog: the DDA has walked too far
				// from the gate-verified handoff estimate to still be
				// tracking the same carrier — revert to CoarseAFC-
				// alone (the pre-DDA behaviour) so the receiver can
				// never end up worse than it was before the DDA. Hand
				// the DDA's current estimate back to CoarseAFC so the
				// eye stays continuous, then re-arm warmup so
				// CoarseAFC re-converges before the DDA tries again.
				// Issue #402.
				r.afc.SetOffset(r.dda.Offset())
				r.dda.Reset()
				r.ddaActive = false
				r.ddaLearning = false
				r.ddaValidUpdates = 0
				r.ddaTotalUpdates = 0
				r.afcAtHandoff = 0
				r.ddaWarmupDoneAt = r.c4fmSymbolsTotal + ddaWarmupSymbols
				r.ddaRearms++
			}
		}
	}
	if len(r.dibits) == 0 {
		return
	}
	// Per-bit LLRs for the soft-decision TSBK path (C4FM only — the CQPSK
	// path has no real 4-level soft track). Derived from the post-AGC soft
	// symbols against the fixed slicer axes: MSB (sign bit) from the signed
	// distance to 0, LSB (magnitude bit, 1 ⇔ outer ±3) from the distance to
	// the inner/outer threshold, in the framing convention (LLR > 0 ⇒ bit
	// 0). Fired immediately before the aligned DibitSink batch so a
	// consumer can StashSoft-then-Process on the same call.
	if r.bitLLRSink != nil && r.demodMode != DemodCQPSK && len(r.symbols) == len(r.dibits) {
		if cap(r.llrBuf) < 2*len(r.symbols) {
			r.llrBuf = make([]float32, 2*len(r.symbols))
		} else {
			r.llrBuf = r.llrBuf[:2*len(r.symbols)]
		}
		for i, y := range r.symbols {
			ay := y
			if ay < 0 {
				ay = -ay
			}
			r.llrBuf[2*i] = y // MSB = 0 for a positive symbol ⇒ LLR > 0 there
			r.llrBuf[2*i+1] = r.llrThreshold - ay
		}
		r.bitLLRSink(r.llrBuf, r.dibitBase)
	}
	if r.dibitSink != nil {
		r.dibitSink(r.dibits, r.dibitBase)
		r.dibitBase += len(r.dibits)
	}
	if r.assembler != nil {
		r.assembler.Process(r.dibits)
	}
}

// ddaHandoffReady reports whether the learning window has produced
// decisions trustworthy enough to fold CoarseAFC into the DDA and let
// the DDA carry the estimate alone. It requires three things, not just
// the raw accepted-update count the first cut used (issue #402):
//
//   - enough accepted (within-gate) updates — the loop has data;
//   - a high accept-rate over the learning window — the eye isn't
//     grossly broken (many out-of-gate decisions);
//   - a mean accepted residual near zero — the eye isn't uniformly
//     biased. A biased eye still yields within-gate "accepted"
//     residuals, so the count + accept-rate alone can't see it; the
//     residual-mean gate is what stops the handoff from locking onto
//     the stable-but-wrong offset that broke decode in #402.
func (r *Receiver) ddaHandoffReady() bool {
	if r.dda == nil || r.ddaValidUpdates < ddaHandoffSymbols || r.ddaTotalUpdates == 0 {
		return false
	}
	if float64(r.ddaValidUpdates)/float64(r.ddaTotalUpdates) < ddaHandoffMinAcceptRate {
		return false
	}
	return math.Abs(r.dda.AcceptedResidualMean()) <= r.ddaResidMeanGate
}

// DDAActive reports whether the decision-directed AFC has handed off
// and is carrying the estimate alone (CoarseAFC frozen). On a healthy
// lock this goes true shortly after warmup and stays true. Issue #402
// diagnostic.
func (r *Receiver) DDAActive() bool { return r.ddaActive }

// DDARearms returns how many times the post-handoff watchdog has
// reverted the DDA to CoarseAFC-alone (the DDA walked too far from its
// gate-verified handoff value). 0 on a stable lock; a climbing count
// means the DDA can't hold this signal and the receiver is repeatedly
// falling back. Issue #402 diagnostic.
func (r *Receiver) DDARearms() int { return r.ddaRearms }

// AFCBiasRadPerSample returns the C4FM AFC's *total* current DC bias
// estimate on the FM-discriminator output, in radians per sample at
// the receiver's input rate. A clean signal converges to ~0; a static
// carrier offset of Δf Hz leaves the AFC tracking 2π·Δf/SampleRateHz.
// Returns 0 on the CQPSK path (no AFC stage).
//
// During bootstrap (before the decision-directed AFC takes over) the
// returned value is the CoarseAFC's open-loop estimate. After the
// handoff (issue #402) it's the DDA's value — CoarseAFC has been
// zeroed and its previous estimate folded into the DDA. The
// diagnostic line stays meaningful across the handoff: whichever
// stage is steering the matched-filter buffer, this is what the
// slicer sees subtracted.
func (r *Receiver) AFCBiasRadPerSample() float64 {
	var sum float64
	if r.afc != nil {
		sum += r.afc.Offset()
	}
	if r.dda != nil {
		sum += r.dda.Offset()
	}
	return sum
}

// AFCOffsetHz returns the AFC's current estimate of the *true* carrier
// frequency offset in Hz. This is the physically meaningful number — a
// static carrier error of Δf Hz reads back as ≈Δf here.
//
// It is NOT AFCBiasRadPerSample()·Fs/(2π): the C4FM matched filter
// (demod.P25C4FMRxTaps) has a DC gain of sps, so the AFC — which tracks
// on the matched-filter output — carries a value sps× larger than the
// raw FM-discriminator offset. The true offset divides that gain back
// out, which (since sps = Fs/SymbolRate) reduces to
//
//	AFCBiasRadPerSample · SymbolRate / (2π)
//
// independent of the sample rate. Diagnostics that reported
// AFCBiasRadPerSample·Fs/(2π) over-stated the offset by ≈sps (≈10× at
// 48 kHz / 4800 baud) and sent issue #402 chasing a phantom ~10 kHz
// error that was really ~1 kHz. Returns 0 on the CQPSK path (no AFC).
func (r *Receiver) AFCOffsetHz() float64 {
	return r.AFCBiasRadPerSample() * SymbolRate / (2.0 * math.Pi)
}

// AGCLevel returns the C4FM symbol-AGC's current EMA estimate of
// mean|x| on the matched-filter output. Compare to AGCTarget(): the
// effective slicer gain at any instant is target/level. A level that
// diverges far from target after CC lock (or oscillates) points at an
// AGC misbehaviour on the live signal. Returns 0 on the CQPSK path
// (no symbol-AGC stage). Issue #402 diagnostic.
func (r *Receiver) AGCLevel() float64 { return float64(r.agc.Level()) }

// AGCTarget returns the C4FM symbol-AGC's target mean|x|, calibrated
// at construction from the configured DeviationHz / SampleRateHz so
// the matched-filter output lands on the slicer's fixed thresholds.
// Returns 0 on the CQPSK path or when DeviationHz was unset.
func (r *Receiver) AGCTarget() float64 { return float64(r.agc.Target) }

// SlicerLevels returns the adaptive 4-level slicer's current tracked
// symbol levels in −3,−1,+1,+3 order (post-AGC soft units). On a clean
// symmetric eye these sit near the nominal ±slicerScale / ±slicerScale/3;
// an asymmetric site (issue #402) shows one rail stretched. The decision
// thresholds the slicer uses are the midpoints between adjacent levels.
// Returns the zero array on the CQPSK path or the legacy fixed-slicer
// path (no adaptive slicer allocated). Issue #402 diagnostic.
func (r *Receiver) SlicerLevels() [4]float64 {
	if r.slicer == nil {
		return [4]float64{}
	}
	lv := r.slicer.Levels()
	return [4]float64{float64(lv[0]), float64(lv[1]), float64(lv[2]), float64(lv[3])}
}

// SlicerThresholds returns the adaptive slicer's three live decision
// boundaries (negative-outer, zero, positive-outer) — the values the slicer
// actually decides on, which on an asymmetric/spread eye differ from the
// midpoints of the tracked levels (issue #402: the OP asked to see the
// per-second thresholds, not just the levels). Returns the zero array on the
// CQPSK path or when no adaptive slicer is allocated (the fixed-slicer path).
func (r *Receiver) SlicerThresholds() [3]float64 {
	if r.slicer == nil {
		return [3]float64{}
	}
	th := r.slicer.Thresholds()
	return [3]float64{float64(th[0]), float64(th[1]), float64(th[2])}
}

// MMClockMu returns the Mueller-Müller symbol clock's current
// sub-sample phase accumulator (in [-1, sps]). At steady state on a
// noise-free input mu cycles deterministically through the symbol
// period; a slow monotonic drift indicates the receiver's nominal
// SampleRateHz disagrees with the stream's actual sample-rate / baud
// ratio. Returns 0 on the CQPSK path (it uses Gardner instead).
func (r *Receiver) MMClockMu() float64 {
	if r.clock == nil {
		return 0
	}
	return r.clock.Mu()
}

// MMClockSPS returns the Mueller-Müller loop's nominal samples per
// symbol. Paired with MMClockMu() so a diagnostic log can render mu
// as a fraction of the symbol period.
func (r *Receiver) MMClockSPS() float64 {
	if r.clock == nil {
		return 0
	}
	return r.clock.SPS()
}

// GardnerMu returns the CQPSK Gardner timing loop's current sub-sample
// phase accumulator. It is the CQPSK analogue of MMClockMu (the C4FM
// path uses Mueller-Müller; CQPSK uses Gardner), so a replay diagnostic
// can render the actual timing-loop state instead of the all-zero
// Mueller-Müller fields that misdirected issue #492. Returns 0 on the
// C4FM path (no Gardner loop).
func (r *Receiver) GardnerMu() float64 {
	if r.cq == nil {
		return 0
	}
	return r.cq.gardner.Mu()
}

// GardnerSPS returns the CQPSK Gardner loop's nominal samples per
// symbol. Paired with GardnerMu so a diagnostic can render mu as a
// fraction of the symbol period. Returns 0 on the C4FM path.
func (r *Receiver) GardnerSPS() float64 {
	if r.cq == nil {
		return 0
	}
	return r.cq.gardner.SPS()
}

// CQPSKAGCGain returns the CQPSK path's matched-filter AGC gain (the
// scale it currently applies to normalise the symbol amplitude the
// Gardner/CMA loops are tuned against). A gain pinned at 1.0 means the
// AGC never saw signal energy to seed on. Returns 0 on the C4FM path
// (which has its own symbol-AGC reported via AGCLevel/AGCTarget).
func (r *Receiver) CQPSKAGCGain() float64 {
	if r.cq == nil {
		return 0
	}
	return float64(r.cq.agc.Gain())
}

// CMAError returns the CQPSK blind equalizer's most recent |y|²−R²
// convergence proxy. It trends toward 0 as the CMA opens the
// constellation; a value stuck large means the equalizer has not
// converged. Returns 0 on the C4FM path (no CMA stage).
func (r *Receiver) CMAError() float64 {
	if r.cq == nil {
		return 0
	}
	return float64(r.cq.cmaErr)
}

// CQPSKCarrierOffsetHz returns the CQPSK carrier-recovery loop's current
// estimate of the residual carrier-frequency offset in Hz (the coarse
// block seed plus the tracking loop's residual). A static tuner error of
// Δf Hz reads back as ≈Δf here once the loop settles. Returns 0 on the
// C4FM path (which reports its carrier estimate via AFCOffsetHz).
func (r *Receiver) CQPSKCarrierOffsetHz() float64 {
	if r.cq == nil {
		return 0
	}
	return r.cq.carrierOffsetHz()
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so a
// stale FSW match doesn't bleed across the discontinuity, and so the
// DibitSink baseIdx restarts at 0 for downstream consumers that
// track absolute dibit positions.
func (r *Receiver) Reset() {
	if r.assembler != nil {
		r.assembler.Reset()
	}
	r.dibitBase = 0
	if r.cq != nil {
		r.cq.reset()
	}
	if r.afc != nil {
		r.afc.Reset()
	}
	if r.dcBlk != nil {
		r.dcBlk.Reset()
	}
	if r.dda != nil {
		r.dda.Reset()
	}
	if r.slicer != nil {
		r.slicer.Reset()
	}
	r.ddaLearning = false
	r.ddaActive = false
	r.ddaValidUpdates = 0
	r.ddaTotalUpdates = 0
	r.ddaWarmupDoneAt = ddaWarmupSymbols
	r.afcAtHandoff = 0
	r.ddaRearms = 0
	r.c4fmSymbolsTotal = 0
	r.agc.Reset()
	// FM discriminator's `last` is harmless to leave alone — the
	// next sample it processes will produce one slightly-wrong
	// derivative, which the matched filter smooths out.
}
