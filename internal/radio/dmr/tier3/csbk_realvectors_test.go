package tier3

import "testing"

// TestParseCSBKRealOffAirVectors pins the CSBK CRC convention (CRC-CCITT
// poly 0x1021, init 0x0000, XOR-out mask 0x5A5A) and the C_ALOHA opcode
// against real off-air ETSI Tier III control-channel bursts.
//
// These 12-byte info blocks were recovered from a 2 MS/s SDR capture of
// a live DMR Tier III TSCC (440.5625 MHz): they decode cleanly through
// BPTC(196,96) — zero corrections — and repeat identically across the
// capture, so they are genuine CSBKs, not noise. Under the previous
// "init 0xFFFF, store the bitwise complement" convention every one of
// them failed CRC (the control channel never locked); they validate
// under the 0x5A5A mask. Keeping them here as a fixture means a future
// regression of the CRC mask or the Aloha opcode trips this test offline
// without needing the RF capture.
func TestParseCSBKRealOffAirVectors(t *testing.T) {
	cases := []struct {
		name   string
		bytes  []byte
		opcode CSBKOpcode
	}{
		{
			// C_ALOHA — the periodic TSCC beacon the CC state machine
			// locks on.
			name:   "aloha",
			bytes:  []byte{0x99, 0x00, 0x49, 0x01, 0x16, 0x40, 0x03, 0x00, 0x00, 0x00, 0x05, 0x9f},
			opcode: OpAloha,
		},
		{
			// Preamble CSBK — announces following content; already had
			// the correct opcode value, included so the fixture covers
			// both repeating bursts in the capture.
			name:   "preamble",
			bytes:  []byte{0xa8, 0x00, 0x08, 0x7b, 0xd6, 0x40, 0x03, 0x06, 0x00, 0x60, 0x61, 0x5b},
			opcode: OpPreamble,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csbk, err := ParseCSBK(tc.bytes)
			if err != nil {
				t.Fatalf("ParseCSBK rejected a real off-air %s CSBK: %v (CRC mask regressed?)", tc.name, err)
			}
			if csbk.Opcode != tc.opcode {
				t.Errorf("opcode = %#02x (%s), want %#02x", uint8(csbk.Opcode), csbk.Opcode, uint8(tc.opcode))
			}
			if csbk.FID != 0x00 {
				t.Errorf("FID = %#02x, want 0x00 (standard ETSI)", csbk.FID)
			}
		})
	}
}
