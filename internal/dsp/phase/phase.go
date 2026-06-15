// Package phase holds the small phase-domain helpers the rotation analyses use
// — the native Go equivalents of numpy.unwrap and the per-symbol differential
// phase that makes a π/4-DQPSK constellation's rotation visible. They are
// stdlib-only and operate on plain float64 / complex128 slices.
package phase

import (
	"math"
	"math/cmplx"
)

// Unwrap removes 2π discontinuities from a sequence of phase angles in radians,
// matching numpy.unwrap with the default discontinuity of π: wherever the step
// between consecutive samples exceeds π in magnitude, a multiple of 2π is added
// to bring it back into (-π, π]. The first sample is left untouched. Returns a
// new slice; the input is not modified.
func Unwrap(p []float64) []float64 {
	out := make([]float64, len(p))
	copy(out, p)
	if len(p) < 2 {
		return out
	}
	correction := 0.0
	for i := 1; i < len(p); i++ {
		d := p[i] - p[i-1]
		// Remainder maps d into [-π, π]; nudge the -π boundary to +π for a
		// positive jump so the result lies in (-π, π], as numpy does.
		wrapped := math.Remainder(d, 2*math.Pi)
		if wrapped == -math.Pi && d > 0 {
			wrapped = math.Pi
		}
		correction += wrapped - d
		out[i] = p[i] + correction
	}
	return out
}

// FromComplex returns the wrapped instantaneous phase (atan2(Im, Re), in
// (-π, π]) of each sample in z. Returns a new slice.
func FromComplex(z []complex128) []float64 {
	out := make([]float64, len(z))
	for i, v := range z {
		out[i] = math.Atan2(imag(v), real(v))
	}
	return out
}

// Diff returns the per-sample differential phase angle(z[i]·conj(z[i-1])) in
// (-π, π] — the rotation between consecutive samples, i.e. the π/4-DQPSK
// rotation signal. The output has length len(z)-1; it is nil when z has fewer
// than two samples.
func Diff(z []complex128) []float64 {
	if len(z) < 2 {
		return nil
	}
	out := make([]float64, len(z)-1)
	for i := 1; i < len(z); i++ {
		d := z[i] * cmplx.Conj(z[i-1])
		out[i-1] = math.Atan2(imag(d), real(d))
	}
	return out
}
