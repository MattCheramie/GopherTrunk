package hunt

import (
	"math"
	"math/rand"
	"testing"
)

// TestDetectOfflineCarriers plants two carriers in a wideband buffer — one
// above and one below the recorded centre — and asserts the offline sweep finds
// both at the right signed offsets. The sub-centre carrier exercises the
// signed-offset path (a naive uint32 absolute frequency would underflow).
func TestDetectOfflineCarriers(t *testing.T) {
	const (
		rate    = 2_000_000
		n       = 262144
		hiHz    = 300_000.0  // above centre
		loHz    = -625_000.0 // below centre (the motivating off-centre CC)
		hiAmp   = 1.0
		loAmp   = 0.6
		noiseSD = 0.03
	)
	rng := rand.New(rand.NewSource(1))
	iq := make([]complex64, n)
	for i := range iq {
		ph1 := 2 * math.Pi * hiHz * float64(i) / rate
		ph2 := 2 * math.Pi * loHz * float64(i) / rate
		re := hiAmp*math.Cos(ph1) + loAmp*math.Cos(ph2) + noiseSD*rng.NormFloat64()
		im := hiAmp*math.Sin(ph1) + loAmp*math.Sin(ph2) + noiseSD*rng.NormFloat64()
		iq[i] = complex64(complex(re, im))
	}

	carriers := detectOfflineCarriers(iq, rate, 4096, PeakOptions{ThresholdDb: 10, MinSpacingHz: 12_500})
	if len(carriers) < 2 {
		t.Fatalf("detected %d carriers, want ≥ 2: %+v", len(carriers), carriers)
	}

	foundHi, foundLo := false, false
	for _, c := range carriers {
		if math.Abs(float64(c.OffsetHz)-hiHz) < 2000 {
			foundHi = true
		}
		if math.Abs(float64(c.OffsetHz)-loHz) < 2000 {
			foundLo = true
		}
	}
	if !foundHi {
		t.Errorf("did not find the +300 kHz carrier; got %+v", carriers)
	}
	if !foundLo {
		t.Errorf("did not find the -625 kHz carrier; got %+v", carriers)
	}
}

// A narrowband (already-channelised) buffer or an empty one yields no carriers
// rather than panicking.
func TestDetectOfflineCarriersDegenerate(t *testing.T) {
	if c := detectOfflineCarriers(nil, 2_000_000, 4096, PeakOptions{}); c != nil {
		t.Errorf("nil input: got %+v, want nil", c)
	}
	if c := detectOfflineCarriers(make([]complex64, 100), 48_000, 4096, PeakOptions{}); c != nil {
		t.Errorf("too-short input: got %+v, want nil", c)
	}
}
