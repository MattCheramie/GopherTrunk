package fft

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	const n = 256
	p := New(n)
	in := make([]complex128, n)
	for i := range in {
		in[i] = complex(math.Sin(2*math.Pi*float64(i)*5/n), 0)
	}
	freq := p.Forward(nil, in)
	back := p.Inverse(nil, freq)
	for i := range in {
		if cmplx.Abs(back[i]-in[i]) > 1e-9 {
			t.Errorf("round-trip mismatch at %d: %v vs %v", i, back[i], in[i])
		}
	}
}

func TestTonePeak(t *testing.T) {
	const n = 512
	const bin = 17
	p := New(n)
	in := make([]complex128, n)
	for i := range in {
		theta := 2 * math.Pi * float64(bin) * float64(i) / float64(n)
		in[i] = complex(math.Cos(theta), math.Sin(theta))
	}
	freq := p.Forward(nil, in)
	var maxIdx int
	var maxMag float64
	for i, c := range freq {
		m := cmplx.Abs(c)
		if m > maxMag {
			maxMag = m
			maxIdx = i
		}
	}
	if maxIdx != bin {
		t.Errorf("peak at bin %d, want %d", maxIdx, bin)
	}
}

func TestC64Roundtrip(t *testing.T) {
	src := []complex64{1 + 2i, 3 + 4i, 5 + 6i}
	mid := Cmplx64ToCmplx128(nil, src)
	back := Cmplx128ToCmplx64(nil, mid)
	for i := range src {
		if back[i] != src[i] {
			t.Errorf("[%d] %v != %v", i, back[i], src[i])
		}
	}
}
