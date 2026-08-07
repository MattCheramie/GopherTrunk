package nxdn

import (
	"fmt"
	"math/bits"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// AMBE+2 on-air forward-error-correction for NXDN VCH voice frames.
//
// NXDN's VCH carries the AMBE+2 3600×2450 vocoder — the SAME codec and
// SAME 72-bit on-air frame DMR uses. The FEC that wraps the 49-bit
// vocoder payload (Golay(23,12) over the C0/C1 sub-vectors, the
// C0-seeded pseudo-random descramble of C1, and the C0:12 + C1:12 +
// C2:11 + C3:14 = 49-bit assembly) is a property of the AMBE+2 codec,
// not of the radio protocol, so it is identical to
// internal/radio/dmr/voice/ambefec.go and is reproduced here reusing
// the shared framing.Golay23_12 primitives.
//
// The ONE protocol-specific piece is how the 72 on-air bits are
// interleaved into the four C0..C3 sub-vectors. DMR uses the
// rW/rX/rY/rZ schedule; NXDN uses its own.
//
// ⚠️ UNVERIFIED ON AIR. GopherTrunk has no NXDN voice capture, so the
// deinterleave table below is a documented *placeholder* (a direct
// sequential split C0|C1|C2|C3) — enough to exercise the FEC machinery
// end to end via the synthetic round-trip in voice_ambe_test.go, but
// NOT confirmed against a real NXDN VCH burst. The first thing a
// capture confirms is this table (and whether the codebook is 3600×2450
// vs 3600×2400). Until then, treat live NXDN audio as experimental. See
// the WS2a plan and CLAUDE.md's TETRA-CRC "self-consistent bug" lesson.

const (
	nxdnAMBEOnAirBits = 72 // on-air bits per AMBE+2 VCH voice frame
	nxdnAMBEInfoBits  = 49 // vocoder-payload bits per frame (ambe_d)
)

// nxdnAMBEDeinterleave maps the 72 on-air bits (one bit per byte,
// MSB-first) into the four C0..C3 sub-vectors carried in fr[4][24]:
// C0 = fr[0][0..23] (24 bits: fr[0][0] is the Golay(24) overall-parity
// cell, fr[0][1..23] the 23-bit codeword), C1 = fr[1][0..22] (23 bits),
// C2 = fr[2][0..10] (11 bits), C3 = fr[3][0..13] (14 bits). Total
// 24+23+11+14 = 72.
//
// PLACEHOLDER: a direct sequential split. Replace with the real NXDN
// VCH interleave table once a capture is available (the encoder inverse
// below must be updated in lock-step). Isolated here so that swap is a
// one-function change.
func nxdnAMBEDeinterleave(frame []byte) [4][24]uint8 {
	var fr [4][24]uint8
	p := 0
	get := func() uint8 { b := frame[p] & 1; p++; return b }
	for j := 0; j < 24; j++ {
		fr[0][j] = get()
	}
	for j := 0; j < 23; j++ {
		fr[1][j] = get()
	}
	for j := 0; j < 11; j++ {
		fr[2][j] = get()
	}
	for j := 0; j < 14; j++ {
		fr[3][j] = get()
	}
	return fr
}

// nxdnAMBEInterleave is the inverse of nxdnAMBEDeinterleave, packing the
// four sub-vectors back into 72 on-air bits. Kept in lock-step with the
// deinterleave placeholder so the round-trip test can exercise the FEC.
func nxdnAMBEInterleave(fr [4][24]uint8) []byte {
	out := make([]byte, nxdnAMBEOnAirBits)
	p := 0
	put := func(b uint8) { out[p] = b & 1; p++ }
	for j := 0; j < 24; j++ {
		put(fr[0][j])
	}
	for j := 0; j < 23; j++ {
		put(fr[1][j])
	}
	for j := 0; j < 11; j++ {
		put(fr[2][j])
	}
	for j := 0; j < 14; j++ {
		put(fr[3][j])
	}
	return out
}

// nxdnC1Keystream returns the pseudo-random sequence AMBE+2 XORs onto
// the C1 sub-vector, seeded by the 12-bit C0 data word. Entries 1..23
// are valid (index 0 unused). This is the mbelib
// mbe_demodulateAmbe3600x2450Data keystream — a property of the AMBE+2
// 3600×2450 codec, identical to the DMR path.
func nxdnC1Keystream(c0data uint16) [24]uint8 {
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

// DecodeVCHFrame decodes one 72-bit on-air AMBE+2 VCH voice frame (72
// bits, one bit per byte MSB-first) into the 49-bit vocoder payload
// (one bit per byte). It returns the payload and the number of Golay
// errors corrected across C0 and C1 (negative counts from an
// uncorrectable sub-vector are clamped to 0). Pack the payload MSB-first
// to 7 bytes (framing.PackBitsMSB) and hand it to the ambe2 decoder.
func DecodeVCHFrame(frame []byte) ([]byte, int, error) {
	if len(frame) != nxdnAMBEOnAirBits {
		return nil, 0, fmt.Errorf("nxdn: VCH AMBE frame must be %d bits, got %d", nxdnAMBEOnAirBits, len(frame))
	}
	fr := nxdnAMBEDeinterleave(frame)

	// C0: Golay(23,12) over fr[0][1..23] (fr[0][0] is the ext-Golay
	// overall-parity cell, unused by the 23-bit decode).
	var c0cw uint32
	for j := 0; j < 23; j++ {
		c0cw |= uint32(fr[0][j+1]) << uint(j)
	}
	c0data, c0errs := framing.GolayDecode23_12(c0cw)

	// C1: descramble with the C0-seeded keystream, then Golay(23,12).
	ks := nxdnC1Keystream(c0data)
	for j := 0; j <= 22; j++ {
		fr[1][j] ^= ks[23-j]
	}
	var c1cw uint32
	for j := 0; j < 23; j++ {
		c1cw |= uint32(fr[1][j]) << uint(j)
	}
	c1data, c1errs := framing.GolayDecode23_12(c1cw)

	// Assemble ambe_d: C0(12) + C1(12) + C2(11) + C3(14).
	out := make([]byte, nxdnAMBEInfoBits)
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
	return out, clampErrs(c0errs) + clampErrs(c1errs), nil
}

// clampErrs turns a Golay decoder's -1 (uncorrectable) into 0 so the
// returned corrected-bit count stays a non-negative sum.
func clampErrs(e int) int {
	if e < 0 {
		return 0
	}
	return e
}

// EncodeVCHFrame is the inverse of DecodeVCHFrame: it wraps a 49-bit
// vocoder payload back into a 72-bit on-air AMBE+2 VCH frame. It exists
// so the FEC chain can be exercised by round-trip and bit-error tests.
func EncodeVCHFrame(info []byte) ([]byte, error) {
	if len(info) != nxdnAMBEInfoBits {
		return nil, fmt.Errorf("nxdn: VCH AMBE payload must be %d bits, got %d", nxdnAMBEInfoBits, len(info))
	}
	var fr [4][24]uint8

	var c0data, c1data uint16
	for k := 0; k < 12; k++ {
		c0data |= uint16(info[k]&1) << uint(11-k)
		c1data |= uint16(info[12+k]&1) << uint(11-k)
	}

	c0cw := framing.GolayEncode23_12(c0data)
	for j := 0; j < 23; j++ {
		fr[0][j+1] = uint8((c0cw >> uint(j)) & 1)
	}
	fr[0][0] = uint8(bits.OnesCount32(c0cw) & 1) // ext-Golay overall parity

	c1cw := framing.GolayEncode23_12(c1data)
	for j := 0; j < 23; j++ {
		fr[1][j] = uint8((c1cw >> uint(j)) & 1)
	}
	ks := nxdnC1Keystream(c0data)
	for j := 0; j <= 22; j++ {
		fr[1][j] ^= ks[23-j]
	}

	for k := 0; k < 11; k++ {
		fr[2][10-k] = info[24+k] & 1
	}
	for k := 0; k < 14; k++ {
		fr[3][13-k] = info[35+k] & 1
	}

	return nxdnAMBEInterleave(fr), nil
}
