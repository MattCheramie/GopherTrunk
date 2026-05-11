package framing

// BCH(63,16,11) is the binary BCH code that protects the P25 Phase 1
// Network ID (NID) field on every transmitted frame. The code carries 16
// information bits in a 63-bit codeword (47 parity bits) and corrects up
// to 11 bit errors per codeword. P25 then appends a single even-parity
// bit to form the 64-bit NID actually transmitted on-air.
//
// Generator polynomial (TIA-102.BAAA-A §6.2):
//
//	g(x) = x^47 + x^46 + x^45 + x^44 + x^41 + x^40 + x^39 + x^36 + x^32
//	     + x^31 + x^30 + x^29 + x^25 + x^23 + x^22 + x^21 + x^20 + x^17
//	     + x^16 + x^14 + x^11 + x^9  + x^8  + x^7  + x^4  + x^3  + 1
//
// Codewords are stored systematically with the 16 information bits in
// positions 62..47 (bit 62 = MSB of info, bit 47 = LSB) and the 47
// parity bits in positions 46..0. We pack the 63-bit codeword into the
// low bits of a uint64 with bit 0 = lowest-order x^0 coefficient.

const bch6316Generator uint64 = 0xF391E2F34B99

// BCHEncode63_16 encodes 16 information bits into a 63-bit BCH codeword.
// Only the low 16 bits of data are used. The result occupies the low 63
// bits of the returned uint64.
func BCHEncode63_16(data uint16) uint64 {
	info := uint64(data) & 0xFFFF
	rem := info << 47
	for i := 62; i >= 47; i-- {
		if rem&(uint64(1)<<uint(i)) != 0 {
			rem ^= bch6316Generator << uint(i-47)
		}
	}
	return (info << 47) | (rem & ((uint64(1) << 47) - 1))
}

// BCHDecode63_16 decodes a 63-bit BCH codeword by minimum-Hamming-
// distance search across all 2^16 valid codewords. Returns (data,
// errors) where errors is the bit-error count corrected, or -1 if the
// closest valid codeword is more than 11 bits away (uncorrectable; data
// is the best guess but should not be trusted).
func BCHDecode63_16(cw uint64) (uint16, int) {
	cw &= (uint64(1) << 63) - 1
	var bestData uint16
	bestDist := 64
	for d := uint32(0); d < 1<<16; d++ {
		c := BCHEncode63_16(uint16(d))
		dist := PopCount64(c ^ cw)
		if dist < bestDist {
			bestDist = dist
			bestData = uint16(d)
			if dist == 0 {
				return bestData, 0
			}
		}
	}
	if bestDist > 11 {
		return bestData, -1
	}
	return bestData, bestDist
}

// BCH6316ParityBit returns the even-parity bit over the 63 codeword
// bits — the trailing bit P25 appends to form the 64-bit NID.
func BCH6316ParityBit(cw uint64) byte {
	return byte(PopCount64(cw&((uint64(1)<<63)-1)) & 1)
}

// BCHEncode64_16 encodes 16 information bits into a 64-bit BCH
// codeword: the 63-bit BCH(63,16,11) codeword from BCHEncode63_16
// plus a trailing overall-even-parity bit. This is the FEC layer
// used by Motorola Type II / SmartZone control-channel OSWs (each
// OSW frame carries two BCH(64,16,11) codewords concatenated; two
// codewords' 16 info bits each combine into the OSW's 32-bit
// {Address, Command} field).
//
// Bit 0 of the returned uint64 is the parity bit; bits 1..63 are
// the 63-bit BCH codeword (info in the high 16 bits of that
// range, parity in the low 47). Encoders that want the canonical
// MSB-first 64-bit wire representation should pack [parity, bch63
// high → low].
func BCHEncode64_16(data uint16) uint64 {
	cw63 := BCHEncode63_16(data)
	parity := uint64(BCH6316ParityBit(cw63))
	return (cw63 << 1) | parity
}

// BCHDecode64_16 decodes a 64-bit BCH codeword by extracting the
// 63-bit BCH(63,16,11) codeword (top 63 bits) and the trailing
// parity bit, running BCHDecode63_16 on the 63-bit codeword, and
// reporting the corrected info + total error count.
//
// Returns (data, errors) where errors is the bit-error count
// corrected. errors = -1 means uncorrectable (the inner BCH
// decoder reported > 11 errors); the returned data is the closest
// codeword's info but should not be trusted.
//
// The trailing parity bit is included in the error count: if the
// 63-bit decode reports E errors AND the recomputed parity over
// the corrected 63-bit codeword doesn't match the received
// parity, the total is E + 1 (still uncorrectable if E > 10).
func BCHDecode64_16(cw uint64) (uint16, int) {
	parity := byte(cw & 1)
	cw63 := (cw >> 1) & ((uint64(1) << 63) - 1)
	data, errs := BCHDecode63_16(cw63)
	if errs < 0 {
		return data, -1
	}
	// Recompute parity over the corrected 63-bit codeword.
	gotParity := BCH6316ParityBit(BCHEncode63_16(data))
	if gotParity != parity {
		errs++
		if errs > 11 {
			return data, -1
		}
	}
	return data, errs
}
