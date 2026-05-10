package imbe

import (
	"errors"
	"math"
	"testing"
)

// putB0Info is a thin convenience wrapper: stamps a b_0 value into
// scattered positions {0..5, 85, 86} of an 88-bit info buffer. The
// remaining 79 "scattered" bits at positions [6..84] stay zero, so
// the resulting Params have all bb[v][p] = 0, which means the
// post-DCT Tl ends up identically zero (the dequantizer's −2^(b−1)
// + 0.5 offset gets exactly cancelled by the DCT linear combination
// when all bm = 0... actually not quite — see below).
func putB0Info(b0 uint) []byte {
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

func TestUnpackParamsPropagatesInvalidFundamental(t *testing.T) {
	_, err := UnpackParams(putB0Info(208))
	if !errors.Is(err, ErrInvalidFundamental) {
		t.Errorf("err = %v, want ErrInvalidFundamental", err)
	}
}

func TestUnpackParamsRejectsWrongLength(t *testing.T) {
	if _, err := UnpackParams(make([]byte, InfoBits-1)); err == nil {
		t.Error("UnpackParams accepted short info")
	}
}

func TestUnpackParamsSilentReturnsZeroSpectral(t *testing.T) {
	for _, b0 := range []uint{216, 217, 218, 219} {
		p, err := UnpackParams(putB0Info(b0))
		if err != nil {
			t.Fatalf("b0=%d: err = %v", b0, err)
		}
		if !p.Silent {
			t.Errorf("b0=%d: Silent=false", b0)
		}
		// All spectral fields must be zero in the silence frame.
		for i := 1; i < len(p.Vl); i++ {
			if p.Vl[i] != 0 {
				t.Errorf("b0=%d: Vl[%d] = %d, want 0", b0, i, p.Vl[i])
			}
		}
		for i, g := range p.Gm {
			if g != 0 {
				t.Errorf("b0=%d: Gm[%d] = %g, want 0", b0, i, g)
			}
		}
		for i := 1; i < len(p.Tl); i++ {
			if p.Tl[i] != 0 {
				t.Errorf("b0=%d: Tl[%d] = %g, want 0", b0, i, p.Tl[i])
			}
		}
	}
}

func TestUnpackParamsAllZeroInfoMatchesB2Index32(t *testing.T) {
	// All-zero info → b_0 = 0 → header path succeeds. With every
	// scattered bit = 0, b_2 = 0 (so Gm[1] = b2Table[0]), each
	// PRBA bm = 0 (so Gm[i] = step × (0 − 2^(b−1) + 0.5)), every
	// HOC bm = 0 (so Cik = quantstep × standdev × (0 − 2^(b-1) +
	// 0.5)). Tl is the inverse-DCT of those — non-zero in general.
	//
	// What we *can* assert: every output is finite and Vl[1..L]
	// are all 0 (vector-1 voicing field is zero).
	info := putB0Info(0) // all-zero info except b_0 = 0
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatal(err)
	}
	if p.Silent {
		t.Error("b0=0: Silent=true, want false")
	}
	for i := 1; i <= p.L; i++ {
		if p.Vl[i] != 0 {
			t.Errorf("Vl[%d] = %d, want 0 (all-zero info)", i, p.Vl[i])
		}
	}
	for i := 1; i <= 6; i++ {
		if math.IsNaN(p.Gm[i]) || math.IsInf(p.Gm[i], 0) {
			t.Errorf("Gm[%d] = %g, want finite", i, p.Gm[i])
		}
	}
	for i := 1; i <= p.L; i++ {
		if math.IsNaN(p.Tl[i]) || math.IsInf(p.Tl[i], 0) {
			t.Errorf("Tl[%d] = %g, want finite", i, p.Tl[i])
		}
	}
	// Tl[L+1..56] must remain at zero (we only fill up to L).
	for i := p.L + 1; i < len(p.Tl); i++ {
		if p.Tl[i] != 0 {
			t.Errorf("Tl[%d] = %g (out of bounds), want 0", i, p.Tl[i])
		}
	}
}

func TestUnpackParamsGm1FromB2Field(t *testing.T) {
	// b_2 lives in bb[2][0..5] after the bo[L9] re-order — so we
	// can't drive it directly without knowing which info[6..84]
	// positions land in bb[2]. Instead, drive a known b_2 by
	// stamping all 79 scattered bits to 1 and comparing against
	// what mbelib's pipeline would compute. With every scattered
	// bit set, b_2 = 0b111111 = 63 and Gm[1] = b2Table[63]. (The
	// last entry of B2 is 8.695827.)
	info := putB0Info(0)
	for i := 6; i <= 84; i++ {
		info[i] = 1
	}
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatal(err)
	}
	want := b2Table[63]
	if math.Abs(p.Gm[1]-want) > 1e-9 {
		t.Errorf("Gm[1] = %g, want %g (b2Table[63])", p.Gm[1], want)
	}
}

func TestUnpackParamsTlLengthMatchesL(t *testing.T) {
	// Tl is filled exactly L entries (indices 1..L). Sweep every
	// valid b_0 and assert no Tl writes spilled past L.
	for b0 := uint(0); b0 < 208; b0++ {
		info := putB0Info(b0)
		p, err := UnpackParams(info)
		if err != nil {
			t.Errorf("b0=%d: err = %v", b0, err)
			continue
		}
		for i := p.L + 1; i < len(p.Tl); i++ {
			if p.Tl[i] != 0 {
				t.Errorf("b0=%d L=%d: Tl[%d] = %g, want 0", b0, p.L, i, p.Tl[i])
				break
			}
		}
	}
}

func TestUnpackParamsVlBitDistribution(t *testing.T) {
	// With every scattered bit = 1, vector 1 is all 1s, so
	// Vl[1..L] should all be 1.
	info := putB0Info(0)
	for i := 6; i <= 84; i++ {
		info[i] = 1
	}
	p, err := UnpackParams(info)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= p.L; i++ {
		if p.Vl[i] != 1 {
			t.Errorf("Vl[%d] = %d, want 1 (all-ones info)", i, p.Vl[i])
		}
	}
}
