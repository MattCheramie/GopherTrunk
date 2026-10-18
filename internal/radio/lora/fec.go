package lora

import "math/bits"

// This file implements the LoRa symbol<->bit forward-error-correction
// transforms in the Semtech air format: Hamming(4+cr, 4) coding, the
// diagonal interleaver, Gray mapping and the chip-value mapping (including
// the +1 chip offset and the reduced-rate x4 scaling). Every transform is
// pinned by literal vectors from two independent open implementations
// (gr-lora_sdr and jkadbear/LoRaPHY) in bitexact_test.go; a round trip
// alone cannot catch a layout that the encoder and decoder share.

// --- Hamming(4+cr, 4) ----------------------------------------------------
//
// With the nibble bits n0 (LSB) .. n3, the codeword read MSB first is
//
//	4/8: n0 n1 n2 n3 p0 p1 p2 p3     p0 = n0^n1^n2, p1 = n1^n2^n3,
//	4/7: n0 n1 n2 n3 p0 p1 p2        p2 = n0^n1^n3, p3 = n0^n2^n3
//	4/6: n0 n1 n2 n3 p0 p1
//	4/5: n0 n1 n2 n3 p               p  = n0^n1^n2^n3
//
// i.e. the data nibble sits bit-reversed in the top four bits.

// HammingEncode4 encodes a 4-bit nibble into its (4+cr)-bit codeword.
func HammingEncode4(nibble uint8, cr int) uint16 {
	n0 := uint16(nibble) & 1
	n1 := uint16(nibble>>1) & 1
	n2 := uint16(nibble>>2) & 1
	n3 := uint16(nibble>>3) & 1
	if cr == 1 {
		return n0<<4 | n1<<3 | n2<<2 | n3<<1 | (n0 ^ n1 ^ n2 ^ n3)
	}
	p0 := n0 ^ n1 ^ n2
	p1 := n1 ^ n2 ^ n3
	p2 := n0 ^ n1 ^ n3
	p3 := n0 ^ n2 ^ n3
	cw := n0<<7 | n1<<6 | n2<<5 | n3<<4 | p0<<3 | p1<<2 | p2<<1 | p3
	return cw >> uint(4-cr)
}

// hammingData extracts the (unchecked) data nibble from a codeword.
func hammingData(cw uint16, cr int) uint8 {
	w := uint(4 + cr)
	b := func(i uint) uint8 { return uint8(cw>>(w-1-i)) & 1 }
	return b(0) | b(1)<<1 | b(2)<<2 | b(3)<<3
}

// HammingDecode4 decodes one (4+cr)-bit codeword to its 4-bit nibble by
// minimum distance over the 16 codewords. corrected reports the outcome:
// 0 = clean, 1 = a single bit error was corrected (4/7, 4/8 only), -1 = an
// error was detected but not corrected (nibble is then the raw data bits).
// 4/5 and 4/6 only detect.
func HammingDecode4(cw uint16, cr int) (nibble uint8, corrected int) {
	if cr < 1 || cr > 4 {
		return hammingData(cw, 4), -1
	}
	mask := uint16(1)<<uint(4+cr) - 1
	cw &= mask
	best, bestDist, ties := uint8(0), 99, 0
	for n := uint8(0); n < 16; n++ {
		d := bits.OnesCount16(cw ^ HammingEncode4(n, cr))
		switch {
		case d < bestDist:
			best, bestDist, ties = n, d, 1
		case d == bestDist:
			ties++
		}
	}
	switch {
	case bestDist == 0:
		return best, 0
	case bestDist == 1 && cr >= 3 && ties == 1:
		return best, 1
	default:
		return hammingData(cw, cr), -1
	}
}

// --- diagonal interleaver -----------------------------------------------

// Interleave maps rows (sf_app) codewords of cwLen bits into cwLen
// symbols of rows bits. With bit positions counted MSB first (codeword
// bit 0 = its MSB, symbol bit 0 = its MSB):
//
//	symbol[i] bit j = codeword[(i-j-1) mod rows] bit i
//
// Missing codewords (a short final block) are zero.
func Interleave(codewords []uint16, rows, cwLen int) []uint16 {
	syms := make([]uint16, cwLen)
	for i := 0; i < cwLen; i++ {
		for j := 0; j < rows; j++ {
			r := modN(i-j-1, rows)
			if r < len(codewords) && codewords[r]>>uint(cwLen-1-i)&1 != 0 {
				syms[i] |= 1 << uint(rows-1-j)
			}
		}
	}
	return syms
}

// Deinterleave inverts Interleave: cwLen symbols of rows bits -> rows
// codewords of cwLen bits.
func Deinterleave(syms []uint16, rows, cwLen int) []uint16 {
	codewords := make([]uint16, rows)
	for i := 0; i < cwLen && i < len(syms); i++ {
		for j := 0; j < rows; j++ {
			if syms[i]>>uint(rows-1-j)&1 != 0 {
				codewords[modN(i-j-1, rows)] |= 1 << uint(cwLen-1-i)
			}
		}
	}
	return codewords
}

// --- Gray mapping and chip values ---------------------------------------

// grayEncode maps a binary value to its Gray code (the RX direction: a
// demodulated value is Gray-encoded to recover the interleaver symbol).
func grayEncode(v uint16) uint16 { return v ^ (v >> 1) }

// grayDecode inverts grayEncode (the TX direction): cumulative XOR.
func grayDecode(g uint16) uint16 {
	g ^= g >> 8
	g ^= g >> 4
	g ^= g >> 2
	g ^= g >> 1
	return g
}

// symbolToChip maps an interleaver symbol to the transmitted chip value:
// Gray-decode, scale by 4 in a reduced-rate block (the two low chip bits
// carry no data, which is what makes the block tolerate a ±1-bin drift),
// then add one. In gr-lora_sdr the reduced-rate symbol carries a parity
// bit and a zero in its two low bits before an SF-bit Gray decode; that
// parity is chosen so those two bits Gray-decode to zero, which makes it
// identical to LoRaPHY's num*4+1 used here (bitexact_test.go pins both).
func symbolToChip(sym uint16, sf int, reduced bool) uint16 {
	v := grayDecode(sym)
	if reduced {
		v <<= 2
	}
	return uint16(modN(int(v)+1, 1<<uint(sf)))
}

// chipToSymbol inverts symbolToChip on a demodulated bin: subtract one,
// divide a reduced-rate block's value by 4, Gray-encode. The division
// ROUNDS to the nearest multiple of four (gr-lora_sdr floors): a
// reduced-rate chip is always 4k+1, so a bin one off either way still
// reads k — the drift tolerance LDRO exists for. Flooring would read a
// one-bin-low peak as k-1. Identical on every exact chip.
func chipToSymbol(chip uint16, sf int, reduced bool) uint16 {
	n := 1 << uint(sf)
	v := modN(int(chip)-1, n)
	if reduced {
		v = ((v + 2) >> 2) % (n >> 2)
	}
	return grayEncode(uint16(v))
}
