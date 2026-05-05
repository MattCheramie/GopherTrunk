package tier3

import "testing"

func TestCSBKAssembleParseRoundTrip(t *testing.T) {
	in := CSBK{
		LB:     true,
		PF:     false,
		Opcode: OpTVGrant,
		FID:    0x00,
		Payload: [8]byte{0x80, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF, 0x00},
	}
	bytes := AssembleCSBK(in)
	if len(bytes) != 12 {
		t.Fatalf("AssembleCSBK = %d bytes, want 12", len(bytes))
	}
	out, err := ParseCSBK(bytes)
	if err != nil {
		t.Fatalf("ParseCSBK: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestCSBKDetectsCRCError(t *testing.T) {
	in := CSBK{LB: true, Opcode: OpAloha, FID: 0x00, Payload: [8]byte{1, 2, 3, 4, 5, 6, 7, 8}}
	b := AssembleCSBK(in)
	b[5] ^= 0x10
	_, err := ParseCSBK(b)
	if err != CRCError {
		t.Errorf("err = %v, want CRCError", err)
	}
}

func TestParseTVGrant(t *testing.T) {
	p := [8]byte{0x80, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF, 0x00}
	g := ParseTVGrant(p)
	if g.ServiceOptions != 0x80 {
		t.Errorf("ServiceOptions = %02X", g.ServiceOptions)
	}
	if g.GroupAddress != 0x123456 {
		t.Errorf("Group = %06X, want 123456", g.GroupAddress)
	}
	if g.SourceID != 0xABCDEF {
		t.Errorf("Source = %06X, want ABCDEF", g.SourceID)
	}
}

func TestParseSystemInfoBroadcast(t *testing.T) {
	p := [8]byte{0x12, 0x34, 0x05, 0x09, 0xFF, 0x00, 0xDE, 0xAD}
	si := ParseSystemInfoBroadcast(p)
	if si.SystemID != 0x1234 {
		t.Errorf("SystemID = %04X", si.SystemID)
	}
	if si.RFSSID != 0x05 || si.SiteID != 0x09 {
		t.Errorf("RFSS/Site = %X/%X", si.RFSSID, si.SiteID)
	}
	if si.NetMask != 0xFF {
		t.Errorf("NetMask = %02X", si.NetMask)
	}
}

func TestInfoBitsToBytes(t *testing.T) {
	bits := make([]byte, 96)
	bits[0] = 1   // → byte 0 bit 7
	bits[7] = 1   // → byte 0 bit 0
	bits[8] = 1   // → byte 1 bit 7
	bits[95] = 1  // → byte 11 bit 0
	got := InfoBitsToBytes(bits)
	want := [12]byte{0x81, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("byte %d = %02X, want %02X", i, got[i], want[i])
		}
	}
}

func TestOpcodeString(t *testing.T) {
	cases := map[CSBKOpcode]string{
		OpAloha:     "Aloha",
		OpTVGrant:   "TalkGroupVoiceChannelGrant",
		OpSysInfo:   "SystemInfoBroadcast",
		OpAdjStatus: "AdjacentSiteStatus",
		CSBKOpcode(0xFE): "CSBKOpcode(FE)",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("Opcode(%02X).String() = %s, want %s", uint8(op), got, want)
		}
	}
}
