package ambe2

import (
	"math"
	"testing"
)

func TestKnoxToneDefaultsToSilence(t *testing.T) {
	// Fresh state — no overrides registered. Every knox index must
	// report (0, 0, false) so Decode falls through to the silence
	// branch instead of synthesising bogus tones.
	ClearKnoxTones()
	for b1 := KnoxIndexLow; b1 <= KnoxIndexHigh; b1++ {
		fA, fB, ok := KnoxTone(b1)
		if ok {
			t.Errorf("KnoxTone(%d) = (%.2f, %.2f, true); want (_, _, false) for unset", b1, fA, fB)
		}
	}
}

func TestKnoxToneRoundTripsOverride(t *testing.T) {
	ClearKnoxTones()
	cases := []struct {
		b1           int
		freqA, freqB float64
	}{
		{144, 1000, 1700},
		{150, 1100, 1750},
		{163, 1300, 1850},
	}
	for _, tc := range cases {
		if err := SetKnoxTone(tc.b1, tc.freqA, tc.freqB); err != nil {
			t.Fatalf("SetKnoxTone(%d, %v, %v): %v", tc.b1, tc.freqA, tc.freqB, err)
		}
	}
	for _, tc := range cases {
		fA, fB, ok := KnoxTone(tc.b1)
		if !ok {
			t.Errorf("KnoxTone(%d) ok = false, want true", tc.b1)
			continue
		}
		if fA != tc.freqA || fB != tc.freqB {
			t.Errorf("KnoxTone(%d) = (%v, %v), want (%v, %v)",
				tc.b1, fA, fB, tc.freqA, tc.freqB)
		}
	}
	ClearKnoxTones()
	// After clear, every entry must report unset again.
	for _, tc := range cases {
		if _, _, ok := KnoxTone(tc.b1); ok {
			t.Errorf("KnoxTone(%d) still ok after ClearKnoxTones", tc.b1)
		}
	}
}

func TestSetKnoxToneRejectsOutOfRange(t *testing.T) {
	cases := []int{0, 127, 128, 143, 164, 200, 255, -1}
	for _, b1 := range cases {
		if err := SetKnoxTone(b1, 1000, 1700); err == nil {
			t.Errorf("SetKnoxTone(%d, ...) accepted out-of-range index", b1)
		}
	}
}

func TestSetKnoxToneZeroPairClearsEntry(t *testing.T) {
	ClearKnoxTones()
	if err := SetKnoxTone(150, 1100, 1750); err != nil {
		t.Fatalf("SetKnoxTone: %v", err)
	}
	if _, _, ok := KnoxTone(150); !ok {
		t.Fatal("override not set")
	}
	if err := SetKnoxTone(150, 0, 0); err != nil {
		t.Fatalf("SetKnoxTone(_, 0, 0): %v", err)
	}
	if _, _, ok := KnoxTone(150); ok {
		t.Error("override not cleared by (0, 0) pair")
	}
}

// TestDecodeKnoxSilenceWithoutOverride asserts the integration: a
// tone frame with a knox b1 routes through the silence branch
// (PCM all-zero or fade-out, never a sinewave) when no override is
// registered.
func TestDecodeKnoxSilenceWithoutOverride(t *testing.T) {
	ClearKnoxTones()
	d := New()
	defer d.Close()

	// Build a 7-byte frame with the AMBE+2 tone-frame magic + a knox
	// b1 index (150) + b2 amplitude (4). The tone-frame bit layout
	// is hot-encoded in params.go; we use the in-package frame
	// builder to avoid duplicating the encoding here.
	frame := buildKnoxToneFrame(t, 150, 200)
	pcm, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if peak := absMaxInt16(pcm); peak > 100 {
		t.Errorf("knox without override produced peak amplitude %d; want near-silence (< 100)", peak)
	}
}

// TestDecodeKnoxSynthesisesWithOverride confirms that after
// SetKnoxTone(150, 1100, 1750) a tone frame at b1=150 produces
// audible PCM (not silence). The actual spectral verification is
// intentionally light — sample-level synthesis is unit-tested
// elsewhere; this test verifies the wire-up.
func TestDecodeKnoxSynthesisesWithOverride(t *testing.T) {
	ClearKnoxTones()
	if err := SetKnoxTone(150, 1100, 1750); err != nil {
		t.Fatalf("SetKnoxTone: %v", err)
	}
	defer ClearKnoxTones()

	d := New()
	defer d.Close()

	frame := buildKnoxToneFrame(t, 150, 200)
	pcm, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	// With dual-tone synthesis the AGC scales output toward int16
	// range — peak should comfortably exceed any reasonable silence
	// threshold.
	if peak := absMaxInt16(pcm); peak < 1000 {
		t.Errorf("knox with override produced peak amplitude %d; want > 1000 (synthesis active)", peak)
	}
}

// buildKnoxToneFrame constructs a 7-byte AMBE+2 tone frame for a
// knox b1 index in [144, 163] and 8-bit volume b2. Matches the bit
// packing used by the in-package dualToneFrame helper
// (decoder_test.go), extended to the knox range — bit 4 of
// (b1 - 128) is 1 so info[9] is set in addition to the lower 4 bits.
func buildKnoxToneFrame(t *testing.T, b1, b2 int) []byte {
	t.Helper()
	if b1 < KnoxIndexLow || b1 > KnoxIndexHigh {
		t.Fatalf("buildKnoxToneFrame: b1 = %d outside knox range [%d, %d]",
			b1, KnoxIndexLow, KnoxIndexHigh)
	}
	info := make([]byte, InfoBits)
	for i := 0; i <= 5; i++ {
		info[i] = 1 // tone-frame protection-bit pattern
	}
	// t-table idx = 0 → info[6,7,8] all zero (default), giving the
	// high-3 bits of b1 = 1 0 0 (i.e. b1 base = 128).
	low := b1 - 128
	info[9] = byte((low >> 4) & 1) // always 1 for knox range
	info[42] = byte((low >> 3) & 1)
	info[43] = byte((low >> 2) & 1)
	info[10] = byte((low >> 1) & 1)
	info[11] = byte(low & 1)
	// b2 packing mirrors dualToneFrame in decoder_test.go.
	info[12] = byte((b2 >> 7) & 1)
	info[13] = byte((b2 >> 6) & 1)
	info[14] = byte((b2 >> 5) & 1)
	info[15] = byte((b2 >> 4) & 1)
	info[16] = byte((b2 >> 3) & 1)
	info[44] = byte((b2 >> 2) & 1)
	info[45] = byte((b2 >> 1) & 1)
	info[17] = byte(b2 & 1)
	frame := make([]byte, FrameBytes)
	for i := 0; i < InfoBits; i++ {
		frame[i/8] |= info[i] << (7 - uint(i)%8)
	}
	return frame
}

func absMaxInt16(samples []int16) int {
	peak := 0
	for _, s := range samples {
		v := int(math.Abs(float64(s)))
		if v > peak {
			peak = v
		}
	}
	return peak
}
