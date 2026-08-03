package ccdecoder

import "testing"

// TestBucketErrorRate pins the shared clean/marginal/poor thresholds the live
// signal-quality indicator uses for both P25 (TSBK) and TETRA (BSCH) error rates.
func TestBucketErrorRate(t *testing.T) {
	cases := []struct {
		rate float64
		want string
	}{
		{0, "clean"},
		{ccQualityCleanMaxPct, "clean"}, // boundary ≤ 1.0
		{ccQualityCleanMaxPct + 0.01, "marginal"},
		{ccQualityMarginalMaxPct, "marginal"}, // boundary ≤ 5.0
		{ccQualityMarginalMaxPct + 0.01, "poor"},
		{100, "poor"},
	}
	for _, c := range cases {
		if got := bucketErrorRate(c.rate); got != c.want {
			t.Errorf("bucketErrorRate(%.2f) = %q, want %q", c.rate, got, c.want)
		}
	}
}
