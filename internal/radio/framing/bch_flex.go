package framing

// FLEX paging protects every 32-bit word with the same BCH(31,21) code as
// POCSAG (generator x^10+x^9+x^8+x^6+x^5+x^3+1 = 0x769, corrects two bit
// errors) plus an even-parity bit, but lays the codeword out differently:
//
//	bits  0..20  information, LSB = the highest polynomial coefficient
//	bits 21..30  BCH parity
//	bit  31      even parity over bits 0..30
//
// As a polynomial, codeword bit b is the coefficient of x^(30-b): information
// bit 0 is x^30 and the last parity bit (bit 30) is x^0. That is multimon-ng's
// layout (bch.c: "recv[i] corresponds to bit (30-i)"), and it is what real air
// carries — the literal on-air words in bch_flex_test.go have a zero syndrome
// under it. An earlier version of this file reused the POCSAG primitive by
// reversing all 31 bits, which left the 21 information bits reversed (and
// decoded by a 2^21-codeword search, milliseconds per word — a 6400 bps frame
// carries 352 words).
//
// Decoding is a syndrome lookup: the 10-bit remainder of the received word
// names the error pattern directly. The 1 + 31 + 465 patterns of weight ≤ 2
// have distinct syndromes (minimum distance 5); any other syndrome is
// uncorrectable.

const flexBCHGen = 0x769

// flexSynBit[b] is the syndrome of a word with only bit b set.
var flexSynBit [31]uint16

// flexErrPattern maps a syndrome to the error pattern (bits 0..30) of weight
// ≤ 2 that produces it; flexErrWeight is that pattern's weight, or -1 when no
// such pattern exists.
var (
	flexErrPattern [1024]uint32
	flexErrWeight  [1024]int8
)

func init() {
	// x^k mod g for k = 0..30; bit b holds x^(30-b).
	rem := uint32(1)
	for k := 0; k <= 30; k++ {
		flexSynBit[30-k] = uint16(rem)
		rem <<= 1
		if rem&0x400 != 0 {
			rem ^= flexBCHGen
		}
	}
	for i := range flexErrWeight {
		flexErrWeight[i] = -1
	}
	flexErrWeight[0] = 0
	for a := 0; a < 31; a++ {
		flexErrPattern[flexSynBit[a]] = 1 << uint(a)
		flexErrWeight[flexSynBit[a]] = 1
	}
	for a := 0; a < 31; a++ {
		for b := a + 1; b < 31; b++ {
			s := flexSynBit[a] ^ flexSynBit[b]
			flexErrPattern[s] = 1<<uint(a) | 1<<uint(b)
			flexErrWeight[s] = 2
		}
	}
}

// flexSyndrome returns the 10-bit remainder of the low 31 bits of cw.
func flexSyndrome(cw uint32) uint16 {
	var s uint16
	for b := 0; b < 31; b++ {
		if cw&(1<<uint(b)) != 0 {
			s ^= flexSynBit[b]
		}
	}
	return s
}

// FLEXBCHEncode21 encodes 21 information bits into a 32-bit FLEX word:
// information in bits 0..20, BCH parity in 21..30, even parity in bit 31.
func FLEXBCHEncode21(info uint32) uint32 {
	cw := info & 0x1FFFFF
	// The remainder's coefficient of x^k belongs in bit 30-k.
	s := flexSyndrome(cw)
	for k := 0; k < 10; k++ {
		if s&(1<<uint(k)) != 0 {
			cw |= 1 << uint(30-k)
		}
	}
	if popCount32(cw)&1 != 0 {
		cw |= 1 << 31
	}
	return cw
}

// FLEXBCHDecode32 decodes a 32-bit FLEX word, returning the 21-bit
// information field and the number of bit errors corrected (-1 if the word is
// uncorrectable, in which case the information is not to be trusted). The
// even-parity bit (31) is ignored, as the reference decoders do.
func FLEXBCHDecode32(cw uint32) (uint32, int) {
	cw &= 0x7FFFFFFF
	s := flexSyndrome(cw)
	w := flexErrWeight[s]
	if w < 0 {
		return cw & 0x1FFFFF, -1
	}
	return (cw ^ flexErrPattern[s]) & 0x1FFFFF, int(w)
}
