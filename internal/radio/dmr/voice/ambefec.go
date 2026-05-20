package voice

import (
	"fmt"
	"math/bits"
)

// AMBE+2 on-air forward-error-correction for DMR voice frames.
//
// Each 72-bit on-air AMBE+2 frame DMR carries (the 3600 bps
// "3600x2450" variant) wraps 49 bits of vocoder payload in FEC. This
// file undoes that wrapping: it deinterleaves the 72 bits into the
// four C0..C3 sub-vectors (C0:24, C1:23, C2:11, C3:14 bits), runs
// Golay(23,12) error correction over C0 and C1, descrambles C1 with
// the C0-seeded pseudo-random sequence, and assembles the 49-bit
// `ambe_d` payload (C0:12 + C1:12 + C2:11 + C3:14).
//
// Algorithmic reference, ported with bit layouts preserved 1:1:
//   - C0/C1/C2/C3 ECC, descramble and assembly: szechyjs/mbelib's
//     mbe_processAmbe3600x2450Frame (ambe3600x2450.c) and
//     mbe_golay2312 / mbe_checkGolayBlock (ecc.c) — ISC-licensed.
//   - the 72-bit on-air → C0..C3 deinterleave schedule (rW/rX/rY/rZ):
//     szechyjs/dsd's dmr_const.h + dmr_voice.c — ISC-licensed.

const (
	ambeOnAirBits = 72 // on-air bits per AMBE+2 voice frame
	ambeInfoBits  = 49 // vocoder-payload bits per frame (ambe_d)
)

// rW/rX/rY/rZ are the DMR AMBE deinterleave schedule. For dibit i of
// the 36-dibit frame, the high bit goes to ambe_fr[rW[i]][rX[i]] and
// the low bit to ambe_fr[rY[i]][rZ[i]] (verbatim from szechyjs/dsd).
var (
	rW = [36]int{
		0, 1, 0, 1, 0, 1,
		0, 1, 0, 1, 0, 1,
		0, 1, 0, 1, 0, 1,
		0, 1, 0, 1, 0, 2,
		0, 2, 0, 2, 0, 2,
		0, 2, 0, 2, 0, 2,
	}
	rX = [36]int{
		23, 10, 22, 9, 21, 8,
		20, 7, 19, 6, 18, 5,
		17, 4, 16, 3, 15, 2,
		14, 1, 13, 0, 12, 10,
		11, 9, 10, 8, 9, 7,
		8, 6, 7, 5, 6, 4,
	}
	rY = [36]int{
		0, 2, 0, 2, 0, 2,
		0, 2, 0, 3, 0, 3,
		1, 3, 1, 3, 1, 3,
		1, 3, 1, 3, 1, 3,
		1, 3, 1, 3, 1, 3,
		1, 3, 1, 3, 1, 3,
	}
	rZ = [36]int{
		5, 3, 4, 2, 3, 1,
		2, 0, 1, 13, 0, 12,
		22, 11, 21, 10, 20, 9,
		19, 8, 18, 7, 17, 6,
		16, 5, 15, 4, 14, 3,
		13, 2, 12, 1, 11, 0,
	}
)

// golayGen is mbelib's Golay(23,12) generator: golayGen[i] is the
// 11-bit parity contribution of data bit i (bit 11-i of the data
// word). Verbatim from mbelib ecc_const.h.
var golayGen = [12]uint16{
	0x63a, 0x31d, 0x7b4, 0x3da, 0x1ed, 0x6cc,
	0x366, 0x1b3, 0x6e3, 0x54b, 0x49f, 0x475,
}

// golaySyndrome maps an 11-bit Golay(23,12) syndrome to the 12-bit
// data-error coset leader. Built at init from golayGen — equivalent
// to mbelib's precomputed golayMatrix[2048] (the (23,12,7) Golay is a
// perfect code, so the 2048 syndromes biject with the weight-≤3
// error patterns).
var golaySyndrome [2048]uint16

// golayParity returns the 11-bit expected parity of a 12-bit data
// word (data bit i, i.e. bit 11-i of data, contributes golayGen[i]).
func golayParity(data uint16) uint16 {
	var p uint16
	for i := 0; i < 12; i++ {
		if data&(1<<uint(11-i)) != 0 {
			p ^= golayGen[i]
		}
	}
	return p
}

func init() {
	apply := func(e uint32) {
		de := uint16(e>>11) & 0x0FFF
		pe := uint16(e) & 0x07FF
		golaySyndrome[golayParity(de)^pe] = de
	}
	apply(0)
	for a := 0; a < 23; a++ {
		apply(1 << uint(a))
		for b := a + 1; b < 23; b++ {
			apply(1<<uint(a) | 1<<uint(b))
			for c := b + 1; c < 23; c++ {
				apply(1<<uint(a) | 1<<uint(b) | 1<<uint(c))
			}
		}
	}
}

// golayDecode2312 corrects a 23-bit Golay(23,12) codeword — data in
// bits 22..11, parity in bits 10..0 — returning the 12 data bits and
// the count of corrected data-bit errors.
func golayDecode2312(cw uint32) (uint16, int) {
	de := uint16(cw>>11) & 0x0FFF
	pe := uint16(cw) & 0x07FF
	corrected := de ^ golaySyndrome[golayParity(de)^pe]
	return corrected, bits.OnesCount16(de ^ corrected)
}

