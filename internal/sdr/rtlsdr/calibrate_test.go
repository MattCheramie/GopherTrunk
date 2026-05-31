package rtlsdr

import (
	"math"
	"math/rand"
	"testing"
)

// makeIQ builds a circularly-symmetric complex stream (ideal: E[I²]=E[Q²],
// E[I·Q]=0) and applies a gain+cross-leakage imbalance Q_m = gain·Q + leak·I.
// leak injects an I→Q correlation (a phase-skew stand-in); gain≠1 injects an
// I/Q power imbalance.
func makeIQ(rng *rand.Rand, n int, gain, leak float64) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		ii := rng.NormFloat64()
		qq := rng.NormFloat64()
		out[i] = complex(float32(ii), float32(gain*qq+leak*ii))
	}
	return out
}

func TestIQImbalanceStatsMeasuresInjectedImbalance(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	// Balanced stream reads ~0 imbalance and a high image-rejection.
	var bal IQImbalanceStats
	bal.Observe(makeIQ(rng, 200_000, 1.0, 0.0))
	if db := math.Abs(bal.GainImbalanceDB()); db > 0.1 {
		t.Errorf("balanced gain imbalance = %.3f dB, want ~0", db)
	}
	if deg := math.Abs(bal.PhaseImbalanceDeg()); deg > 0.5 {
		t.Errorf("balanced phase imbalance = %.3f°, want ~0", deg)
	}
	if irr := bal.ImageRejectionDB(); irr < 35 {
		t.Errorf("balanced image rejection = %.1f dB, want ≳ 35", irr)
	}

	// Imbalanced stream (Q hot by gain 1.25, plus I→Q leakage) reads clearly
	// non-zero gain + phase imbalance and a degraded image rejection.
	var imb IQImbalanceStats
	imb.Observe(makeIQ(rng, 200_000, 1.25, 0.12))
	if db := imb.GainImbalanceDB(); db > -1.0 { // Q hotter ⇒ negative
		t.Errorf("imbalanced gain imbalance = %.3f dB, want clearly < -1", db)
	}
	if deg := imb.PhaseImbalanceDeg(); deg < 3 {
		t.Errorf("imbalanced phase imbalance = %.3f°, want clearly > 3", deg)
	}
	if irr := imb.ImageRejectionDB(); irr > 25 {
		t.Errorf("imbalanced image rejection = %.1f dB, want degraded (< 25)", irr)
	}
}

func TestIQImbalanceCorrectorRemovesImbalance(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	const gain, leak = 1.25, 0.12

	// Stats on the raw imbalanced stream, for the before/after comparison.
	var before IQImbalanceStats
	before.Observe(makeIQ(rand.New(rand.NewSource(2)), 300_000, gain, leak))

	c := NewIQImbalanceCorrector()
	var after IQImbalanceStats
	const chunk = 8192
	total := 600_000
	for done := 0; done < total; done += chunk {
		n := chunk
		if done+n > total {
			n = total - done
		}
		buf := makeIQ(rng, n, gain, leak)
		c.Process(buf) // corrects in place
		if done >= total/2 {
			after.Observe(buf) // score the converged tail
		}
	}

	t.Logf("gain imbalance dB: before=%.3f after=%.3f", before.GainImbalanceDB(), after.GainImbalanceDB())
	t.Logf("phase imbalance °: before=%.3f after=%.3f", before.PhaseImbalanceDeg(), after.PhaseImbalanceDeg())
	t.Logf("image rejection dB: before=%.1f after=%.1f", before.ImageRejectionDB(), after.ImageRejectionDB())

	// The corrector must drive the residual I/Q correlation and gain imbalance
	// far toward zero — at least a 4× reduction on each.
	if math.Abs(after.PhaseImbalanceDeg())*4 > math.Abs(before.PhaseImbalanceDeg()) {
		t.Errorf("phase imbalance not reduced ≥4×: before=%.3f° after=%.3f°", before.PhaseImbalanceDeg(), after.PhaseImbalanceDeg())
	}
	if math.Abs(after.GainImbalanceDB())*4 > math.Abs(before.GainImbalanceDB()) {
		t.Errorf("gain imbalance not reduced ≥4×: before=%.3f dB after=%.3f dB", before.GainImbalanceDB(), after.GainImbalanceDB())
	}
	if after.ImageRejectionDB() < 35 {
		t.Errorf("corrected image rejection = %.1f dB, want ≳ 35", after.ImageRejectionDB())
	}

	// A balanced stream must pass through ~untouched (no spurious correction).
	cb := NewIQImbalanceCorrector()
	var passthru IQImbalanceStats
	for done := 0; done < total; done += chunk {
		buf := makeIQ(rng, chunk, 1.0, 0.0)
		cb.Process(buf)
		if done >= total/2 {
			passthru.Observe(buf)
		}
	}
	if irr := passthru.ImageRejectionDB(); irr < 35 {
		t.Errorf("balanced stream image rejection after corrector = %.1f dB, want ≳ 35", irr)
	}
}

func TestDCBlockerRemovesBias(t *testing.T) {
	const n = 4096
	in := make([]complex64, n)
	for i := range in {
		in[i] = complex(0.5, -0.3) // pure DC
	}
	d := NewDCBlocker(0.05)
	for k := 0; k < 20; k++ {
		buf := append([]complex64(nil), in...)
		d.Process(buf)
		if k == 19 {
			// After convergence, output should be near zero.
			var maxAbs float32
			for _, s := range buf {
				a := absComplex(s)
				if a > maxAbs {
					maxAbs = a
				}
			}
			if maxAbs > 0.05 {
				t.Errorf("residual after convergence = %f, want < 0.05", maxAbs)
			}
		}
	}
}

func TestPPMToHz(t *testing.T) {
	got := PPMToHz(50, 851_000_000)
	if got != 42_550 {
		t.Errorf("PPMToHz = %d, want 42550", got)
	}
}

func absComplex(c complex64) float32 {
	return float32(math.Hypot(float64(real(c)), float64(imag(c))))
}
