// Package receiver wires the IQ → C4FM dibit chain that feeds the
// DMR control-channel state machines (Tier II conventional + Tier III
// trunked). It composes primitives that already live in internal/dsp
// + internal/radio/dmr:
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → RRC matched filter (internal/dsp/demod.C4FM)
//	  → symbol AGC: level normalisation (c4fmSymbolAGC, local)
//	  → 4-level slicer (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → 4-level symbol → 0..3 dibit (SymbolToDibit, local)
//	  → dmr.DibitSink (cross-call indexed by absolute dibit position)
//
// The symbol-AGC stage is what lets the receiver decode a real SDR
// capture rather than only the zero-offset, level-matched synthesized
// fixtures. The RRC matched filter is normalised to unit *energy*, so it
// has a DC gain of ~3.1: on a real FM-discriminator stream the matched
// filter leaves the 4-level symbol centres well above the slicer's fixed
// thresholds, the slicer collapses the inner (±1) symbols onto the outer
// (±3) rails, and every BPTC(196,96) payload comes back uncorrectable
// while the more forgiving sync + slot-type FEC still pass — exactly the
// "Tier III BPTC uncorrectable / Tier II instalock then nothing" field
// failure. The AGC renormalises the running mean|x| to the level a
// balanced 4-level stream should sit at so the slicer decides correctly,
// mirroring the same calibration the P25 Phase 1 C4FM receiver runs
// (internal/radio/p25/phase1/receiver, issue #275).
//
// DMR runs the same 4800-baud 4-level C4FM modulation as P25 Phase 1
// and YSF, so the matched-filter parameters (RRC α = 0.20, 4800
// sym/s) are identical. The downstream framing — 132-dibit TDMA
// bursts with a 24-dibit sync field bracketed by 2 × 10-bit slot-
// type fields — lives in the parent dmr package and the tier2 /
// tier3 control-channel state machines.
//
// The receiver is stateful and not safe for concurrent Process calls.
// Instantiate one per tuned frequency / per call chain. All
// primitives it composes own their own internal history so chunk
// boundaries do not corrupt the stream.
package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// DMR on-air parameters (per ETSI TS 102 361-1).
const (
	// SymbolRate is the channel symbol rate. Each symbol carries one
	// dibit (2 bits) on the wire; with 2 TDMA slots the total channel
	// capacity is 9600 bps.
	SymbolRate = 4800.0
	// RolloffAlpha is the RRC roll-off the matched filter is designed
	// around. 0.20 matches the standard DMR / P25 Phase 1 / YSF
	// receiver pulse shape.
	RolloffAlpha = 0.20
	// PulseSpanSymbols is the half-span of the RRC pulse on each side
	// of the symbol time. 8 symbols (16 total) is the standard
	// receiver-side compromise between truncation noise and CPU cost.
	PulseSpanSymbols = 8
)

// Options configures a Receiver. Zero-valued fields fall back to the
// DMR defaults above.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate.
	SampleRateHz float64
	// DibitSink receives the raw dibit stream the receiver decodes
	// from IQ. The connector wraps a DMR ControlChannel adapter
	// here once that lands; this PR ships the symbol-domain glue
	// in isolation so the connector + adapter can land together.
	// Required.
	DibitSink dmr.DibitSink
	// PulseSpanSymbols overrides the RRC half-span. <= 0 uses
	// PulseSpanSymbols.
	PulseSpanSymbols int
	// Alpha overrides the RRC roll-off. <= 0 uses RolloffAlpha.
	Alpha float64
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
	// DeviationHz is the peak frequency deviation of the C4FM
	// signal at symbol ±3 (1944 Hz on DMR per ETSI TS 102 361-1
	// §6.3). Used to calibrate the slicer thresholds against the
	// FM-discriminator output level (slicer scale = 2π ·
	// DeviationHz / SampleRateHz). <= 0 falls back to the legacy
	// slicerScale = 1.0 — back-compat with fixtures that pre-scale
	// their FM levels.
	DeviationHz float64

	// SoftSink, when set, receives the post-AFC/AGC 1-sample-per-symbol
	// soft waveform (the 4-level C4FM soft track) each batch, aligned with
	// the DibitSink batch fired on the same Process call. Optional; used by
	// the diagnostic symbol scope. Signature matches the P25 Phase 1 C4FM
	// receiver so the scope routes DMR and P25 identically.
	SoftSink func(softSamples []float32)
	// SymbolSink is stored for API parity with the P25 receiver but is
	// never fired on the DMR path (pure C4FM has no complex CQPSK symbol
	// domain; the soft track is surfaced via SoftSink). Optional.
	SymbolSink func(symbols []complex64)
	// EyeSink, when set, receives the oversampled matched-filter output
	// (sps samples/symbol) scaled to the soft-level units and recentred for
	// the residual carrier offset, so folding sps samples reconstructs the
	// 4-level eye. Optional; used by the diagnostic symbol scope.
	EyeSink func(oversampled []float32, sps int)
}

