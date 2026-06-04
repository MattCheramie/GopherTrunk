package framing

// Hamming(13,9,3): a single-error-correcting linear block code used by
// DMR's BPTC(196,96) column code (ETSI TS 102 361-1 Annex B.1.2). 9
// information bits become a 13-bit codeword via four parity bits.
//
// The parity equations are the canonical DMR ones (matching the reference
// MMDVM CHamming::encode1393 / decode1393, which is the de-facto on-air
// implementation used by DMR hotspots). In the spec the codeword cells are
// d[0..8] = data, d[9..12] = parity, with:
//
//   d[9]  = d0 ^ d1 ^ d3 ^ d5 ^ d6
//   d[10] = d0 ^ d1 ^ d2 ^ d4 ^ d6 ^ d7
//   d[11] = d0 ^ d1 ^ d2 ^ d3 ^ d5 ^ d7 ^ d8
//   d[12] = d0 ^ d2 ^ d4 ^ d5 ^ d8
//
// This file packs the same code into a uint16 codeword: bits 12..4 hold
// d8..d0 (data, d0 at bit 4), and bits 3..0 hold the parity in the order
// p0=d[9], p1=d[10], p2=d[11], p3=d[12]. The BPTC column pass in bptc.go
// relies on that p0..p3 == d[9]..d[12] ordering so the column cells
// m[9..12][c] map straight onto the spec parity cells.

// HammingEncode13_9 encodes 9 data bits (in the low 9 bits of input) into
// a 13-bit codeword.
func HammingEncode13_9(data uint16) uint16 {
	d := data & 0x01FF
	bit := func(i int) uint16 { return (d >> i) & 1 }
	p0 := bit(0) ^ bit(1) ^ bit(3) ^ bit(5) ^ bit(6)
	p1 := bit(0) ^ bit(1) ^ bit(2) ^ bit(4) ^ bit(6) ^ bit(7)
	p2 := bit(0) ^ bit(1) ^ bit(2) ^ bit(3) ^ bit(5) ^ bit(7) ^ bit(8)
	p3 := bit(0) ^ bit(2) ^ bit(4) ^ bit(5) ^ bit(8)
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
