// Package fft provides a swappable FFT abstraction. The default backend wraps
// gonum's fourier.CmplxFFT; future backends (FFTW, hand-rolled SIMD) can
// implement the Plan interface without disturbing call sites.
package fft

import (
	"gonum.org/v1/gonum/dsp/fourier"
)

type Plan interface {
	Size() int
	Forward(out, in []complex128) []complex128
	Inverse(out, in []complex128) []complex128
}

type gonumPlan struct {
	n   int
	fft *fourier.CmplxFFT
}

func New(n int) Plan {
	return &gonumPlan{n: n, fft: fourier.NewCmplxFFT(n)}
}

func (p *gonumPlan) Size() int { return p.n }

func (p *gonumPlan) Forward(out, in []complex128) []complex128 {
	if len(in) != p.n {
		panic("fft: input length != plan size")
	}
	if cap(out) < p.n {
		out = make([]complex128, p.n)
	} else {
		out = out[:p.n]
	}
	res := p.fft.Coefficients(out, in)
	return res
}

func (p *gonumPlan) Inverse(out, in []complex128) []complex128 {
	if len(in) != p.n {
		panic("fft: input length != plan size")
	}
	if cap(out) < p.n {
		out = make([]complex128, p.n)
	} else {
		out = out[:p.n]
	}
	res := p.fft.Sequence(out, in)
	// gonum's inverse returns N*x; normalize.
	scale := complex(1/float64(p.n), 0)
	for i := range res {
		res[i] *= scale
	}
	return res
}

// Cmplx64ToCmplx128 / Cmplx128ToCmplx64 are convenience converters for the
// IQ pipeline (which is complex64) at FFT boundaries (gonum is complex128).
func Cmplx64ToCmplx128(dst []complex128, src []complex64) []complex128 {
	if cap(dst) < len(src) {
		dst = make([]complex128, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		dst[i] = complex(float64(real(s)), float64(imag(s)))
	}
	return dst
}

func Cmplx128ToCmplx64(dst []complex64, src []complex128) []complex64 {
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		dst[i] = complex(float32(real(s)), float32(imag(s)))
	}
	return dst
}
