package filter

import (
	"math"
	"testing"
	"time"
)

func TestDeEmphasisDCPassThrough(t *testing.T) {
	// A constant-DC input must converge to the same DC output. The
	// filter's DC gain is exactly unity by construction.
	d := NewDeEmphasis(DeEmphasis75us, 48_000)
	in := make([]float32, 4096)
	for i := range in {
		in[i] = 0.5
	}
	out := d.Process(nil, in)
	last := out[len(out)-1]
	if math.Abs(float64(last-0.5)) > 1e-3 {
		t.Errorf("DC steady-state = %f, want 0.5", last)
	}
}

func TestDeEmphasisStepResponseMonotonic(t *testing.T) {
	d := NewDeEmphasis(DeEmphasis75us, 48_000)
	in := make([]float32, 1024)
	for i := range in {
		in[i] = 1.0
	}
	out := d.Process(nil, in)
	for i := 1; i < len(out); i++ {
		if out[i] < out[i-1] {
			t.Fatalf("step response non-monotonic at i=%d: %f → %f", i, out[i-1], out[i])
		}
	}
	if out[len(out)-1] < 0.99 {
		t.Errorf("step response did not converge: last sample = %f", out[len(out)-1])
	}
}

func TestDeEmphasisAttenuatesHighFreq(t *testing.T) {
	// At 75 µs the −3 dB cutoff is fc = 1 / (2π × 75e-6) ≈ 2.122 kHz.
	// A 100 Hz sinusoid is well below cutoff and should pass with
	// near-unit gain; a 6 kHz sinusoid is well above and should drop
	// by ≥ 8 dB. We measure the ratio of mean-squared output to mean-
	// squared input over the steady-state portion of the response.
	const fs = 48_000.0
	const settling = 4096 // samples to skip while the IIR settles

	gain := func(freq float64) float64 {
		d := NewDeEmphasis(DeEmphasis75us, fs)
		n := settling + 8192
		in := make([]float32, n)
		for i := range in {
			in[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / fs))
		}
		out := d.Process(nil, in)
		var inE, outE float64
		for i := settling; i < n; i++ {
			inE += float64(in[i]) * float64(in[i])
			outE += float64(out[i]) * float64(out[i])
		}
		return math.Sqrt(outE / inE)
	}

	low := gain(100)
	high := gain(6_000)
	if low < 0.97 || low > 1.01 {
		t.Errorf("100 Hz gain = %.3f, want ≈1.0", low)
	}
	if high > 0.4 {
		t.Errorf("6 kHz gain = %.3f, want < 0.4 (≥ 8 dB attenuation)", high)
	}
	if high >= low {
		t.Errorf("expected high freq attenuated more than low: low=%.3f high=%.3f", low, high)
	}
}

func TestDeEmphasisReset(t *testing.T) {
	d := NewDeEmphasis(DeEmphasis75us, 48_000)
	in := make([]float32, 1024)
	for i := range in {
		in[i] = 1.0
	}
	d.Process(nil, in)
	d.Reset()
	out := d.Process(nil, []float32{0})
	if out[0] != 0 {
		t.Errorf("after Reset, first output for x=0 should be 0, got %f", out[0])
	}
}

func TestDeEmphasisInPlace(t *testing.T) {
	d := NewDeEmphasis(DeEmphasis75us, 48_000)
	buf := []float32{1, 1, 1, 1}
	out := d.Process(buf, buf)
	if &out[0] != &buf[0] {
		t.Error("Process(buf, buf) should reuse buf for dst")
	}
	if out[0] == 1 {
		t.Errorf("step input transient should be < 1 at first sample, got %f", out[0])
	}
}

func TestDeEmphasisSampleRateCoeffs(t *testing.T) {
	// Half the sample rate → α moves toward 0; oneMinus toward 1.
	// The point of the check is to make sure the constructor actually
	// uses fs (a regression guard), not to pin α to a specific value.
	d48 := NewDeEmphasis(DeEmphasis75us, 48_000)
	d24 := NewDeEmphasis(DeEmphasis75us, 24_000)
	if d24.alpha >= d48.alpha {
		t.Errorf("alpha at 24 kHz (%f) should be < alpha at 48 kHz (%f)", d24.alpha, d48.alpha)
	}
}

func TestDeEmphasisRejectsBadParams(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-positive time constant")
		}
	}()
	_ = NewDeEmphasis(0, 48_000)
}

func TestDeEmphasisRejectsZeroSampleRate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-positive sample rate")
		}
	}()
	_ = NewDeEmphasis(time.Millisecond, 0)
}

func TestDeEmphasisUSAlias(t *testing.T) {
	a := NewDeEmphasisUS(48_000)
	b := NewDeEmphasis(DeEmphasis75us, 48_000)
	if a.alpha != b.alpha || a.oneMinus != b.oneMinus {
		t.Errorf("US alias coefficients differ from explicit 75µs construction: %+v vs %+v", a, b)
	}
}