// Receiver is the composed IQ → dibit pipeline. Process is the only
// hot path; instantiate once per call chain and reuse.
type Receiver struct {
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	afc       *demod.CoarseAFC
	agc       demod.C4FMSymbolAGC
	dibitSink dmr.DibitSink
	dibitBase int

	// Optional diagnostic taps (symbol scope). nil = no-op.
	softSink   func([]float32)
	symbolSink func([]complex64) // parity only; never fired (pure C4FM)
	eyeSink    func([]float32, int)
	eyeSPS     int
	eyeBuf     []float32

	disc    []float32
	matched []float32
	symbols []float32
	sliced  []int8
	dibits  []uint8
}

// New constructs a Receiver from opts. Panics if SampleRateHz or
// DibitSink are unset, or the resulting samples-per-symbol is below 2
// (the Mueller-Müller loop's minimum).
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.DibitSink == nil {
		panic("receiver: DibitSink is required")
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
	// Slicer thresholds: when DeviationHz is set, calibrate
	// against the physical FM-discriminator level (same fix as
	// the P25 P1 / NXDN receivers). Legacy fixtures that
	// pre-scale their FM output to ±1 stay green via the
	// fallback.
	slicerScale := 1.0
	if opts.DeviationHz > 0 {
		slicerScale = 2.0 * math.Pi * opts.DeviationHz / opts.SampleRateHz
	}

	// Coarse carrier-offset correction (issue #836). An uncorrected tuner
	// ppm error leaves the FM discriminator as a constant DC bias that
	// slides the 4-level eye off the slicer's fixed thresholds; the
	// narrowband DMR demod tolerates only ~±75 Hz before decode collapses.
	// CoarseAFC tracks and subtracts that bias, recentring the eye — the
	// same primitive the P25 Phase 1 C4FM receiver runs (issue #275), which
	// DMR was the one 4800-baud C4FM receiver to skip.
	//
	// Unlike P25, DMR applies the correction on the recovered SYMBOL stream
	// (post-clock, pre-slicer), not on the pre-clock matched-filter stream.
	// The offset's DC bias is present identically in both domains, but the
	// coarse open-loop estimate wanders by a few Hz as it chases the data
	// mean, and feeding that time-varying bias into the Mueller-Müller
	// timing loop destabilises symbol timing on a clean signal (verified:
	// it drove synthetic-decode sync to zero). Correcting on the symbols
	// recentres the eye for the fixed-threshold slicer while leaving the
	// timing loop's input untouched. That keeps large offsets (where the
	// clock itself needs help) bounded to the timing loop's own pull-in
	// range — a P25-style before-clock/freeze stage is the follow-up for
	// grossly-mistuned dongles, which still want sdr.ppm today.
	//
	// sps=1: the symbol stream is one sample per symbol, so the 64-symbol
	// time constant is NewCoarseAFC(1). Allocated only on the calibrated
	// DeviationHz>0 path so legacy pre-scaled fixtures stay byte-identical.
	var afc *demod.CoarseAFC
	if opts.DeviationHz > 0 {
		afc = demod.NewCoarseAFC(1)
	}

	return &Receiver{
		fm:    demod.NewFM(),
		mf:    demod.NewC4FM(int(sps+0.5), span, alpha, slicerScale),
		clock: sync.NewMuellerMuller(sps, gain),
		afc:   afc,
		// Symbol-AGC bridges the level mismatch between the unit-energy
		// RRC matched filter and the 4-level slicer's fixed thresholds.
		// The RRC has a DC gain of ~3.1 (it is normalised to unit
		// energy, not unit DC), so on a real signal the matched-filter
		// symbol centres land well above slicerScale and the fixed
		// slicer collapses inner symbols onto the outer rails — every
		// BPTC payload then comes back uncorrectable. The AGC normalises
		// the running mean|x| to the level a balanced 4-level stream
		// should sit at (slicerScale·2/3), so the slicer decides
		// correctly regardless of the absolute matched-filter level.
		// Same calibration the P25 Phase 1 C4FM receiver runs (issue
		// #275). Disabled on the legacy DeviationHz<=0 path (slicerScale
		// == 1.0) so pre-scaled fixtures stay byte-identical.
		agc: demod.C4FMSymbolAGC{
			Target: float32(demod.C4FMAGCTarget(slicerScale, opts.DeviationHz)),
			Rate:   1.0 / 256.0,
		},
		dibitSink:  opts.DibitSink,
		softSink:   opts.SoftSink,
		symbolSink: opts.SymbolSink,
		eyeSink:    opts.EyeSink,
		eyeSPS:     int(sps + 0.5),
	}
}

