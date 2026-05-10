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
