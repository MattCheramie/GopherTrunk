package receiver

import (
	"math"
	"testing"
)

// TestC4FMSymbolAGCNormalisesLevel pins the symbol-AGC that lets the
// receiver decode real captures: the RRC matched filter is unit-energy
// (DC gain ~3.1), so on a real FM-discriminator stream the 4-level
// symbol centres land well above the slicer's fixed thresholds and the
// inner symbols collapse onto the outer rails. The AGC must pull the
// running mean|x| back to its target (slicerScale·2/3) regardless of the
// absolute input level, while preserving the relative 4-level structure
// the slicer decides on.
func TestC4FMSymbolAGCNormalisesLevel(t *testing.T) {
	const target = 0.17
	a := c4fmSymbolAGC{target: target, rate: 1.0 / 256.0}

	// A balanced 4-level stream at ~3.1× the level the slicer expects —
	// the over-scale a real matched-filter output carries.
	const scale = 3.1
	levels := []float32{scale * target, scale * target / 3, -scale * target / 3, -scale * target}
	buf := make([]float32, 4096)
	for i := range buf {
		buf[i] = levels[i%4]
	}
	a.process(buf)

	// After the EMA settles, mean|x| over the tail should track target.
	var sum float64
	tail := buf[len(buf)-1024:]
	for _, x := range tail {
		sum += math.Abs(float64(x))
	}
	got := sum / float64(len(tail))
	// The AGC drives the running mean|x| of its output to target, so the
	// settled tail mean|x| should be ≈ target regardless of input scale.
	if math.Abs(got-target) > 0.02*target {
		t.Errorf("settled mean|x| = %.4f, want ≈ %.4f (AGC not normalising)", got, target)
	}

	// Relative structure preserved: outer/inner magnitude ratio stays 3:1.
	outer := math.Abs(float64(tail[0]))
	inner := math.Abs(float64(tail[1]))
	if r := outer / inner; math.Abs(r-3.0) > 0.05 {
		t.Errorf("outer/inner ratio = %.3f, want ≈ 3.0 (AGC distorted the eye)", r)
	}
}

// TestC4FMSymbolAGCDisabled confirms the legacy pre-scaled-fixture path
// (target<=0) is a no-op so existing fixtures stay byte-identical.
func TestC4FMSymbolAGCDisabled(t *testing.T) {
	a := c4fmSymbolAGC{target: 0}
	buf := []float32{0.5, -0.3, 1.2, -0.9}
	want := append([]float32(nil), buf...)
	a.process(buf)
	for i := range buf {
		if buf[i] != want[i] {
			t.Errorf("buf[%d] = %v, want %v (disabled AGC must be a no-op)", i, buf[i], want[i])
		}
	}
}
