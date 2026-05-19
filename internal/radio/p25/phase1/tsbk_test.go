package phase1

import (
	"testing"
)

func TestTSBKAssembleParseRoundTrip(t *testing.T) {
	in := TSBK{
		LB:     true,
		P:      false,
		Opcode: OpGroupVoiceChannelGrant,
		MFID:   0x00,
		Payload: [8]byte{
			0x80, 0x10, 0x05, 0x12, 0x34, 0xAB, 0xCD, 0xEF,
		},
	}
	bytes := AssembleTSBK(in)
	if len(bytes) != 12 {
		t.Fatalf("AssembleTSBK = %d bytes, want 12", len(bytes))
	}
	out, err := ParseTSBK(bytes)
	if err != nil {
		t.Fatalf("ParseTSBK: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestTSBKDetectsCRCError(t *testing.T) {
	in := TSBK{LB: true, Opcode: OpRFSSStatusBroadcast}
	b := AssembleTSBK(in)
	b[5] ^= 0x01
	_, err := ParseTSBK(b)
	if err != CRCError {
		t.Errorf("expected CRCError, got %v", err)
	}
}

func TestParseGroupVoiceChannelGrant(t *testing.T) {
	p := [8]byte{0x80, 0x40, 0x05, 0x12, 0x34, 0xAB, 0xCD, 0xEF}
	g := ParseGroupVoiceChannelGrant(p)
	if g.ServiceOptions != 0x80 {
		t.Errorf("ServiceOptions = %02X", g.ServiceOptions)
	}
	if g.ChannelID != 0x4 || g.ChannelNumber != 0x005 {
		t.Errorf("Channel = %X.%03X, want 4.005", g.ChannelID, g.ChannelNumber)
	}
	if g.GroupAddress != 0x1234 {
		t.Errorf("Group = %04X, want 1234", g.GroupAddress)
	}
	if g.SourceID != 0xABCDEF {
		t.Errorf("Source = %06X, want ABCDEF", g.SourceID)
	}
}

func TestParseGroupAffiliationResponse(t *testing.T) {
	// Response = 0x2 (denied), AnnGroup = 0xAABB, Group = 0x1234,
	// Target = 0xABCDEF. Top 6 bits of byte 0 are reserved; we set
	// them to 1s to confirm the parser masks them off.
	p := [8]byte{0xFE, 0xAA, 0xBB, 0x12, 0x34, 0xAB, 0xCD, 0xEF}
	a := ParseGroupAffiliationResponse(p)
	if a.Response != 0x2 {
		t.Errorf("Response = %X, want 2", a.Response)
	}
	if a.AnnouncementGroup != 0xAABB {
		t.Errorf("AnnouncementGroup = %04X, want AABB", a.AnnouncementGroup)
	}
	if a.GroupAddress != 0x1234 {
		t.Errorf("GroupAddress = %04X, want 1234", a.GroupAddress)
	}
	if a.TargetID != 0xABCDEF {
		t.Errorf("TargetID = %06X, want ABCDEF", a.TargetID)
	}
}

func TestParseUnitRegistrationResponse(t *testing.T) {
	// Response = 0x0 (accepted). WACN = 0xBEE08 (20-bit) packed as
	// byte1 = 0xBE, byte2 = 0xE0, top nibble of byte3 = 0x8.
	// SystemID = 0x534 (12-bit) packed as bottom nibble of byte3 = 0x5,
	// byte4 = 0x34. Source = 0x112233.
	p := [8]byte{0x00, 0xBE, 0xE0, 0x85, 0x34, 0x11, 0x22, 0x33}
	u := ParseUnitRegistrationResponse(p)
	if u.Response != 0x0 {
		t.Errorf("Response = %X, want 0", u.Response)
	}
	if u.WACN != 0xBEE08 {
		t.Errorf("WACN = %05X, want BEE08", u.WACN)
	}
	if u.SystemID != 0x534 {
		t.Errorf("SystemID = %03X, want 534", u.SystemID)
	}
	if u.SourceID != 0x112233 {
		t.Errorf("SourceID = %06X, want 112233", u.SourceID)
	}
}

func TestParseNetworkStatusBroadcast(t *testing.T) {
	// WACN = 0xABCDE (20-bit), SystemID = 0x123 (12-bit).
	p := [8]byte{0xAB, 0xCD, 0xE1, 0x23, 0x40, 0x10, 0x00, 0x42}
	n := ParseNetworkStatusBroadcast(p)
	if n.WACN != 0xABCDE {
		t.Errorf("WACN = %05X, want ABCDE", n.WACN)
	}
	if n.SystemID != 0x123 {
		t.Errorf("SystemID = %03X, want 123", n.SystemID)
	}
	if n.ChannelID != 0x4 || n.ChannelNumber != 0x010 {
		t.Errorf("Channel = %X.%03X", n.ChannelID, n.ChannelNumber)
	}
}

func TestOpcodeString(t *testing.T) {
	cases := map[Opcode]string{
		OpGroupVoiceChannelGrant: "GroupVoiceChannelGrant",
		OpRFSSStatusBroadcast:    "RFSSStatusBroadcast",
		OpNetworkStatusBroadcast: "NetworkStatusBroadcast",
		Opcode(0x42):             "Opcode(42)",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("Opcode(%02X).String() = %s, want %s", uint8(op), got, want)
		}
	}
}
