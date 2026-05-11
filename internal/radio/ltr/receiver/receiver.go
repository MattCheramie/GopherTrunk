// Package receiver wires the IQ → sub-audible bit chain that feeds
// the LTR per-repeater state machine. LTR transmits a 41-bit Status
// word at 300 baud underneath the in-band voice — sub-audible (the
// bulk of the signal energy sits below 300 Hz), so the receiver
// extracts it via FM demod + a narrow low-pass filter, then runs
// symbol-rate clock recovery + 2-level slicing.
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → real audio
//	  → narrow lowpass (sub-audible band, ~300 Hz cutoff)
//	  → Mueller-Müller symbol clock recovery at 300 baud
//	  → 2-level slicer at zero
//	  → ltr.BitSink
//
// The receiver emits raw bits (each byte is 0 or 1) via
// ltr.BitSink. Manchester decoding (if the system uses it) plus the
// 41-bit Status-word framing live in a future
// ControlChannel.Process adapter and aren't wired by this package.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ltr"
)

// LTR on-air parameters.
const (
	// SymbolRate is the on-air bit rate of the LTR Status-word
	// channel. Manchester-encoded systems double the transition
	// rate but the symbol clock here is the underlying bit clock.
	SymbolRate = 300.0
	// LPFCutoffHz is the default sub-audible cutoff. The Status
	// word's energy sits below 300 Hz; everything above belongs to
	// the voice and should be rejected.
	LPFCutoffHz = 300.0
	// LPFLen is the default Kaiser FIR length for the sub-audible
	// LPF. 101 taps at 48 kHz gives a transition width of ~430 Hz
	// (β = 8.6, ~85 dB stopband) so the voice spectrum above the
	// cutoff is heavily attenuated.
	LPFLen = 101
	// LPFBeta is the Kaiser β for the LPF.
	LPFBeta = 8.6
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate.
	SampleRateHz float64
	// BitSink receives the raw bit stream the receiver decodes
	// from IQ. Required.
	BitSink ltr.BitSink
	// LPFCutoffHz overrides the sub-audible cutoff. <= 0 uses
	// LPFCutoffHz.
	LPFCutoffHz float64
	// LPFLen overrides the LPF length. <= 0 uses LPFLen.
	LPFLen int
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
}

// Receiver is the composed IQ → bit pipeline.
type Receiver struct {
	fm      *demod.FM
	lpf     *filter.RealFIR
	clock   *sync.MuellerMuller
	bitSink ltr.BitSink
	bitBase int

	disc    []float32
	low     []float32
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
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (600 Hz)")
	}

	cutoff := opts.LPFCutoffHz
	if cutoff <= 0 {
		cutoff = LPFCutoffHz
	}
	n := opts.LPFLen
	if n <= 0 {
		n = LPFLen
	}
	if n%2 == 0 {
		n++
	}
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}

	lpfTaps := filter.LowpassKaiser(n, cutoff/opts.SampleRateHz, LPFBeta)

	return &Receiver{
		fm:      demod.NewFM(),
		lpf:     filter.NewRealFIR(lpfTaps),
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
	r.low = r.lpf.Process(r.low, r.disc)
	r.symbols = r.clock.Process(r.symbols, r.low)
	if len(r.symbols) == 0 {
		return
	}
	if cap(r.bits) < len(r.symbols) {
		r.bits = make([]byte, len(r.symbols))
	} else {
		r.bits = r.bits[:len(r.symbols)]
	}
	for i, s := range r.symbols {
		if s > 0 {
			r.bits[i] = 1
		} else {
			r.bits[i] = 0
		}
	}
	r.bitSink(r.bits, r.bitBase)
	r.bitBase += len(r.bits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the BitSink baseIdx restarts at 0 and the LPF sheds its history.
func (r *Receiver) Reset() {
	r.bitBase = 0
	r.lpf.Reset()
}
