package imbe

// P25-faithful per-vector FEC for the IMBE 4400 channel coding
// (TIA-102.BABA §7.3). This is deliberately a *separate* Golay /
// Hamming implementation from internal/radio/framing: the framing
// package's Golay(24,12) / Hamming(15,11) are systematic codes built
// for DMR / P25 link-control framing, and although framing's Golay
// generator list equals golayGenerator[i] shifted by one bit, the two
// associate generator rows with data bits in opposite order, so they
// are *different* codes in practice — a clean real-air IMBE codeword
// decodes to the wrong data under framing's Golay (verified against a
// real P25 voice capture and the mbelib reference decoder, issue
// #489). These functions reproduce the exact code real P25
// transmitters use, transcribed from mbelib's ecc.c
// (mbe_golay2312 / mbe_hamming1511) and ecc_const.h.
//
// Bit convention: a vector's channel bits are addressed by column
// index, column c carrying codeword bit c (LSB-first). The 12 Golay
// / 11 Hamming data bits occupy the *high* columns; the data value
// returned/accepted here is MSB-first (bit 11 / bit 10 = the
// most-significant data bit = highest column). DecodeChannel /
// EncodeChannel own the column↔info-bit mapping.

// golayGenerator is the 11-bit parity (ecc) contribution of each of
// the 12 Golay(23,12) data bits, MSB-first (golayGenerator[0] is the
// most-significant data bit's contribution). From mbelib ecc_const.h.
var golayGenerator = [12]uint16{
	0x63a, 0x31d, 0x7b4, 0x3da, 0x1ed, 0x6cc,
	0x366, 0x1b3, 0x6e3, 0x54b, 0x49f, 0x475,
}

// golay23Codewords is the full set of 4096 valid Golay(23,12)
// codewords, indexed by data value (MSB-first 12-bit). Built once at
// init so golay23Decode can do an optimal nearest-codeword search —
// for ≤3 bit errors the minimum-Hamming-distance codeword is the
// unique correct one, identical to mbelib's syndrome-table decode
// (verified exhaustively over 0..3 error patterns).
var golay23Codewords [4096]uint32

func init() {
	for d := uint16(0); d < 4096; d++ {
		golay23Codewords[d] = golay23Encode(d)
	}
}

// golay23Encode encodes a 12-bit data value (MSB-first: bit 11 is the
// most-significant data bit / highest column) into a 23-bit codeword
// with the data in bits 22..11 and the 11 parity bits in 10..0.
func golay23Encode(data uint16) uint32 {
	d := data & 0x0FFF
	var ecc uint16
	for i := 0; i < 12; i++ {
		if d&(1<<uint(11-i)) != 0 {
			ecc ^= golayGenerator[i]
		}
	}
	return uint32(d)<<11 | uint32(ecc&0x7FF)
}

// golay23Decode decodes a 23-bit codeword (data in bits 22..11) to
// its 12-bit data value (MSB-first) and the number of corrected bit
// errors, or -1 when the codeword is further than the guaranteed
// correction radius (3) from every codeword.
func golay23Decode(cw uint32) (uint16, int) {
	cw &= 0x7FFFFF
	bestData := uint16(0)
	bestDist := 24
	for d := 0; d < 4096; d++ {
		dist := popcount32(golay23Codewords[d] ^ cw)
		if dist < bestDist {
			bestDist = dist
			bestData = uint16(d)
			if dist == 0 {
				return bestData, 0
			}
		}
	}
	if bestDist > 3 {
		return bestData, -1
	}
	return bestData, bestDist
}

// hammingGenerator is the four parity-check rows of the P25 IMBE
// Hamming(15,11) code over a 15-bit block whose column c carries bit
// c (data in the high columns 14..4, four parity columns in 3..0).
// From mbelib ecc_const.h.
var hammingGenerator = [4]uint16{0x7f08, 0x78e4, 0x66d2, 0x55b1}

// hamming1511Codewords is the 2048-codeword set indexed by data value
// (MSB-first 11-bit), so hamming1511Decode can do an optimal
// nearest-codeword search. This corrects every single-bit error,
// including data-bit errors — mbelib's compact hammingMatrix table
// only resolves the four parity-bit syndromes and leaves single
// data-bit errors uncorrected, but clean frames decode identically.
var hamming1511Codewords [2048]uint16

func init() {
	for d := uint16(0); d < 2048; d++ {
		hamming1511Codewords[d] = hamming1511Encode(d)
	}
}

// hamming1511Decode decodes a 15-bit codeword (column c = bit c, data
// in bits 14..4) to its 11-bit data value (MSB-first: bit 10 = column
// 14) plus the corrected-error count (0 or 1; -1 when the codeword is
// more than one bit from every codeword).
func hamming1511Decode(block uint16) (uint16, int) {
	block &= 0x7FFF
	bestData := uint16(0)
	bestDist := 16
	for d := 0; d < 2048; d++ {
		dist := popcount32(uint32(hamming1511Codewords[d] ^ block))
		if dist < bestDist {
			bestDist = dist
			bestData = uint16(d)
			if dist == 0 {
				return bestData, 0
			}
		}
	}
	if bestDist > 1 {
		return bestData, -1
	}
	return bestData, bestDist
}

// hamming1511Encode encodes an 11-bit data value (MSB-first: bit 10 =
// column 14) into the 15-bit codeword (data in bits 14..4, four
// parity bits in 3..0) that hamming1511Decode accepts cleanly.
func hamming1511Encode(data uint16) uint16 {
	block := (data & 0x07FF) << 4
	for par := uint16(0); par < 16; par++ {
		cw := block | par
		ok := true
		for i := 0; i < 4; i++ {
			if parity16(cw&hammingGenerator[i]) != 0 {
				ok = false
				break
			}
		}
		if ok {
			return cw
		}
	}
	return block // unreachable for a valid systematic code
}

// parity16 returns the XOR (even/odd parity) of the bits of v.
func parity16(v uint16) uint16 {
	v ^= v >> 8
	v ^= v >> 4
	v ^= v >> 2
	v ^= v >> 1
	return v & 1
}

// popcount32 returns the number of set bits in v.
func popcount32(v uint32) int {
	n := 0
	for v != 0 {
		v &= v - 1
		n++
	}
	return n
}
