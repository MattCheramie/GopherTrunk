package fft

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// Radix2 must match the gonum backend (the package's reference transform)
// to float32 precision at every size a power spectrum uses, and match a
// direct DFT at small sizes so the reference itself is not assumed.
func TestRadix2MatchesReference(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for logN := 1; logN <= 14; logN++ {
		n := 1 << logN
		p, err := NewRadix2(n)
		if err != nil {
			t.Fatal(err)
		}
		src := make([]complex64, n)
		ref := make([]complex128, n)
		for i := range src {
			src[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
			ref[i] = complex128(src[i])
		}
		want := New(n).Forward(nil, ref)
		if n <= 64 {
			for k := range want { // direct DFT
				var s complex128
				for j, x := range ref {
					s += x * cmplx.Exp(complex(0, -2*math.Pi*float64(k*j)/float64(n)))
				}
				if cmplx.Abs(s-want[k]) > 1e-9*float64(n) {
					t.Fatalf("n=%d: gonum reference disagrees with the DFT at bin %d", n, k)
				}
			}
		}
		got := make([]complex64, n)
		p.Forward(got, src)
		var errPow, refPow float64
		for k := range want {
			d := complex128(got[k]) - want[k]
			errPow += real(d)*real(d) + imag(d)*imag(d)
			refPow += real(want[k])*real(want[k]) + imag(want[k])*imag(want[k])
		}
		if rel := math.Sqrt(errPow / refPow); rel > 1e-5 {
			t.Errorf("n=%d: relative RMS error %.2e against the reference, want < 1e-5", n, rel)
		}
	}
}

func TestNewRadix2RejectsBadSizes(t *testing.T) {
	for _, n := range []int{0, 1, 3, 6, 1000} {
		if _, err := NewRadix2(n); err == nil {
			t.Errorf("NewRadix2(%d) accepted a size that is not a power of two ≥ 2", n)
		}
	}
}

func BenchmarkForward1024(b *testing.B) {
	const n = 1024
	r := rand.New(rand.NewSource(1))
	src := make([]complex64, n)
	src128 := make([]complex128, n)
	for i := range src {
		src[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
		src128[i] = complex128(src[i])
	}
	b.Run("gonum", func(b *testing.B) {
		p, out := New(n), make([]complex128, n)
		b.SetBytes(n)
		for i := 0; i < b.N; i++ {
			p.Forward(out, src128)
		}
	})
	b.Run("radix2", func(b *testing.B) {
		p, _ := NewRadix2(n)
		out := make([]complex64, n)
		b.SetBytes(n)
		for i := 0; i < b.N; i++ {
			p.Forward(out, src)
		}
	})
}
