package framing

// MPT 1327 codeword check.
//
// Per the most-cited public reference (sdrtrunk's CRCFleetsync
// implementation, which Fleetsync and MPT 1327 share), a 64-bit
// codeword is laid out as:
//
//	bits  0..47   48-bit information field
//	bits 48..62   15-bit BCH check (CRC-15 with polynomial 0x6815,
//	              initial fill 0x0001)
//	bit  63       overall even parity over the 63-bit body
//
// The generator polynomial is:
//
//	g(x) = x^15 + x^14 + x^13 + x^11 + x^4 + x^2 + 1
//
// representable as the 16-bit constant 0xE815 (full degree-15
// polynomial including the implicit x^15 leading term), or 0x6815
// when the implicit leading bit is dropped — the form most CRC
// implementations use. The 0x0001 initial value seeds the
// computation so the all-zero codeword isn't trivially valid.
//
// The BCH structure corrects up to one bit error within the 64
// codeword bits — a single-bit flip changes both the BCH syndrome
// (which identifies the position within bits 0..62) and the
// overall parity (which catches a flip at position 63).

const (
	bchMPT1327PolyHigh uint16 = 0x6815 // generator without the implicit x^15
	bchMPT1327Init     uint16 = 0x0001 // initial-fill XOR
)

// bchMPT1327Syndromes[i] is x^i mod g(x) for i in 0..47, computed
// at package init time. Equivalent to sdrtrunk's sCHECKSUMS[] table
// for the message-bit half.
var bchMPT1327Syndromes [48]uint16

func init() {
	// x^0 .. x^14 are just 1 << i since they fit entirely below the
	// polynomial's degree.
	for i := 0; i < 15; i++ {
		bchMPT1327Syndromes[i] = uint16(1) << uint(i)
	}
	// x^15 ≡ 0x6815 (mod g). For i = 16..47, multiply the previous
	// syndrome by x (= left shift by 1) and reduce when the
	// resulting x^15 term needs eliminating.
	v := bchMPT1327PolyHigh
	bchMPT1327Syndromes[15] = v
	for i := 16; i < 48; i++ {
		if v&(1<<14) != 0 {
			v = ((v << 1) & 0x7FFF) ^ bchMPT1327PolyHigh
		} else {
			v = (v << 1) & 0x7FFF
		}
		bchMPT1327Syndromes[i] = v
	}
}

// BCHEncodeMPT1327 builds a 64-bit MPT 1327 codeword from 48
// information bits. Only the low 48 bits of info are used.
// The returned codeword has the layout described in the package
// header: info in bits 0..47, BCH check in bits 48..62, overall
// even parity in bit 63.
func BCHEncodeMPT1327(info uint64) uint64 {
	info &= (uint64(1) << 48) - 1
	cs := bchMPT1327Init
	for i := 0; i < 48; i++ {
		if info&(uint64(1)<<uint(i)) != 0 {
			cs ^= bchMPT1327Syndromes[i]
		}
	}
	cw := info | (uint64(cs) << 48)
	if PopCount64(cw)&1 == 1 {
		cw |= uint64(1) << 63
	}
	return cw
}

// BCHDecodeMPT1327 validates and (where possible) corrects a 64-bit
// MPT 1327 codeword. Returns (info, errs) where:
//
//   - errs = 0: codeword passed the BCH check + overall parity
//     cleanly; info is the recovered 48-bit information field.
//   - errs = 1: a single-bit error was detected and corrected;
//     info reflects the corrected information field.
//   - errs = -1: the codeword carries more than one bit error in
//     positions that BCH(64,48,2) can't isolate; info is the
//     received information field but should not be trusted.
//
// Single-error positions covered:
//
//   - bits 0..47: info-bit error — XOR the corresponding syndrome
//     column out of the received checksum, then flip the matching
//     info bit.
//   - bits 48..62: BCH-bit error — syndrome equals 1 << (j-48),
//     info field is untouched.
//   - bit 63: overall parity bit — syndrome is zero but parity
//     mismatches.
func BCHDecodeMPT1327(cw uint64) (uint64, int) {
	info := cw & ((uint64(1) << 48) - 1)
	txCS := uint16((cw >> 48) & 0x7FFF)
	txParity := uint8((cw >> 63) & 1)

	// Recompute the expected BCH check.
	cs := bchMPT1327Init
	for i := 0; i < 48; i++ {
		if info&(uint64(1)<<uint(i)) != 0 {
			cs ^= bchMPT1327Syndromes[i]
		}
	}

	// Recompute overall parity over the 63-bit body.
	body := cw & ((uint64(1) << 63) - 1)
	expectedParity := uint8(PopCount64(body) & 1)

	csMatch := cs == txCS
	parityMatch := expectedParity == txParity

	if csMatch && parityMatch {
		return info, 0
	}
	if csMatch && !parityMatch {
		// Only the overall parity bit is wrong.
		return info, 1
	}

	syndrome := cs ^ txCS

	// A single error in any of the 63 body bits flips both the
	// syndrome and the overall parity. If only the syndrome is
	// off (parity matches), there are at least two errors → not
	// correctable by BCH(64,48,2).
	if !parityMatch {
		// Try matching against an info-bit error (positions 0..47).
		for i := 0; i < 48; i++ {
			if bchMPT1327Syndromes[i] == syndrome {
				return info ^ (uint64(1) << uint(i)), 1
			}
		}
		// Try matching against a CRC-bit error (positions 48..62).
		for j := 0; j < 15; j++ {
			if uint16(1)<<uint(j) == syndrome {
				return info, 1
			}
		}
	}

	return info, -1
}
