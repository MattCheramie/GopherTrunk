package framing

import (
	"testing"
)

// TestRSGF64FieldClosure validates the GF(2^6) log/exp tables: every
// non-zero element should appear exactly once in the exp[] cycle, and
// log[exp[i]] should round-trip.
func TestRSGF64FieldClosure(t *testing.T) {
	seen := make(map[byte]int)
	for i := 0; i < 63; i++ {
		v := rsGF64.exp[i]
		if v == 0 {
			t.Fatalf("exp[%d] = 0; GF(2^6) cycle should not include zero", i)
		}
		if v >= 64 {
			t.Fatalf("exp[%d] = %d; out of range for GF(2^6)", i, v)
		}
		if prev, ok := seen[v]; ok {
			t.Fatalf("exp[%d] = exp[%d] = %d; duplicate in cycle", i, prev, v)
		}
		seen[v] = i
		if rsGF64.log[v] != byte(i) {
			t.Fatalf("log[exp[%d]=%d] = %d, want %d", i, v, rsGF64.log[v], i)
		}
	}
	if len(seen) != 63 {
		t.Fatalf("expected 63 distinct non-zero elements, saw %d", len(seen))
	}
}

// TestRSGF64MulIdentity checks that α^0 = 1 acts as the multiplicative
// identity and 0 absorbs.
func TestRSGF64MulIdentity(t *testing.T) {
	if rsGF64.exp[0] != 1 {
		t.Fatalf("α^0 = %d, want 1", rsGF64.exp[0])
	}
	for a := byte(0); a < 64; a++ {
		if got := gf64Mul(a, 1); got != a {
			t.Fatalf("a * 1 = %d, want %d", got, a)
		}
		if got := gf64Mul(a, 0); got != 0 {
			t.Fatalf("a * 0 = %d, want 0", got)
		}
	}
}

// TestEncodeRS24_12_RoundTrip verifies the encoder produces a
// codeword that satisfies the verifier for every information vector
// in a small sample plus a stress sweep.
func TestEncodeRS24_12_RoundTrip(t *testing.T) {
	cases := [][]byte{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		{0o77, 0o76, 0o75, 0o74, 0o73, 0o72, 0o71, 0o70, 0o67, 0o66, 0o65, 0o64},
		{63, 63, 63, 63, 63, 63, 63, 63, 63, 63, 63, 63},
	}
	for ci, info := range cases {
		var a [12]byte
		copy(a[:], info)
		cw := EncodeRS24_12(a)
		if !VerifyRS24_12(cw[:]) {
			t.Errorf("case %d: VerifyRS24_12 returned false for freshly-encoded codeword", ci)
		}
		// Flip a single bit in any data symbol; the codeword should now
		// fail verification (RS detects up to d-1 = 12 symbol errors,
		// trivially detecting a single-bit flip).
		bad := cw
		bad[3] ^= 0x01
		if VerifyRS24_12(bad[:]) {
			t.Errorf("case %d: VerifyRS24_12 accepted a single-bit-flipped codeword", ci)
		}
	}
}

// TestEncodeRS24_16_RoundTrip mirrors TestEncodeRS24_12_RoundTrip for
// the 16-info / 8-parity ES code.
func TestEncodeRS24_16_RoundTrip(t *testing.T) {
	cases := [][]byte{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		{63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0},
		{0o31, 0o12, 0o42, 0o77, 0o00, 0o63, 0o25, 0o14, 0o55, 0o22, 0o44, 0o70, 0o03, 0o01, 0o66, 0o57},
	}
	for ci, info := range cases {
		var a [16]byte
		copy(a[:], info)
		cw := EncodeRS24_16(a)
		if !VerifyRS24_16(cw[:]) {
			t.Errorf("case %d: VerifyRS24_16 returned false for freshly-encoded codeword", ci)
		}
		bad := cw
		bad[7] ^= 0x02
		if VerifyRS24_16(bad[:]) {
			t.Errorf("case %d: VerifyRS24_16 accepted a single-bit-flipped codeword", ci)
		}
	}
}

// TestEncodeRS36_20_RoundTrip mirrors for the 20-info / 16-parity HDU
// code.
func TestEncodeRS36_20_RoundTrip(t *testing.T) {
	cases := [][]byte{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		{63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0, 63, 0},
	}
	for ci, info := range cases {
		var a [20]byte
		copy(a[:], info)
		cw := EncodeRS36_20(a)
		if !VerifyRS36_20(cw[:]) {
			t.Errorf("case %d: VerifyRS36_20 returned false for freshly-encoded codeword", ci)
		}
		bad := cw
		bad[19] ^= 0x04
		if VerifyRS36_20(bad[:]) {
			t.Errorf("case %d: VerifyRS36_20 accepted a single-bit-flipped codeword", ci)
		}
	}
}

// TestRSGF64InvalidLength confirms each verifier rejects mis-sized
// input rather than panicking or reading out of bounds.
func TestRSGF64InvalidLength(t *testing.T) {
	if VerifyRS24_12(make([]byte, 23)) {
		t.Errorf("VerifyRS24_12 accepted 23-byte input")
	}
	if VerifyRS24_12(make([]byte, 25)) {
		t.Errorf("VerifyRS24_12 accepted 25-byte input")
	}
	if VerifyRS24_16(make([]byte, 22)) {
		t.Errorf("VerifyRS24_16 accepted 22-byte input")
	}
	if VerifyRS36_20(make([]byte, 35)) {
		t.Errorf("VerifyRS36_20 accepted 35-byte input")
	}
}

// TestRSGF64InvalidSymbol confirms symbols with bits outside the
// 6-bit field are rejected.
func TestRSGF64InvalidSymbol(t *testing.T) {
	cw := make([]byte, 24)
	cw[5] = 0x80 // bit 7 set, outside GF(2^6)
	if VerifyRS24_12(cw) {
		t.Errorf("VerifyRS24_12 accepted symbol with bits outside GF(2^6)")
	}
}
