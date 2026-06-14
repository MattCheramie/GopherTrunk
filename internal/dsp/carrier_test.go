package dsp

import (
	"math"
	"math/rand"
	"testing"
)

// EstimateCarrierOffsetHz should recover a planted carrier offset to within
// a few Hz, even buried in additive noise and at a negative offset.
func TestEstimateCarrierOffset(t *testing.T) {
	const fs = 48000.0
	for _, want := range []float64{750, -1234, 3000} {
		rng := rand.New(rand.NewSource(1))
		n := 200000
		iq := make([]complex64, n)
		for i := range iq {
			ph := 2 * math.Pi * want * float64(i) / fs
			noiseI := 0.3 * rng.NormFloat64()
			noiseQ := 0.3 * rng.NormFloat64()
			iq[i] = complex64(complex(math.Cos(ph)+noiseI, math.Sin(ph)+noiseQ))
		}
		got := EstimateCarrierOffsetHz(iq, fs, fs*0.5)
		if math.Abs(got-want) > 5 {
			t.Errorf("offset %.0f Hz: estimated %.1f Hz (err %.1f, want <5)", want, got, got-want)
		}
	}
}

// Degenerate inputs return 0 rather than panicking.
func TestEstimateCarrierOffsetEdgeCases(t *testing.T) {
	if got := EstimateCarrierOffsetHz(nil, 48000, 24000); got != 0 {
		t.Errorf("empty input: got %.3f, want 0", got)
	}
	if got := EstimateCarrierOffsetHz([]complex64{complex(1, 0)}, 48000, 0); got != 0 {
		t.Errorf("zero searchHz: got %.3f, want 0", got)
	}
	if got := EstimateCarrierCandidatesHz(nil, 48000, 24000, 0, 0); got != nil {
		t.Errorf("empty input: got %v, want nil", got)
	}
}

// twoCarrier builds an fs-rate capture with two complex tones (strong + weak)
// plus additive noise — a stand-in for a wideband grab whose control channel
// is not the dominant carrier.
func twoCarrier(n int, fs, strongHz, strongAmp, weakHz, weakAmp, noise float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	iq := make([]complex64, n)
	for i := range iq {
		ps := 2 * math.Pi * strongHz * float64(i) / fs
		pw := 2 * math.Pi * weakHz * float64(i) / fs
		re := strongAmp*math.Cos(ps) + weakAmp*math.Cos(pw) + noise*rng.NormFloat64()
		im := strongAmp*math.Sin(ps) + weakAmp*math.Sin(pw) + noise*rng.NormFloat64()
		iq[i] = complex64(complex(re, im))
	}
	return iq
}

// EstimateCarrierCandidatesHz must surface BOTH carriers (the motivating
// wideband case: a strong +560 kHz voice carrier and a weaker -625 kHz
// control channel), ranked strongest-first with the right relative power.
func TestEstimateCarrierCandidatesTwoCarriers(t *testing.T) {
	const fs = 2_000_000.0
	// weakAmp 0.5 ⇒ power 1/4 of strong ⇒ ≈ -6 dB.
	iq := twoCarrier(262144, fs, 560_000, 1.0, -625_000, 0.5, 0.05, 7)
	cands := EstimateCarrierCandidatesHz(iq, fs, fs*0.5, 0, 6)
	if len(cands) < 2 {
		t.Fatalf("got %d candidates, want ≥ 2: %+v", len(cands), cands)
	}
	if math.Abs(cands[0].OffsetHz-560_000) > 1000 {
		t.Errorf("strongest offset = %.0f Hz, want ≈ 560000", cands[0].OffsetHz)
	}
	if math.Abs(cands[1].OffsetHz-(-625_000)) > 1000 {
		t.Errorf("second offset = %.0f Hz, want ≈ -625000", cands[1].OffsetHz)
	}
	if cands[0].RelPowerDb != 0 {
		t.Errorf("strongest RelPowerDb = %.2f, want 0", cands[0].RelPowerDb)
	}
	if d := cands[1].RelPowerDb; d < -9 || d > -3 {
		t.Errorf("second RelPowerDb = %.2f dB, want ≈ -6", d)
	}
	// The single-carrier wrapper still returns the dominant carrier.
	if got := EstimateCarrierOffsetHz(iq, fs, fs*0.5); math.Abs(got-560_000) > 1000 {
		t.Errorf("EstimateCarrierOffsetHz = %.0f Hz, want ≈ 560000", got)
	}
}

// Carriers closer than the min spacing collapse to the stronger; a carrier far
// below the floor is dropped.
func TestEstimateCarrierCandidatesSpacingAndFloor(t *testing.T) {
	const fs = 2_000_000.0
	// A strong carrier and one 30 dB down (amp ≈ 0.0316) — below the 20 dB floor.
	iq := twoCarrier(262144, fs, 100_000, 1.0, -700_000, 0.0316, 0.0, 3)
	cands := EstimateCarrierCandidatesHz(iq, fs, fs*0.5, 0, 6)
	for _, c := range cands {
		if math.Abs(c.OffsetHz-(-700_000)) < 5000 {
			t.Errorf("carrier 30 dB down should be dropped, but found %+v", c)
		}
	}

	// Two equal carriers within the min spacing collapse to one.
	iq2 := twoCarrier(262144, fs, 300_000, 1.0, 305_000, 1.0, 0.0, 5)
	c2 := EstimateCarrierCandidatesHz(iq2, fs, fs*0.5, 12_500, 6)
	near := 0
	for _, c := range c2 {
		if c.OffsetHz > 290_000 && c.OffsetHz < 315_000 {
			near++
		}
	}
	if near != 1 {
		t.Errorf("carriers 5 kHz apart within 12.5 kHz spacing: got %d candidates in band, want 1 (%+v)", near, c2)
	}
}
