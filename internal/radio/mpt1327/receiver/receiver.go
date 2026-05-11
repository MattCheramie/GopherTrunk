// Package receiver wires the IQ → FFSK bit chain that feeds the
// MPT 1327 control-channel state machine. MPT 1327 carries 1200-baud
// CCIR FFSK signalling on top of a narrowband-FM audio channel —
// mark = 1200 Hz = binary 1, space = 1800 Hz = binary 0.
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → real audio
//	  → FFSK tone discriminator (internal/dsp/demod.FFSK)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → mpt1327.BitSink
//
// The receiver is 2-level so it emits raw bits (each byte is 0 or 1)
// via mpt1327.BitSink. The downstream framing (24-bit preamble +
// 64-bit codeword units, BCH(63,38) parity) lives in the parent
// mpt1327 package + upstream FEC.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
)

// MPT 1327 on-air parameters (CCIR FFSK).
const (
	// SymbolRate is the bit rate on the FFSK signalling layer.
	SymbolRate = 1200.0
	// MarkHz / SpaceHz are the CCIR FFSK tone frequencies inside
	// the FM audio. Mark conventionally encodes binary 1.
	MarkHz  = 1200.0
	SpaceHz = 1800.0
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SpaceHz so the FFSK
	// helper's complex mixer can reach the upper tone.
	SampleRateHz float64
	// BitSink receives the raw bit stream the receiver decodes
	// from IQ. Required.
	BitSink mpt1327.BitSink
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
}

// Receiver is the composed IQ → bit pipeline.
type Receiver struct {
	fm      *demod.FM
	ffsk    *demod.FFSK
	clock   *sync.MuellerMuller
	bitSink mpt1327.BitSink
	bitBase int

	disc    []float32
	tone    []float32
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
	if opts.SampleRateHz < 2*SpaceHz {
		panic("receiver: SampleRateHz must be >= 2*SpaceHz (3600 Hz)")
	}
	sps := opts.SampleRateHz / SymbolRate
	if sps < 2 {
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (2400 Hz)")
	}
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}

	return &Receiver{
		fm:      demod.NewFM(),
		ffsk:    demod.NewFFSK(opts.SampleRateHz, MarkHz, SpaceHz),
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
	r.tone = r.ffsk.Discriminate(r.tone, r.disc)
	r.symbols = r.clock.Process(r.symbols, r.tone)
	if len(r.symbols) == 0 {
		return
	}
	if cap(r.bits) < len(r.symbols) {
		r.bits = make([]byte, len(r.symbols))
	} else {
		r.bits = r.bits[:len(r.symbols)]
	}
	for i, s := range r.symbols {
		r.bits[i] = byte(r.ffsk.Slice(s))
	}
	r.bitSink(r.bits, r.bitBase)
	r.bitBase += len(r.bits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the BitSink baseIdx restarts at 0 and the FFSK chain sheds its
// mixer phase + LPF history.
func (r *Receiver) Reset() {
	r.bitBase = 0
	r.ffsk.Reset()
}
