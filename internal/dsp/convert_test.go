package dsp

import (
	"math"
	"testing"
)

func TestRMS(t *testing.T) {
	if got := RMS(nil); got != 0 {
		t.Errorf("RMS(nil) = %v, want 0", got)
	}
	// A full-scale complex sinusoid has |x| = 1 at every sample → RMS 1.0.
	n := 1024
	iq := make([]complex64, n)
	for i := range iq {
		ph := 2 * math.Pi * 5 * float64(i) / float64(n)
		iq[i] = complex64(complex(math.Cos(ph), math.Sin(ph)))
	}
	if got := RMS(iq); math.Abs(got-1.0) > 1e-6 {
		t.Errorf("RMS(full-scale tone) = %v, want 1.0", got)
	}
	// Half-amplitude → RMS 0.5 → -6.02 dBFS.
	for i := range iq {
		iq[i] *= 0.5
	}
	if got := Mag2dB(RMS(iq)); math.Abs(got-(-6.0206)) > 1e-3 {
		t.Errorf("RMS dBFS(half scale) = %.4f, want -6.02", got)
	}
}

func TestPowerAndMagdB(t *testing.T) {
	if got := Power2dB(1); got != 0 {
		t.Errorf("Power2dB(1) = %v, want 0", got)
	}
	if got := Power2dB(100); math.Abs(got-20) > 1e-9 {
		t.Errorf("Power2dB(100) = %v, want 20", got)
	}
	if got := Mag2dB(10); math.Abs(got-20) > 1e-9 {
		t.Errorf("Mag2dB(10) = %v, want 20", got)
	}
	if got := Power2dB(0); got != -300 {
		t.Errorf("Power2dB(0) = %v, want -300 floor", got)
	}
	if got := Mag2dB(-1); got != -300 {
		t.Errorf("Mag2dB(neg) = %v, want -300 floor", got)
	}
}
