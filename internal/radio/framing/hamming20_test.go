package framing

import "testing"

func TestHamming20_8RoundTrip(t *testing.T) {
	for d := 0; d < 256; d++ {
		cw := HammingEncode20_8(uint8(d))
		got, errs := HammingDecode20_8(cw)
		if errs != 0 {
			t.Errorf("d=%02x clean codeword reported %d errors", d, errs)
		}
		if int(got) != d {
			t.Errorf("d=%02x: got=%02x", d, got)
		}
	}
}

func TestHamming20_8CodewordIsSystematic(t *testing.T) {
	for _, d := range []uint8{0x00, 0x01, 0x80, 0xAB, 0xFF} {
		cw := HammingEncode20_8(d)
		gotInfo := uint8((cw >> 12) & 0xFF)
		if gotInfo != d {
			t.Errorf("info bits not systematic: d=%02x cw>>12 = %02x", d, gotInfo)
		}
	}
}

func TestHamming20_8CorrectsSingleErrors(t *testing.T) {
	const d uint8 = 0xAB
	cw := HammingEncode20_8(d)
	for bit := 0; bit < 20; bit++ {
		corrupted := cw ^ (uint32(1) << uint(bit))
		got, errs := HammingDecode20_8(corrupted)
		if got != d {
			t.Errorf("bit=%d: got=%02x want=%02x", bit, got, d)
		}
		if errs != 1 {
			t.Errorf("bit=%d: errs=%d want=1", bit, errs)
		}
	}
}

func TestHamming20_8CorrectsTripleErrors(t *testing.T) {
	const d uint8 = 0x5A
	cw := HammingEncode20_8(d)
	// Flip 3 bits — at the t=3 correction radius.
	corrupted := cw ^ 0b101010 // bits 1, 3, 5
	got, errs := HammingDecode20_8(corrupted)
	if got != d {
		t.Fatalf("triple-error decode: got %02x want %02x", got, d)
	}
	if errs != 3 {
		t.Errorf("errs=%d want=3", errs)
	}
}

func TestHamming20_8RejectsFourErrors(t *testing.T) {
	const d uint8 = 0x33
	cw := HammingEncode20_8(d)
	// Flip 4 bits — beyond the correction radius. The decoder should
	// either flag uncorrectable (errs == -1) or land on a closer-but-
	// wrong codeword; either way errs must NOT be in [0..3] with the
	// original data, since that would mislead the caller.
	corrupted := cw ^ 0b10101010 // bits 1, 3, 5, 7
	got, errs := HammingDecode20_8(corrupted)
	if errs >= 0 && errs <= 3 && got == d {
		t.Errorf("4 errors mis-decoded as %d-error correction to original data", errs)
	}
}

func TestHamming20_8MinimumDistance(t *testing.T) {
	// ETSI references the slot-type FEC as a (20,8,7) shortened Hamming
	// code. With the published parity equations the actual achieved
	// minimum distance is 8 (an extended-Hamming property), giving the
	// decoder a stricter detection radius than the spec's lower bound.
	// We pin the value here so a regression in the parity matrix would
	// surface as a distance drop.
	codewords := make([]uint32, 256)
	for i := 0; i < 256; i++ {
		codewords[i] = HammingEncode20_8(uint8(i))
	}
	minDist := 21
	for i := 0; i < 256; i++ {
		for j := i + 1; j < 256; j++ {
			d := PopCount64(uint64(codewords[i] ^ codewords[j]))
			if d < minDist {
				minDist = d
			}
		}
	}
	if minDist != 8 {
		t.Errorf("min distance = %d, want 8", minDist)
	}
}
