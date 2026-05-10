package framing

import (
	"testing"
)

func TestRS129EncodeAllZeroIsAllZero(t *testing.T) {
	// Linearity sanity: encoding all-zero data must produce all-zero
	// parity (otherwise the encoder's shift register has a bug).
	cw := EncodeRS12_9([9]byte{})
	for i, b := range cw {
		if b != 0 {
			t.Errorf("cw[%d] = %#x, want 0 for all-zero data", i, b)
		}
	}
}

func TestRS129VerifyEncodedCodewordWithoutSeed(t *testing.T) {
	// Round-trip: anything we encode should verify with the no-seed
	// path. Tries a handful of representative payloads.
	cases := [][9]byte{
		{0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF},
		{0x42, 0x10, 0x00, 0x00, 0x42, 0x00, 0x00, 0x00, 0x42},
		{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7},
		{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x55},
	}
	for _, data := range cases {
		cw := EncodeRS12_9(data)
		if !VerifyRS12_9(cw[:], RS129SeedNone) {
			t.Errorf("encoded codeword failed verify: data=%x cw=%x", data, cw)
		}
	}
}

func TestRS129RoundTripWithDMRSeeds(t *testing.T) {
	// Every DMR Voice context applies its own XOR seed to the parity
	// bytes; verify the round-trip works for each.
	data := [9]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	seeds := []struct {
		name string
		seed [3]byte
	}{
		{"VoiceLCHeader", RS129SeedVoiceLCHeader},
		{"TerminatorLC", RS129SeedTerminatorLC},
		{"EmbeddedLC", RS129SeedEmbeddedLC},
	}
	for _, s := range seeds {
		t.Run(s.name, func(t *testing.T) {
			cw := EncodeRS12_9(data)
			// Apply the seed exactly the way the DMR transmitter
			// would.
			for i := 0; i < 3; i++ {
				cw[9+i] ^= s.seed[i]
			}
			if !VerifyRS12_9(cw[:], s.seed) {
				t.Errorf("round-trip failed for seed %s: cw=%x", s.name, cw)
			}
			// Wrong seed must fail.
			if VerifyRS12_9(cw[:], RS129SeedNone) {
				t.Errorf("wrong seed (none) accepted for %s", s.name)
			}
		})
	}
}

func TestRS129DetectsSingleByteCorruption(t *testing.T) {
	// The code's nominal capability is t = 1 error correction (3
	// parity symbols, distance 4). Verify-only here just needs to
	// catch any non-zero number of errors. Flip every position once
	// and confirm the verifier reports invalid.
	data := [9]byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x55}
	clean := EncodeRS12_9(data)
	for pos := 0; pos < 12; pos++ {
		corrupt := clean
		corrupt[pos] ^= 0x42 // flip a chunk of bits in one octet
		if VerifyRS12_9(corrupt[:], RS129SeedNone) {
			t.Errorf("verify accepted corruption at byte %d (cw=%x)", pos, corrupt)
		}
	}
}

func TestRS129DetectsSingleBitFlip(t *testing.T) {
	data := [9]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x55}
	clean := EncodeRS12_9(data)
	for pos := 0; pos < 12; pos++ {
		for bit := 0; bit < 8; bit++ {
			corrupt := clean
			corrupt[pos] ^= 1 << uint(bit)
			if VerifyRS12_9(corrupt[:], RS129SeedNone) {
				t.Errorf("verify accepted single-bit flip at byte %d bit %d", pos, bit)
			}
		}
	}
}

func TestRS129RejectsWrongLength(t *testing.T) {
	if VerifyRS12_9(make([]byte, 11), RS129SeedNone) {
		t.Error("verify accepted 11-byte input")
	}
	if VerifyRS12_9(make([]byte, 13), RS129SeedNone) {
		t.Error("verify accepted 13-byte input")
	}
}

func TestRS129VerifyDoesntMutateInput(t *testing.T) {
	data := [9]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	cw := EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= RS129SeedVoiceLCHeader[i]
	}
	cwCopy := cw
	_ = VerifyRS12_9(cw[:], RS129SeedVoiceLCHeader)
	if cw != cwCopy {
		t.Errorf("VerifyRS12_9 mutated input buffer: was %x got %x", cwCopy, cw)
	}
}

func TestGFMul2Order(t *testing.T) {
	// α has order 255 in GF(2^8) under a primitive polynomial. Verify
	// the multiplicative cycle covers all 255 non-zero elements.
	x := byte(1)
	seen := map[byte]bool{1: true}
	for i := 1; i < 255; i++ {
		x = gfMul2(x)
		if seen[x] {
			t.Fatalf("α=2 cycle repeats early at step %d (val %#x)", i, x)
		}
		seen[x] = true
	}
	// After 255 multiplications we're back at 1.
	x = gfMul2(x)
	if x != 1 {
		t.Errorf("α^255 = %#x, want 1 (α primitive in GF(2^8))", x)
	}
}
