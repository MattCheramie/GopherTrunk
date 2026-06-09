package lora

// This file implements the LoRa symbol<->bit forward-error-correction
// transforms as exact-inverse pairs: Gray mapping, the diagonal
// interleaver, and Hamming(4+cr, 4) coding. The encode (TX) and decode
// (RX) directions are inverses, so the chain is verifiable by round-trip
// tests; CR 4/7 and 4/8 additionally correct a single bit error per
// codeword. Exact bit-level interoperability with Semtech silicon
// (interleaver rotation sign, codeword bit order) is a calibration step
// gated behind captured golden vectors — see the package tests.

// grayEncode maps a binary value to its Gray code (TX direction).
func grayEncode(v uint16) uint16 { return v ^ (v >> 1) }

// grayDecode inverts grayEncode (RX direction): cumulative XOR.
func grayDecode(g uint16) uint16 {
	g ^= g >> 8
	g ^= g >> 4
	g ^= g >> 2
	g ^= g >> 1
	return g
}

// --- diagonal interleaver -----------------------------------------------

// Interleave maps ppm data codewords, each (4+cr) bits, into (4+cr)
// symbols, each ppm bits, using a diagonal rotation. cr is 1..4. The
// inverse is Deinterleave.
//
//	symbol[k] bit ((r+k) mod ppm) = codeword[r] bit k
//
// codewords[r] holds bits 0..(4+cr-1) (LSB = bit 0). The returned symbols
// hold bits 0..(ppm-1).
func Interleave(codewords []uint16, ppm, cr int) []uint16 {
	cw := 4 + cr
	syms := make([]uint16, cw)
	for r := 0; r < ppm && r < len(codewords); r++ {
		for k := 0; k < cw; k++ {
			if codewords[r]&(1<<uint(k)) != 0 {
				dst := (r + k) % ppm
				syms[k] |= 1 << uint(dst)
			}
		}
	}
	return syms
}

// Deinterleave inverts Interleave: (4+cr) symbols of ppm bits -> ppm
// codewords of (4+cr) bits.
func Deinterleave(syms []uint16, ppm, cr int) []uint16 {
	cw := 4 + cr
	codewords := make([]uint16, ppm)
	for k := 0; k < cw && k < len(syms); k++ {
		for r := 0; r < ppm; r++ {
			dst := (r + k) % ppm
			if syms[k]&(1<<uint(dst)) != 0 {
				codewords[r] |= 1 << uint(k)
			}
		}
	}
	return codewords
}

// --- Hamming(4+cr, 4) ----------------------------------------------------
//
// Codeword bit layout (LSB-first within a uint16):
//
//	bit0..3 = data nibble d0..d3
//	bit4..  = parity, per coding rate (see below)
//
// 4/5 (cr=1): bit4 = even parity of d0..d3            (detect only)
// 4/6 (cr=2): bit4 = d0^d1^d3, bit5 = d0^d2^d3        (detect)
// 4/7 (cr=3): Hamming(7,4) parities p1,p2,p3          (single-error correct)
// 4/8 (cr=4): Hamming(7,4) + bit7 overall parity      (SEC-DED)

// HammingEncode4 encodes a 4-bit nibble into its (4+cr)-bit codeword.
func HammingEncode4(nibble uint8, cr int) uint16 {
	d0 := uint16(nibble>>0) & 1
	d1 := uint16(nibble>>1) & 1
	d2 := uint16(nibble>>2) & 1
	d3 := uint16(nibble>>3) & 1
	cw := d0 | d1<<1 | d2<<2 | d3<<3
	switch cr {
	case 1:
		cw |= (d0 ^ d1 ^ d2 ^ d3) << 4
	case 2:
		cw |= (d0 ^ d1 ^ d3) << 4
		cw |= (d0 ^ d2 ^ d3) << 5
	case 3, 4:
		p1 := d0 ^ d1 ^ d3
		p2 := d0 ^ d2 ^ d3
		p3 := d1 ^ d2 ^ d3
		cw |= p1<<4 | p2<<5 | p3<<6
		if cr == 4 {
			// overall (extended) parity over the 7 bits.
			ov := d0 ^ d1 ^ d2 ^ d3 ^ p1 ^ p2 ^ p3
			cw |= ov << 7
		}
	}
	return cw
}

