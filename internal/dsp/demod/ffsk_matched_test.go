package demod

import (
	"math"
	"testing"
)

// TestFFSKMatchedFilterPassthroughWhenDisabled: MatchedFilter must be a safe
// no-op copy until EnableMatchedFilter is called, so the many NewFFSK callers
// that don't opt in (APRS / DSC / MDC1200 AFSK) are unaffected.
func TestFFSKMatchedFilterPassthroughWhenDisabled(t *testing.T) {
	f := NewFFSK(48000, 1200, 1800)
	src := []float32{0.5, -0.3, 0.9, -1.0, 0.1}
	out := f.MatchedFilter(nil, src)
	if len(out) != len(src) {
		t.Fatalf("len(out) = %d, want %d", len(out), len(src))
	}
	for i := range src {
		if out[i] != src[i] {
			t.Errorf("out[%d] = %v, want passthrough %v", i, out[i], src[i])
		}
	}
}

// TestFFSKMatchedFilterUnityDCGain: a sustained level must pass through the
// boxcar unchanged once the ring buffer fills (unity DC gain), so the
// zero-threshold slicer and soft magnitudes keep their scale.
func TestFFSKMatchedFilterUnityDCGain(t *testing.T) {
	f := NewFFSK(48000, 1200, 1800)
	const taps = 40
	f.EnableMatchedFilter(taps)
	src := make([]float32, 200)
	for i := range src {
		src[i] = 0.7
	}
	out := f.MatchedFilter(nil, src)
	// After the first `taps` samples the window is full of 0.7 → output 0.7.
	for i := taps; i < len(out); i++ {
		if math.Abs(float64(out[i]-0.7)) > 1e-4 {
			t.Fatalf("out[%d] = %v, want ~0.7 (unity DC gain)", i, out[i])
		}
	}
}
