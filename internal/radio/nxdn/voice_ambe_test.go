package nxdn

import (
	"testing"
)

// mkInfo builds a deterministic 49-bit AMBE payload from a seed (a
// simple LCG), mirroring the DMR ambefec_test helper.
func mkInfo(seed uint32) []byte {
	out := make([]byte, nxdnAMBEInfoBits)
	x := seed | 1
	for i := range out {
		x = 1664525*x + 1013904223
		out[i] = byte((x >> 24) & 1)
	}
	return out
}

// TestVCHFrameRoundTrip pins the AMBE FEC machinery: Encode→Decode over
// many payloads must reproduce the exact 49-bit input. This validates
// the Golay(23,12) coding, the C0-seeded C1 descramble, and the
// C0/C1/C2/C3 assembly — everything except the (capture-unknown) on-air
// interleave table, which round-trips by construction here.
func TestVCHFrameRoundTrip(t *testing.T) {
	for seed := uint32(0); seed < 128; seed++ {
		info := mkInfo(seed)
		frame, err := EncodeVCHFrame(info)
		if err != nil {
			t.Fatalf("EncodeVCHFrame(seed=%d): %v", seed, err)
		}
		if len(frame) != nxdnAMBEOnAirBits {
			t.Fatalf("encoded frame = %d bits, want %d", len(frame), nxdnAMBEOnAirBits)
		}
		got, _, err := DecodeVCHFrame(frame)
		if err != nil {
			t.Fatalf("DecodeVCHFrame(seed=%d): %v", seed, err)
		}
		for i := range info {
			if got[i] != info[i] {
				t.Fatalf("seed=%d bit %d: got %d want %d", seed, i, got[i], info[i])
			}
		}
	}
}

// TestVCHFrameCorrectsErrors confirms the Golay(23,12) layer corrects up
// to 3 bit errors injected into each of the C0 and C1 sub-vectors (the
// FEC-protected regions), recovering the exact payload.
func TestVCHFrameCorrectsErrors(t *testing.T) {
	info := mkInfo(42)
	frame, err := EncodeVCHFrame(info)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// C0 occupies on-air positions 0..23, C1 24..46 (sequential
	// placeholder layout). Flip 3 bits in C0's codeword and 3 in C1's.
	corrupt := append([]byte(nil), frame...)
	for _, pos := range []int{1, 5, 12, 24, 30, 40} {
		corrupt[pos] ^= 1
	}
	got, errs, err := DecodeVCHFrame(corrupt)
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

func TestDecodeVCHFrameRejectsBadLength(t *testing.T) {
	if _, _, err := DecodeVCHFrame(make([]byte, 71)); err == nil {
		t.Error("expected error for 71-bit frame")
	}
	if _, err := EncodeVCHFrame(make([]byte, 48)); err == nil {
		t.Error("expected error for 48-bit payload")
	}
}
