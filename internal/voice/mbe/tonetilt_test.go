package mbe

import (
	"math"
	"testing"
)

// bandPower drives the filter with a sine at freqHz and returns the
// steady-state output RMS (skipping a settling region).
func toneBandPower(t *ToneTilt, freqHz, sampleRate float64) float64 {
	const n = 8000
	in := make([]float64, n)
	for i := range in {
		in[i] = math.Sin(2 * math.Pi * freqHz * float64(i) / sampleRate)
	}
	t.Process(in)
	var sumSq float64
	for i := n / 2; i < n; i++ { // second half = settled
		sumSq += in[i] * in[i]
	}
	return math.Sqrt(sumSq / float64(n/2))
}

// TestToneTiltPassesLowAttenuatesHigh confirms the high-shelf shape: a
// low tone passes near unity, a tone well above the corner is attenuated
// by ~shelfDb. This is the deterministic frequency-response check behind
// the opt-in "warm" DMR decoder (issue #644).
func TestToneTiltPassesLowAttenuatesHigh(t *testing.T) {
	const fs = 8000.0
	const corner = 900.0
	const shelfDb = 3.5 // first-order asymptote at high frequency
	const unitRMS = math.Sqrt2 / 2

	gainDb := func(freq float64) float64 {
		return 20 * math.Log10(toneBandPower(NewToneTilt(fs, corner, shelfDb), freq, fs)/unitRMS)
	}
	low := gainDb(150)
	mid := gainDb(1500)
	high := gainDb(3200)

	// Low end essentially untouched.
	if math.Abs(low) > 0.3 {
		t.Errorf("low (150 Hz) gain %.2f dB, want ≈0 (passband)", low)
	}
	// In-band cut is meaningful but never exceeds the first-order asymptote.
	if high > -1.2 || high < -shelfDb {
		t.Errorf("high (3200 Hz) gain %.2f dB, want in (-%.1f, -1.2) dB", high, shelfDb)
	}
	// Monotonic high-shelf: more cut as frequency rises.
	if !(low > mid && mid > high) {
		t.Errorf("response not monotonically decreasing: low=%.2f mid=%.2f high=%.2f", low, mid, high)
	}
}

// TestToneTiltDisabledIsPassThrough: a zero/negative shelf, a zero
// sample rate, and a nil receiver all leave the signal untouched, so the
// default decoders (which hold a nil *ToneTilt) are unaffected.
func TestToneTiltDisabledIsPassThrough(t *testing.T) {
	in := []float64{1, -2, 3, -4, 5, 0, -7, 8}
	for _, tt := range []*ToneTilt{
		NewToneTilt(8000, 1500, 0),  // no cut
		NewToneTilt(8000, 1500, -3), // negative shelf treated as disabled
		NewToneTilt(0, 1500, 2.5),   // invalid rate
		nil,                         // nil receiver
	} {
		got := append([]float64(nil), in...)
		tt.Process(got)
		for i := range in {
			if got[i] != in[i] {
				t.Errorf("pass-through filter changed sample %d: %v != %v", i, got[i], in[i])
			}
		}
	}
}

// TestToneTiltResetClearsState: Reset zeroes the one-pole memory so a
// re-synced stream starts clean (and is a no-op on a nil receiver).
func TestToneTiltResetClearsState(t *testing.T) {
	f := NewToneTilt(8000, 1500, 2.5)
	f.Process([]float64{1000, 1000, 1000, 1000})
	if f.lp == 0 {
		t.Fatal("expected non-zero state after processing")
	}
	f.Reset()
	if f.lp != 0 {
		t.Errorf("Reset did not clear state: lp=%v", f.lp)
	}
	var nilFilter *ToneTilt
	nilFilter.Reset() // must not panic
}
