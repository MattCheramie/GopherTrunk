package demod

import (
	"math"
	"testing"
)

func TestFMDemodLinearChirp(t *testing.T) {
	// Generate a complex exponential whose phase advances by a constant rate.
	const N = 4096
	const rate = 0.1 // radians per sample
	in := make([]complex64, N)
	phi := 0.0
	for i := 0; i < N; i++ {
		in[i] = complex(float32(math.Cos(phi)), float32(math.Sin(phi)))
		phi += rate
	}
	d := NewFM()
	out := d.Process(nil, in)
	// Skip first sample (depends on init). Rest should be ~rate.
	for i := 1; i < N; i++ {
		if math.Abs(float64(out[i])-rate) > 1e-3 {
			t.Fatalf("FM out[%d] = %f, want %f", i, out[i], rate)
		}
	}
}

func TestC4FMSlicer(t *testing.T) {
	// Deviation = 3.0 → outer-symbol threshold ±2.0.
	c := NewC4FM(8, 8, 0.2, 3.0)
	cases := []struct {
		in   float32
		want int
	}{
		{2.5, 3}, {1.0, 1}, {0.5, 1}, {0.01, 1},
		{-0.01, -1}, {-1.0, -1}, {-2.5, -3},
		{2.001, 3}, {1.999, 1}, // threshold corner
	}
	for _, tc := range cases {
		got := c.Slice(tc.in)
		if got != tc.want {
			t.Errorf("Slice(%f) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestDQPSKDibitsMatchPhaseSteps(t *testing.T) {
	d := NewDQPSK()
	// Build a signal whose phase advances by π/2 per symbol → dibit "01".
	const N = 64
	in := make([]complex64, N)
	phi := 0.0
	for i := 0; i < N; i++ {
		in[i] = complex(float32(math.Cos(phi)), float32(math.Sin(phi)))
		phi += math.Pi / 2
	}
	out := d.Decode(nil, in)
	for i := 1; i < N; i++ {
		if out[i] != 0b01 {
			t.Errorf("out[%d] = %02b, want 01", i, out[i])
		}
	}
}
