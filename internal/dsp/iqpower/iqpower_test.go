package iqpower

import (
	"math"
	"testing"
)

func TestMeanPowerDbFS(t *testing.T) {
	// A constant unit-magnitude tone has mean |x|² = 1 → 0 dBFS.
	tone := make([]complex64, 1000)
	for i := range tone {
		tone[i] = complex(1, 0)
	}
	if got := MeanPowerDbFS(tone); math.Abs(got) > 1e-6 {
		t.Errorf("MeanPowerDbFS(unit tone) = %v dBFS, want ~0", got)
	}

	// Half-amplitude tone: mean power 0.25 → 10*log10(0.25) ≈ -6.0206 dB.
	half := make([]complex64, 500)
	for i := range half {
		half[i] = complex(0.5, 0)
	}
	if got := MeanPowerDbFS(half); math.Abs(got-(-6.0206)) > 1e-3 {
		t.Errorf("MeanPowerDbFS(half tone) = %v dBFS, want ~-6.02", got)
	}

	// Silence clamps to the epsilon floor, not -Inf.
	silent := make([]complex64, 100)
	floor := 10 * math.Log10(powerEpsilon)
	if got := MeanPowerDbFS(silent); math.Abs(got-floor) > 1e-9 {
		t.Errorf("MeanPowerDbFS(silence) = %v, want clamped floor %v", got, floor)
	}

	// Empty slice also returns the floor instead of NaN.
	if got := MeanPowerDbFS(nil); math.Abs(got-floor) > 1e-9 {
		t.Errorf("MeanPowerDbFS(nil) = %v, want clamped floor %v", got, floor)
	}
}

func TestAccumulatorWindowingAndReset(t *testing.T) {
	var a Accumulator
	if a.Samples() != 0 {
		t.Fatalf("fresh Accumulator Samples() = %d, want 0", a.Samples())
	}

	tone := make([]complex64, 600)
	for i := range tone {
		tone[i] = complex(1, 0)
	}
	a.Add(tone[:200])
	a.Add(tone[200:]) // fold across two calls
	if a.Samples() != 600 {
		t.Errorf("Samples() = %d, want 600", a.Samples())
	}

	dbfs, n := a.MeanDbFSAndReset()
	if n != 600 {
		t.Errorf("drained samples = %d, want 600", n)
	}
	if math.Abs(dbfs) > 1e-6 {
		t.Errorf("windowed mean = %v dBFS, want ~0", dbfs)
	}

	// Reset leaves the accumulator empty.
	if a.Samples() != 0 {
		t.Errorf("after reset Samples() = %d, want 0", a.Samples())
	}
	if _, n := a.MeanDbFSAndReset(); n != 0 {
		t.Errorf("empty drain samples = %d, want 0", n)
	}
}
