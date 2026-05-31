package dsp

import "math"

// NCO is a numerically-controlled oscillator for frequency-shifting a
// complex baseband stream — i.e. "tuning" a channel that sits at a
// non-zero offset down to 0 Hz. Mix multiplies each input sample by
// e^{-j·2π·offsetHz·n/Fs}, so a spectral component at +offsetHz lands at
// DC.
//
// The phasor is generated recursively (one complex multiply per sample)
// rather than via a Cos/Sin call. Repeated float32 multiplies let the
// magnitude drift off the unit circle, so renorm folds it back every
// renormEveryN samples — one sqrt amortised over thousands of samples,
// well below the surrounding numeric noise floor.
//
// This mirrors the private oscillator inside internal/dsp/tuner's DDC
// bank, exported here as a standalone primitive the replay path and the
// single-channel down-converter (ccdecoder.Downconverter) reuse to tune
// an off-centre recorded capture without standing up a whole tuner bank.
type NCO struct {
	phasor       complex64
	step         complex64
	sinceRenorm  int
	renormEveryN int
}

// NewNCO returns an oscillator that shifts a component at +offsetHz down
// to DC at the given sample rate. A zero offset yields an identity mix.
func NewNCO(offsetHz, sampleRateHz float64) *NCO {
	n := &NCO{phasor: complex(1, 0), renormEveryN: 4096}
	n.SetOffset(offsetHz, sampleRateHz)
	return n
}

// SetOffset retunes the oscillator. The phasor (current phase) is left
// untouched so a mid-stream retune stays phase-continuous.
func (n *NCO) SetOffset(offsetHz, sampleRateHz float64) {
	theta := -2 * math.Pi * offsetHz / sampleRateHz
	n.step = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
}

// Reset returns the phasor to 1+0j (zero phase).
func (n *NCO) Reset() {
	n.phasor = complex(1, 0)
	n.sinceRenorm = 0
}

// Mix writes the frequency-shifted src into dst (reused when it has
// capacity) and returns it. src is not read after the matching dst
// element is written, so in-place operation (dst aliasing src) is safe.
func (n *NCO) Mix(dst, src []complex64) []complex64 {
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, x := range src {
		dst[i] = x * n.phasor
		n.phasor *= n.step
		n.sinceRenorm++
		if n.sinceRenorm >= n.renormEveryN {
			n.renorm()
		}
	}
	return dst
}

func (n *NCO) renorm() {
	r := float64(real(n.phasor))
	im := float64(imag(n.phasor))
	mag := math.Sqrt(r*r + im*im)
	if mag > 0 {
		n.phasor = complex(float32(r/mag), float32(im/mag))
	} else {
		n.phasor = complex(1, 0)
	}
	n.sinceRenorm = 0
}
