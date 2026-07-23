package acelp

import "testing"

// TestBasicOperators checks the documented edge cases of the fixed-point
// operators (from the ETSI/ITU-T definitions) that make the decoder bit-exact.
func TestBasicOperators(t *testing.T) {
	if absS(minWord16) != maxWord16 {
		t.Errorf("absS(-32768) = %d, want 32767", absS(minWord16))
	}
	if addOp(maxWord16, 1) != maxWord16 {
		t.Errorf("addOp saturate high failed")
	}
	if addOp(minWord16, -1) != minWord16 {
		t.Errorf("addOp saturate low failed")
	}
	if subOp(minWord16, 1) != minWord16 {
		t.Errorf("subOp saturate low failed")
	}
	if negate(minWord16) != maxWord16 {
		t.Errorf("negate(-32768) = %d, want 32767", negate(minWord16))
	}
	if mult(minWord16, minWord16) != maxWord16 {
		t.Errorf("mult(-32768,-32768) = %d, want 32767", mult(minWord16, minWord16))
	}
	if multR(minWord16, minWord16) != maxWord16 {
		t.Errorf("multR(-32768,-32768) = %d, want 32767", multR(minWord16, minWord16))
	}
	if got := mult(16384, 16384); got != 8192 { // 0.5 * 0.5 = 0.25 in Q15
		t.Errorf("mult(16384,16384) = %d, want 8192", got)
	}
	if lMult(minWord16, minWord16) != maxWord32 {
		t.Errorf("lMult(-32768,-32768) = %d, want MAX_32", lMult(minWord16, minWord16))
	}
	if got := lMult(16384, 16384); got != 0x20000000 { // (0.5*0.5)<<1 in Q31
		t.Errorf("lMult(16384,16384) = %#x, want 0x20000000", got)
	}
	if lAdd(maxWord32, 1) != maxWord32 {
		t.Errorf("lAdd saturate high failed")
	}
	if lSub(minWord32, 1) != minWord32 {
		t.Errorf("lSub saturate low failed")
	}
	if got := etsiRound(0x00018000); got != 2 { // round 1.5 -> 2 at the 16-MSB point
		t.Errorf("etsiRound(0x00018000) = %d, want 2", got)
	}
	if got := extractH(0x12345678); got != 0x1234 {
		t.Errorf("extractH = %#x, want 0x1234", got)
	}
}

// TestShiftsAndNorm checks the shift and normalisation operators, including the
// shl(v, norm_s(v)) normalisation identity.
func TestShiftsAndNorm(t *testing.T) {
	if got := shl(1, 3); got != 8 {
		t.Errorf("shl(1,3) = %d, want 8", got)
	}
	if got := shl(16384, 2); got != maxWord16 { // overflow saturates positive
		t.Errorf("shl overflow = %d, want 32767", got)
	}
	if got := shr(-8, 2); got != -2 { // arithmetic right shift
		t.Errorf("shr(-8,2) = %d, want -2", got)
	}
	if got := shr(v16(1), -3); got != 8 { // negative count = left shift
		t.Errorf("shr(1,-3) = %d, want 8", got)
	}
	if got := lShr(-8, 2); got != -2 {
		t.Errorf("lShr(-8,2) = %d, want -2", got)
	}
	if got := lShl(1, 4); got != 16 {
		t.Errorf("lShl(1,4) = %d, want 16", got)
	}
	// normalisation identity: shl(v, normS(v)) lands in [0x4000,0x7fff] magnitude.
	for _, v := range []int16{1, 100, -100, 0x0123, -0x0123, 0x4000} {
		n := normS(v)
		norm := shl(v, n)
		if v > 0 && (norm < 0x4000 || norm > 0x7fff) {
			t.Errorf("normS(%d): shl gave %#x, not normalised", v, norm)
		}
	}
	if normL(0) != 0 || normL(-1) != 31 {
		t.Errorf("normL edge cases failed")
	}
	if got := divS(16384, 32767); got <= 0 || got >= maxWord16 {
		t.Errorf("divS(16384,32767) = %d, out of range", got)
	}
	if divS(100, 100) != maxWord16 {
		t.Errorf("divS(a,a) should be 32767")
	}
}

func v16(x int16) int16 { return x }
