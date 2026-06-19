package phase1

import (
	"testing"

	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

// TestStatusBroadcastLayoutMatchesPhase2 feeds the same argument bytes to
// the Phase 1 status-broadcast parsers and the independent Phase 2
// decoders (phase2/mac.go, phase2/mac_standard.go) and asserts they agree
// on WACN / System ID / RFSS / Site. The two share the TIA-102.AABF
// layout; this guards against the two implementations drifting apart again
// (the Phase 1 side previously omitted the leading LRA byte).
func TestStatusBroadcastLayoutMatchesPhase2(t *testing.T) {
	// Network Status Broadcast: LRA 0x07, WACN 0xABCDE, SystemID 0x123,
	// channel 4.010. Phase 2 carries a 12-bit Color Code where Phase 1
	// carries the service class, so compare only the shared identity fields.
	nsb := [8]byte{0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x40, 0x10, 0x42}
	p1n := ParseNetworkStatusBroadcast(nsb)
	p2n, ok := p25phase2.MACPDU{
		Opcode:  p25phase2.OpNetworkStatusBroadcastUpdate,
		Payload: append(nsb[:], 0x00), // Phase 2 NSB-Update is one byte longer
	}.AsNetworkStatusBroadcast()
	if !ok {
		t.Fatal("phase2 AsNetworkStatusBroadcast returned !ok")
	}
	if p1n.LRA != p2n.LRA || p1n.WACN != p2n.WACN || p1n.SystemID != p2n.SystemID {
		t.Errorf("NSB phase1=%+v disagrees with phase2 LRA/WACN/SystemID %d/%05X/%03X",
			p1n, p2n.LRA, p2n.WACN, p2n.SystemID)
	}

	// RFSS Status Broadcast: LRA 0x09, SystemID 0x123, RFSS 4, Site 7.
	rfss := [8]byte{0x09, 0x01, 0x23, 0x04, 0x07, 0x40, 0x10, 0x42}
	p1r := ParseRFSSStatusBroadcast(rfss)
	p2r, ok := p25phase2.MACPDU{
		Opcode:  p25phase2.OpRFSSStatusBroadcastUpdate,
		Payload: rfss[:],
	}.AsRFSSStatusBroadcast()
	if !ok {
		t.Fatal("phase2 AsRFSSStatusBroadcast returned !ok")
	}
	if p1r.LRA != p2r.LRA || p1r.SystemID != p2r.SystemID || p1r.RFSS != p2r.RFSS || p1r.Site != p2r.Site {
		t.Errorf("RFSS phase1=%+v disagrees with phase2 %+v", p1r, p2r)
	}
}

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

// TestTSBKAcceptsMtAnakieOnAirVector pins the augmented-CRC convention
// the P25 TSBK trailer actually uses against three ground-truth on-air
// frames recovered from the Mt Anakie capture (issue #275). Each is a
// metric=0 clean-trellis decode; under the prior CRC-CCITT/FALSE +
// inverted-trailer implementation all three were rejected as CRC
// errors, and 195 of 197 TSBKs on the capture met the same fate. A
// regression on the CRC algorithm would un-decode them again.
func TestTSBKAcceptsMtAnakieOnAirVector(t *testing.T) {
	vectors := [][12]byte{
		{0x16, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x01, 0x5F, 0xCB},
		{0x34, 0x00, 0xE5, 0x05, 0xC0, 0x64, 0x01, 0xEE, 0x89, 0x90, 0x7F, 0xA6},
		{0x3C, 0x00, 0x00, 0x31, 0x64, 0x04, 0x2E, 0xF0, 0x65, 0x70, 0x78, 0x47},
	}
	for i, v := range vectors {
		_, err := ParseTSBK(v[:])
		if err != nil {
			t.Errorf("vector %d (% 02X): ParseTSBK returned %v, want nil", i, v[:], err)
		}
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
	// LRA = 0x07, WACN = 0xABCDE (20-bit), SystemID = 0x123 (12-bit),
	// channel = 4.010, service class = 0x42. Layout per TIA-102.AABF:
	// byte 0 LRA, bytes 1-3 WACN, bytes 3-4 SystemID, bytes 5-6 channel,
	// byte 7 service class — matching phase2/mac.go AsNetworkStatusBroadcast.
	p := [8]byte{0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x40, 0x10, 0x42}
	n := ParseNetworkStatusBroadcast(p)
	if n.LRA != 0x07 {
		t.Errorf("LRA = %02X, want 07", n.LRA)
	}
	if n.WACN != 0xABCDE {
		t.Errorf("WACN = %05X, want ABCDE", n.WACN)
	}
	if n.SystemID != 0x123 {
		t.Errorf("SystemID = %03X, want 123", n.SystemID)
	}
	if n.ChannelID != 0x4 || n.ChannelNumber != 0x010 {
		t.Errorf("Channel = %X.%03X", n.ChannelID, n.ChannelNumber)
	}
	if n.ServiceClass != 0x42 {
		t.Errorf("ServiceClass = %02X, want 42", n.ServiceClass)
	}
}

// TestParseRFSSStatusBroadcast pins the corrected byte layout (LRA, then
// 12-bit SystemID, then RFSS at byte 3 and Site at byte 4) against the
// independent Phase 2 decoder in phase2/mac_standard.go.
func TestParseRFSSStatusBroadcast(t *testing.T) {
	// LRA = 0x09, SystemID = 0x123, RFSS = 4, Site = 7, channel = 4.010.
	p := [8]byte{0x09, 0x01, 0x23, 0x04, 0x07, 0x40, 0x10, 0x42}
	r := ParseRFSSStatusBroadcast(p)
	if r.LRA != 0x09 {
		t.Errorf("LRA = %02X, want 09", r.LRA)
	}
	if r.SystemID != 0x123 {
		t.Errorf("SystemID = %03X, want 123", r.SystemID)
	}
	if r.RFSS != 4 || r.Site != 7 {
		t.Errorf("RFSS/Site = %d/%d, want 4/7", r.RFSS, r.Site)
	}
	if r.ChannelID != 0x4 || r.ChannelNumber != 0x010 {
		t.Errorf("Channel = %X.%03X, want 4.010", r.ChannelID, r.ChannelNumber)
	}
}

// TestParseAdjacentSiteStatusBroadcast checks the corrected neighbour
// layout via an Assemble/Parse round-trip plus the real on-air Mt Anakie
// frame already pinned for CRC in TestTSBKAcceptsMtAnakieOnAirVector.
func TestParseAdjacentSiteStatusBroadcast(t *testing.T) {
	in := AdjacentSiteStatusBroadcast{
		LRA: 0x0A, SystemID: 0x164, RFSS: 4, Site: 8, ChannelID: 1, ChannelNumber: 300,
	}
	out := ParseAdjacentSiteStatusBroadcast(AssembleAdjacentSiteStatusBroadcast(in))
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}

	// Mt Anakie on-air ADJ_STS_BCST payload (the 8 argument bytes of the
	// 0x3C frame in TestTSBKAcceptsMtAnakieOnAirVector). LRA 0, SystemID
	// 0x164, RFSS 4, Site 0x2E, channel 0xF.065.
	p := [8]byte{0x00, 0x31, 0x64, 0x04, 0x2E, 0xF0, 0x65, 0x70}
	a := ParseAdjacentSiteStatusBroadcast(p)
	if a.SystemID != 0x164 || a.RFSS != 4 || a.Site != 0x2E {
		t.Errorf("on-air ADJ = SystemID %03X RFSS %d Site %02X, want 164/4/2E",
			a.SystemID, a.RFSS, a.Site)
	}
	if a.ChannelID != 0xF || a.ChannelNumber != 0x065 {
		t.Errorf("on-air ADJ channel = %X.%03X, want F.065", a.ChannelID, a.ChannelNumber)
	}
}

func TestOpcodeString(t *testing.T) {
	cases := map[Opcode]string{
		OpGroupVoiceChannelGrant:  "GRP_V_CH_GRANT",
		OpUnitToUnitAnswerRequest: "UU_ANS_REQ",
		OpRFSSStatusBroadcast:     "RFSS_STS_BCST",
		OpNetworkStatusBroadcast:  "NET_STS_BCST",
		Opcode(0x42):              "OSP(0x42)",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("Opcode(%02X).String() = %s, want %s", uint8(op), got, want)
		}
	}
}
