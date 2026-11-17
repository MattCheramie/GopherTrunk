package dpmr

import (
	"encoding/hex"
	"testing"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
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

// TestTCHFrameRoundTrip checks that Encode→Decode over many payloads
// reproduces the exact 49-bit input. It is self-consistent: it passed
// while both sides used the wrong Golay code (#1374), so it cannot pin
// the FEC to the codec. TestTCHFrameDecodesDMRSilenceWithMbelibGolay does.
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
		got, _, err := DecodeTCHFrame(frame)
		if err != nil {
			t.Fatalf("DecodeTCHFrame(seed=%d): %v", seed, err)
		}
		for i := range info {
			if got[i] != info[i] {
				t.Fatalf("seed=%d bit %d: got %d want %d", seed, i, got[i], info[i])
			}
		}
	}
}

// TestTCHFrameCorrectsErrors confirms the Golay(23,12) layer corrects up
// to 3 bit errors injected into each of the C0 and C1 sub-vectors (the
// FEC-protected regions), recovering the exact payload.
func TestTCHFrameCorrectsErrors(t *testing.T) {
	info := mkTCHInfo(42)
	frame, err := EncodeTCHFrame(info)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// C0 occupies on-air positions 0..23, C1 24..46 (sequential
	// placeholder layout). Flip 3 bits in C0's codeword and 3 in C1's.
	corrupt := append([]byte(nil), frame...)
	for _, pos := range []int{1, 5, 12, 24, 30, 40} {
		corrupt[pos] ^= 1
	}
	got, errs, err := DecodeTCHFrame(corrupt)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errs == 0 {
		t.Errorf("expected corrected-error count > 0, got 0")
	}
	for i := range info {
		if got[i] != info[i] {
			t.Fatalf("bit %d not corrected: got %d want %d", i, got[i], info[i])
		}
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

// dmrSilenceSubVectors are the C0..C3 sub-vectors of MMDVMHost's
// DMR_SILENCE AMBE+2 frame (on-air bytes B9 E8 81 52 61 73 00 2A 6B),
// deinterleaved with DSD's DMR rW/rX/rY/rZ schedule: cell j of each
// string is fr[k][j]. They are codec-level vectors, the same for every
// protocol that carries AMBE+2 3600×2450, so they are exact mbelib
// Golay(23,12) codewords whatever the on-air interleave.
var dmrSilenceSubVectors = [4]string{
	"000100101000000000011111",
	"10110000101010000010101",
	"00111111001",
	"10000011100110",
}

// TestTCHFrameDecodesDMRSilenceWithMbelibGolay pins dPMR's AMBE+2 FEC to
// mbelib's. Until this test, DecodeTCHFrame and EncodeTCHFrame used
// framing.GolayDecode23_12 / GolayEncode23_12, a different (23,12) Golay
// code: the round trip above passed because both sides shared it, while a
// real AMBE+2 frame decoded to the wrong vocoder parameters. Laid out with
// dPMR's own interleave, the silence frame's sub-vectors must decode to the
// same 49 bits the DMR decoder (bit-identical to mbelib) gives for
// DMR_SILENCE, with no Golay correction, and re-encode bit for bit.
func TestTCHFrameDecodesDMRSilenceWithMbelibGolay(t *testing.T) {
	dmrSilence, _ := hex.DecodeString("b9e88152617300" + "2a6b")
	dmrBits := make([]byte, 72)
	for i := range dmrBits {
		dmrBits[i] = (dmrSilence[i/8] >> uint(7-i%8)) & 1
	}
	want, errs, err := dmrvoice.DecodeAMBEFrame(dmrBits)
	if err != nil || errs != 0 {
		t.Fatalf("DMR silence decode: errs=%d err=%v", errs, err)
	}

	var fr [4][24]uint8
	for k, s := range dmrSilenceSubVectors {
		for j, c := range s {
			fr[k][j] = uint8(c - '0')
		}
	}
	frame := dpmrAMBEInterleave(fr)

	got, corrected, err := DecodeTCHFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if corrected != 0 {
		t.Errorf("silence frame needed %d Golay corrections, want 0", corrected)
	}
	if string(got) != string(want) {
		t.Fatalf("payload %v\nwant (DMR silence) %v", got, want)
	}

	enc, err := EncodeTCHFrame(want)
	if err != nil {
		t.Fatal(err)
	}
	for i := range frame {
		if enc[i] != frame[i] {
			t.Fatalf("re-encoded silence frame differs at on-air bit %d", i)
		}
	}
}
