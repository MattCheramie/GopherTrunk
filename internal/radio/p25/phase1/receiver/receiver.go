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
//	    → RRC matched filter (internal/dsp/demod.C4FM)
//	    → coarse AFC: residual carrier-offset removal
//	      (internal/dsp/demod.CoarseAFC)
//	    → Mueller-Müller symbol clock recovery (sync.MuellerMuller)
//	    → 4-level slicer → C4FM symbol → 0..3 dibit
//	      (internal/dsp/demod.C4FM, phase1.SymbolToDibit)
//
//	DemodCQPSK (LSM / simulcast):
//	  IQ
//	    → complex RRC matched filter (demod.PiOver4DQPSK, rotation=π/4)
//	    → Gardner timing recovery on complex IQ (sync.Gardner)
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
// caller leaves Options.GardnerGain at zero. Matches the value Phase 2
// and TETRA settled on after live-capture tuning (internal/scanner/
// ccdecoder/pipelines.go).
const defaultGardnerGain = 0.03

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
	// — required for simulcast P25 sites whose control channel is
	// on the wire as LSM rather than C4FM. See modes.go.
	DemodMode DemodMode
	// GardnerGain overrides the Gardner loop step used by the
	// CQPSK path. <=0 uses defaultGardnerGain. Ignored when
	// DemodMode == DemodC4FM (the C4FM path uses Mueller-Müller).
	GardnerGain float64
}

// Receiver is the composed IQ → dibit → LDU pipeline. Process is the
// only hot path; instantiate once per call chain and reuse.
type Receiver struct {
	demodMode DemodMode

	// C4FM path: FM discriminator → real RRC matched filter →
	// coarse-AFC carrier-offset removal → Mueller-Müller → 4-level
	// slicer. Allocated only when demodMode == DemodC4FM.
	fm    *demod.FM
	mf    *demod.C4FM
	afc   *demod.CoarseAFC
	clock *sync.MuellerMuller

	// CQPSK / LSM path: complex RRC + Gardner + DQPSK quadrant
	// decode + LSM dibit remap. Allocated only when demodMode ==
	// DemodCQPSK.
	cq *cqpskDemod

	assembler *phase1.LDUAssembler
	dibitSink phase1.DibitSink
	dibitBase int

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

	// Slicer thresholds are normalised to the FM-discriminator's
	// output range so ±1 maps to ±deviation. With the FM
	// discriminator in rad/sample, the FM peak at symbol ±3 is
	// 2π · DeviationHz / SampleRateHz, so we pass that value
	// (times the symbol-magnitude reference of 1.0 — symbol +3
	// produces a peak ±value, +1 produces ±value/3, etc.) as the
	// "deviation" arg to NewC4FM. The slicer then puts the
	// +1/+3 boundary at 2/3 of this and the -1/-3 boundary at
	// -2/3, both proportional to the physical signal level.
	//
	// Callers that don't supply DeviationHz fall back to the
	// legacy slicerScale = 1.0 — matches the existing synthesized
	// fixture tests that pre-scale their FM levels into ±1.
	slicerScale := 1.0
	if opts.DeviationHz > 0 {
		slicerScale = 2.0 * math.Pi * opts.DeviationHz / opts.SampleRateHz
	}

	r := &Receiver{
		demodMode: opts.DemodMode,
		dibitSink: opts.DibitSink,
	}
	switch opts.DemodMode {
	case DemodCQPSK:
		r.cq = newCQPSKDemod(int(sps+0.5), span, alpha, opts.GardnerGain)
	default:
		r.fm = demod.NewFM()
		r.mf = demod.NewC4FM(int(sps+0.5), span, alpha, slicerScale)
		r.afc = demod.NewCoarseAFC(sps)
		r.clock = sync.NewMuellerMuller(sps, gain)
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
	if r.demodMode == DemodCQPSK {
		// cq.process returns its internal dibit buffer; we hand
		// it directly to the sinks below — both consume
		// synchronously before the next Process call.
		r.dibits = r.cq.process(iq)
	} else {
		r.disc = r.fm.Process(r.disc, iq)
		r.matched = r.mf.MatchedFilter(r.matched, r.disc)
		// Coarse AFC: track and subtract the residual carrier-offset
		// DC bias before the symbol clock + slicer see it, so a real
		// tuner's frequency error doesn't shift the 4-level eye off
		// the slicer's fixed thresholds (issue #275).
		r.afc.Process(r.matched)
		r.symbols = r.clock.Process(r.symbols, r.matched)
		if len(r.symbols) == 0 {
			return
		}
		r.sliced = r.mf.SliceMany(r.sliced, r.symbols)
		if cap(r.dibits) < len(r.sliced) {
			r.dibits = make([]uint8, len(r.sliced))
		} else {
			r.dibits = r.dibits[:len(r.sliced)]
		}
		for i, sym := range r.sliced {
			r.dibits[i] = phase1.SymbolToDibit(sym)
		}
	}
	if len(r.dibits) == 0 {
		return
	}
	if r.dibitSink != nil {
		r.dibitSink(r.dibits, r.dibitBase)
		r.dibitBase += len(r.dibits)
	}
	if r.assembler != nil {
		r.assembler.Process(r.dibits)
	}
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
	// FM discriminator's `last` is harmless to leave alone — the
	// next sample it processes will produce one slightly-wrong
	// derivative, which the matched filter smooths out.
}
