package dstar

import (
	"fmt"
	"math/bits"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

// AMBE on-air forward-error-correction for D-STAR DV voice frames.
//
// D-STAR's DV mode carries the original AMBE 3600×2400 vocoder — the
// codec internal/voice/ambe2's base decoder implements (params.go is a
// 1:1 port of mbelib's mbe_decodeAmbe2400Parms from ambe3600x2400.c).
// The FEC that wraps the vocoder payload is structurally identical to
// the AMBE+2 3600×2450 layer DMR / NXDN / dPMR use (mbelib's
// ambe3600x2400 and ambe3600x2450 ECC paths share it): Golay(23,12)
// over the C0/C1 sub-vectors (with an extended-Golay overall-parity
// cell on C0), a C0-seeded pseudo-random descramble of C1, and the
// C0:12 + C1:12 + C2:11 + C3:14 = 49-bit ambe_d assembly the vocoder
// unpacks. It is reproduced here with mbelib's Golay(23,12), the one
// internal/radio/dmr/voice/ambefec.go carries. Until #1261 this file
// used framing.GolayDecode23_12, a different (23,12) Golay code, so even
// a clean frame decoded to the wrong parameters.
//
// The ONE protocol-specific piece is how the 72 on-air bits are
// interleaved into the four C0..C3 sub-vectors: DSD's D-STAR schedule
// (dstar_const.h dW / dX, ISC licence): on-air bit i, in time order and
// in MMDVM's polarity, is fr[dstarDW[i]][dstarDX[i]]. GT carried a
// sequential C0|C1|C2|C3 placeholder here until #1261. The schedule is
// pinned by the null AMBE frame MMDVMHost and the MMDVM firmware send
// in a silent DV frame (9E 8D 32 88 26 1A 3F 61 E8, bytes LSB first):
// it decodes with no Golay correction in C0 or in the descrambled C1,
// which a wrong schedule does about once in four million frames
// (voice_ambe_test.go).

const (
	dstarAMBEOnAirBits = 72 // on-air bits per AMBE DV voice frame
	dstarAMBEInfoBits  = 49 // vocoder-payload bits per frame (ambe_d)
)

// dstarDW / dstarDX are DSD's dW / dX: the sub-vector and the cell of
// each on-air voice bit.
var dstarDW = [dstarAMBEOnAirBits]uint8{
	0, 0, 3, 2, 1, 1, 0, 0, 1, 1, 0, 0,
	3, 2, 1, 1, 3, 2, 1, 1, 0, 0, 3, 2,
	0, 0, 3, 2, 1, 1, 0, 0, 1, 1, 0, 0,
	3, 2, 1, 1, 3, 2, 1, 1, 0, 0, 3, 2,
	0, 0, 3, 2, 1, 1, 0, 0, 1, 1, 0, 0,
	3, 2, 1, 1, 3, 3, 2, 1, 0, 0, 3, 3,
}

var dstarDX = [dstarAMBEOnAirBits]uint8{
	10, 22, 11, 9, 10, 22, 11, 23, 8, 20, 9, 21,
	10, 8, 9, 21, 8, 6, 7, 19, 8, 20, 9, 7,
	6, 18, 7, 5, 6, 18, 7, 19, 4, 16, 5, 17,
	6, 4, 5, 17, 4, 2, 3, 15, 4, 16, 5, 3,
	2, 14, 3, 1, 2, 14, 3, 15, 0, 12, 1, 13,
	2, 0, 1, 13, 0, 12, 10, 11, 0, 12, 1, 13,
}

// dstarAMBEDeinterleave maps the 72 on-air voice bits (one bit per
// byte, time order) into the four C0..C3 sub-vectors carried in
// fr[4][24], mbelib's ambe_fr layout: C0 = fr[0][0..23] (fr[0][0] is
// the Golay(24) overall-parity cell, fr[0][1..23] the 23-bit
// codeword), C1 = fr[1][0..22], C2 = fr[2][0..10], C3 = fr[3][0..13].
func dstarAMBEDeinterleave(frame []byte) [4][24]uint8 {
	var fr [4][24]uint8
	for i := 0; i < dstarAMBEOnAirBits; i++ {
		fr[dstarDW[i]][dstarDX[i]] = frame[i] & 1
	}
	return fr
}

// dstarAMBEInterleave is the inverse of dstarAMBEDeinterleave.
func dstarAMBEInterleave(fr [4][24]uint8) []byte {
	out := make([]byte, dstarAMBEOnAirBits)
	for i := 0; i < dstarAMBEOnAirBits; i++ {
		out[i] = fr[dstarDW[i]][dstarDX[i]] & 1
	}
	return out
}

// dstarC1Keystream returns the pseudo-random sequence AMBE XORs onto
// the C1 sub-vector, seeded by the 12-bit C0 data word. Entries 1..23
// are valid (index 0 unused). This is mbelib's
// mbe_demodulateAmbe3600x2400Data keystream — identical to the
// 3600×2450 one the DMR / NXDN / dPMR paths use.
func dstarC1Keystream(c0data uint16) [24]uint8 {
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

// DecodeDVVoiceBits decodes the 72 voice bits of one D-STAR DV frame
// (one bit per byte, time order) into the 49-bit vocoder payload (one bit
// per byte). It returns the payload and the number of Golay errors
// corrected across C0 and C1 (negative counts from an uncorrectable
// sub-vector are clamped to 0). Pack the payload MSB-first to 7 bytes
// (framing.PackBitsMSB) and hand it to the base ambe2 decoder.
func DecodeDVVoiceBits(frame []byte) ([]byte, int, error) {
	if len(frame) != dstarAMBEOnAirBits {
		return nil, 0, fmt.Errorf("dstar: DV AMBE frame must be %d bits, got %d", dstarAMBEOnAirBits, len(frame))
	}
	fr := dstarAMBEDeinterleave(frame)

	// C0: Golay(23,12) over fr[0][1..23] (fr[0][0] is the ext-Golay
	// overall-parity cell, unused by the 23-bit decode).
	var c0cw uint32
	for j := 0; j < 23; j++ {
		c0cw |= uint32(fr[0][j+1]) << uint(j)
	}
	c0data, c0errs := dmrvoice.GolayDecode2312(c0cw)

	// C1: descramble with the C0-seeded keystream, then Golay(23,12).
	ks := dstarC1Keystream(c0data)
	for j := 0; j <= 22; j++ {
		fr[1][j] ^= ks[23-j]
	}
	var c1cw uint32
	for j := 0; j < 23; j++ {
		c1cw |= uint32(fr[1][j]) << uint(j)
	}
	c1data, c1errs := dmrvoice.GolayDecode2312(c1cw)

	// Assemble ambe_d: C0(12) + C1(12) + C2(11) + C3(14).
	out := make([]byte, dstarAMBEInfoBits)
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
	return out, clampGolayErrs(c0errs) + clampGolayErrs(c1errs), nil
}

// clampGolayErrs turns a Golay decoder's -1 (uncorrectable) into 0 so
// the returned corrected-bit count stays a non-negative sum.
func clampGolayErrs(e int) int {
	if e < 0 {
		return 0
	}
	return e
}

// EncodeDVVoiceBits is the inverse of DecodeDVVoiceBits: it wraps a
// 49-bit vocoder payload back into the 72 on-air voice bits of a DV
// frame. It exists so the FEC chain can be exercised by round-trip and
// bit-error tests.
func EncodeDVVoiceBits(info []byte) ([]byte, error) {
	if len(info) != dstarAMBEInfoBits {
		return nil, fmt.Errorf("dstar: DV AMBE payload must be %d bits, got %d", dstarAMBEInfoBits, len(info))
	}
	var fr [4][24]uint8

	var c0data, c1data uint16
	for k := 0; k < 12; k++ {
		c0data |= uint16(info[k]&1) << uint(11-k)
		c1data |= uint16(info[12+k]&1) << uint(11-k)
	}

	c0cw := dmrvoice.GolayEncode2312(c0data)
	for j := 0; j < 23; j++ {
		fr[0][j+1] = uint8((c0cw >> uint(j)) & 1)
	}
	fr[0][0] = uint8(bits.OnesCount32(c0cw) & 1) // ext-Golay overall parity

	c1cw := dmrvoice.GolayEncode2312(c1data)
	for j := 0; j < 23; j++ {
		fr[1][j] = uint8((c1cw >> uint(j)) & 1)
	}
	ks := dstarC1Keystream(c0data)
	for j := 0; j <= 22; j++ {
		fr[1][j] ^= ks[23-j]
	}

	for k := 0; k < 11; k++ {
		fr[2][10-k] = info[24+k] & 1
	}
	for k := 0; k < 14; k++ {
		fr[3][13-k] = info[35+k] & 1
	}

	return dstarAMBEInterleave(fr), nil
}
