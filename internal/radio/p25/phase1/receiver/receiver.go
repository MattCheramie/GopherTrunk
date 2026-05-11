// Package receiver wires the IQ → C4FM dibit chain that feeds the
// P25 Phase 1 LDU assembler. It composes primitives that already
// live in internal/dsp + internal/radio/p25/phase1:
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → RRC matched filter + 4-level slicer (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → C4FM symbol → 0..3 dibit (phase1.SymbolToDibit)
//	  → LDU assembler (phase1.LDUAssembler)
//
// The receiver is stateful and not safe for concurrent Process calls.
// Instantiate one per tuned frequency / per call chain. All primitives
// it composes own their own internal history, so chunk boundaries do
// not corrupt the stream.
//
// This package closes the last symbol-domain gap in the README's
// roadmap: the LDU assembler and everything downstream (LDU layout,
// IMBE channel decoding, recorder integration) was already in place;
// what was missing was a glue layer that takes captured baseband IQ
// and produces the dibit stream the assembler consumes.
package receiver

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
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
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization (e.g. the polyphase channelizer's per-channel
	// output rate). Required; must be ≥ 2 * SymbolRate.
	SampleRateHz float64
	// Sink receives complete 1728-bit LDU buffers ready for
	// phase1.ExtractVoiceFrames. Required.
	Sink phase1.LDUSink
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
}

// Receiver is the composed IQ → dibit → LDU pipeline. Process is the
// only hot path; instantiate once per call chain and reuse.
type Receiver struct {
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	assembler *phase1.LDUAssembler

	// Reusable scratch slices so Process doesn't allocate per call.
	disc     []float32
	matched  []float32
	symbols  []float32
	sliced   []int8
	dibits   []uint8
}

// New constructs a Receiver from opts. Panics if SampleRateHz or Sink
// are unset, or the resulting samples-per-symbol is below 2 (the
// Mueller-Müller loop's minimum).
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.Sink == nil {
		panic("receiver: Sink is required")
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
	// output range so ±1 maps to ±deviation. Passing 1.0 sets the
	// inner thresholds at ±2/3 and the outer at >2/3, matching the
	// {-3,-1,+1,+3} symbol alphabet after the discriminator scales
	// the four C4FM levels into a real-valued ramp around zero.
	const slicerScale = 1.0

	return &Receiver{
		fm:        demod.NewFM(),
		mf:        demod.NewC4FM(int(sps+0.5), span, alpha, slicerScale),
		clock:     sync.NewMuellerMuller(sps, gain),
		assembler: phase1.NewLDUAssembler(opts.Sink, opts.Tolerance),
	}
}

// Process pushes one chunk of complex64 IQ samples through the chain.
// Zero or more LDUs may be emitted to the sink during the call,
// matching the standard "data-driven, callback per complete unit"
// pattern the rest of the radio packages use.
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
	r.sliced = r.mf.SliceMany(r.sliced, r.symbols)
	if cap(r.dibits) < len(r.sliced) {
		r.dibits = make([]uint8, len(r.sliced))
	} else {
		r.dibits = r.dibits[:len(r.sliced)]
	}
	for i, sym := range r.sliced {
		r.dibits[i] = phase1.SymbolToDibit(sym)
	}
	r.assembler.Process(r.dibits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so a
// stale FSW match doesn't bleed across the discontinuity.
func (r *Receiver) Reset() {
	r.assembler.Reset()
	// FM discriminator's `last` is harmless to leave alone — the
	// next sample it processes will produce one slightly-wrong
	// derivative, which the matched filter smooths out.
}
