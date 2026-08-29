// Package receiver wires the IQ → FSK bit chain that feeds the
// Motorola Type II / SmartZone control-channel framer.
//
//	IQ samples (channelized, SampleRateHz)
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → slow DC tracker (carrier-offset removal)
//	  → boxcar symbol filter (one symbol wide)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync)
//	  → zero-threshold slicer → motorola.BitSink
//
// The SmartNet control channel is 3600-baud binary FSK with ~±1.2 kHz
// deviation. This chain mirrors trunk-recorder's proven
// smartnet_fsk2_demod (PLL frequency detector → AGC → one-symbol
// averaging filter → two-level symbol tracker → binary slicer): the
// FM discriminator is the open-loop equivalent of its PLL frequency
// detector, and the DC tracker stands in for the PLL's carrier
// pull-in — at ±1.2 kHz deviation a few hundred hertz of residual
// carrier offset is a large slicer bias, so it cannot be ignored the
// way wider-deviation protocols can. The slicer threshold is zero and
// the M&M timing error is amplitude-normalised, so no AGC is needed.
//
// The receiver emits raw bits (each byte is 0 or 1) via
// motorola.BitSink. The downstream framing — 8-bit sync, 76-bit
// interleaved payload, convolutional-parity ECC, CRC-10 — lives in
// the parent motorola package (frame.go / process.go).
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
	// DeviationHz is the nominal FSK deviation (±), for modulators
	// and documentation; the receive chain is deviation-agnostic.
	DeviationHz = 1200.0
	// dcTrackSeconds is the time constant of the discriminator DC
	// tracker. Long against the 84-bit / ~23 ms frame so NRZ data
	// content is untouched, short enough to pull in oscillator
	// drift within a couple hundred milliseconds.
	dcTrackSeconds = 0.1
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after upstream
	// channelization. Required; must be ≥ 2 × SymbolRate (7200 Hz).
	// The production DDC delivers 18 kHz (5 samples/symbol).
	SampleRateHz float64
	// BitSink receives the raw bit stream the receiver decodes
	// from IQ. Required.
	BitSink motorola.BitSink
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64
}

// Receiver is the composed IQ → bit pipeline.
type Receiver struct {
	fm      *demod.FM
	clock   *sync.MuellerMuller
	bitSink motorola.BitSink
	bitBase int

	// DC tracker state: one-pole mean estimate of the discriminator
	// output (the carrier-offset term), subtracted before filtering.
	dcAlpha float32
	dcMean  float32

	// Boxcar matched filter: N-sample moving average.
	boxTaps int
	boxHist []float32
	boxPos  int
	boxSum  float32

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
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}
	taps := int(sps + 0.5)
	return &Receiver{
		fm:      demod.NewFM(),
		clock:   sync.NewMuellerMuller(sps, gain),
		bitSink: opts.BitSink,
		dcAlpha: float32(1.0 / (dcTrackSeconds * opts.SampleRateHz)),
		boxTaps: taps,
		boxHist: make([]float32, taps),
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
	// Remove the carrier-offset DC term with a slow one-pole
	// tracker, then apply the one-symbol boxcar matched filter.
	if cap(r.matched) < len(r.disc) {
		r.matched = make([]float32, len(r.disc))
	} else {
		r.matched = r.matched[:len(r.disc)]
	}
	inv := 1.0 / float32(r.boxTaps)
	for i, x := range r.disc {
		r.dcMean += r.dcAlpha * (x - r.dcMean)
		x -= r.dcMean
		r.boxSum += x - r.boxHist[r.boxPos]
		r.boxHist[r.boxPos] = x
		r.boxPos++
		if r.boxPos == r.boxTaps {
			r.boxPos = 0
		}
		r.matched[i] = r.boxSum * inv
	}
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
// the BitSink baseIdx restarts at 0 and the filters shed their
// history.
func (r *Receiver) Reset() {
	r.bitBase = 0
	r.dcMean = 0
	r.boxSum = 0
	r.boxPos = 0
	for i := range r.boxHist {
		r.boxHist[i] = 0
	}
}
