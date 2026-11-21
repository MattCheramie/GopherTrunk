package dpmr

import (
	"encoding/hex"
	"testing"
)

// mkTCHInfo builds a deterministic 49-bit AMBE payload from a seed (a
// simple LCG), mirroring the DMR ambefec_test / NXDN voice_ambe_test
// helpers.
func mkTCHInfo(seed uint32) []byte {
	out := make([]byte, dpmrAMBEInfoBits)
	x := seed | 1
	for i := range out {
		x = 1664525*x + 1013904223
		out[i] = byte((x >> 24) & 1)
	}
	return out
}

// ambeSilenceOnAir is the 72-bit frame an AMBE+2 3600x2450 vocoder puts
// on the air for silence, as MMDVMHost carries it (DMR_SILENCE). dPMR
// fills the rest of a superframe with it when the PTT is released
// (TS 102 658 clause 11: "silence data is the vocoder output data when no
// sound is input"), so it is a frame a real dPMR radio transmits.
const ambeSilenceOnAir = "b9e881526173002a6b"

// ambeSilencePayload is the 49-bit vocoder payload of that frame. Its b0
// (bits 0..3 and 37..39) is 124, the AMBE+2 silence code.
const ambeSilencePayload = "1111100000000001101010011001111110001100111000001"

func bitsOfHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]byte, 8*len(raw))
	for i := range out {
		out[i] = raw[i/8] >> uint(7-i%8) & 1
	}
	return out
}

func bitString(b []byte) string {
	s := make([]byte, len(b))
	for i, v := range b {
		s[i] = '0' + v&1
	}
	return string(s)
}

// ambeSchedule is the vocoder-frame interleave schedule the two decoders
// that receive dPMR voice on the air carry for dPMR: dsdcc (dpmr.cpp, rW..rZ)
// and DSD-FME (dpmr_const.h, dPmrW..dPmrZ), which agree element for
// element. Dibit i of the 36 puts its high bit in cell [W[i]][X[i]] and its
// low bit in cell [Y[i]][Z[i]] of the four sub-vectors C0..C3.
var ambeSchedule = [4][36]int{
	{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 2, 0, 2, 0, 2, 0, 2, 0, 2, 0, 2, 0, 2},
	{23, 10, 22, 9, 21, 8, 20, 7, 19, 6, 18, 5, 17, 4, 16, 3, 15, 2, 14, 1, 13, 0, 12, 10, 11, 9, 10, 8, 9, 7, 8, 6, 7, 5, 6, 4},
	{0, 2, 0, 2, 0, 2, 0, 2, 0, 3, 0, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3},
	{5, 3, 4, 2, 3, 1, 2, 0, 1, 13, 0, 12, 22, 11, 21, 10, 20, 9, 19, 8, 18, 7, 17, 6, 16, 5, 15, 4, 14, 3, 13, 2, 12, 1, 11, 0},
}

// ambeSilenceSubVectors are the C0..C3 sub-vectors of the silence frame
// (cell j of each string is C_k[j]): C0 and C1 are Golay (23,12) codewords
// of the vocoder's own FEC, so they do not depend on any radio protocol.
var ambeSilenceSubVectors = [4]string{
	"000100101000000000011111",
	"10110000101010000010101",
	"00111111001",
	"10000011100110",
}

