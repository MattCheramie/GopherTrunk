package motorola

import "testing"

// TestOutboundSyncBitsMatchReference pins the sync word against the
// OP25 literal (frame_sync_magics.h: SMARTNET_SYNC_MAGIC = 0xAC,
// 8 bits) — the only thing that can catch constant drift, per the
// SoapyRemote-opcode lesson.
func TestOutboundSyncBitsMatchReference(t *testing.T) {
	want := []byte{1, 0, 1, 0, 1, 1, 0, 0}
	got := OutboundSyncBits()
	if len(got) != len(want) {
		t.Fatalf("sync length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sync bit %d = %d, want %d (0xAC MSB-first)", i, got[i], want[i])
		}
	}
}

// TestDeinterleavePermutationMatchesReference pins the interleave
// against OP25's documented shuffle: incoming wire bits {1,2,3,…}
// deinterleave to {1,20,39,58,2,21,40,59,…} (1-based) — i.e.
// deinterleaved position k*4+l reads wire position k + l*19.
func TestDeinterleavePermutationMatchesReference(t *testing.T) {
	wire := make([]byte, PayloadBits)
	// Tag each wire position with its own parity-of-index pattern via
	// a position-unique marker: set exactly one bit and find it.
	for probe := 0; probe < PayloadBits; probe++ {
		for i := range wire {
			wire[i] = 0
		}
		wire[probe] = 1
		out := deinterleave76(nil, wire)
		k, l := probe%19, probe/19
		wantIdx := k*4 + l
		for i, b := range out {
			if (b == 1) != (i == wantIdx) {
				t.Fatalf("wire bit %d landed at %d, want %d", probe, i, wantIdx)
			}
		}
	}
}

// TestXORMasksMatchReference pins the field-inversion masks against
// the OP25 literals: ID_XOR 0x33C7, CMD_XOR 0x32A, applied as their
// complements to the (inverted) wire fields.
func TestXORMasksMatchReference(t *testing.T) {
	if idXORMask != 0xCC38 {
		t.Errorf("idXORMask = %#x, want 0xCC38 (~0x33C7)", idXORMask)
	}
	if cmdXORMask != 0x0D5 {
		t.Errorf("cmdXORMask = %#x, want 0x0D5 (~0x32A & 0x3ff)", cmdXORMask)
	}
}

func TestOSWFrameRoundTrip(t *testing.T) {
	cases := []OSW{
		{Address: 0x4567, Group: false, Command: CmdFirstNormal},
		{Address: 0x1F00, Group: false, Command: 0x8E},
		{Address: 0xB010, Group: true, Command: 0x2A},
		{Address: 0x0000, Group: true, Command: 0x000},
		{Address: 0xFFFF, Group: false, Command: 0x3FF},
	}
	for _, in := range cases {
		frame := EncodeOSWFrame(in)
		if len(frame) != FrameBits {
			t.Fatalf("frame length = %d, want %d", len(frame), FrameBits)
		}
		got, ok := DecodeOSWPayload(frame[SyncBits:])
		if !ok {
			t.Fatalf("CRC rejected round-trip of %+v", in)
		}
		if got != in {
			t.Errorf("round-trip = %+v, want %+v", got, in)
		}
	}
}

// TestECCCorrectsSingleInfoBitError flips one info-position wire bit;
// the doubled-parity syndrome must correct it and the CRC pass.
func TestECCCorrectsSingleInfoBitError(t *testing.T) {
	in := OSW{Address: 0xB010, Group: true, Command: 0x2A}
	frame := EncodeOSWFrame(in)
	// Interleaved position of info pair k=5 (wire index for seq
	// position 2k=10: k*4+l = 10 → k=2,l=2 → wire 2+2*19 = 40).
	payload := append([]byte(nil), frame[SyncBits:]...)
	payload[40] ^= 1
	got, flips, ok := DecodeOSWPayloadDetail(payload)
	if !ok {
		t.Fatal("CRC rejected a single-info-bit error the ECC should correct")
	}
	if flips != 1 {
		t.Errorf("flips = %d, want 1", flips)
	}
	if got != in {
		t.Errorf("corrected decode = %+v, want %+v", got, in)
	}
}

// TestCRCRejectsCorruptPayload flips a parity bit and an info bit in
// neighbouring pairs — a pattern the pairwise syndrome miscorrects —
// and expects a CRC reject rather than a wrong OSW.
func TestCRCRejectsCorruptPayload(t *testing.T) {
	in := OSW{Address: 0xB010, Group: true, Command: 0x2A}
	frame := EncodeOSWFrame(in)
	payload := append([]byte(nil), frame[SyncBits:]...)
	payload[21] ^= 1 // seq position 9: pair 4's parity
	payload[40] ^= 1 // seq position 10: pair 5's info
	if osw, ok := DecodeOSWPayload(payload); ok {
		t.Fatalf("CRC accepted a two-bit corrupt payload as %+v", osw)
	}
}
