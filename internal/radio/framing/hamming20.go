package framing

// (20,8,7) shortened Hamming / Golay code — a binary linear block code
// used to protect the DMR slot-type field that frames every burst (10
// bits before sync + 10 bits after sync = 20 bits surrounding the sync
// pattern). Per ETSI TS 102 361-1 §B.1.1 ("DMR AI spec page 134"), the
// 8 information bits are followed by 12 parity bits computed from the
// generator matrix below. Some references call this code Hamming(20,8),
// others call it Golay(20,8); the wire format is identical either way
// and the file is named after the README's pre-existing terminology.
//
// Minimum distance is 7, so the decoder corrects up to 3 bit errors per
// codeword and detects up to 4. We brute-force min-distance decode over
// all 256 valid codewords because that table fits easily in cache and
// keeps the implementation auditable against the spec without a
// syndrome lookup.
//
// Codeword layout (low 20 bits of uint32, MSB-first):
//
//	bits 19..12  info  i[0]..i[7]  (MSB first)
//	bits 11..0   parity p[0]..p[11] (MSB first, p[0] at bit 11)

// hamming20_8ParityMasks holds, for each parity bit p[k], the bitmask
// over info bits whose XOR equals p[k]. Info bit i[n] is at bit (7-n)
// of the input byte, so e.g. p[0] = i[1] ⊕ i[4] ⊕ i[5] ⊕ i[6] ⊕ i[7]
// becomes mask 0b01001111 = 0x4F.
var hamming20_8ParityMasks = [12]byte{
	0x4F, // p[0]  = i1 ⊕ i4 ⊕ i5 ⊕ i6 ⊕ i7
	0x68, // p[1]  = i1 ⊕ i2 ⊕ i4
	0xB4, // p[2]  = i0 ⊕ i2 ⊕ i3 ⊕ i5
	0xDA, // p[3]  = i0 ⊕ i1 ⊕ i3 ⊕ i4 ⊕ i6
	0xED, // p[4]  = i0 ⊕ i1 ⊕ i2 ⊕ i4 ⊕ i5 ⊕ i7
	0xB9, // p[5]  = i0 ⊕ i2 ⊕ i3 ⊕ i4 ⊕ i7
	0x13, // p[6]  = i3 ⊕ i6 ⊕ i7
	0xC6, // p[7]  = i0 ⊕ i1 ⊕ i5 ⊕ i6
	0xE3, // p[8]  = i0 ⊕ i1 ⊕ i2 ⊕ i6 ⊕ i7
	0x3E, // p[9]  = i2 ⊕ i3 ⊕ i4 ⊕ i5 ⊕ i6
	0x9F, // p[10] = i0 ⊕ i3 ⊕ i4 ⊕ i5 ⊕ i6 ⊕ i7
	0x75, // p[11] = i1 ⊕ i2 ⊕ i3 ⊕ i5 ⊕ i7
}

// HammingEncode20_8 encodes 8 information bits into a 20-bit codeword.
// The codeword occupies the low 20 bits of the returned uint32, with
// info in bits 19..12 and parity in bits 11..0.
func HammingEncode20_8(data uint8) uint32 {
	cw := uint32(data) << 12
	for k := 0; k < 12; k++ {
		if PopCount64(uint64(data&hamming20_8ParityMasks[k]))&1 != 0 {
			cw |= uint32(1) << uint(11-k)
		}
	}
	return cw
}

// HammingDecode20_8 decodes a 20-bit codeword (low 20 bits of cw) by
// minimum-Hamming-distance search across all 256 valid codewords.
// Returns (data, errors) where errors is the corrected bit count, or
// -1 if the closest codeword is more than 3 bits away (uncorrectable —
// data is the best guess but should not be trusted).
func HammingDecode20_8(cw uint32) (uint8, int) {
	cw &= 0xFFFFF
	var bestData uint8
	bestDist := 21
	for d := 0; d < 256; d++ {
		c := HammingEncode20_8(uint8(d))
		dist := PopCount64(uint64(c ^ cw))
		if dist < bestDist {
			bestDist = dist
			bestData = uint8(d)
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