// TestTCHFrameIsTheAMBESilenceFrame pins the layout of a dPMR voice TCH
// block. Three literals have to agree: the silence frame as transmitted,
// its sub-vectors, and the schedule that maps one to the other. The frame
// must then decode with no Golay correction to the silence payload.
//
// With the sequential C0|C1|C2|C3 split this package used as a placeholder,
// the same 72 bits are not Golay codewords at all and decode to a different
// payload.
func TestTCHFrameIsTheAMBESilenceFrame(t *testing.T) {
	onAir := bitsOfHex(t, ambeSilenceOnAir)

	// The schedule takes the transmitted frame to the sub-vectors.
	var sub [4][24]byte
	for i := 0; i < 36; i++ {
		sub[ambeSchedule[0][i]][ambeSchedule[1][i]] = onAir[2*i]
		sub[ambeSchedule[2][i]][ambeSchedule[3][i]] = onAir[2*i+1]
	}
	for k, want := range ambeSilenceSubVectors {
		if got := bitString(sub[k][:len(want)]); got != want {
			t.Fatalf("sub-vector C%d = %s, want %s", k, got, want)
		}
	}

	payload, corrected, err := DecodeTCHFrame(onAir)
	if err != nil {
		t.Fatal(err)
	}
	if corrected != 0 {
		t.Errorf("silence frame needed %d Golay corrections, want 0", corrected)
	}
	if got := bitString(payload); got != ambeSilencePayload {
		t.Fatalf("payload %s\nwant    %s", got, ambeSilencePayload)
	}
	b0 := 0
	for _, i := range []int{0, 1, 2, 3, 37, 38, 39} {
		b0 = b0<<1 | int(payload[i])
	}
	if b0 != 124 {
		t.Errorf("b0 = %d, want 124 (AMBE+2 silence)", b0)
	}

	enc, err := EncodeTCHFrame(payload)
	if err != nil {
		t.Fatal(err)
	}
	if bitString(enc) != bitString(onAir) {
		t.Errorf("re-encoded silence frame differs from the transmitted one")
	}
}

// TestTCHFrameRoundTrip checks that Encode then Decode reproduces the
// 49-bit input. It is self-consistent and cannot pin the layout;
// TestTCHFrameIsTheAMBESilenceFrame does.
func TestTCHFrameRoundTrip(t *testing.T) {
	for seed := uint32(0); seed < 128; seed++ {
		info := mkTCHInfo(seed)
		frame, err := EncodeTCHFrame(info)
		if err != nil {
			t.Fatalf("EncodeTCHFrame(seed=%d): %v", seed, err)
		}
		if len(frame) != dpmrAMBEOnAirBits {
			t.Fatalf("encoded frame = %d bits, want %d", len(frame), dpmrAMBEOnAirBits)
		}
		got, errs, err := DecodeTCHFrame(frame)
		if err != nil {
			t.Fatalf("DecodeTCHFrame(seed=%d): %v", seed, err)
		}
		if errs != 0 {
			t.Fatalf("seed=%d: clean frame needed %d corrections", seed, errs)
		}
		if bitString(got) != bitString(info) {
			t.Fatalf("seed=%d: payload changed in the round trip", seed)
		}
	}
}

// TestTCHFrameCorrectsErrors flips three bits in each of the two Golay
// codewords. In the schedule the high bits of the even dibits 0..22 belong
// to C0 and those of the odd dibits 1..21 to C1.
func TestTCHFrameCorrectsErrors(t *testing.T) {
	info := mkTCHInfo(42)
	frame, err := EncodeTCHFrame(info)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	corrupt := append([]byte(nil), frame...)
	for _, dibit := range []int{0, 4, 8, 1, 5, 9} {
		corrupt[2*dibit] ^= 1
	}
	got, errs, err := DecodeTCHFrame(corrupt)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The count is of corrected data bits, as mbelib reports it, so it is
	// at most the six flipped and at least one.
	if errs < 1 || errs > 6 {
		t.Errorf("corrected %d bits, want 1..6", errs)
	}
	if bitString(got) != bitString(info) {
		t.Fatalf("payload not recovered")
	}
}

func TestDecodeTCHFrameRejectsBadLength(t *testing.T) {
	if _, _, err := DecodeTCHFrame(make([]byte, 71)); err == nil {
		t.Error("expected error for 71-bit frame")
	}
	if _, err := EncodeTCHFrame(make([]byte, 48)); err == nil {
		t.Error("expected error for 48-bit payload")
	}
}
