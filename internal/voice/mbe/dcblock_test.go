package mbe

import (
	"math"
	"testing"
)

// TestDCBlockRemovesConstantOffset: a constant input decays to ~0 (DC is
// removed) after the filter settles.
func TestDCBlockRemovesConstantOffset(t *testing.T) {
	var d DCBlock
	in := make([]float64, 2000)
	for i := range in {
		in[i] = 5000 // pure DC
	}
	d.Process(in)
	// After settling, output should be near zero.
	tail := in[len(in)-1]
	if math.Abs(tail) > 1.0 {
		t.Errorf("DC not removed: tail sample = %v, want ~0", tail)
	}
}

// TestDCBlockPassesAC: a mid-band sinusoid passes with most of its
// amplitude intact (the high-pass only removes near-DC).
func TestDCBlockPassesAC(t *testing.T) {
	var d DCBlock
	const n = 1600
	in := make([]float64, n)
	for i := range in {
		in[i] = 10000 * math.Sin(2*math.Pi*float64(i)/16) // 500 Hz at 8 kHz
	}
	d.Process(in)
	var peak float64
	for _, v := range in[n/2:] { // after settling
		if a := math.Abs(v); a > peak {
			peak = a
		}
	}
	if peak < 9000 {
		t.Errorf("AC over-attenuated: peak = %v, want > 9000", peak)
	}
}

// TestDCBlockReset clears state.
func TestDCBlockReset(t *testing.T) {
	var d DCBlock
	d.Process([]float64{1, 2, 3})
	d.Reset()
	if d.prevX != 0 || d.prevY != 0 {
		t.Errorf("Reset left state %v/%v", d.prevX, d.prevY)
	}
}
