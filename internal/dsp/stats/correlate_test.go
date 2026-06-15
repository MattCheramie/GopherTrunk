package stats

import (
	"math"
	"testing"
)

func TestCorrelateAutocorrPeak(t *testing.T) {
	a := []float64{1, -2, 3, 0, 1}
	corr, lag := Correlate(a, a)
	if lag != 0 {
		t.Errorf("autocorrelation peak lag = %d, want 0", lag)
	}
	// Normalized autocorrelation peaks at exactly 1.0 at zero lag.
	peak := corr[len(corr)/2]
	if math.Abs(peak-1.0) > 1e-12 {
		t.Errorf("autocorrelation peak = %v, want 1.0", peak)
	}
	for _, v := range corr {
		if v > peak+1e-12 {
			t.Errorf("found correlation %v exceeding the zero-lag peak %v", v, peak)
		}
	}
}

func TestCorrelateRecoversShift(t *testing.T) {
	// The pattern b sits inside a starting at index 3.
	b := []float64{1, 2, 3}
	a := []float64{0, 0, 0, 1, 2, 3, 0}
	_, lag := Correlate(a, b)
	if lag != 3 {
		t.Errorf("peak lag = %d, want 3 (pattern start index)", lag)
	}
}

func TestCorrelateEmpty(t *testing.T) {
	if c, lag := Correlate(nil, []float64{1}); c != nil || lag != 0 {
		t.Errorf("Correlate(nil,_) = (%v,%v), want (nil,0)", c, lag)
	}
}

func TestCorrelateLength(t *testing.T) {
	corr, _ := Correlate([]float64{1, 2}, []float64{1, 2, 3})
	if len(corr) != 2+3-1 {
		t.Errorf("len(corr) = %d, want %d", len(corr), 2+3-1)
	}
}
