package tier3

import "testing"

// TestParseCSBKRealOffAirVectors pins the CSBK CRC convention (CRC-CCITT
// poly 0x1021, init 0x0000, XOR-out mask 0x5A5A) and the standard CSBKO
// opcode values against real off-air ETSI Tier III control-channel bursts.
//
// These 12-byte info blocks were recovered from 2 MS/s SDR captures of
// live DMR Tier III TSCCs (440.5625 MHz and 440.2625 MHz): they decode
// cleanly through BPTC(196,96) — zero corrections — and repeat identically
// across the capture, so they are genuine CSBKs, not noise. Under the
// previous "init 0xFFFF, store the bitwise complement" convention every
// one of them failed CRC (the control channel never locked); they validate
// under the 0x5A5A mask. The opcode bytes also pin the corrected opcode
// table: 0x28 is C_BCAST (was mislabelled "Preamble"), 0x1C is C_AHOY,
// 0x20 is C_ACKD. A regression of the CRC mask or the opcode constants
// trips this test offline without needing the RF capture.
func TestParseCSBKRealOffAirVectors(t *testing.T) {
	cases := []struct {
		name   string
		bytes  []byte
		opcode CSBKOpcode
	}{
		{
			// C_ALOHA — the periodic TSCC beacon the CC state machine
			// locks on (440.5625 MHz capture).
			name:   "aloha",
			bytes:  []byte{0x99, 0x00, 0x49, 0x01, 0x16, 0x40, 0x03, 0x00, 0x00, 0x00, 0x05, 0x9f},
			opcode: OpAloha,
		},
		{
			// C_BCAST (Gen_Site_Params) — previously decoded as the
			// non-existent "Preamble" opcode because 0x28 was wrongly
			// mapped. dsd-neo decodes the identical burst as
			// "Announcements (C_BCAST) General Site Parameters".
			name:   "bcast_gen_site",
			bytes:  []byte{0xa8, 0x00, 0x3a, 0x00, 0xb1, 0x40, 0x15, 0x00, 0x00, 0x80, 0x83, 0x86},
			opcode: OpBcast,
		},
		{
			// C_BCAST (CallTimer_Parms) — same 0x28 opcode, different
			// anncd_type sub-field.
			name:   "bcast_call_timer",
			bytes:  []byte{0xa8, 0x00, 0x0f, 0xff, 0xf1, 0x40, 0x15, 0xff, 0xff, 0xff, 0x01, 0x61},
			opcode: OpBcast,
		},
		{
			// C_AHOY — outbound poll; previously surfaced as the unnamed
			// CSBKOpcode(1C) because OpAhoy was wrongly 0x18.
			name:   "ahoy",
			bytes:  []byte{0x9c, 0x00, 0x00, 0x0e, 0x00, 0x02, 0xd0, 0xff, 0xfe, 0xca, 0x94, 0x25},
			opcode: OpAhoy,
		},
		{
			// C_ACKD — acknowledged downlink response; previously the
			// unnamed CSBKOpcode(20).
			name:   "ackd",
			bytes:  []byte{0xa0, 0x00, 0x00, 0xc4, 0x00, 0x03, 0x45, 0xff, 0xfe, 0xc6, 0x4a, 0x9c},
			opcode: OpAckD,
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
