package framing

// Hamming(13,9,3): a single-error-correcting linear block code used by
// DMR's BPTC(196,96) column code (ETSI TS 102 361-1 Annex B). 9 information
// bits become a 13-bit codeword via four parity bits.
//
// The parity-check matrix uses the 9 nonzero, non-unit 4-bit vectors of
// weight ≥ 2 as data-column patterns, guaranteeing minimum distance 3:
//
//   data bit  pattern (bit3 bit2 bit1 bit0)
//   d0        0011        d4        1001
//   d1        0101        d5        1010
//   d2        0110        d6        1011
//   d3        0111        d7        1100
//                         d8        1101
//
// Parity equations (LSB-indexed information bits d0..d8):
//   p0 = d0 ^ d1 ^ d3 ^ d4 ^ d6 ^ d8
//   p1 = d0 ^ d2 ^ d3 ^ d5 ^ d6
//   p2 = d1 ^ d2 ^ d3 ^ d7 ^ d8
//   p3 = d4 ^ d5 ^ d6 ^ d7 ^ d8
//
// Codeword layout: bits 12..4 = d8..d0, bits 3..0 = p3..p0.

// HammingEncode13_9 encodes 9 data bits (in the low 9 bits of input) into
// a 13-bit codeword.
func HammingEncode13_9(data uint16) uint16 {
	d := data & 0x01FF
	bit := func(i int) uint16 { return (d >> i) & 1 }
	p0 := bit(0) ^ bit(1) ^ bit(3) ^ bit(4) ^ bit(6) ^ bit(8)
	p1 := bit(0) ^ bit(2) ^ bit(3) ^ bit(5) ^ bit(6)
	p2 := bit(1) ^ bit(2) ^ bit(3) ^ bit(7) ^ bit(8)
	p3 := bit(4) ^ bit(5) ^ bit(6) ^ bit(7) ^ bit(8)
	return d<<4 | p3<<3 | p2<<2 | p1<<1 | p0
}

// HammingDecode13_9 decodes a 13-bit codeword. Returns (data, errors).
// errors is 0 (clean), 1 (single bit corrected), or -1 (uncorrectable).
func HammingDecode13_9(cw uint16) (uint16, int) {
	cw &= 0x1FFF
	expected := HammingEncode13_9(cw >> 4)
	if (expected^cw)&0x0F == 0 {
		return cw >> 4, 0
	}
	for i := 0; i < 13; i++ {
		flipped := cw ^ (1 << uint(i))
		if (HammingEncode13_9(flipped>>4)^flipped)&0x0F == 0 {
			return flipped >> 4, 1
		}
	}
	return cw >> 4, -1
}
