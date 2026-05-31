package dsp

import (
	"math"
	"math/rand"
	"testing"
)

// EstimateCarrierOffsetHz should recover a planted carrier offset to within
// a few Hz, even buried in additive noise and at a negative offset.
func TestEstimateCarrierOffset(t *testing.T) {
	const fs = 48000.0
	for _, want := range []float64{750, -1234, 3000} {
		rng := rand.New(rand.NewSource(1))
		n := 200000
		iq := make([]complex64, n)
		for i := range iq {
			ph := 2 * math.Pi * want * float64(i) / fs
			noiseI := 0.3 * rng.NormFloat64()
			noiseQ := 0.3 * rng.NormFloat64()
			iq[i] = complex64(complex(math.Cos(ph)+noiseI, math.Sin(ph)+noiseQ))
		}
		got := EstimateCarrierOffsetHz(iq, fs, fs*0.5)
		if math.Abs(got-want) > 5 {
			t.Errorf("offset %.0f Hz: estimated %.1f Hz (err %.1f, want <5)", want, got, got-want)
		}
	}
}

// Degenerate inputs return 0 rather than panicking.
func TestEstimateCarrierOffsetEdgeCases(t *testing.T) {
	if got := EstimateCarrierOffsetHz(nil, 48000, 24000); got != 0 {
		t.Errorf("empty input: got %.3f, want 0", got)
	}
	if got := EstimateCarrierOffsetHz([]complex64{complex(1, 0)}, 48000, 0); got != 0 {
		t.Errorf("zero searchHz: got %.3f, want 0", got)
	}
}
