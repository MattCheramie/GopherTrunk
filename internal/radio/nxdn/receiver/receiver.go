// Package receiver wires the IQ → C4FM dibit chain that feeds the
// NXDN control-channel state machine for the 9600-baud 4-FSK
// variant (the most common deployment; the 4800-baud BFSK variant
// uses a 2-level slicer and lives in a follow-up).
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → RRC matched filter + 4-level slicer (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → 4-level symbol → 0..3 dibit (SymbolToDibit, local)
//	  → nxdn.DibitSink
//
// NXDN's 9600-baud channel rate is 4800 sym/s 4-FSK with α = 0.20
// RRC pulse shaping — the same modulation P25 Phase 1, DMR, and YSF
// use, so the matched-filter parameters carry over unchanged. Only
// the downstream framing (192-dibit / 80 ms frames, 8-dibit FSW,
// LICH / SACCH / Info field layout) differs and lives in the parent
// nxdn package.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
)

// NXDN on-air parameters (4-FSK 9600-baud variant).
const (
	// SymbolRate is the channel symbol rate. Each symbol carries one
	// dibit (2 bits) on the wire for a total channel capacity of
	// 9600 bps. The BFSK variant uses the same symbol rate but only
	// 1 bit per symbol — a separate receiver path.
	SymbolRate = 4800.0
	// RolloffAlpha matches the standard NXDN / P25 Phase 1 / YSF /
	// DMR receiver pulse shape.
	RolloffAlpha = 0.20
	// PulseSpanSymbols is the half-span of the RRC pulse on each
	// side of the symbol time.
	PulseSpanSymbols = 8
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate.
	SampleRateHz float64
	// DibitSink receives the raw dibit stream the receiver decodes
	// from IQ. Required.
	DibitSink nxdn.DibitSink
	// PulseSpanSymbols overrides the RRC half-span. <= 0 uses
	// PulseSpanSymbols.
	PulseSpanSymbols int
	// Alpha overrides the RRC roll-off. <= 0 uses RolloffAlpha.
	Alpha float64
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
}

// Receiver is the composed IQ → dibit pipeline.
type Receiver struct {
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	dibitSink nxdn.DibitSink
	dibitBase int

	disc    []float32
	matched []float32
	symbols []float32
	sliced  []int8
	dibits  []uint8
}

// New constructs a Receiver. Panics if SampleRateHz or DibitSink are
// unset, or the resulting samples-per-symbol is below 2.
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
	const slicerScale = 1.0

	return &Receiver{
		fm:        demod.NewFM(),
		mf:        demod.NewC4FM(int(sps+0.5), span, alpha, slicerScale),
		clock:     sync.NewMuellerMuller(sps, gain),
		dibitSink: opts.DibitSink,
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

// Reset returns the receiver to its initial state.
func (r *Receiver) Reset() {
	r.dibitBase = 0
}

// SymbolToDibit maps a C4FM slicer output ({-3, -1, +1, +3}) to a
// dibit value (0..3). Uses the same Gray-coded convention as the
// P25 Phase 1 / DMR / YSF receivers: +3 → 01, +1 → 00, -1 → 10,
// -3 → 11. NXDN's on-air symbol mapping in the Common Air Interface
// specification matches this convention; the mapping is pinned by
// unit test so a future spec re-read doesn't silently desync from
// the FSW patterns in the parent nxdn package.
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
