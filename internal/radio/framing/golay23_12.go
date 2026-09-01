package framing

// Golay (23, 12, 7) — the non-extended sibling of GolayEncode24_12 /
// GolayDecode24_12. Same 12 information bits, same triple-error
// correction radius, but no overall-parity bit. P25 Phase 1 IMBE
// channel coding (TIA-102.BABA §7.3.1) uses (23, 12) for the four
// most-protected u_n vectors.
//
// We piggy-back on the existing extended-Golay machinery: the
// extended (24, 12, 8) code is the (23, 12, 7) code plus one
// even-parity bit, so a 23-bit codeword corresponds to either of
// two 24-bit extended codewords (one for each possible parity bit).
// To decode, append both possible parity bits, run the extended
// decoder for each, and pick the survivor with the lower error
// count. One of the two trial bits will always match the true
// extended-Golay parity for the underlying data, so a clean 23-bit
// channel ≤ 3-error event always decodes to ≤ 3 errors via at
// least one branch.

// GolayEncode23_12 encodes 12 data bits (low 12 bits of input) into
// a 23-bit codeword. Output layout: [data(12) | parity(11)] with
// data in bits 22..11 and the 11 parity bits in 10..0. Built by
// dropping the overall-parity LSB from the extended (24, 12)
// encoding so the systematic data layout stays consistent with the
// 24-bit version.
func GolayEncode23_12(data uint16) uint32 {
	return GolayEncode24_12(data) >> 1
}

// GolayDecode23_12 decodes a 23-bit codeword (low 23 bits of input).
// Returns (data, errors) where errors is the corrected bit count
// (-1 if both append-parity branches exceed the ext-Golay
// correction radius, which means > 3 real errors in the 23-bit
// codeword).
func GolayDecode23_12(cw uint32) (uint16, int) {
	cw &= 0x7FFFFF
	cw24 := cw << 1
	d0, e0 := GolayDecode24_12(cw24)
	d1, e1 := GolayDecode24_12(cw24 | 1)
	switch {
	case e0 < 0 && e1 < 0:
		return d0, -1
	case e1 < 0:
		return d0, e0
	case e0 < 0:
		return d1, e1
	case e0 <= e1:
		return d0, e0
	default:
		return d1, e1
	}
}

// Cyclic Golay(23, 12) as P25 and the AMBE+2 vocoder use it — generator
// polynomial x^11+x^9+x^7+x^6+x^5+1 (0xC75), systematic as
// [12 information bits | 11 parity bits], most-significant bit first.
//
// This is NOT interchangeable with GolayDecode24_12 above. Both are Golay
// codes and both correct three errors, but they are different equivalent
// forms of it, and a codeword from one looks like noise to the other: on a
// real P25 Phase 2 voice burst the AMBE c0 field decodes at a median Hamming
// distance of 0 under this decoder and 4 — the covering radius, i.e. pure
// noise — under GolayDecode24_12. Use this one for anything carrying P25 or
// AMBE codewords.
//
// The (23,12) Golay code is perfect: every one of the 2^11 syndromes is the
// syndrome of exactly one error pattern of weight ≤ 3, so the coset-leader
// table below is a complete decoder with no search and no failure case.

const golayCyclicGen = 0xC75

// golayCyclicSyndromes maps each syndrome to its coset leader — the unique
// error pattern of weight ≤ 3 that produces it.
var golayCyclicSyndromes [1 << 11]uint32

func init() {
	set := func(pattern uint32) {
		golayCyclicSyndromes[golayCyclicSyndrome(pattern)] = pattern
	}
	set(0)
	for i := 0; i < 23; i++ {
		set(1 << uint(i))
		for j := i + 1; j < 23; j++ {
			set(1<<uint(i) | 1<<uint(j))
			for k := j + 1; k < 23; k++ {
				set(1<<uint(i) | 1<<uint(j) | 1<<uint(k))
			}
		}
	}
}

// golayCyclicSyndrome returns the remainder of a 23-bit word modulo the
// generator polynomial.
func golayCyclicSyndrome(cw uint32) uint32 {
	r := cw & 0x7FFFFF
	for i := 22; i >= 11; i-- {
		if r&(1<<uint(i)) != 0 {
			r ^= golayCyclicGen << uint(i-11)
		}
	}
	return r & 0x7FF
}

// GolayDecodeCyclic23_12 corrects up to three bit errors in a 23-bit cyclic
// Golay codeword and returns its 12 information bits together with the number
// of errors corrected. It never fails: the code is perfect, so every received
// word lies within distance 3 of exactly one codeword.
func GolayDecodeCyclic23_12(cw uint32) (uint16, int) {
	cw &= 0x7FFFFF
	e := golayCyclicSyndromes[golayCyclicSyndrome(cw)]
	corrected := cw ^ e
	return uint16(corrected >> 11), PopCount64(uint64(e))
}

// GolayEncodeCyclic23_12 is the systematic encoder matching
// GolayDecodeCyclic23_12.
func GolayEncodeCyclic23_12(data uint16) uint32 {
	r := uint32(data&0xFFF) << 11
	for i := 22; i >= 11; i-- {
		if r&(1<<uint(i)) != 0 {
			r ^= golayCyclicGen << uint(i-11)
		}
	}
	return uint32(data&0xFFF)<<11 | r&0x7FF
}
