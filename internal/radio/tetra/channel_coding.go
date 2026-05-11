package tetra

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TETRA per-channel encode/decode helpers per ETSI EN 300 392-2
// §8.3.1, composing the framing primitives (RCPC mother + puncture,
// (30,14) RM, block interleaver, scrambler) shipped in PRs #137 +
// #138 into the full type-1 → type-5 chain for each π/4-DQPSK
// signaling channel.
//
// Channels covered:
//
//	AACH      14 → 30 bits — (30,14) RM + scramble (§8.3.1.1)
//	BSCH      60 → 120     — CRC-16 + tail + RCPC R=2/3 + (120,11)
//	                          interleave + scramble (colour=0
//	                          per §8.2.5.2) (§8.3.1.2)
//	SCH/HD    124 → 216    — CRC-16 + tail + RCPC R=2/3 + (216,101)
//	                          interleave + scramble (§8.3.1.4.1)
//	                          (also covers BNCH + STCH — same chain)
//	SCH/HU    92 → 168     — CRC-16 + tail + RCPC R=2/3 + (168,13)
//	                          interleave + scramble (§8.3.1.4.3)
//	SCH/F     268 → 432    — CRC-16 + tail + RCPC R=2/3 + (432,103)
//	                          interleave + scramble (§8.3.1.4.5)
//
// All bit slices are 0/1 byte values, MSB-first per spec convention.
//
// The CRC-16 used by the (K1+16, K1) block code in §8.2.3.3 is the
// standard CRC-CCITT with polynomial G(X) = X^16 + X^12 + X^5 + 1
// (= 0x1021), initial fill 0xFFFF, and final XOR 0xFFFF — that's
// the spec's eq. 8.15 unpacked into the textbook
// "CRC-CCITT-FALSE / FFFF" convention. The Decode helpers verify
// this CRC; failure returns (info, false).

// crcTetraK1Plus16 computes the 16-bit CRC for the (K1+16, K1) block
// code per §8.2.3.3 eq. 8.14-8.18. Processes the K1-bit input
// MSB-first with init=0xFFFF, applies final XOR=0xFFFF. Output is
// the 16-bit CRC to append (MSB first) to the K1 information bits
// to form the K1+16 block-encoded bits.
func crcTetraK1Plus16(bits []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range bits {
		topBit := (crc >> 15) & 1
		crc = (crc << 1) & 0xFFFF
		if topBit^(uint16(b&1)) != 0 {
			crc ^= 0x1021
		}
	}
	return crc ^ 0xFFFF
}

// appendCRC16 appends the 16-bit CRC of bits (computed via
// crcTetraK1Plus16) as 16 MSB-first bit entries.
func appendCRC16(bits []byte) []byte {
	crc := crcTetraK1Plus16(bits)
	out := make([]byte, len(bits)+16)
	copy(out, bits)
	for i := 0; i < 16; i++ {
		out[len(bits)+i] = byte((crc >> uint(15-i)) & 1)
	}
	return out
}

// appendTailBits appends n zero bits to the slice.
func appendTailBits(bits []byte, n int) []byte {
	out := make([]byte, len(bits)+n)
	copy(out, bits)
	return out
}

// encodeRCPCRate23 runs len(input) type-2 bits through the K=5
// R=1/4 mother code and rate-2/3 puncturing, producing (3/2)*
// len(input) type-3 bits. Both sizes must be exact.
func encodeRCPCRate23(type2 []byte) []byte {
	mother := framing.EncodeRCPCTetraSigMother(type2)
	k3 := (3 * len(type2)) / 2
	return framing.PunctureRCPCTetraSig(mother, framing.RCPCTetraSigPuncture23, k3, nil)
}

// decodeRCPCRate23 inverts encodeRCPCRate23: depuncture +
// Viterbi-decode len(type3)*2/3 stages from the type-3 bits.
// Returns the recovered K2-bit type-2 stream (still including any
// tail bits the encoder appended).
func decodeRCPCRate23(type3 []byte) []byte {
	stages := (2 * len(type3)) / 3
	motherLen := 4 * stages
	mother := framing.DepunctureRCPCTetraSig(type3, framing.RCPCTetraSigPuncture23, motherLen, nil)
	out, _ := framing.DecodeRCPCTetraSigMother(mother, stages)
	return out
}

// signalingEncode runs the common SCH/BSCH chain — CRC-16 +
// 4 zero tail bits + RCPC R=2/3 + (K, a) interleave + scramble —
// producing K3 type-5 output bits from K1 type-1 input bits.
func signalingEncode(type1 []byte, interleaverK, interleaverA int, colourCode uint32) []byte {
	withCRC := appendCRC16(type1)                // K1 + 16
	type2 := appendTailBits(withCRC, 4)          // K1 + 20
	type3 := encodeRCPCRate23(type2)             // (K1 + 20) * 3 / 2
	type4 := framing.BlockInterleaveTetra(type3, interleaverK, interleaverA)
	type5 := framing.ScrambleTetra(type4, colourCode)
	return type5
}

