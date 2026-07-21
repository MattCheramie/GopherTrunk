package main

import (
	"math"
	"strings"
	"testing"
)

// synthTone builds n IQ samples of a complex exponential at offsetHz plus a
// little deterministic noise, so the noise floor is non-zero (no math/rand, so
// it's reproducible without seeding concerns).
func synthTone(n int, offsetHz, rate, noise float64) []complex64 {
	iq := make([]complex64, n)
	x := uint32(0x9E3779B9)
	nz := func() float64 {
		x = x*1664525 + 1013904223
		return (float64(x>>8)/float64(1<<24) - 0.5) * noise
	}
	for i := 0; i < n; i++ {
		th := 2 * math.Pi * offsetHz * float64(i) / rate
		iq[i] = complex(float32(math.Cos(th)+nz()), float32(math.Sin(th)+nz()))
	}
	return iq
}

func TestCaptureCarrierOffsetHz(t *testing.T) {
	const rate = 50000.0
	for _, off := range []float64{0, 5000, -8700, 12000} {
		got, ok := captureCarrierOffsetHz(synthTone(1<<15, off, rate, 0.1), rate)
		if !ok {
			t.Fatalf("off=%.0f: estimate not ok", off)
		}
		if math.Abs(got-off) > 200 {
			t.Errorf("off=%.0f Hz: got %.0f Hz, want within 200 Hz", off, got)
		}
	}
}

func TestCaptureCarrierOffsetHzTooShort(t *testing.T) {
	if _, ok := captureCarrierOffsetHz(make([]complex64, 500), 50000); ok {
		t.Error("expected ok=false for a window shorter than the minimum FFT size")
	}
	if _, ok := captureCarrierOffsetHz(synthTone(4096, 0, 50000, 0.1), 0); ok {
		t.Error("expected ok=false for a zero sample rate")
	}
}

func TestCarrierOffsetWarning(t *testing.T) {
	const rate = 50000.0
	const freq = 467_913_000

	// A small offset is pulled in by the decoder's AFC — stay silent.
	if w := carrierOffsetWarning(synthTone(1<<15, 500, rate, 0.1), rate, freq); w != "" {
		t.Errorf("500 Hz offset should not warn, got: %s", w)
	}

	// A multi-kHz offset fires and names the fix.
	w := carrierOffsetWarning(synthTone(1<<15, -8700, rate, 0.1), rate, freq)
	if w == "" {
		t.Fatal("an 8.7 kHz offset should warn")
	}
	for _, sub := range []string{"centre", "ppm", "-ppm", "below"} {
		if !strings.Contains(w, sub) {
			t.Errorf("warning missing %q: %s", sub, w)
		}
	}
}

func TestFormatCarrierOffsetWarning(t *testing.T) {
	below := formatCarrierOffsetWarning(-8700, -18.6)
	if !strings.Contains(below, "8.7 kHz below centre") {
		t.Errorf("below-centre wording wrong: %s", below)
	}
	if !strings.Contains(below, "18.6 ppm") {
		t.Errorf("missing ppm figure: %s", below)
	}
	above := formatCarrierOffsetWarning(3200, 6.8)
	if !strings.Contains(above, "3.2 kHz above centre") {
		t.Errorf("above-centre wording wrong: %s", above)
	}
}
