package tier3

import "testing"

func TestCSBKAssembleParseRoundTrip(t *testing.T) {
	in := CSBK{
		LB:      true,
		PF:      false,
		Opcode:  OpTVGrant,
		FID:     0x00,
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
	// LPCN 0x015 (21) in payload bits 0-11, TS2 (bit 12 set), group
	// 0x123456 at octets 2-4, source 0xABCDEF at octets 5-7.
	//   p[0] = 0x015 >> 4              = 0x01
	//   p[1] = (0x015 & 0xF)<<4 | 1<<3 = 0x58
	p := [8]byte{0x01, 0x58, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF}
	g := ParseTVGrant(p)
	if g.GroupAddress != 0x123456 {
		t.Errorf("Group = %06X, want 123456", g.GroupAddress)
	}
	if g.SourceID != 0xABCDEF {
		t.Errorf("Source = %06X, want ABCDEF", g.SourceID)
	}
	if g.LCN != 0x015 {
		t.Errorf("LCN = %d, want 21 (0x015)", g.LCN)
	}
	if g.Timeslot != 1 {
		t.Errorf("Timeslot = %d, want 1 (TS2)", g.Timeslot)
	}
}

func TestParsePVGrant(t *testing.T) {
	// LPCN 5, TS1, dst 0x001234, src 0x005678.
	//   p[0] = 5 >> 4         = 0x00
	//   p[1] = (5 & 0xF) << 4 = 0x50
	p := [8]byte{0x00, 0x50, 0x00, 0x12, 0x34, 0x00, 0x56, 0x78}
	g := ParsePVGrant(p)
	if g.DestinationID != 0x001234 || g.SourceID != 0x005678 {
		t.Errorf("dst/src = %X/%X", g.DestinationID, g.SourceID)
	}
	if g.LCN != 5 || g.Timeslot != 0 {
		t.Errorf("LCN/TS = %d/%d, want 5/0", g.LCN, g.Timeslot)
	}
}

// TestParseTVGrantLCNIndependentOfSource is the regression guard for
// issue #639: the LPCN lives in the leading payload bits, so two grants
// for the same channel but different transmitting radios (different
// source addresses) must decode to the SAME LCN. The pre-fix code read
// the LCN from the source address's low byte, so it changed on every
// transmission.
func TestParseTVGrantLCNIndependentOfSource(t *testing.T) {
	// LPCN 2, TS1, group 0x000064. Only the 24-bit source differs.
	mk := func(source uint32) [8]byte {
		return [8]byte{
			0x00, 0x20, // LPCN 0x002, TS1
			0x00, 0x00, 0x64,
			byte(source >> 16), byte(source >> 8), byte(source),
		}
	}
	g1 := ParseTVGrant(mk(0x111111))
	g2 := ParseTVGrant(mk(0x222222))
	if g1.LCN != 2 || g2.LCN != 2 {
		t.Fatalf("LCN leaked from source: g1=%d g2=%d, want both 2", g1.LCN, g2.LCN)
	}
	if g1.SourceID == g2.SourceID {
		t.Fatal("test bug: sources should differ")
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
	bits[0] = 1  // → byte 0 bit 7
	bits[7] = 1  // → byte 0 bit 0
	bits[8] = 1  // → byte 1 bit 7
	bits[95] = 1 // → byte 11 bit 0
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
		OpAloha:          "C_ALOHA",
		OpAhoy:           "C_AHOY",
		OpAckD:           "C_ACKD",
		OpBcast:          "C_BCAST",
		OpTVGrant:        "TV_GRANT",
		OpPVGrant:        "PV_GRANT",
		OpPreamble:       "Preamble",
		CSBKOpcode(0xFE): "CSBKOpcode(FE)",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("Opcode(%02X).String() = %s, want %s", uint8(op), got, want)
		}
	}
}
