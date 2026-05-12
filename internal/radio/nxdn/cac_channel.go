package nxdn

import (
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// CAC channel coding per NXDN Technical Specification NXDN-TS-1-A
// rev 1.3 §4.5.1.1 (CAC outbound on the RCCH).
//
// The CAC carries 152 user bits (8 SR + 144 L3 Data) per burst.
// Three Null zeros pad it to 155, a 16-bit CRC-CCITT covers the
// 155-bit info block, four zero tail bits flush the K=5 encoder,
// the 175-bit input is convolution-encoded at R = 1/2, punctured
// at a fixed 50/350 rate, and the surviving 300 bits run through
// a 25×12 block interleaver. The output is 300 channel bits = 150
// dibits transmitted in the CAC slot of the RCCH frame
// (§4.6, "RCCH Outbound": FSW 20 + LICH 16 + CAC 300 + E 24 +
// Post 24).
//
// Spec parameters:
//
//	K = 5, rate-½ convolutional code (same primitive as SACCH):
//	   g1(D) = 1 + D³ + D⁴   (0x19, octal 31)
//	   g2(D) = 1 + D + D² + D⁴ (0x17, octal 27)
//	Output bits read alternately G1, G2.
//
//	Puncture matrix (period 7):
//	   row G1: 1111111  (always keep G1)
//	   row G2: 1011101  (drop G2 at sub-columns 1 and 5 mod 7)
//	2 drops × 25 periods = 50 dropped pre-puncture positions →
//	350 → 300 channel bits.
//
//	Interleave 25 × 12: write the 300 pre-interleave bits as 25
//	rows × 12 columns row-by-row, read out column-by-column.
//	Equivalent to: channel[k] = pre[(k % 25) * 12 + (k / 25)]
//	for k = 0..299.
//
//	CRC-16: poly X¹⁶ + X¹² + X⁵ + 1 (= 0x1021), init 0xFFFF, no
//	reflection, no final XOR. Same convention as the existing
//	AssembleCAC byte-level helper; here it's computed bit-level
//	over exactly 155 bits since 155 isn't byte-aligned.

// CACInfoBits is the number of user information bits the CAC
// carries before the CRC trailer is appended on the encode side
// (8-bit SR + 144-bit L3 Data + 3 Null zeros, per Figure 4.5-1
// step ②). DecodeCACChannel returns exactly this many bits when
// the CRC validates.
const CACInfoBits = 155

// CACChannelBits is the number of bits transmitted on-air for one
// CAC slot. 300 / 2 = 150 dibits per the RCCH outbound layout in
// §4.6.
const CACChannelBits = 300

// cacTailBits is the number of zero tail bits appended to the
// CRC-suffixed info block so the K=5 encoder ends in state 0.
const cacTailBits = 4

// cacPuncturePositions enumerates the 50 pre-puncture bit indices
// (within the 350-bit K=5 R=1/2 encoder output) the spec's
// 1111111/1011101 puncturing matrix drops. Each entry is the
// position of a G2 bit at encoder step `i` where `i mod 7 ∈ {1, 5}`
// (the two zero-columns in the G2 row). Total 50 positions
// (25 periods × 2 drops). Sorted ascending.
var cacPuncturePositions = computeCACPuncturePositions()

func computeCACPuncturePositions() []int {
	out := make([]int, 0, 50)
	for i := 0; i < CACInfoBits+16+cacTailBits; i++ {
		switch i % 7 {
		case 1, 5:
			// G2 bit at encoder step i lives at pre-puncture
			// position 2i+1 (each step emits G1 then G2).
			out = append(out, 2*i+1)
		}
	}
	return out
}

// cacInterleavePerm[k] = j means channel[k] = punctured[j] on the
// encoder side, and equivalently depunctured[j] = received[k] on
// the decoder side. The permutation is the column-major readout
// of a 25-row × 12-column matrix written row-by-row:
//
//	channel[k] = pre[(k % 25) * 12 + (k / 25)]
//
// 300 entries covering both axes exactly.
var cacInterleavePerm = computeCACInterleavePerm()

func computeCACInterleavePerm() [CACChannelBits]int {
	var out [CACChannelBits]int
	for k := 0; k < CACChannelBits; k++ {
		out[k] = (k%25)*12 + (k / 25)
	}
	return out
}

// EncodeCACChannel encodes the spec-correct CAC outbound coding
// chain: CACInfoBits user bits → +16-bit CRC → +4 zero tail →
// K=5 R=1/2 convolutional encode → 50-position puncture →
// 25×12 block interleave. Returns CACChannelBits on-air bits.
//
// Input length must be exactly CACInfoBits; returns nil for any
// other length so callers can spot misuse early.
func EncodeCACChannel(info []byte) []byte {
	if len(info) != CACInfoBits {
		return nil
	}
	// CRC over the 155-bit info block (bit-level since 155 isn't
	// byte-aligned).
	crc := cacCRC16(info)
	// Build the 175-bit pre-encode input: info ‖ CRC ‖ tail zeros.
	const preEncodeLen = CACInfoBits + 16 + cacTailBits
	src := make([]byte, preEncodeLen)
	copy(src, info)
	for i := 0; i < 16; i++ {
		src[CACInfoBits+i] = byte((crc >> uint(15-i)) & 1)
	}
	// tail bits already zero (make zero-fills).
	channel := framing.EncodeK5(src) // 350 bits
	// Puncture: drop the 50 positions enumerated by
	// cacPuncturePositions, preserving order of the survivors.
	punctured := make([]byte, 0, CACChannelBits)
	punc := 0
	for i, b := range channel {
		if punc < len(cacPuncturePositions) && cacPuncturePositions[punc] == i {
			punc++
			continue
		}
		punctured = append(punctured, b)
	}
	// Interleave: channel[k] = punctured[perm[k]].
	out := make([]byte, CACChannelBits)
	for k := 0; k < CACChannelBits; k++ {
		out[k] = punctured[cacInterleavePerm[k]]
	}
	return out
}

// DecodeCACChannel runs the inverse of the §4.5.1.1 outbound
// chain: deinterleave → depuncture (insert DepunctureMark at the
// 50 drop positions so ViterbiK5 skips their cost contribution) →
// K=5 Viterbi decode 175 stages with terminal-state-0 constraint →
// strip the 4 tail bits → verify the 16-bit CRC trailer over the
// 155-bit info block.
//
// Returns the 155-bit info block (excluding the recovered CRC and
// tail) and an ok flag. ok == false means the CRC didn't match —
// the returned slice is still populated with the Viterbi-corrected
// info bits so callers can log the rejected frame if useful.
func DecodeCACChannel(channel []byte) ([]byte, bool) {
	if len(channel) != CACChannelBits {
		return nil, false
	}
	// Inverse interleave: punctured[perm[k]] = channel[k].
	punctured := make([]byte, CACChannelBits)
	for k := 0; k < CACChannelBits; k++ {
		punctured[cacInterleavePerm[k]] = channel[k]
	}
	// Depuncture: insert DepunctureMark at the 50 drop positions
	// so the Viterbi metric ignores them, restoring the 350-bit
	// stream the K=5 decoder consumes.
	const preDepunctureLen = 2 * (CACInfoBits + 16 + cacTailBits)
	depunctured := make([]byte, preDepunctureLen)
	src := 0
	punc := 0
	for i := 0; i < preDepunctureLen; i++ {
		if punc < len(cacPuncturePositions) && cacPuncturePositions[punc] == i {
			depunctured[i] = framing.DepunctureMark
			punc++
			continue
		}
		depunctured[i] = punctured[src]
		src++
	}
	// K=5 Viterbi over CACInfoBits + 16 CRC + 4 tail = 175 stages.
	const stages = CACInfoBits + 16 + cacTailBits
	all, _ := framing.ViterbiK5(depunctured, stages)
	// Verify CRC: recompute over the first CACInfoBits and
	// compare to the 16 recovered CRC bits.
	info := all[:CACInfoBits]
	want := cacCRC16(info)
	var got uint16
	for i := 0; i < 16; i++ {
		got = (got << 1) | uint16(all[CACInfoBits+i]&1)
	}
	return info, got == want
}

// cacCRC16 computes the 16-bit CRC-CCITT over the supplied bit
// slice (each entry 0/1, MSB-first). Polynomial 0x1021, init
// 0xFFFF, no reflection, no final XOR — same convention as
// framing.CRCCCITT, evaluated bit-level since the CAC's
// CACInfoBits (155) isn't byte-aligned.
func cacCRC16(bits []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range bits {
		topBit := (crc >> 15) & 1
		crc = (crc << 1) & 0xFFFF
		if topBit^uint16(b&1) != 0 {
			crc ^= 0x1021
		}
	}
	return crc
}

// init guards the spec invariants at package load time so a
// future edit that breaks the algebra fails loudly instead of
// silently producing wrong CRC / wrong channel lengths.
func init() {
	if len(cacPuncturePositions) != 50 {
		panic(fmt.Sprintf("nxdn: cacPuncturePositions length = %d, want 50", len(cacPuncturePositions)))
	}
	const preEncodeLen = CACInfoBits + 16 + cacTailBits
	if preEncodeLen != 175 {
		panic(fmt.Sprintf("nxdn: CAC pre-encode length = %d, want 175", preEncodeLen))
	}
	if 2*preEncodeLen-len(cacPuncturePositions) != CACChannelBits {
		panic(fmt.Sprintf("nxdn: 2*%d - %d = %d, want %d",
			preEncodeLen, len(cacPuncturePositions),
			2*preEncodeLen-len(cacPuncturePositions), CACChannelBits))
	}
}
