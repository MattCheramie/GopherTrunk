package framing

import (
	"errors"
	"math/rand"
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

// TestRSGF64Inverse checks gf64Inv: a · a^-1 = 1 for every non-zero
// element, and the inverse of 1 is 1.
func TestRSGF64Inverse(t *testing.T) {
	if gf64Inv(1) != 1 {
		t.Fatalf("gf64Inv(1) = %d, want 1", gf64Inv(1))
	}
	for a := byte(1); a < 64; a++ {
		if got := gf64Mul(a, gf64Inv(a)); got != 1 {
			t.Fatalf("a=%d: a * a^-1 = %d, want 1", a, got)
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

// randInfo returns n random GF(2^6) symbols seeded deterministically.
func randInfo(rng *rand.Rand, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(rng.Intn(64))
	}
	return out
}

// corruptSymbols flips `count` distinct codeword positions to random
// *different* GF(2^6) values, returning the corrupted copy.
func corruptSymbols(rng *rand.Rand, cw []byte, count int) []byte {
	bad := append([]byte(nil), cw...)
	perm := rng.Perm(len(cw))
	for k := 0; k < count; k++ {
		pos := perm[k]
		for {
			v := byte(rng.Intn(64))
			if v != bad[pos] {
				bad[pos] = v
				break
			}
		}
	}
	return bad
}

// TestDecodeRS24_12_Correction sweeps random information vectors,
// injects 0..t (t=6) symbol errors, and confirms the decoder recovers
// the original information and reports the right error count. Injecting
// t+1 errors must return ErrRSUncorrectable rather than a wrong word.
func TestDecodeRS24_12_Correction(t *testing.T) {
	rng := rand.New(rand.NewSource(0x2412))
	for iter := 0; iter < 400; iter++ {
		info := randInfo(rng, 12)
		var a [12]byte
		copy(a[:], info)
		cw := EncodeRS24_12(a)

		for nerr := 0; nerr <= 6; nerr++ {
			bad := corruptSymbols(rng, cw[:], nerr)
			got, corrected, err := DecodeRS24_12(bad)
			if err != nil {
				t.Fatalf("iter %d nerr %d: unexpected error %v", iter, nerr, err)
			}
			if corrected != nerr {
				t.Fatalf("iter %d: reported %d corrected, want %d", iter, corrected, nerr)
			}
			if got != a {
				t.Fatalf("iter %d nerr %d: recovered %v, want %v", iter, nerr, got, a)
			}
		}
	}
}

// TestDecodeRS24_12_Uncorrectable confirms t+1 errors are flagged rather
// than silently miscorrected. (A small fraction of 7-error patterns can
// land on another codeword; the defensive re-syndrome guarantees we
// never return a word that fails to recover the original silently — we
// assert it either errors or, in the rare aliasing case, does not return
// the wrong info as if it were correct.)
func TestDecodeRS24_12_Uncorrectable(t *testing.T) {
	rng := rand.New(rand.NewSource(0x99))
	uncorrectable := 0
	for iter := 0; iter < 500; iter++ {
		info := randInfo(rng, 12)
		var a [12]byte
		copy(a[:], info)
		cw := EncodeRS24_12(a)
		bad := corruptSymbols(rng, cw[:], 7)
		got, _, err := DecodeRS24_12(bad)
		if err != nil {
			if !errors.Is(err, ErrRSUncorrectable) {
				t.Fatalf("iter %d: unexpected error %v", iter, err)
			}
			uncorrectable++
			continue
		}
		// No error returned: the only acceptable outcome is that the
		// decoder landed on a *valid* codeword (aliasing). It must not
		// claim to have recovered our original from 7 errors via ≤6
		// corrections, which would be a miscorrection bug.
		if got == a {
			t.Fatalf("iter %d: 7-error pattern 'recovered' the original — impossible within t=6", iter)
		}
	}
	if uncorrectable == 0 {
		t.Fatalf("expected most 7-error patterns to be flagged uncorrectable, got 0/500")
	}
}

// TestDecodeRS24_16_Correction sweeps the ES code (t=4).
func TestDecodeRS24_16_Correction(t *testing.T) {
	rng := rand.New(rand.NewSource(0x2416))
	for iter := 0; iter < 400; iter++ {
		info := randInfo(rng, 16)
		var a [16]byte
		copy(a[:], info)
		cw := EncodeRS24_16(a)
		for nerr := 0; nerr <= 4; nerr++ {
			bad := corruptSymbols(rng, cw[:], nerr)
			got, corrected, err := DecodeRS24_16(bad)
			if err != nil {
				t.Fatalf("iter %d nerr %d: unexpected error %v", iter, nerr, err)
			}
			if corrected != nerr {
				t.Fatalf("iter %d: reported %d corrected, want %d", iter, corrected, nerr)
			}
			if got != a {
				t.Fatalf("iter %d nerr %d: recovered %v, want %v", iter, nerr, got, a)
			}
		}
	}
}

// TestDecodeRS36_20_Correction sweeps the HDU code (t=8).
func TestDecodeRS36_20_Correction(t *testing.T) {
	rng := rand.New(rand.NewSource(0x3620))
	for iter := 0; iter < 300; iter++ {
		info := randInfo(rng, 20)
		var a [20]byte
		copy(a[:], info)
		cw := EncodeRS36_20(a)
		for nerr := 0; nerr <= 8; nerr++ {
			bad := corruptSymbols(rng, cw[:], nerr)
			got, corrected, err := DecodeRS36_20(bad)
			if err != nil {
				t.Fatalf("iter %d nerr %d: unexpected error %v", iter, nerr, err)
			}
			if corrected != nerr {
				t.Fatalf("iter %d: reported %d corrected, want %d", iter, corrected, nerr)
			}
			if got != a {
				t.Fatalf("iter %d nerr %d: recovered %v, want %v", iter, nerr, got, a)
			}
		}
	}
}
