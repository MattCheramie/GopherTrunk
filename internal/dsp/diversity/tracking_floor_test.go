package diversity

import (
	"math"
	"math/rand"
	"testing"
)

// TestTrackingAppliedGainMagnitudeFloor is the regression for the 19 Aug field
// log's branch_gain_db=-34.3 with calibrated=true: the divergence bounds gate
// only the per-window PROPOSAL, while the smoothed APPLIED gain is a vector
// blend that can random-walk its magnitude below the -30 dB floor — at which
// point the second branch contributes nothing and the combiner has silently
// degenerated to single-branch. The applied gain must never sit below the
// divergence floor after an accepted update (and never above the ceiling).
func TestTrackingAppliedGainMagnitudeFloor(t *testing.T) {
	const window = 4096
	rng := rand.New(rand.NewSource(505))
	ref := genSignal(rng, window)
	x := addNoise(rng, rotate(ref, 0.4), 0.2)

	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 0.2})
	c.Observe([][]complex64{ref, x})
	if !c.Calibrated() {
		t.Fatal("did not calibrate on clean data")
	}

	// Poison the applied weight to -40 dB — the state a long adverse-phase
	// random walk reaches — keeping its phase.
	ph := math.Atan2(float64(imag(c.applied[1])), float64(real(c.applied[1])))
	c.applied[1] = complex(float32(0.01*math.Cos(ph)), float32(0.01*math.Sin(ph)))

	// One accepted good window must pull it back inside the divergence bounds
	// immediately — the floor overrides the per-step ratio clamp, which alone
	// would need many windows to climb 10 dB.
	ref2 := genSignal(rng, window)
	x2 := addNoise(rng, rotate(ref2, 0.4), 0.2)
	res := c.Observe([][]complex64{ref2, x2})
	if !res.Updated {
		t.Fatal("clean window was not accepted")
	}
	g := c.Gains()[1]
	magSq := float64(real(g))*float64(real(g)) + float64(imag(g))*float64(imag(g))
	if magSq < trackingDivergeFloorSq {
		t.Errorf("applied |gain| = %.1f dB after an accepted update, want >= %.1f dB (the divergence floor)",
			10*math.Log10(magSq), 10*math.Log10(trackingDivergeFloorSq))
	}
	if magSq > trackingDivergeCeilSq {
		t.Errorf("applied |gain| = %.1f dB above the divergence ceiling", 10*math.Log10(magSq))
	}
}
