package spectrum

import (
	"math"
	"testing"
)

// makeBins returns n dBFS bins at floorDB, with the given bin indices set to
// peakDB — a tiny synthetic spectrum for the occupancy math.
func makeBins(n int, floorDB, peakDB float32, peaks ...int) []float32 {
	b := make([]float32, n)
	for i := range b {
		b[i] = floorDB
	}
	for _, p := range peaks {
		if p >= 0 && p < n {
			b[p] = peakDB
		}
	}
	return b
}

func TestComputeOccupancyToneIsNarrowAndPeaky(t *testing.T) {
	const n, fs = 256, 48000.0
	// A narrow carrier hump straddling DC (bin n/2), everything else at floor.
	bins := makeBins(n, -80, 0, 126, 127, 128, 129, 130)
	occ := ComputeOccupancy(bins, fs, 2000, 12500, 2000)

	// 99% bandwidth should be a handful of bins wide, not the whole span.
	if occ.OccupiedBandwidthHz <= 0 || occ.OccupiedBandwidthHz > 2000 {
		t.Errorf("tone occupied bandwidth = %.1f Hz, want narrow (0,2000]", occ.OccupiedBandwidthHz)
	}
	// A tone is the opposite of flat.
	if occ.SpectralFlatness >= 0.1 {
		t.Errorf("tone spectral flatness = %.4f, want ≪ 1", occ.SpectralFlatness)
	}
	// Adjacent channels hold only noise floor ⇒ strongly negative ACPR.
	if occ.ACPRUpperDB >= -40 || occ.ACPRLowerDB >= -40 {
		t.Errorf("tone ACPR upper=%.1f lower=%.1f dB, want both < -40", occ.ACPRUpperDB, occ.ACPRLowerDB)
	}
}

func TestComputeOccupancyWhiteNoiseIsWideAndFlat(t *testing.T) {
	const n, fs = 256, 48000.0
	// Every bin equal ⇒ white noise.
	bins := makeBins(n, -50, -50)
	occ := ComputeOccupancy(bins, fs, 2000, 12500, 2000)

	// 99% of a flat spectrum spans nearly the whole band.
	if occ.OccupiedBandwidthHz < 0.95*fs {
		t.Errorf("white-noise occupied bandwidth = %.1f Hz, want ≈ full span (%.0f)", occ.OccupiedBandwidthHz, fs)
	}
	// Equal bins ⇒ geometric mean == arithmetic mean ⇒ flatness ≈ 1.
	if math.Abs(occ.SpectralFlatness-1) > 1e-6 {
		t.Errorf("white-noise spectral flatness = %.6f, want ≈ 1", occ.SpectralFlatness)
	}
}

func TestComputeOccupancyEmptyOrInvalid(t *testing.T) {
	if got := ComputeOccupancy(nil, 48000, 2000, 12500, 2000); got != (Occupancy{}) {
		t.Errorf("nil bins = %+v, want zero Occupancy", got)
	}
	if got := ComputeOccupancy(makeBins(64, -50, -50), 0, 2000, 12500, 2000); got != (Occupancy{}) {
		t.Errorf("zero sample rate = %+v, want zero Occupancy", got)
	}
}
