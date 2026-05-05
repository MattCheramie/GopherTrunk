// Package demod contains baseband demodulators that convert IQ streams into
// real-valued symbol streams (or audio, for FM).
package demod

import "math"

// FM is a quadrature FM discriminator. Output is the instantaneous phase
// derivative computed via arg(z[n] * conj(z[n-1])), scaled into the range
// [-pi, +pi] radians/sample. Multiply by Fs / (2π * Δf) to get audio.
type FM struct {
	last complex64
}

func NewFM() *FM { return &FM{last: complex(1, 0)} }

func (f *FM) Process(dst []float32, src []complex64) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		// z[n] * conj(z[n-1]).
		ar := real(s)*real(f.last) + imag(s)*imag(f.last)
		ai := imag(s)*real(f.last) - real(s)*imag(f.last)
		dst[i] = float32(math.Atan2(float64(ai), float64(ar)))
		f.last = s
	}
	return dst
}
