package equalizer

import (
	"math"
	"testing"
)

// complexChannel applies a 2-tap multipath with a genuinely COMPLEX echo
// coefficient: y[n] = x[n] + alpha·x[n-1], alpha ∈ ℂ with imag ≠ 0. This is
// the case the original real-only TestLMSConvergesOnTwoTapChannel could not
// exercise — and the case that exposed the conjugation sign error in the LMS
// weight update (e·conj(x), not x·conj(e)).
func complexChannel(x []complex64, alpha complex64) []complex64 {
	y := make([]complex64, len(x))
	for i := range x {
		y[i] = x[i]
		if i > 0 {
			y[i] += alpha * x[i-1]
		}
	}
	return y
}

// TestLMSConvergesOnComplexChannel is the regression guard for the complex-LMS
// conjugation fix. With a complex echo coefficient the wrong-sign update
// (w += μ·x·conj(e)) diverges to NaN; the correct update (w += μ·e·conj(x))
// converges. We train on known symbols and require the post-convergence MSE to
// be both finite and small.
func TestLMSConvergesOnComplexChannel(t *testing.T) {
	const n = 8000
	tx := genQPSK(n, 99)
	rx := complexChannel(tx, complex(0.5, 0.35)) // complex echo: imag ≠ 0

	eq := NewLMS(11, 0.02)
	const D = 11 / 2 // centre-tap delay: align desired to the FIR group delay

	// Train.
	for i := 0; i < n-1000; i++ {
		var desired complex64
		if i-D >= 0 {
			desired = tx[i-D]
		}
		eq.Process(rx[i], desired)
	}

	// Taps must be finite (the buggy update diverged to NaN here).
	for i, tap := range eq.Taps() {
		if math.IsNaN(float64(real(tap))) || math.IsNaN(float64(imag(tap))) ||
			math.IsInf(float64(real(tap)), 0) || math.IsInf(float64(imag(tap)), 0) {
			t.Fatalf("tap[%d] diverged to %v — complex LMS update is unstable", i, tap)
		}
	}

	// Post-convergence MSE on a held-out window (no further updates: step 0).
	eq.SetStepSize(0)
	var mse float64
	cnt := 0
	for i := n - 1000; i < n; i++ {
		y, _ := eq.Process(rx[i], 0)
		if i-D < 0 {
			continue
		}
		dr := float64(real(tx[i-D]) - real(y))
		di := float64(imag(tx[i-D]) - imag(y))
		mse += dr*dr + di*di
		cnt++
	}
	mse /= float64(cnt)
	if mse > 0.05 {
		t.Errorf("complex-channel post-convergence MSE = %g, want < 0.05", mse)
	}
}
