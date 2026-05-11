// Package receiver wires the IQ → C4FM dibit chain that feeds the
// DMR control-channel state machines (Tier II conventional + Tier III
// trunked). It composes primitives that already live in internal/dsp
// + internal/radio/dmr:
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → RRC matched filter + 4-level slicer (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → 4-level symbol → 0..3 dibit (SymbolToDibit, local)
//	  → dmr.DibitSink (cross-call indexed by absolute dibit position)
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
}

// Receiver is the composed IQ → dibit pipeline. Process is the only
// hot path; instantiate once per call chain and reuse.
type Receiver struct {
	fm        *demod.FM
	mf        *demod.C4FM
	clock     *sync.MuellerMuller
	dibitSink dmr.DibitSink
	dibitBase int

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

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the DibitSink baseIdx restarts at 0.
func (r *Receiver) Reset() {
	r.dibitBase = 0
}

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
