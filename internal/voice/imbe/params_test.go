package imbe

import (
	"errors"
	"math"
	"testing"
)

// putB0 stamps an 8-bit b_0 value into the scattered positions
// {0..5, 85, 86} of an 88-bit info buffer, leaving the other bits
// at zero.
func putB0(b0 uint) []byte {
	info := make([]byte, InfoBits)
	info[0] = byte((b0 >> 7) & 1)
	info[1] = byte((b0 >> 6) & 1)
	info[2] = byte((b0 >> 5) & 1)
	info[3] = byte((b0 >> 4) & 1)
	info[4] = byte((b0 >> 3) & 1)
	info[5] = byte((b0 >> 2) & 1)
	info[85] = byte((b0 >> 1) & 1)
	info[86] = byte(b0 & 1)
	return info
}

func TestUnpackHeaderRejectsWrongLength(t *testing.T) {
	if _, err := UnpackHeader(make([]byte, InfoBits-1)); err == nil {
		t.Error("UnpackHeader accepted short info")
	}
	if _, err := UnpackHeader(make([]byte, InfoBits+1)); err == nil {
		t.Error("UnpackHeader accepted long info")
	}
}

func TestUnpackHeaderInvalidFundamentalAbove207(t *testing.T) {
	for _, b0 := range []uint{208, 215, 220, 255} {
		_, err := UnpackHeader(putB0(b0))
		if !errors.Is(err, ErrInvalidFundamental) {
			t.Errorf("b0=%d: err = %v, want ErrInvalidFundamental", b0, err)
		}
	}
}

func TestUnpackHeaderSilenceWindow(t *testing.T) {
	for _, b0 := range []uint{216, 217, 218, 219} {
		h, err := UnpackHeader(putB0(b0))
		if err != nil {
			t.Errorf("b0=%d: err = %v, want nil", b0, err)
			continue
		}
		if !h.Silent {
			t.Errorf("b0=%d: Silent=false, want true", b0)
		}
		if h.L != 0 || h.K != 0 || h.W0 != 0 {
			t.Errorf("b0=%d: header = %+v, want zero W0/L/K", b0, h)
		}
	}
}

func TestUnpackHeaderW0FormulaMatchesSpec(t *testing.T) {
	// w0 = 4π / (b0 + 39.5). Sample a couple of valid b0 values and
	// compare to the closed-form expectation.
	cases := []uint{0, 50, 100, 200}
	for _, b0 := range cases {
		h, err := UnpackHeader(putB0(b0))
		if err != nil {
			t.Fatalf("b0=%d: err = %v", b0, err)
		}
		want := 4 * math.Pi / (float64(b0) + 39.5)
		if math.Abs(h.W0-want) > 1e-9 {
			t.Errorf("b0=%d: W0 = %g, want %g", b0, h.W0, want)
		}
	}
}

func TestUnpackHeaderLFromW0(t *testing.T) {
	// L = floor(0.9254 * floor(π/w0 + 0.25)). Pin to
	// mbe_decodeImbe4400Parms's exact integer-truncation order so a
	// future refactor that "simplifies" the formula trips this
	// regression.
	cases := []struct {
		b0   uint
		wantL int
	}{
		{0, 9},   // smallest b_0 → highest fundamental → smallest L
		{207, 56}, // largest valid b_0 → lowest fundamental → largest L
	}
	for _, tc := range cases {
		h, err := UnpackHeader(putB0(tc.b0))
		if err != nil {
			t.Errorf("b0=%d: err = %v", tc.b0, err)
			continue
		}
		if h.L != tc.wantL {
			t.Errorf("b0=%d: L = %d, want %d", tc.b0, h.L, tc.wantL)
		}
	}
}

func TestUnpackHeaderKFromL(t *testing.T) {
	// K = floor((L+2)/3) for L < 37, else 12 — preserving mbelib's
	// integer-truncating cast in mbe_decodeImbe4400Parms (the C
	// `(int) ((float)(L+2)/3)` form). Sample a handful of L values
	// via b_0 to exercise both branches.
	cases := []struct {
		b0    uint
		wantK int
	}{
		{207, 12}, // L = 56 → K = 12 (large-L branch)
		{0, 3},    // L = 9  → K = ⌊11/3⌋ = 3
	}
	for _, tc := range cases {
		h, err := UnpackHeader(putB0(tc.b0))
		if err != nil {
			t.Fatalf("b0=%d: err = %v", tc.b0, err)
		}
		if h.K != tc.wantK {
			t.Errorf("b0=%d: K = %d, want %d", tc.b0, h.K, tc.wantK)
		}
	}
}

func TestUnpackHeaderLInBoundsForAllValidB0(t *testing.T) {
	// Sweep every valid b_0 (0..207) and assert L stays in [9, 56].
	// Catches off-by-ones in the floor() chain.
	for b0 := uint(0); b0 < 208; b0++ {
		h, err := UnpackHeader(putB0(b0))
		if err != nil {
			t.Errorf("b0=%d: err = %v", b0, err)
			continue
		}
		if h.L < 9 || h.L > 56 {
			t.Errorf("b0=%d: L = %d out of [9, 56]", b0, h.L)
		}
	}
}

func TestTablesSizesMatchSpec(t *testing.T) {
	// Cheap regression guard against typos in the transcribed
	// tables.go: the dimensions must exactly match TIA-102.BABA
	// (and mbelib's reference implementation).
	if len(quantstep) != 11 {
		t.Errorf("quantstep len = %d, want 11", len(quantstep))
	}
	if len(standdev) != 9 {
		t.Errorf("standdev len = %d, want 9", len(standdev))
	}
	if len(b2Table) != 64 {
		t.Errorf("b2Table len = %d, want 64", len(b2Table))
	}
	if len(imbeJi) != 48 {
		t.Errorf("imbeJi len = %d, want 48", len(imbeJi))
	}
	if len(ba) != 48 {
		t.Errorf("ba len = %d, want 48", len(ba))
	}
	if len(hoba) != 48 {
		t.Errorf("hoba len = %d, want 48", len(hoba))
	}
	if len(bo) != 48 {
		t.Errorf("bo len = %d, want 48", len(bo))
	}
}
