package framing

import "testing"

// TestBCH6416RoundTrip: every clean BCH(64,16,11) codeword
// decodes back to its source info with zero errors.
func TestBCH6416RoundTrip(t *testing.T) {
	for d := uint32(0); d < 1<<16; d += 257 {
		cw := BCHEncode64_16(uint16(d))
		got, errs := BCHDecode64_16(cw)
		if errs != 0 {
			t.Errorf("d=%04x: clean decode reported %d errors", d, errs)
		}
		if uint32(got) != d {
			t.Errorf("d=%04x: decode round-trip = %04x", d, got)
		}
	}
}

// TestBCH6416CorrectsSingleBitErrors: flip any single bit in a
// 64-bit codeword and confirm the decoder still recovers the
// original info with errors == 1.
func TestBCH6416CorrectsSingleBitErrors(t *testing.T) {
	data := uint16(0xABCD)
	cw := BCHEncode64_16(data)
	for bit := 0; bit < 64; bit++ {
		corrupted := cw ^ (uint64(1) << uint(bit))
		got, errs := BCHDecode64_16(corrupted)
		if errs != 1 {
			t.Errorf("bit=%d: errs = %d, want 1", bit, errs)
		}
		if got != data {
			t.Errorf("bit=%d: data = %04x, want %04x", bit, got, data)
		}
	}
}

// TestBCH6416DetectsParityFlip: flipping the trailing parity bit
// alone is uncorrectable by the 63-bit BCH but the decoder must
// still report errs >= 1 (not a clean decode).
func TestBCH6416DetectsParityFlip(t *testing.T) {
	data := uint16(0x5A5A)
	cw := BCHEncode64_16(data)
	corrupted := cw ^ 1 // flip the parity bit
	got, errs := BCHDecode64_16(corrupted)
	if errs < 1 {
		t.Errorf("errs = %d, want >= 1", errs)
	}
	if got != data {
		t.Errorf("data = %04x, want %04x (parity flip should still recover info)",
			got, data)
	}
}

// TestBCH6416RejectsManyErrors: corrupt 12 bits in a 64-bit
// codeword (one past the 11-bit correction limit) and confirm
// the decoder reports errs == -1.
func TestBCH6416RejectsManyErrors(t *testing.T) {
	data := uint16(0xDEAD)
	cw := BCHEncode64_16(data)
	// Flip bits 0..11 — 12 bit flips.
	for bit := 0; bit < 12; bit++ {
		cw ^= uint64(1) << uint(bit)
	}
	_, errs := BCHDecode64_16(cw)
	if errs != -1 {
		t.Errorf("errs = %d, want -1 (uncorrectable)", errs)
	}
}

func TestBCH6316RoundTrip(t *testing.T) {
	// Sparse sweep across the 65536-entry info space; brute-force decode
	// is ~1 ms per call so an exhaustive sweep would dominate the test
	// suite without adding meaningful coverage.
	for d := uint32(0); d < 1<<16; d += 257 {
		cw := BCHEncode63_16(uint16(d))
		got, errs := BCHDecode63_16(cw)
		if errs != 0 {
			t.Errorf("d=%04x: clean decode reported %d errors", d, errs)
		}
		if uint32(got) != d {
			t.Errorf("d=%04x: got=%04x", d, got)
		}
	}
}

func TestBCH6316CodewordIsSystematic(t *testing.T) {
	// Info bits should land verbatim in positions 62..47 of the codeword.
	for _, d := range []uint16{0x0000, 0x0001, 0x8000, 0xABCD, 0xFFFF} {
		cw := BCHEncode63_16(d)
		gotInfo := uint16((cw >> 47) & 0xFFFF)
		if gotInfo != d {
			t.Errorf("info bits not systematic: d=%04x, cw>>47 = %04x", d, gotInfo)
		}
	}
}

func TestBCH6316CorrectsSingleErrors(t *testing.T) {
	const d uint16 = 0xABCD
	cw := BCHEncode63_16(d)
	for bit := 0; bit < 63; bit++ {
		corrupted := cw ^ (uint64(1) << uint(bit))
		got, errs := BCHDecode63_16(corrupted)
		if got != d {
			t.Errorf("bit=%d: got=%04x want=%04x", bit, got, d)
		}
		if errs != 1 {
			t.Errorf("bit=%d: errs=%d want=1", bit, errs)
		}
	}
}

func TestBCH6316CorrectsElevenErrors(t *testing.T) {
	const d uint16 = 0x1234
	cw := BCHEncode63_16(d)
	// Flip 11 bits — within the t=11 correction radius.
	mask := uint64(0)
	for bit := 0; bit < 11; bit++ {
		mask |= uint64(1) << uint(bit*5)
	}
	got, errs := BCHDecode63_16(cw ^ mask)
	if got != d {
		t.Fatalf("11-error decode: got=%04x want=%04x", got, d)
	}
	if errs != 11 {
		t.Errorf("errs=%d want=11", errs)
	}
}

func TestBCH6316RejectsTwelveErrors(t *testing.T) {
	const d uint16 = 0x5A5A
	cw := BCHEncode63_16(d)
	// Flip 12 bits — beyond the correction radius. The decoder should
	// either flag the codeword as uncorrectable (errs == -1) or settle
	// on a closer-but-wrong codeword; we accept either outcome and only
	// require that the caller can distinguish "trustworthy" from "not"
	// via the errs value.
	mask := uint64(0)
	for bit := 0; bit < 12; bit++ {
		mask |= uint64(1) << uint(bit*5)
	}
	_, errs := BCHDecode63_16(cw ^ mask)
	if errs >= 0 && errs <= 11 {
		t.Errorf("12 errors should be flagged uncorrectable, got errs=%d", errs)
	}
}

func TestBCH6316MinimumDistance(t *testing.T) {
	// Two distinct codewords must differ in >=23 bit positions (designed
	// distance d=23, equivalent to t=11 correction).
	a := BCHEncode63_16(0x0000)
	b := BCHEncode63_16(0x0001)
	dist := PopCount64(a ^ b)
	if dist < 23 {
		t.Errorf("d(c0, c1) = %d, want >= 23", dist)
	}
}

func TestBCH6316ParityBit(t *testing.T) {
	// Parity bit is even parity over the 63 codeword bits.
	for _, d := range []uint16{0x0000, 0xFFFF, 0xAAAA, 0x5555} {
		cw := BCHEncode63_16(d)
		p := BCH6316ParityBit(cw)
		// Re-XOR every set bit to verify parity.
		want := byte(PopCount64(cw) & 1)
		if p != want {
			t.Errorf("d=%04x parity=%d want=%d", d, p, want)
		}
	}
}
