package dsp

import (
	"math"
	"testing"
)

func TestAGCConvergence(t *testing.T) {
	a := NewAGC(1.0, 5e-3, 1e6)
	in := make([]complex64, 100_000)
	for i := range in {
		// Constant amplitude 0.01 — AGC should drive to ~100x gain.
		in[i] = complex(0.01, 0)
	}
	out := a.Process(nil, in)
	tail := out[len(out)-1024:]
	var avg float64
	for _, s := range tail {
		avg += math.Hypot(float64(real(s)), float64(imag(s)))
	}
	avg /= float64(len(tail))
	if avg < 0.9 || avg > 1.1 {
		t.Errorf("AGC settled magnitude = %f, want ~1.0", avg)
	}
}

func TestResamplerRateRoughlyMatches(t *testing.T) {
	// 3/2 upsample: 1000 in → ~1500 out (after warmup).
	r := NewResampler(3, 2, 8, 8.6)
	in := make([]complex64, 4000)
	for i := range in {
		theta := 2 * math.Pi * 0.05 * float64(i)
		in[i] = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	}
	out := r.Process(nil, in)
	want := len(in) * 3 / 2
	diff := want - len(out)
	if diff < 0 {
		diff = -diff
	}
	if diff > 4 {
		t.Errorf("len(out) = %d, want %d ± 4", len(out), want)
	}
}

func TestResamplerReset(t *testing.T) {
	r := NewResampler(3, 2, 8, 8.6)
	in := make([]complex64, 2000)
	for i := range in {
		theta := 2 * math.Pi * 0.05 * float64(i)
		in[i] = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	}
	first := append([]complex64(nil), r.Process(nil, in)...)

	// After Reset the resampler must reproduce a fresh run bit-for-
	// bit — proving the sample history + commutator state cleared.
	r.Reset()
	second := r.Process(nil, in)

	if len(first) != len(second) {
		t.Fatalf("len after reset = %d, want %d", len(second), len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("sample %d after reset = %v, want %v", i, second[i], first[i])
		}
	}
}
