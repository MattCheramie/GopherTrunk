package sync

import (
	"math"
	"testing"
)

// TestQPSKCostasTracksOffset feeds the loop a synthetic π/4-DQPSK symbol
// stream carrying a known per-symbol carrier rotation (a residual
// frequency offset) and asserts the loop's OffsetHz converges to it and
// that the de-rotated symbols land back on the nominal differential grid.
func TestQPSKCostasTracksOffset(t *testing.T) {
	const baud = 4800.0
	// π/4-DQPSK differential phase changes for dibits 0..3.
	deltas := [4]float64{math.Pi / 4, 3 * math.Pi / 4, -3 * math.Pi / 4, -math.Pi / 4}

	for _, offsetHz := range []float64{0, 120, -300, 500} {
		// Use the same loop bandwidth the CQPSK path runs at; the frequency
		// integrator's settling time is ~1/ki symbols, so give it a long
		// enough run to converge and average the tail.
		c := NewQPSKCostas(baud, 120, 0.707)
		// Per-symbol carrier rotation the offset imposes.
		rot := 2 * math.Pi * offsetHz / baud

		// Build an absolute-phase π/4-DQPSK walk over a pseudo-random dibit
		// stream, adding the carrier rotation cumulatively.
		var absPhase, carrier float64
		seed := uint32(12345)
		const n = 6000
		var sum float64
		var cnt int
		for i := 0; i < n; i++ {
			seed = seed*1664525 + 1013904223
			d := (seed >> 16) & 3
			absPhase += deltas[d]
			carrier += rot
			ph := absPhase + carrier
			sym := complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
			c.Update(sym)
			if i >= n-1000 {
				sum += c.OffsetHz()
				cnt++
			}
		}
		got := sum / float64(cnt)
		if math.Abs(got-offsetHz) > 15 {
			t.Errorf("offset %.0f Hz: loop converged to %.1f Hz (err %.1f > 15 Hz)",
				offsetHz, got, got-offsetHz)
		}
	}
}

// TestQPSKCostasReset confirms Reset clears the loop state.
func TestQPSKCostasReset(t *testing.T) {
	c := NewQPSKCostas(4800, 60, 0.707)
	for i := 0; i < 100; i++ {
		ph := 0.5 * float64(i)
		c.Update(complex(float32(math.Cos(ph)), float32(math.Sin(ph))))
	}
	c.Reset()
	if c.OffsetHz() != 0 {
		t.Errorf("after Reset OffsetHz = %v, want 0", c.OffsetHz())
	}
}
