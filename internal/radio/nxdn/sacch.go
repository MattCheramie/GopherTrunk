package nxdn

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// NXDN Slow Associated Control Channel (SACCH) channel coding, per
// NXDN Forum technical specification rev 1.4 §6.6 (cross-checked
// against MMDVMHost NXDNSACCH.cpp).
//
// The SACCH carries per-frame signaling on RTCH/RDCH frames. Each
// transmitted SACCH block is 60 bits, produced from 32 information
// bits (26 payload + 6-bit CRC trailer) as follows:
//
//	32-bit info  (26 payload bits + 6-bit CRC over those 26)
//	      │  append 4 zero tail bits
//	      ▼
//	36-bit input
//	      │  K=5 1/2-rate convolutional encode (g1=0o31, g2=0o27)
//	      ▼
//	72 channel bits
//	      │  drop 12 puncture positions
//	      ▼
//	60 bits
//	      │  sub-frame interleave (60-position permutation)
//	      ▼
//	60 bits transmitted on-air
//
// The receive pipeline runs the inverse: deinterleave → depuncture
// (fill punctured positions with a sentinel that contributes zero cost
// in the Viterbi metric) → Viterbi decode 36 stages with end-state
// constrained to 0 → strip the 4 tail bits → CRC-6 verify the 32-bit
// info block.
//
// Tables sourced from MMDVMHost (NXDNConvolution.cpp, NXDNSACCH.cpp)
// and f4exb/dsdcc (nxdn.cpp). The SACCH FEC parameters were
// cross-checked against the published NXDN technical specification.

// SACCHChannelBits is the number of bits transmitted per SACCH block.
const SACCHChannelBits = 60

// SACCHInfoBits is the number of information bits per SACCH block
// (26 payload + 6 CRC trailer). Tail bits are an internal detail.
const SACCHInfoBits = 32

// sacchTailBits is the number of zero tail bits appended to the info
// before convolutional encoding so the encoder ends in state 0 and the
// receiver's Viterbi can use the terminal-state constraint.
const sacchTailBits = 4

// sacchPayloadBits is the number of bits the CRC-6 trailer protects
// (the first 26 bits of the 32-bit info block).
const sacchPayloadBits = 26

// sacchInterleavePerm[i] = j means channel[i] = punctured[j] on the
// encoder side. Equivalently, depunctured[j] = received[i] on the
// decoder side after deinterleaving via the inverse permutation.
//
// Sourced from f4exb/dsdcc nxdn.cpp DSDNXDN::SACCH::m_Interleave[60].
var sacchInterleavePerm = [60]int{
	0, 12, 24, 36, 48,
	1, 13, 25, 37, 49,
	2, 14, 26, 38, 50,
	3, 15, 27, 39, 51,
	4, 16, 28, 40, 52,
	5, 17, 29, 41, 53,
	6, 18, 30, 42, 54,
	7, 19, 31, 43, 55,
	8, 20, 32, 44, 56,
	9, 21, 33, 45, 57,
	10, 22, 34, 46, 58,
	11, 23, 35, 47, 59,
}

// sacchPuncturePositions are the indexes (within the 80-bit pre-
// puncture stream) of the bits that get DROPPED to produce the 60-bit
// transmitted block. Sourced from dsdcc m_PunctureList[12].
var sacchPuncturePositions = [12]int{5, 11, 17, 23, 29, 35, 41, 47, 53, 59, 65, 71}

// nxdn K=5 1/2-rate convolutional code generators (NXDN spec §6.6.2).
//
//	g1(D) = 1 + D^3 + D^4   (octal 31, hex 0x19)
//	g2(D) = 1 + D + D^2 + D^4 (octal 27, hex 0x17)
//
// See MMDVMHost NXDNConvolution::encode.
const (
	nxdnConvG1 = 0x19
	nxdnConvG2 = 0x17
)

// EncodeSACCH turns 32 information bits into the 60-bit SACCH channel
// block. The input slice is bits (each entry 0/1), output is the same.
// Useful for synthetic test streams.
func EncodeSACCH(info []byte) []byte {
	if len(info) != SACCHInfoBits {
		panic(fmt.Sprintf("nxdn: SACCH info must be %d bits, got %d", SACCHInfoBits, len(info)))
	}
	const inputLen = SACCHInfoBits + sacchTailBits // 36 input bits
	const preLen = inputLen * 2                    // 72 channel bits before puncturing
	pre := make([]byte, preLen)
	state := 0
	for i := 0; i < inputLen; i++ {
		var d byte
		if i < SACCHInfoBits {
			d = info[i] & 1
		} // tail bits are zero by default
		d1 := (state >> 3) & 1
		d2 := (state >> 2) & 1
		d3 := (state >> 1) & 1
		d4 := state & 1
		g1 := byte(int(d)^d3^d4) & 1
		g2 := byte(int(d)^d1^d2^d4) & 1
		pre[2*i] = g1
		pre[2*i+1] = g2
		state = (int(d) << 3) | (d1 << 2) | (d2 << 1) | d3
	}
	punctured := puncture(pre)
	out := make([]byte, SACCHChannelBits)
	for i := 0; i < SACCHChannelBits; i++ {
		out[i] = punctured[sacchInterleavePerm[i]]
	}
	return out
}

