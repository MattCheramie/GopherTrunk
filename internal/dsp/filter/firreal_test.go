package filter

import (
	"math"
	"testing"
)

func TestRealFIRDelta(t *testing.T) {
	// A delta input through a unit-tap filter must reproduce the delta.
	f := NewRealFIR([]float32{1})
	out := f.Process(nil, []float32{1, 0, 0, 0})
	if out[0] != 1 || out[1] != 0 || out[2] != 0 || out[3] != 0 {
		t.Errorf("unit-tap delta response = %v", out)
	}
}

func TestRealFIRImpulseMatchesTaps(t *testing.T) {
	// Impulse response of an N-tap FIR is the tap sequence itself.
	taps := []float32{0.1, 0.2, 0.3, 0.2, 0.1}
	f := NewRealFIR(taps)
	in := make([]float32, len(taps)+4)
	in[0] = 1
	out := f.Process(nil, in)
	for i, want := range taps {
		if math.Abs(float64(out[i]-want)) > 1e-6 {
			t.Errorf("impulse[%d] = %f, want %f", i, out[i], want)
		}
	}
	for i := len(taps); i < len(in); i++ {
		if math.Abs(float64(out[i])) > 1e-6 {
			t.Errorf("impulse tail [%d] = %f, want 0", i, out[i])
		}
	}
}

func TestRealFIRLPFKaiserPassbandStopband(t *testing.T) {
	// 81-tap Kaiser LPF with fc = 0.1 (normalized, so 4.8 kHz at fs =
	// 48 kHz). A 1 kHz sinusoid should pass with near-unit gain; a 12
	// kHz sinusoid should be heavily attenuated.
	const fs = 48_000.0
	taps := LowpassKaiser(81, 4_800.0/fs, 8.6)
	settling := len(taps) * 2

	gainAt := func(freq float64) float64 {
		f := NewRealFIR(taps)
		n := settling + 4096
		in := make([]float32, n)
		for i := range in {
			in[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / fs))
		}
		out := f.Process(nil, in)
		var inE, outE float64
		for i := settling; i < n; i++ {
			inE += float64(in[i]) * float64(in[i])
			outE += float64(out[i]) * float64(out[i])
		}
		return math.Sqrt(outE / inE)
	}

	pass := gainAt(1_000)
	stop := gainAt(12_000)
	if pass < 0.95 || pass > 1.05 {
		t.Errorf("1 kHz passband gain = %.3f, want ≈1.0", pass)
	}
	if stop > 0.05 {
		t.Errorf("12 kHz stopband gain = %.3f, want < 0.05 (≥ 26 dB)", stop)
	}
}

func TestRealFIRReset(t *testing.T) {
	f := NewRealFIR([]float32{0.5, 0.5})
	f.Process(nil, []float32{1, 1, 1, 1})
	f.Reset()
	out := f.Process(nil, []float32{1})
	// First sample after reset, with [0.5, 0.5] taps and zero history,
	// should be 0.5 — not affected by the prior call's residue.
	if math.Abs(float64(out[0]-0.5)) > 1e-6 {
		t.Errorf("post-reset first sample = %f, want 0.5", out[0])
	}
}

func TestRealFIRInPlace(t *testing.T) {
	f := NewRealFIR([]float32{1})
	buf := []float32{1, 2, 3}
	out := f.Process(buf, buf)
	if &out[0] != &buf[0] {
		t.Error("Process(buf, buf) should reuse the slice")
	}
}

func TestRealFIRRejectsEmptyTaps(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty taps")
		}
	}()
	_ = NewRealFIR(nil)
}