// signalingDecode reverses signalingEncode. Returns the recovered
// K1-bit type-1 block plus a CRC-pass flag.
func signalingDecode(type5 []byte, interleaverK, interleaverA int, colourCode uint32, k1 int) ([]byte, bool) {
	type4 := framing.DescrambleTetra(type5, colourCode)
	type3 := framing.BlockDeinterleaveTetra(type4, interleaverK, interleaverA)
	type2 := decodeRCPCRate23(type3) // K1 + 20
	if len(type2) < k1+20 {
		return nil, false
	}
	// Strip 4 tail bits → K1 + 16 block-encoded bits
	blockEncoded := type2[:k1+16]
	info := blockEncoded[:k1]
	receivedCRC := uint16(0)
	for i := 0; i < 16; i++ {
		receivedCRC = (receivedCRC << 1) | uint16(blockEncoded[k1+i]&1)
	}
	expected := crcTetraK1Plus16(info)
	if expected != receivedCRC {
		return info, false
	}
	return info, true
}

// EncodeSCHHD runs 124 type-1 bits through the SCH/HD chain per
// §8.3.1.4.1 — also valid for BNCH and STCH which share the same
// coding. Returns 216 type-5 bits.
func EncodeSCHHD(type1 []byte, colourCode uint32) []byte {
	if len(type1) != 124 {
		return nil
	}
	return signalingEncode(type1, framing.InterleaveKSCHHD, framing.InterleaveASCHHD, colourCode)
}

// DecodeSCHHD inverts EncodeSCHHD. Returns the 124 recovered type-1
// bits and a CRC-pass flag; if the flag is false the info bits
// are the best Viterbi guess but should not be trusted.
func DecodeSCHHD(type5 []byte, colourCode uint32) ([]byte, bool) {
	if len(type5) != 216 {
		return nil, false
	}
	return signalingDecode(type5, framing.InterleaveKSCHHD, framing.InterleaveASCHHD, colourCode, 124)
}

// EncodeSCHF runs 268 type-1 bits through the SCH/F chain per
// §8.3.1.4.5. Returns 432 type-5 bits.
func EncodeSCHF(type1 []byte, colourCode uint32) []byte {
	if len(type1) != 268 {
		return nil
	}
	return signalingEncode(type1, framing.InterleaveKSCHF, framing.InterleaveASCHF, colourCode)
}

// DecodeSCHF inverts EncodeSCHF.
func DecodeSCHF(type5 []byte, colourCode uint32) ([]byte, bool) {
	if len(type5) != 432 {
		return nil, false
	}
	return signalingDecode(type5, framing.InterleaveKSCHF, framing.InterleaveASCHF, colourCode, 268)
}

// EncodeSCHHU runs 92 type-1 bits through the SCH/HU chain per
// §8.3.1.4.3. Returns 168 type-5 bits.
func EncodeSCHHU(type1 []byte, colourCode uint32) []byte {
	if len(type1) != 92 {
		return nil
	}
	return signalingEncode(type1, framing.InterleaveKSCHHU, framing.InterleaveASCHHU, colourCode)
}

// DecodeSCHHU inverts EncodeSCHHU.
func DecodeSCHHU(type5 []byte, colourCode uint32) ([]byte, bool) {
	if len(type5) != 168 {
		return nil, false
	}
	return signalingDecode(type5, framing.InterleaveKSCHHU, framing.InterleaveASCHHU, colourCode, 92)
}

// EncodeBSCH runs 60 type-1 bits through the BSCH chain per
// §8.3.1.2. Colour code is always zero for BSCH per §8.2.5.2.
// Returns 120 type-5 bits.
func EncodeBSCH(type1 []byte) []byte {
	if len(type1) != 60 {
		return nil
	}
	return signalingEncode(type1, framing.InterleaveKBSCH, framing.InterleaveABSCH, 0)
}

// DecodeBSCH inverts EncodeBSCH.
func DecodeBSCH(type5 []byte) ([]byte, bool) {
	if len(type5) != 120 {
		return nil, false
	}
	return signalingDecode(type5, framing.InterleaveKBSCH, framing.InterleaveABSCH, 0, 60)
}

// EncodeAACH runs 14 type-1 bits through the AACH chain per
// §8.3.1.1 — shorter than the other signaling channels because
// AACH skips RCPC + interleaving entirely. The (30,14) RM block
// code directly yields the type-2 / type-3 / type-4 bits, then
// scrambling produces 30 type-5 bits.
func EncodeAACH(type1 []byte, colourCode uint32) []byte {
	if len(type1) != 14 {
		return nil
	}
	type2 := framing.EncodeRM3014Tetra(type1)
	return framing.ScrambleTetra(type2, colourCode)
}

// DecodeAACH inverts EncodeAACH. Returns the recovered 14 type-1
// bits plus the Viterbi-style Hamming-distance metric from the
// (30,14) RM decoder (0 = clean codeword, ≥ 1 = corrections
// applied or uncorrectable).
func DecodeAACH(type5 []byte, colourCode uint32) ([]byte, int) {
	if len(type5) != 30 {
		return nil, -1
	}
	type4 := framing.DescrambleTetra(type5, colourCode)
	return framing.DecodeRM3014Tetra(type4)
}
