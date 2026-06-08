package framing

// FLEX paging uses the same BCH(31,21) error-correcting code as POCSAG
// (generator 0x769, corrects up to 2 bit errors) plus an overall
// even-parity bit, but lays the codeword out differently on the wire:
// FLEX puts the 21 information bits in the LOW bits (0..20) and the 10
// BCH parity bits in the high bits (21..30), with the even-parity bit
// at bit 31. POCSAG's BCHEncode31_21 / BCHDecode31_21 (this package)
// put the info in bits 30..10. The two layouts are a bit-reversal of
// the 31-bit codeword apart, and bit-reversal preserves Hamming
// distance, so we reuse the tested POCSAG primitive by reversing into
// and out of its layout. The recovered 21-bit info integer is what the
// FLEX BIW / VIW / message-field formulas operate on.
//
// Wire calibration: the FLEX generator polynomial and codeword bit
// order are reproduced here as the POCSAG-compatible (31,21,5) code;
// the encode/decode round-trip and 2-bit correction are unit-tested,
// but the exact on-wire bit order of real FLEX traffic should be
// confirmed against a captured signal.

// reverse31 reverses the low 31 bits of x.
func reverse31(x uint32) uint32 {
	var r uint32
	for i := 0; i < 31; i++ {
		r = (r << 1) | (x & 1)
		x >>= 1
	}
	return r
}

// FLEXBCHEncode21 encodes 21 information bits into a 32-bit FLEX
// codeword: info in bits 0..20, 10 BCH parity bits in 21..30, and the
// overall even-parity bit in bit 31.
func FLEXBCHEncode21(info uint32) uint32 {
	cw := reverse31(BCHEncode31_21(info & 0x1FFFFF))
	if popCount32(cw&0x7FFFFFFF)&1 != 0 {
		cw |= 1 << 31
	}
	return cw
}

// FLEXBCHDecode32 decodes a 32-bit FLEX codeword, returning the 21-bit
// information field and the BCH error count (-1 if uncorrectable). The
// even-parity bit (31) is ignored; the BCH layer carries the
// correction.
func FLEXBCHDecode32(cw uint32) (uint32, int) {
	data, errs := BCHDecode31_21(reverse31(cw & 0x7FFFFFFF))
	return data & 0x1FFFFF, errs
}
