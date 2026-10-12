package fft

import (
	"fmt"
	"math"
	"math/bits"
)

// Radix2 is an in-place, iterative radix-2 forward FFT on complex64 for
// power-of-two sizes. It exists for throughput, not generality: the gonum
// backend (New) converts every sample to complex128 and walks FFTPACK's
// accessor abstractions, which on a 32-bit Android phone could not keep up
// with a 2.4 MS/s RTL-SDR in `gophertrunk power` (#1230). Radix2 keeps the
// samples in float32, precomputes every twiddle once, and lets the caller
// write its (windowed) input straight into bit-reversed order via Rev, so
// the transform itself is a single pass of butterflies per stage.
//
// Twiddles are computed in float64 and rounded once, so the error is that of
// float32 butterflies: about 1e-6 relative for the sizes a power spectrum
// uses — far below anything a dB display can show.
type Radix2 struct {
	n   int
	rev []int32
	tw  []complex64 // stage h (h = 1, 2, 4, …, n/2) at tw[h-1 : 2h-1]
}

// NewRadix2 plans an n-point transform; n must be a power of two ≥ 2.
func NewRadix2(n int) (*Radix2, error) {
	if n < 2 || n&(n-1) != 0 {
		return nil, fmt.Errorf("fft: radix-2 size %d is not a power of two ≥ 2", n)
	}
	logN := bits.TrailingZeros(uint(n))
	p := &Radix2{n: n, rev: make([]int32, n), tw: make([]complex64, n-1)}
	for i := range p.rev {
		p.rev[i] = int32(bits.Reverse32(uint32(i)) >> (32 - logN))
	}
	for h := 1; h < n; h <<= 1 {
		for k := 0; k < h; k++ {
			s, c := math.Sincos(-math.Pi * float64(k) / float64(h))
			p.tw[h-1+k] = complex(float32(c), float32(s))
		}
	}
	return p, nil
}

// Size is the transform length.
func (p *Radix2) Size() int { return p.n }

// Rev maps a natural-order input index to its slot in the bit-reversed
// buffer TransformBitReversed expects: write x[i] to buf[Rev()[i]].
func (p *Radix2) Rev() []int32 { return p.rev }

// TransformBitReversed runs the forward DFT, X[k] = Σ x[n]·e^(−2πikn/N), in
// place on x, which must hold the input in bit-reversed order (see Rev). On
// return x holds the spectrum in natural order. It panics if len(x) is not
// the plan size.
func (p *Radix2) TransformBitReversed(x []complex64) {
	n := p.n
	if len(x) != n {
		panic("fft: radix-2 input length != plan size")
	}
	// Stage 1: every twiddle is 1.
	for i := 0; i+1 < n; i += 2 {
		a, b := x[i], x[i+1]
		x[i], x[i+1] = a+b, a-b
	}
	if n == 2 {
		return
	}
	// Stage 2: twiddles 1 and −i; multiplying by −i swaps the parts.
	for i := 0; i+3 < n; i += 4 {
		a0, a1, b0, b1 := x[i], x[i+1], x[i+2], x[i+3]
		t1 := complex(imag(b1), -real(b1))
		x[i], x[i+2] = a0+b0, a0-b0
		x[i+1], x[i+3] = a1+t1, a1-t1
	}
	for h := 4; h < n; h <<= 1 {
		tw := p.tw[h-1 : 2*h-1]
		for start := 0; start < n; start += 2 * h {
			lo := x[start : start+h]
			hi := x[start+h : start+2*h]
			lo, hi = lo[:len(tw)], hi[:len(tw)] // bounds-check elimination
			for k, w := range tw {
				// Spelled out in float32: Go evaluates a complex64
				// product in float64 (to limit cancellation), and the
				// widening/narrowing dominated this loop on 32-bit
				// targets. One rounding per operation is plenty for a
				// power spectrum.
				br, bi := real(hi[k]), imag(hi[k])
				wr, wi := real(w), imag(w)
				tr := br*wr - bi*wi
				ti := br*wi + bi*wr
				ar, ai := real(lo[k]), imag(lo[k])
				lo[k] = complex(ar+tr, ai+ti)
				hi[k] = complex(ar-tr, ai-ti)
			}
		}
	}
}

// Forward writes the DFT of src to dst (both natural order, length Size),
// using dst as the working buffer. src is left unchanged.
func (p *Radix2) Forward(dst, src []complex64) {
	if len(src) != p.n || len(dst) != p.n {
		panic("fft: radix-2 input length != plan size")
	}
	for i, s := range src {
		dst[p.rev[i]] = s
	}
	p.TransformBitReversed(dst)
}