// ErrSACCHCRCFail is returned by DecodeSACCH when the K=5 Viterbi
// decode succeeds structurally but the 6-bit CRC trailer does not
// match the recovered payload — the corrected bits are still returned
// so callers can log the rejected message.
var ErrSACCHCRCFail = errors.New("nxdn: SACCH CRC-6 mismatch")

// DecodeSACCH runs deinterleave → depuncture → Viterbi → CRC-6 over
// the 60 received channel bits and returns the 36 decoded information
// bits, the Viterbi path metric (sum of dibit-distance penalties; 0
// means clean), and an error chained from ErrSACCHCRCFail if the CRC
// trailer doesn't match.
func DecodeSACCH(channel []byte) ([]byte, int, error) {
	if len(channel) != SACCHChannelBits {
		return nil, -1, fmt.Errorf("nxdn: SACCH channel must be %d bits, got %d", SACCHChannelBits, len(channel))
	}
	// Inverse interleave: punctured[perm[i]] = channel[i].
	punctured := make([]byte, SACCHChannelBits)
	for i := 0; i < SACCHChannelBits; i++ {
		punctured[sacchInterleavePerm[i]] = channel[i]
	}
	pre := depuncture(punctured)
	const stages = SACCHInfoBits + sacchTailBits // 36 Viterbi stages
	allBits, metric := framing.ViterbiK5(pre, stages)
	out := allBits[:SACCHInfoBits]
	// CRC-6 over the first 26 bits = 18 SR + ... payload preceding the
	// CRC trailer (per NXDN Single SACCH layout). We don't dictate
	// payload semantics here; the caller selects which prefix to CRC.
	if !VerifySACCHCRC6(out) {
		return out, metric, ErrSACCHCRCFail
	}
	return out, metric, nil
}

// depunctureMark is the sentinel byte the K=5 Viterbi recognizes as
// "no information at this slot". Aliased onto framing.DepunctureMark
// so the puncture / depuncture helpers below stay readable.
const depunctureMark = framing.DepunctureMark

// puncture drops 12 fixed positions from the first 72 of the 80-bit
// encoded stream, plus the trailing 8 tail bits (the encoder ends in
// state 0 so the receiver can re-introduce them as zeros). Output: 60
// transmitted bits.
func puncture(in []byte) []byte {
	out := make([]byte, 0, SACCHChannelBits)
	punc := 0
	for i := 0; i < 72; i++ {
		if punc < len(sacchPuncturePositions) && sacchPuncturePositions[punc] == i {
			punc++
			continue
		}
		out = append(out, in[i])
	}
	return out
}

// depuncture is the receive-side inverse: insert "no-info" markers at
// the 12 puncture positions to produce 72 bits the Viterbi decoder
// consumes as 36 stages.
func depuncture(in []byte) []byte {
	out := make([]byte, 72)
	src := 0
	punc := 0
	for i := 0; i < 72; i++ {
		if punc < len(sacchPuncturePositions) && sacchPuncturePositions[punc] == i {
			out[i] = depunctureMark
			punc++
			continue
		}
		out[i] = in[src]
		src++
	}
	return out
}

// CRC-6 over a bit-stream with NXDN's polynomial g(x) = x^6 + x + 1
// (= 0x43 with the implicit MSB). Standard sliding-XOR style.
const crc6Poly = 0x43 // x^6 + x + 1, plus implicit x^6

// SACCHCRC6 computes the 6-bit CRC over the first `nBits` bits of the
// supplied bit slice (each entry 0/1), MSB-first. The 6-bit result
// occupies the low bits of the returned byte.
func SACCHCRC6(bits []byte, nBits int) byte {
	if nBits > len(bits) {
		nBits = len(bits)
	}
	var reg uint8
	for i := 0; i < nBits; i++ {
		bit := bits[i] & 1
		fb := ((reg >> 5) ^ bit) & 1
		reg = ((reg << 1) | fb) & 0x3F
		if fb != 0 {
			reg ^= crc6Poly & 0x3F
		}
	}
	return reg & 0x3F
}

// VerifySACCHCRC6 checks that the trailing 6 bits of the supplied 32-
// bit info block are the correct CRC-6 over the preceding 26 bits. Per
// NXDN Single SACCH layout (§6.6): bits 0..25 are payload, bits 26..31
// are the CRC-6 trailer.
func VerifySACCHCRC6(info []byte) bool {
	if len(info) != SACCHInfoBits {
		return false
	}
	want := SACCHCRC6(info[:sacchPayloadBits], sacchPayloadBits)
	var got byte
	for i := 0; i < 6; i++ {
		got = (got << 1) | (info[sacchPayloadBits+i] & 1)
	}
	return got == want
}

// AppendSACCHCRC6 returns a 32-bit info block built from the supplied
// 26 payload bits plus a freshly-computed 6-bit CRC trailer. Useful in
// tests and for any future encoder paths.
func AppendSACCHCRC6(payload []byte) []byte {
	if len(payload) != sacchPayloadBits {
		panic("nxdn: SACCH payload must be 26 bits")
	}
	out := make([]byte, SACCHInfoBits)
	copy(out, payload)
	crc := SACCHCRC6(payload, sacchPayloadBits)
	for i := 0; i < 6; i++ {
		out[sacchPayloadBits+i] = (crc >> uint(5-i)) & 1
	}
	return out
}