// HammingDecode4 decodes one (4+cr)-bit codeword to its 4-bit nibble.
// corrected reports the outcome: 0 = clean, 1 = a single bit error was
// corrected, -1 = an uncorrectable error was detected (nibble is the
// best-effort raw data bits).
func HammingDecode4(cw uint16, cr int) (nibble uint8, corrected int) {
	rawNibble := func(c uint16) uint8 { return uint8(c & 0x0f) }
	bit := func(c uint16, i int) uint16 { return (c >> uint(i)) & 1 }

	switch cr {
	case 1:
		par := bit(cw, 0) ^ bit(cw, 1) ^ bit(cw, 2) ^ bit(cw, 3)
		if par != bit(cw, 4) {
			return rawNibble(cw), -1
		}
		return rawNibble(cw), 0
	case 2:
		s0 := bit(cw, 0) ^ bit(cw, 1) ^ bit(cw, 3) ^ bit(cw, 4)
		s1 := bit(cw, 0) ^ bit(cw, 2) ^ bit(cw, 3) ^ bit(cw, 5)
		if s0 != 0 || s1 != 0 {
			return rawNibble(cw), -1
		}
		return rawNibble(cw), 0
	case 3, 4:
		d0, d1, d2, d3 := bit(cw, 0), bit(cw, 1), bit(cw, 2), bit(cw, 3)
		p1, p2, p3 := bit(cw, 4), bit(cw, 5), bit(cw, 6)
		s1 := d0 ^ d1 ^ d3 ^ p1
		s2 := d0 ^ d2 ^ d3 ^ p2
		s3 := d1 ^ d2 ^ d3 ^ p3
		syn := s1 | s2<<1 | s3<<2
		if syn != 0 {
			// Map the syndrome to the flipped bit position and correct.
			// Order chosen to match the parity equations above.
			pos := syndromeToPos(syn)
			if pos >= 0 {
				cw ^= 1 << uint(pos)
			}
			if cr == 4 {
				// Extended parity: distinguish corrected single error
				// (now consistent) from a double error.
				ov := bit(cw, 0) ^ bit(cw, 1) ^ bit(cw, 2) ^ bit(cw, 3) ^
					bit(cw, 4) ^ bit(cw, 5) ^ bit(cw, 6)
				if ov != bit(cw, 7) {
					return rawNibble(cw), -1 // double error
				}
			}
			return rawNibble(cw), 1
		}
		if cr == 4 {
			ov := d0 ^ d1 ^ d2 ^ d3 ^ p1 ^ p2 ^ p3
			if ov != bit(cw, 7) {
				// Clean (7,4) syndrome but the overall-parity bit flipped:
				// the single error is in bit7 itself. Correctable.
				return rawNibble(cw), 1
			}
		}
		return rawNibble(cw), 0
	}
	return rawNibble(cw), -1
}

// syndromeToPos returns the bit index (in the LSB-first codeword) to flip
// for a given non-zero Hamming(7,4) syndrome, or -1 if none.
func syndromeToPos(syn uint16) int {
	// syn = s1 | s2<<1 | s3<<2 with:
	//   d0 in {s1,s2}; d1 in {s1,s3}; d2 in {s2,s3}; d3 in {s1,s2,s3};
	//   p1=s1 only; p2=s2 only; p3=s3 only.
	switch syn {
	case 0b011: // s1,s2 -> d0
		return 0
	case 0b101: // s1,s3 -> d1
		return 1
	case 0b110: // s2,s3 -> d2
		return 2
	case 0b111: // s1,s2,s3 -> d3
		return 3
	case 0b001: // s1 -> p1 (bit4)
		return 4
	case 0b010: // s2 -> p2 (bit5)
		return 5
	case 0b100: // s3 -> p3 (bit6)
		return 6
	}
	return -1
}
