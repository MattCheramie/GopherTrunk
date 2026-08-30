package dstar

import (
	"testing"
)

// mkDVInfo builds a deterministic 49-bit AMBE payload from a seed (a
// simple LCG), mirroring the DMR / NXDN / dPMR test helpers.
func mkDVInfo(seed uint32) []byte {
	out := make([]byte, dstarAMBEInfoBits)
	x := seed | 1
	for i := range out {
		x = 1664525*x + 1013904223
		out[i] = byte((x >> 24) & 1)
	}
	return out
}

// TestDVVoiceBitsRoundTrip pins the AMBE FEC machinery: Encode→Decode
// over many payloads must reproduce the exact 49-bit input. This
// validates the Golay(23,12) coding, the C0-seeded C1 descramble, and
// the C0/C1/C2/C3 assembly — everything except the (capture-unknown)
// on-air interleave table, which round-trips by construction here.
func TestDVVoiceBitsRoundTrip(t *testing.T) {
	for seed := uint32(0); seed < 128; seed++ {
		info := mkDVInfo(seed)
		frame, err := EncodeDVVoiceBits(info)
		if err != nil {
			t.Fatalf("EncodeDVVoiceBits(seed=%d): %v", seed, err)
		}
		if len(frame) != dstarAMBEOnAirBits {
			t.Fatalf("encoded frame = %d bits, want %d", len(frame), dstarAMBEOnAirBits)
		}
		got, _, err := DecodeDVVoiceBits(frame)
		if err != nil {
			t.Fatalf("DecodeDVVoiceBits(seed=%d): %v", seed, err)
		}
		for i := range info {
			if got[i] != info[i] {
				t.Fatalf("seed=%d bit %d: got %d want %d", seed, i, got[i], info[i])
			}
		}
	}
}

// TestDVVoiceBitsCorrectsErrors confirms the Golay(23,12) layer
// corrects up to 3 bit errors injected into each of the C0 and C1
// sub-vectors (the FEC-protected regions), recovering the exact
// payload.
func TestDVVoiceBitsCorrectsErrors(t *testing.T) {
	info := mkDVInfo(42)
	frame, err := EncodeDVVoiceBits(info)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// C0 occupies on-air positions 0..23, C1 24..46 (sequential
	// placeholder layout). Flip 3 bits in C0's codeword and 3 in C1's.
	corrupt := append([]byte(nil), frame...)
	for _, pos := range []int{1, 5, 12, 24, 30, 40} {
		corrupt[pos] ^= 1
	}
	got, errs, err := DecodeDVVoiceBits(corrupt)
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

func TestDecodeDVVoiceBitsRejectsBadLength(t *testing.T) {
	if _, _, err := DecodeDVVoiceBits(make([]byte, 71)); err == nil {
		t.Error("expected error for 71-bit frame")
	}
	if _, err := EncodeDVVoiceBits(make([]byte, 48)); err == nil {
		t.Error("expected error for 48-bit payload")
	}
}