// golayEncode2312 builds the 23-bit systematic Golay(23,12) codeword
// for 12 data bits (data in bits 22..11, parity in 10..0).
func golayEncode2312(data uint16) uint32 {
	data &= 0x0FFF
	return uint32(data)<<11 | uint32(golayParity(data))
}

// c1Keystream returns the pseudo-random sequence DMR XORs onto the C1
// sub-vector, seeded by the 12-bit C0 data word. Entries 1..23 are
// valid (index 0 is unused). Ported from mbelib
// mbe_demodulateAmbe3600x2450Data.
func c1Keystream(c0data uint16) [24]uint8 {
	var pr [24]uint32
	pr[0] = 16 * uint32(c0data&0x0FFF)
	for i := 1; i < 24; i++ {
		pr[i] = (173*pr[i-1] + 13849) & 0xFFFF
	}
	var ks [24]uint8
	for i := 1; i < 24; i++ {
		ks[i] = uint8(pr[i] >> 15)
	}
	return ks
}

// DecodeAMBEFrame decodes one 72-bit on-air AMBE+2 voice frame (72
// bits, one bit per byte MSB-first, as produced by AMBEFrames) into
// the 49-bit vocoder payload. It returns the payload (one bit per
// byte) and the number of Golay errors corrected across C0 and C1.
func DecodeAMBEFrame(frame []byte) ([]byte, int, error) {
	if len(frame) != ambeOnAirBits {
		return nil, 0, fmt.Errorf("dmr/voice: AMBE frame must be %d bits, got %d", ambeOnAirBits, len(frame))
	}
	var fr [4][24]uint8
	for i := 0; i < 36; i++ {
		fr[rW[i]][rX[i]] = frame[2*i] & 1
		fr[rY[i]][rZ[i]] = frame[2*i+1] & 1
	}

	// C0: Golay(23,12) over fr[0][1..23].
	var c0cw uint32
	for j := 0; j < 23; j++ {
		c0cw |= uint32(fr[0][j+1]) << uint(j)
	}
	c0data, c0errs := golayDecode2312(c0cw)

	// C1: descramble with the C0-seeded keystream, then Golay(23,12).
	ks := c1Keystream(c0data)
	for j := 0; j <= 22; j++ {
		fr[1][j] ^= ks[23-j]
	}
	var c1cw uint32
	for j := 0; j < 23; j++ {
		c1cw |= uint32(fr[1][j]) << uint(j)
	}
	c1data, c1errs := golayDecode2312(c1cw)

	// Assemble ambe_d: C0(12) + C1(12) + C2(11) + C3(14).
	out := make([]byte, ambeInfoBits)
	for k := 0; k < 12; k++ {
		out[k] = uint8((c0data >> uint(11-k)) & 1)
		out[12+k] = uint8((c1data >> uint(11-k)) & 1)
	}
	for k := 0; k < 11; k++ {
		out[24+k] = fr[2][10-k]
	}
	for k := 0; k < 14; k++ {
		out[35+k] = fr[3][13-k]
	}
	return out, c0errs + c1errs, nil
}

// EncodeAMBEFrame is the inverse of DecodeAMBEFrame: it wraps a 49-bit
// vocoder payload back into a 72-bit on-air AMBE+2 frame. It exists so
// the FEC chain can be exercised by round-trip and bit-error tests.
func EncodeAMBEFrame(info []byte) ([]byte, error) {
	if len(info) != ambeInfoBits {
		return nil, fmt.Errorf("dmr/voice: AMBE payload must be %d bits, got %d", ambeInfoBits, len(info))
	}
	var fr [4][24]uint8

	var c0data, c1data uint16
	for k := 0; k < 12; k++ {
		c0data |= uint16(info[k]&1) << uint(11-k)
		c1data |= uint16(info[12+k]&1) << uint(11-k)
	}

	c0cw := golayEncode2312(c0data)
	for j := 0; j < 23; j++ {
		fr[0][j+1] = uint8((c0cw >> uint(j)) & 1)
	}
	fr[0][0] = uint8(bits.OnesCount32(c0cw) & 1) // Golay(24) overall parity

	c1cw := golayEncode2312(c1data)
	for j := 0; j < 23; j++ {
		fr[1][j] = uint8((c1cw >> uint(j)) & 1)
	}
	ks := c1Keystream(c0data)
	for j := 0; j <= 22; j++ {
		fr[1][j] ^= ks[23-j]
	}

	for k := 0; k < 11; k++ {
		fr[2][10-k] = info[24+k] & 1
	}
	for k := 0; k < 14; k++ {
		fr[3][13-k] = info[35+k] & 1
	}

	out := make([]byte, ambeOnAirBits)
	for i := 0; i < 36; i++ {
		out[2*i] = fr[rW[i]][rX[i]]
		out[2*i+1] = fr[rY[i]][rZ[i]]
	}
	return out, nil
}
