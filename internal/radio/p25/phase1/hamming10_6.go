package phase1

// Shortened Hamming(10, 6, 3) code — 6 data bits + 4 parity bits,
// single-error-correcting. It is the per-codeword inner FEC the P25
// LDU Link Control / Encryption Sync word uses (24 codewords spanning
// the 240-bit LC/ES field). The framing package's Hamming helpers are
// the (15,11) and (13,9) shortenings, not this one, so it is
// implemented locally — a small, self-contained table-driven code.
//
// hamming10_6ParityCols[i] is the 4-bit parity column for data bit i.
// Together with the four unit columns 0x8/0x4/0x2/0x1 (the parity-bit
// positions) all ten columns are distinct and nonzero, which makes
// single-error correction a syndrome → bit-position lookup.
var hamming10_6ParityCols = [6]uint8{0x7, 0xB, 0xD, 0xE, 0x3, 0x5}

// hamming10_6Parity computes the 4 parity bits over data[0:6] (0/1 per
// byte) as a 4-bit value.
func hamming10_6Parity(data []byte) uint8 {
	var p uint8
	for i := 0; i < 6; i++ {
		if data[i]&1 != 0 {
			p ^= hamming10_6ParityCols[i]
		}
	}
	return p
}

// encodeHamming10_6 encodes 6 data bits into a 10-bit codeword (one bit
// per byte, data in positions 0..5, parity in 6..9).
func encodeHamming10_6(data []byte) []byte {
	cw := make([]byte, 10)
	copy(cw[0:6], data[0:6])
	p := hamming10_6Parity(data)
	cw[6] = (p >> 3) & 1
	cw[7] = (p >> 2) & 1
	cw[8] = (p >> 1) & 1
	cw[9] = p & 1
	return cw
}

// decodeHamming10_6 corrects up to one bit error in a 10-bit codeword
// and returns the 6 data bits plus the count of corrected errors.
func decodeHamming10_6(cw []byte) ([]byte, int) {
	c := make([]byte, 10)
	copy(c, cw)
	rxParity := (c[6]&1)<<3 | (c[7]&1)<<2 | (c[8]&1)<<1 | (c[9] & 1)
	syn := hamming10_6Parity(c[0:6]) ^ rxParity
	errs := 0
	if syn != 0 {
		pos := -1
		for i := 0; i < 6; i++ {
			if hamming10_6ParityCols[i] == syn {
				pos = i
				break
			}
		}
		if pos < 0 {
			for j := 0; j < 4; j++ {
				if uint8(0x8>>uint(j)) == syn {
					pos = 6 + j
					break
				}
			}
		}
		if pos >= 0 {
			c[pos] ^= 1
			errs = 1
		}
	}
	out := make([]byte, 6)
	copy(out, c[0:6])
	return out, errs
}
