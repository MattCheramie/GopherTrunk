package filter

import "testing"

func TestCICDecimatesByR(t *testing.T) {
	c := NewCICDecimator(8, 3)
	in := make([]int64, 8000)
	for i := range in {
		in[i] = 1000
	}
	out := c.ProcessReal(nil, in)
	if got, want := len(out), len(in)/8; got != want {
		t.Fatalf("len(out) = %d, want %d", got, want)
	}
	// After enough samples, DC input × gain == steady-state output (within
	// the comb-warmup transient).
	stable := out[len(out)-1]
	if stable != int64(1000)*c.Gain() {
		t.Errorf("steady-state = %d, want %d", stable, int64(1000)*c.Gain())
	}
}

func TestHalfbandZeroPattern(t *testing.T) {
	taps := HalfbandLowpass(31)
	mid := len(taps) / 2
	for i, t0 := range taps {
		if i == mid {
			continue
		}
		if (i-mid)%2 == 0 && t0 != 0 {
			t.Errorf("expected tap[%d] zero, got %f", i, t0)
		}
	}
}