// Process pushes one chunk of complex64 IQ samples through the chain.
// Zero or more dibit batches may be emitted to DibitSink during the
// call.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	r.disc = r.fm.Process(r.disc, iq)
	r.matched = r.mf.MatchedFilter(r.matched, r.disc)
	r.symbols = r.clock.Process(r.symbols, r.matched)
	if len(r.symbols) == 0 {
		return
	}
	// Subtract the residual carrier-offset DC bias from the recovered
	// symbols before slicing, so a real tuner's frequency error doesn't
	// shift the 4-level eye off the slicer's fixed thresholds and stop the
	// decode (issue #836). Applied here (post-clock) rather than before the
	// timing loop so its small, data-driven wander can't destabilise symbol
	// timing; nil (no-op) on the legacy DeviationHz<=0 path. Runs before the
	// AGC so the level normalisation sees a centred eye.
	if r.afc != nil {
		r.afc.Process(r.symbols)
	}
	// Normalise the symbol level to the slicer's expected scale before
	// slicing, so the unit-energy matched filter's ~3.1× DC gain doesn't
	// push the 4-level eye past the slicer's fixed thresholds (no-op on
	// the legacy DeviationHz<=0 path where target==0). See package doc.
	agcLevel := r.agc.Process(r.symbols)
	// Diagnostic taps (symbol scope). The soft track is the post-AFC/AGC
	// 1/symbol waveform — aligned with the dibit batch fired below. The eye
	// is the oversampled matched buffer the clock loop read read-only,
	// scaled by this batch's AGC gain so its rails line up with the soft
	// levels. Unlike P25, DMR runs the AFC on the post-clock symbol stream
	// (issue #836), so r.matched here is NOT AFC-corrected — subtract the
	// AFC's DC offset (identical bias in the oversampled and decimated
	// domains) to recentre the eye. nil sinks are no-ops.
	if r.softSink != nil {
		r.softSink(r.symbols)
	}
	if r.eyeSink != nil && r.eyeSPS > 0 && len(r.matched) > 0 {
		g := float32(1)
		if r.agc.Target > 0 && agcLevel > 0 {
			g = r.agc.Target / float32(agcLevel)
		}
		off := float32(0)
		if r.afc != nil {
			off = float32(r.afc.Offset())
		}
		if cap(r.eyeBuf) < len(r.matched) {
			r.eyeBuf = make([]float32, len(r.matched))
		} else {
			r.eyeBuf = r.eyeBuf[:len(r.matched)]
		}
		for i, x := range r.matched {
			r.eyeBuf[i] = (x - off) * g
		}
		r.eyeSink(r.eyeBuf, r.eyeSPS)
	}
	r.sliced = r.mf.SliceMany(r.sliced, r.symbols)
	if cap(r.dibits) < len(r.sliced) {
		r.dibits = make([]uint8, len(r.sliced))
	} else {
		r.dibits = r.dibits[:len(r.sliced)]
	}
	for i, sym := range r.sliced {
		r.dibits[i] = SymbolToDibit(sym)
	}
	r.dibitSink(r.dibits, r.dibitBase)
	r.dibitBase += len(r.dibits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the DibitSink baseIdx restarts at 0.
func (r *Receiver) Reset() {
	r.dibitBase = 0
	r.agc.Reset()
	if r.afc != nil {
		r.afc.Reset()
	}
}

// AGCLevel and AGCTarget expose the shared symbol-AGC's running mean|x|
// estimate and its target, for the diagnostic Tuning panel. Both are 0 on the
// legacy pre-scaled-fixture path (AGC disabled).
func (r *Receiver) AGCLevel() float64  { return float64(r.agc.Level()) }
func (r *Receiver) AGCTarget() float64 { return float64(r.agc.Target) }

// MMClockMu and MMClockSPS expose the Mueller-Müller timing loop's fractional
// interpolation index and samples-per-symbol estimate, mirroring the P25
// Phase 1 C4FM receiver's getters so the symbol scope reads them identically.
func (r *Receiver) MMClockMu() float64  { return r.clock.Mu() }
func (r *Receiver) MMClockSPS() float64 { return r.clock.SPS() }

// SymbolToDibit maps a C4FM slicer output ({-3, -1, +1, +3}) to a
// dibit value (0..3). Uses the same Gray-coded convention as
// P25 Phase 1 (TIA-102.BAAA) and YSF: +3 → 01, +1 → 00, -1 → 10,
// -3 → 11. DMR's on-air symbol-to-dibit mapping in ETSI TS 102 361-1
// matches this; the receiver's mapping is pinned by unit test so a
// future spec re-read doesn't silently desync from the published
// sync patterns in the parent dmr package.
func SymbolToDibit(sym int8) uint8 {
	switch sym {
	case 1:
		return 0
	case 3:
		return 1
	case -1:
		return 2
	case -3:
		return 3
	}
	return 0
}
