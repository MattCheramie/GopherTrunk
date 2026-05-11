// Package receiver wires the IQ → MSK bit chain that feeds the
// Motorola Type II / SmartZone control-channel state machine.
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → Gaussian matched filter + 2-level slicer (internal/dsp/demod.GFSK)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → motorola.BitSink
//
// Motorola Type II's control channel is 3600-baud MSK. MSK is a
// special case of CPFSK (modulation index = 0.5) and decodes
// cleanly via an FM-discriminator front-end plus a matched filter —
// the same shape as the GFSK receiver. The GFSK helper's BT-
// configurable Gaussian matched filter is the closest approximation
// of an ideal MSK matched filter; with the default BT it works well
// enough to recover bits from a clean MSK signal, and the
// configurable BT lets the operator fine-tune for noisy or
// off-spec transmitters.
//
// The receiver emits raw bits (each byte is 0 or 1) via
// motorola.BitSink. The downstream framing — 24-bit sync, 84-bit
// OSW with a BCH(64,16,11) FEC reducing to 32 information bits —
// lives in the parent motorola package + upstream FEC.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
)

// Motorola Type II SmartZone on-air parameters.
const (
	// SymbolRate is the control-channel bit rate.
	SymbolRate = 3600.0
	// BT is the Gaussian matched-filter product. 0.5 is the
	// closest fit for an MSK matched filter; vendor receivers
	// sometimes drop to ~0.3 for noisier sites.
	BT = 0.5
	// PulseSpanSymbols is the half-span of the Gaussian pulse on
	// each side of the symbol time. 4 symbols is the typical
	// receiver-side compromise.
	PulseSpanSymbols = 4
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate (7200 Hz).
	SampleRateHz float64
	// BitSink receives the raw bit stream the receiver decodes
	// from IQ. Required.
	BitSink motorola.BitSink
	// PulseSpanSymbols overrides the Gaussian half-span. <= 0
	// uses PulseSpanSymbols.
	PulseSpanSymbols int
	// BTProduct overrides the Gaussian BT. <= 0 uses BT.
	BTProduct float64
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
}

// Receiver is the composed IQ → bit pipeline.
type Receiver struct {
	fm      *demod.FM
	gfsk    *demod.GFSK
	clock   *sync.MuellerMuller
	bitSink motorola.BitSink
	bitBase int

	disc    []float32
	matched []float32
	symbols []float32
	bits    []byte
}

// New constructs a Receiver. Panics if SampleRateHz or BitSink are
// unset, or the resulting samples-per-symbol is below 2.
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.BitSink == nil {
		panic("receiver: BitSink is required")
	}
	sps := opts.SampleRateHz / SymbolRate
	if sps < 2 {
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (7200 Hz)")
	}
	span := opts.PulseSpanSymbols
	if span <= 0 {
		span = PulseSpanSymbols
	}
	bt := opts.BTProduct
	if bt <= 0 {
		bt = BT
	}
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}

	return &Receiver{
		fm:      demod.NewFM(),
		gfsk:    demod.NewGFSK(int(sps+0.5), span, bt),
		clock:   sync.NewMuellerMuller(sps, gain),
		bitSink: opts.BitSink,
	}
}

// Process pushes one chunk of complex64 IQ samples through the
// chain. Zero or more bit batches may be emitted to BitSink during
// the call.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	r.disc = r.fm.Process(r.disc, iq)
	r.matched = r.gfsk.MatchedFilter(r.matched, r.disc)
	r.symbols = r.clock.Process(r.symbols, r.matched)
	if len(r.symbols) == 0 {
		return
	}
	if cap(r.bits) < len(r.symbols) {
		r.bits = make([]byte, len(r.symbols))
	} else {
		r.bits = r.bits[:len(r.symbols)]
	}
	for i, s := range r.symbols {
		r.bits[i] = byte(r.gfsk.Slice(s))
	}
	r.bitSink(r.bits, r.bitBase)
	r.bitBase += len(r.bits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the BitSink baseIdx restarts at 0 and the matched filter sheds
// its history.
func (r *Receiver) Reset() {
	r.bitBase = 0
	r.gfsk.Reset()
}
