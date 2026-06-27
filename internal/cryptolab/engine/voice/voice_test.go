package voice

import (
	"math"
	"testing"
)

func TestSpectralInvertIsSelfInverse(t *testing.T) {
	t.Parallel()
	in := []float64{0.1, -0.4, 0.7, 0.2, -0.9, 0.5, 0.33, -0.1}
	out := SpectralInvert(SpectralInvert(in))
	for i := range in {
		if math.Abs(out[i]-in[i]) > 1e-12 {
			t.Fatalf("double invert[%d] = %v, want %v", i, out[i], in[i])
		}
	}
}

func TestSpectralInvertMovesEnergyHigh(t *testing.T) {
	t.Parallel()
	// A low-frequency tone should land in the high band after inversion.
	const n = 256
	samples := make([]float64, n)
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * 4 * float64(i) / float64(n)) // bin 4 (low)
	}
	lowBefore := BandEnergy(samples, 0, 0.25)
	highBefore := BandEnergy(samples, 0.75, 1.0)
	if lowBefore <= highBefore {
		t.Fatalf("setup: tone not in low band (low=%v high=%v)", lowBefore, highBefore)
	}
	inv := SpectralInvert(samples)
	lowAfter := BandEnergy(inv, 0, 0.25)
	highAfter := BandEnergy(inv, 0.75, 1.0)
	if highAfter <= lowAfter {
		t.Fatalf("inversion did not move energy high (low=%v high=%v)", lowAfter, highAfter)
	}
}

func TestSpectralInvertInt16(t *testing.T) {
	t.Parallel()
	in := []int16{100, 200, -300, 400}
	out := SpectralInvertInt16(in)
	want := []int16{100, -200, -300, -400}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out[%d] = %d, want %d", i, out[i], want[i])
		}
	}
}
