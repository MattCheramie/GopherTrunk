package phase1

import "testing"

// TestAMBTOpcodeLabel pins the AMBT log-naming rule: only the decoded
// standard broadcasts get a mnemonic; every other opcode — and any vendor
// MFID — renders numerically so an AMBT/ISP opcode is never mislabeled with
// a TSBK OSP name that does not apply.
func TestAMBTOpcodeLabel(t *testing.T) {
	cases := []struct {
		h    MBTHeader
		want string
	}{
		{MBTHeader{Opcode: OpNetworkStatusBroadcast}, "NET_STS_BCST"},
		{MBTHeader{Opcode: OpRFSSStatusBroadcast}, "RFSS_STS_BCST"},
		{MBTHeader{Opcode: OpAdjacentSiteStatusBroadcast}, "ADJ_STS_BCST"},
		// An AMBT opcode GT does not decode must NOT borrow a TSBK OSP name
		// (0x00 would otherwise read GRP_V_CH_GRANT).
		{MBTHeader{Opcode: 0x00}, "AMBT(0x00)"},
		{MBTHeader{Opcode: 0x05}, "AMBT(0x05)"},
		// A vendor MFID makes even a "known" opcode value vendor-defined.
		{MBTHeader{MFID: 0x90, Opcode: OpNetworkStatusBroadcast}, "AMBT(0x3B)"},
	}
	for _, tc := range cases {
		if got := ambtOpcodeLabel(tc.h); got != tc.want {
			t.Errorf("ambtOpcodeLabel(mfid=0x%02X op=0x%02X) = %q, want %q",
				tc.h.MFID, uint8(tc.h.Opcode), got, tc.want)
		}
	}
}
