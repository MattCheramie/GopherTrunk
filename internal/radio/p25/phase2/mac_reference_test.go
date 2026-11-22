package phase2

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// The MAC message opcodes and layouts in this file come from outside this
// package: SDRTrunk's MacOpcode enum and its MacStructure classes, whose
// field positions are bit indexes counted from the opcode's first bit, and
// OP25's mac_msg_len table for the message lengths. macField / macMessage
// build a message from those bit indexes directly, so nothing here shares
// code with the parsers under test. One vector is real air: the Motorola
// group regroup an MMR SACCH carries (realair_mmr_test.go).

// macField is one field of a message: the bit index of its first bit, its
// width, and its value.
type macField struct {
	bit, width int
	value      uint32
}

// macMessage builds an n-octet message with the given opcode and fields and
// parses it the way AirMAC.Messages hands a message on.
func macMessage(t *testing.T, n int, opcode uint8, fields ...macField) MACPDU {
	t.Helper()
	b := make([]byte, n)
	b[0] = opcode
	for _, f := range fields {
		for i := 0; i < f.width; i++ {
			if f.value>>uint(f.width-1-i)&1 != 0 {
				b[(f.bit+i)/8] |= 0x80 >> uint((f.bit+i)%8)
			}
		}
	}
	pdu, err := ParseMACPDU(b)
	if err != nil {
		t.Fatal(err)
	}
	return pdu
}

// TestOpcodeTableFollowsTheReferences pins the opcode values this package
// acts on, and the names of the ones it used to get wrong.
func TestOpcodeTableFollowsTheReferences(t *testing.T) {
	values := map[Opcode]uint8{
		OpGroupVoiceChannelUserAbbreviated:             1,   // TDMA_01
		OpUnitToUnitVoiceChannelUserAbbreviated:        2,   // TDMA_02
		OpGroupVoiceChannelGrantUpdateMultiple:         5,   // TDMA_05
		OpGroupVoiceChannelUserExtended:                33,  // TDMA_21
		OpUnitToUnitVoiceChannelUserExtended:           34,  // TDMA_22
		OpGroupVoiceChannelGrantUpdateMultipleExplicit: 37,  // TDMA_25
		OpGroupVoiceChannelGrant:                       64,  // PHASE1_40
		OpGroupVoiceChannelGrantUpdate:                 66,  // PHASE1_42
		OpUnitToUnitVoiceChannelGrant:                  68,  // PHASE1_44
		OpUnitToUnitVoiceChannelGrantUpdate:            70,  // PHASE1_46
		OpGroupAffiliationResponse:                     104, // PHASE1_68
		OpUnitRegistrationResponse:                     108, // PHASE1_6C
		OpSynchronizationBroadcast:                     112, // PHASE1_70
		OpRFSSStatusBroadcastImplicit:                  122, // PHASE1_7A
		OpNetworkStatusBroadcastImplicit:               123, // PHASE1_7B
		OpAdjacentStatusBroadcastImplicit:              124, // PHASE1_7C
		OpIdentifierUpdate:                             125, // PHASE1_7D
		OpGroupVoiceChannelGrantExplicit:               192, // PHASE1_C0
		OpGroupVoiceChannelGrantUpdateExplicit:         195, // PHASE1_C3
		OpRFSSStatusBroadcastUpdate:                    250, // PHASE1_FA
		OpNetworkStatusBroadcastUpdate:                 251, // PHASE1_FB
		OpMotorolaRegroupVoiceChannelUser:              128, // MOTOROLA_80
		OpMotorolaRegroupAdd:                           129, // MOTOROLA_81
		OpMotorolaRegroupDelete:                        137, // MOTOROLA_89
		OpMotorolaAliasHeader:                          145, // MOTOROLA_91
		OpMotorolaAliasData:                            149, // MOTOROLA_95
		OpMotorolaRegroupVoiceChannelUserExtended:      160, // MOTOROLA_A0
	}
	for op, want := range values {
		if uint8(op) != want {
			t.Errorf("%v = 0x%02X, the references have 0x%02X", op, uint8(op), want)
		}
	}
	// Opcodes the older table named as something else. 0x42 and 0x7C are the
	// two the first on-air Phase 2 log printed as bare numbers.
	names := map[Opcode]string{
		0x02: "UnitToUnitVoiceChannelUserAbbreviated", // was MAC_END
		0x03: "TelephoneInterconnectVoiceChannelUser", // was MAC_IDLE
		0x05: "GroupVoiceChannelGrantUpdateMultiple",  // was MAC_HANGTIME
		0x40: "GroupVoiceChannelGrant",                // was the grant update
		0x42: "GroupVoiceChannelGrantUpdate",
		0x44: "UnitToUnitVoiceChannelGrant", // was the group grant
		0x48: "TelephoneInterconnectVoiceChannelGrant",
		0x4C: "RadioUnitMonitorCommand", // was the affiliation response
		0x70: "SynchronizationBroadcast",
		0x7C: "AdjacentStatusBroadcastImplicit",
	}
	for op, want := range names {
		if got := op.String(); got != want {
			t.Errorf("opcode 0x%02X is named %q, want %q", uint8(op), got, want)
		}
	}
	// 0x06 and 0x4D are not messages at all.
	for _, op := range []Opcode{0x06, 0x4D} {
		if op.IsKnown() {
			t.Errorf("opcode 0x%02X is not a defined MAC message", uint8(op))
		}
	}
	// A manufacturer-specific opcode has no name without its MFID.
	if got := Opcode(0x81).String(); got != "Opcode(81)" {
		t.Errorf("Opcode(0x81).String() = %q", got)
	}
	vendor := map[string]MACPDU{
		"Motorola.GroupRegroupAdd":                {Opcode: 0x81, MFID: MFIDMotorola},
		"Motorola.GroupRegroupVoiceChannelUpdate": {Opcode: 0x83, MFID: MFIDMotorola},
		"Motorola.TalkerAliasHeader":              {Opcode: 0x91, MFID: MFIDMotorola},
		"Harris.TalkerAlias":                      {Opcode: 0xA8, MFID: MFIDHarris},
		"Harris.Opcode(81)":                       {Opcode: 0x81, MFID: MFIDHarris},
		"MFID(D8).Opcode(81)":                     {Opcode: 0x81, MFID: 0xD8},
	}
	for want, pdu := range vendor {
		if got := pdu.Name(); got != want {
			t.Errorf("MFID %02X opcode %02X is named %q, want %q", pdu.MFID, uint8(pdu.Opcode), got, want)
		}
	}
}

// TestGroupVoiceChannelGrantLayouts: SDRTrunk GroupVoiceChannelGrantImplicit
// (service options bit 8, band 16, channel 20, group 32, source 48) and
// Explicit (transmit band 16 / channel 20, receive 32 / 36, group 48,
// source 64).
func TestGroupVoiceChannelGrantLayouts(t *testing.T) {
	want := GroupVoiceChannelGrant{ServiceOptions: 0xC5, ChannelID: 0x3, ChannelNumber: 0x4A7, GroupAddress: 0xBEEF, SourceID: 0x123456}
	implicit := macMessage(t, 9, 0x40,
		macField{8, 8, 0xC5}, macField{16, 4, 0x3}, macField{20, 12, 0x4A7},
		macField{32, 16, 0xBEEF}, macField{48, 24, 0x123456})
	if got, ok := implicit.AsGroupVoiceChannelGrant(); !ok || got != want {
		t.Errorf("0x40: %+v ok=%v, want %+v", got, ok, want)
	}
	explicit := macMessage(t, 11, 0xC0,
		macField{8, 8, 0xC5}, macField{16, 4, 0x3}, macField{20, 12, 0x4A7},
		macField{32, 4, 0x9}, macField{36, 12, 0x111},
		macField{48, 16, 0xBEEF}, macField{64, 24, 0x123456})
	if got, ok := explicit.AsGroupVoiceChannelGrant(); !ok || got != want {
		t.Errorf("0xC0: %+v ok=%v, want %+v", got, ok, want)
	}
	// 0x44 is a unit-to-unit grant. The older table had the group grant
	// there and read a talkgroup out of the target address.
	uu := macMessage(t, 9, 0x44, macField{8, 4, 0x3}, macField{12, 12, 0x4A7},
		macField{24, 24, 0xABCDEF}, macField{48, 24, 0x123456})
	if _, ok := uu.AsGroupVoiceChannelGrant(); ok {
		t.Error("0x44 (a unit-to-unit grant) parsed as a group grant")
	}
}

// TestGroupVoiceChannelGrantUpdateLayouts: the four grant-update messages
// (SDRTrunk GroupVoiceChannelGrantUpdateImplicit / Explicit /
// MultipleImplicit / MultipleExplicit).
func TestGroupVoiceChannelGrantUpdateLayouts(t *testing.T) {
	a := GroupVoiceChannelGrant{ChannelID: 0x1, ChannelNumber: 0x0A5, GroupAddress: 0x1111}
	b := GroupVoiceChannelGrant{ChannelID: 0x2, ChannelNumber: 0x7FE, GroupAddress: 0x2222}
	c := GroupVoiceChannelGrant{ChannelID: 0x3, ChannelNumber: 0x123, GroupAddress: 0x3333}
	check := func(name string, pdu MACPDU, want ...GroupVoiceChannelGrant) {
		t.Helper()
		got, ok := pdu.AsGroupVoiceChannelGrantUpdates()
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %+v ok=%v, want %+v", name, got, ok, want)
		}
	}
	// 0x42: band A 8, channel A 12, group A 24, band B 40, channel B 44,
	// group B 56. No service options.
	check("0x42", macMessage(t, 9, 0x42,
		macField{8, 4, 0x1}, macField{12, 12, 0x0A5}, macField{24, 16, 0x1111},
		macField{40, 4, 0x2}, macField{44, 12, 0x7FE}, macField{56, 16, 0x2222}), a, b)
	// A message with one call repeats it.
	check("0x42 repeated", macMessage(t, 9, 0x42,
		macField{8, 4, 0x1}, macField{12, 12, 0x0A5}, macField{24, 16, 0x1111},
		macField{40, 4, 0x1}, macField{44, 12, 0x0A5}, macField{56, 16, 0x1111}), a)

	// 0xC3: service options 8, transmit 16 / 20, receive 32 / 36, group 48.
	ae := a
	ae.ServiceOptions = 0x44
	check("0xC3", macMessage(t, 8, 0xC3,
		macField{8, 8, 0x44}, macField{16, 4, 0x1}, macField{20, 12, 0x0A5},
		macField{32, 4, 0xF}, macField{36, 12, 0xFFF}, macField{48, 16, 0x1111}), ae)

	// 0x05: service options 8 / 48 / 88, band 16 / 56 / 96, channel
	// 20 / 60 / 100, group 32 / 72 / 112.
	a5, b5, c5 := a, b, c
	a5.ServiceOptions, b5.ServiceOptions, c5.ServiceOptions = 0x04, 0x45, 0x86
	check("0x05", macMessage(t, 16, 0x05,
		macField{8, 8, 0x04}, macField{16, 4, 0x1}, macField{20, 12, 0x0A5}, macField{32, 16, 0x1111},
		macField{48, 8, 0x45}, macField{56, 4, 0x2}, macField{60, 12, 0x7FE}, macField{72, 16, 0x2222},
		macField{88, 8, 0x86}, macField{96, 4, 0x3}, macField{100, 12, 0x123}, macField{112, 16, 0x3333}),
		a5, b5, c5)

	// 0x25: service options 8 / 64, transmit 16 / 72 (channel 20 / 76),
	// receive 32 / 88, group 48 / 104.
	check("0x25", macMessage(t, 15, 0x25,
		macField{8, 8, 0x04}, macField{16, 4, 0x1}, macField{20, 12, 0x0A5},
		macField{32, 4, 0xF}, macField{36, 12, 0xFFF}, macField{48, 16, 0x1111},
		macField{64, 8, 0x45}, macField{72, 4, 0x2}, macField{76, 12, 0x7FE},
		macField{88, 4, 0xF}, macField{92, 12, 0xFFF}, macField{104, 16, 0x2222}),
		a5, b5)

	// 0x40 is a grant, not an update.
	if _, ok := macMessage(t, 9, 0x40).AsGroupVoiceChannelGrantUpdates(); ok {
		t.Error("0x40 (a grant) parsed as a grant update")
	}
}

// TestUnitToUnitLayouts: the unit-to-unit grant (SDRTrunk
// UnitToUnitVoiceServiceChannelGrantAbbreviated: band 8, channel 12, target
// 24, source 48) and the in-call user (UnitToUnitVoiceChannelUser
// Abbreviated: service options 8, target 16, source 40).
func TestUnitToUnitLayouts(t *testing.T) {
	wantGrant := UnitToUnitVoiceChannelGrant{ChannelID: 0x3, ChannelNumber: 0x4A7, TargetID: 0xABCDEF, SourceID: 0x123456}
	for _, op := range []uint8{0x44, 0x46} {
		pdu := macMessage(t, 9, op, macField{8, 4, 0x3}, macField{12, 12, 0x4A7},
			macField{24, 24, 0xABCDEF}, macField{48, 24, 0x123456})
		if got, ok := pdu.AsUnitToUnitVoiceChannelGrant(); !ok || got != wantGrant {
			t.Errorf("0x%02X: %+v ok=%v, want %+v", op, got, ok, wantGrant)
		}
	}
	// 0x48 / 0x49 are telephone interconnect grants; the older table had
	// the unit-to-unit grant there.
	if _, ok := macMessage(t, 10, 0x48).AsUnitToUnitVoiceChannelGrant(); ok {
		t.Error("0x48 (a telephone interconnect grant) parsed as a unit-to-unit grant")
	}

	wantUser := UnitToUnitVoiceChannelUser{ServiceOptions: 0x44, TargetID: 0xABCDEF, SourceID: 0x123456}
	for op, n := range map[uint8]int{0x02: 8, 0x22: 15} {
		pdu := macMessage(t, n, op, macField{8, 8, 0x44}, macField{16, 24, 0xABCDEF}, macField{40, 24, 0x123456})
		if got, ok := pdu.AsUnitToUnitVoiceChannelUser(); !ok || got != wantUser {
			t.Errorf("0x%02X: %+v ok=%v, want %+v", op, got, ok, wantUser)
		}
	}
	if _, ok := macMessage(t, 7, 0x01).AsUnitToUnitVoiceChannelUser(); ok {
		t.Error("0x01 (a group voice channel user) parsed as a unit-to-unit user")
	}
}

// TestAffiliationAndRegistrationLayouts: SDRTrunk GroupAffiliationResponse
// Abbreviated (response bits 22-23, announcement group 24, group 40, target
// 56) and UnitRegistrationResponseAbbreviated (response bits 18-19, system
// 20, unit ID 32, assigned address 56).
func TestAffiliationAndRegistrationLayouts(t *testing.T) {
	aff := macMessage(t, 10, 0x68, macField{16, 1, 1}, macField{22, 2, 0x2},
		macField{24, 16, 0x0F0F}, macField{40, 16, 0xBEEF}, macField{56, 24, 0x123456})
	wantAff := GroupAffiliationResponse{Response: 2, AnnouncementGroup: 0x0F0F, GroupAddress: 0xBEEF, TargetID: 0x123456}
	if got, ok := aff.AsGroupAffiliationResponse(); !ok || got != wantAff {
		t.Errorf("0x68: %+v ok=%v, want %+v", got, ok, wantAff)
	}
	if got, ok := EncodeGroupAffiliationResponse(wantAff).AsGroupAffiliationResponse(); !ok || got != wantAff {
		t.Errorf("encoder round trip: %+v ok=%v", got, ok)
	}

	reg := macMessage(t, 10, 0x6C, macField{18, 2, 0x1}, macField{20, 12, 0x164},
		macField{32, 24, 0xABCDEF}, macField{56, 24, 0x123456})
	wantReg := UnitRegistrationResponse{Response: 1, SystemID: 0x164, UnitID: 0xABCDEF, SourceID: 0x123456}
	if got, ok := reg.AsUnitRegistrationResponse(); !ok || got != wantReg {
		t.Errorf("0x6C: %+v ok=%v, want %+v", got, ok, wantReg)
	}
	if got, ok := EncodeUnitRegistrationResponse(wantReg).AsUnitRegistrationResponse(); !ok || got != wantReg {
		t.Errorf("encoder round trip: %+v ok=%v", got, ok)
	}

	// 0x4C (a Radio Unit Monitor Command) and 0x4D (nothing) are where the
	// older table looked for these two.
	if _, ok := macMessage(t, 10, 0x4C).AsGroupAffiliationResponse(); ok {
		t.Error("0x4C parsed as a group affiliation response")
	}
	if _, ok := macMessage(t, 10, 0x4D).AsUnitRegistrationResponse(); ok {
		t.Error("0x4D parsed as a unit registration response")
	}
}

// TestRFSSStatusBroadcastBothForms: SDRTrunk RfssStatusBroadcast (LRA 8,
// System ID 20, RFSS 32, site 40) with the channel at 48 in both the
// implicit (0x7A) and explicit (0xFA) forms.
func TestRFSSStatusBroadcastBothForms(t *testing.T) {
	want := RFSSStatusBroadcast{LRA: 0x2A, SystemID: 0x164, RFSS: 0x07, Site: 0x13, ChannelID: 0x3, ChannelNumber: 0x4A7}
	for op, n := range map[uint8]int{0x7A: 9, 0xFA: 11} {
		pdu := macMessage(t, n, op, macField{8, 8, 0x2A}, macField{20, 12, 0x164},
			macField{32, 8, 0x07}, macField{40, 8, 0x13}, macField{48, 4, 0x3}, macField{52, 12, 0x4A7})
		if got, ok := pdu.AsRFSSStatusBroadcast(); !ok || got != want {
			t.Errorf("0x%02X: %+v ok=%v, want %+v", op, got, ok, want)
		}
	}
}

// TestMotorolaRegroupRealAir: the group regroup message a real MMR SACCH
// carries (realair_mmr_test.go finds these bytes in the capture). The Phase 1
// control channel in the same capture announces supergroup 3202 patching
// talkgroup 201. The older parser read the supergroup from the length octet
// and the octet after it: supergroup 2828 with members 33280 and 51456.
func TestMotorolaRegroupRealAir(t *testing.T) {
	raw, _ := hex.DecodeString("81900b0c8200c900c900c9")
	pdu, err := ParseMACPDU(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := pdu.Name(); got != "Motorola.GroupRegroupAdd" {
		t.Errorf("named %q", got)
	}
	g, ok := pdu.AsMotorolaPatchGroup()
	if !ok {
		t.Fatal("not parsed as a Motorola group regroup")
	}
	if g.SuperGroup != 3202 || !reflect.DeepEqual(g.Patched, []uint16{201}) {
		t.Errorf("supergroup %d members %v, want 3202 and [201]", g.SuperGroup, g.Patched)
	}
	// The same bytes under another MFID are another manufacturer's message.
	other := pdu
	other.MFID = MFIDHarris
	if _, ok := other.AsMotorolaPatchGroup(); ok {
		t.Error("a Harris 0x81 parsed as a Motorola group regroup")
	}
}

// TestMotorolaRegroupLayouts: SDRTrunk MotorolaGroupRegroupAddCommand /
// DeleteCommand (supergroup 24, talkgroups from 40, 16 bits each) and
// MotorolaGroupRegroupVoiceChannelUserAbbreviated (supergroup 24, source 40)
// / Extended (supergroup 32, source 48).
func TestMotorolaRegroupLayouts(t *testing.T) {
	add := macMessage(t, 17, 0x81, macField{8, 8, 0x90}, macField{16, 8, 17}, macField{24, 16, 0x0C82},
		macField{40, 16, 0x0001}, macField{56, 16, 0x0002}, macField{72, 16, 0x0003},
		macField{88, 16, 0x0004}, macField{104, 16, 0x0005}, macField{120, 16, 0x0006})
	g, ok := add.AsMotorolaPatchGroup()
	if !ok || g.SuperGroup != 0x0C82 || !reflect.DeepEqual(g.Patched, []uint16{1, 2, 3, 4, 5, 6}) {
		t.Errorf("0x81: %+v ok=%v", g, ok)
	}
	// The length octet bounds the member list: what follows the message in
	// the PDU is another message's bytes.
	short := macMessage(t, 17, 0x81, macField{8, 8, 0x90}, macField{16, 8, 9}, macField{24, 16, 0x0C82},
		macField{40, 16, 0x0001}, macField{56, 16, 0x0002}, macField{72, 16, 0xFFFF})
	if g, ok := short.AsMotorolaPatchGroup(); !ok || !reflect.DeepEqual(g.Patched, []uint16{1, 2}) {
		t.Errorf("0x81 with length 9: %+v ok=%v, want members [1 2]", g, ok)
	}

	del := macMessage(t, 17, 0x89, macField{8, 8, 0x90}, macField{16, 8, 7}, macField{24, 16, 0x0C82},
		macField{40, 16, 0x00C9})
	if super, ok := del.AsMotorolaPatchDelete(); !ok || super != 0x0C82 {
		t.Errorf("0x89: supergroup %#x ok=%v, want 0xC82", super, ok)
	}
	// 0x83 is a regroup voice channel update (a call in progress on a
	// supergroup). The older table had "patch delete" there, so every such
	// call cancelled its own patch.
	upd := macMessage(t, 7, 0x83, macField{8, 8, 0x90}, macField{24, 16, 0x0C82}, macField{40, 16, 0x34A7})
	if _, ok := upd.AsMotorolaPatchDelete(); ok {
		t.Error("0x83 (a regroup voice channel update) parsed as a patch delete")
	}

	want := MotorolaRegroupVoiceChannelUser{SuperGroup: 0x0C82, SourceID: 0x123456}
	abbrev := macMessage(t, 8, 0x80, macField{8, 8, 0x90}, macField{24, 16, 0x0C82}, macField{40, 24, 0x123456})
	if got, ok := abbrev.AsMotorolaRegroupVoiceChannelUser(); !ok || got != want {
		t.Errorf("0x80: %+v ok=%v, want %+v", got, ok, want)
	}
	ext := macMessage(t, 16, 0xA0, macField{8, 8, 0x90}, macField{16, 8, 16}, macField{24, 8, 0x40},
		macField{32, 16, 0x0C82}, macField{48, 24, 0x123456})
	if got, ok := ext.AsMotorolaRegroupVoiceChannelUser(); !ok || got != want {
		t.Errorf("0xA0: %+v ok=%v, want %+v", got, ok, want)
	}
	notMoto := abbrev
	notMoto.MFID = MFIDHarris
	if _, ok := notMoto.AsMotorolaRegroupVoiceChannelUser(); ok {
		t.Error("a Harris 0x80 parsed as a Motorola regroup voice channel user")
	}
}

// TestControlChannelFollowsTheRealGrantMessages: what a Phase 2 control
// channel publishes for the grant messages at their real opcodes. A grant
// update (0x42) repeats the calls in progress, and is how a receiver that
// missed a grant still finds the call; this package did not parse it at all.
// A unit-to-unit grant (0x44) used to come out as a group grant on a
// talkgroup read from the target radio's address.
func TestControlChannelFollowsTheRealGrantMessages(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})

	cc.Ingest(macMessage(t, 9, 0x42,
		macField{8, 4, 0x1}, macField{12, 12, 0x0A5}, macField{24, 16, 0x1111},
		macField{40, 4, 0x2}, macField{44, 12, 0x7FE}, macField{56, 16, 0x2222}))
	cc.Ingest(macMessage(t, 9, 0x44, macField{8, 4, 0x3}, macField{12, 12, 0x4A7},
		macField{24, 24, 0xABCDEF}, macField{48, 24, 0x123456}))

	grants := countGrants(sub)
	if len(grants) != 3 {
		t.Fatalf("%d grants published, want 3 (two from the update, one unit-to-unit)", len(grants))
	}
	if grants[0].GroupID != 0x1111 || grants[0].ChannelID != 1 || grants[0].ChannelNum != 0x0A5 || grants[0].SourceID != 0 {
		t.Errorf("update, first call: %+v", grants[0])
	}
	if grants[1].GroupID != 0x2222 || grants[1].ChannelID != 2 || grants[1].ChannelNum != 0x7FE {
		t.Errorf("update, second call: %+v", grants[1])
	}
	if grants[2].GroupID != 0xABCDEF || grants[2].SourceID != 0x123456 || grants[2].ChannelID != 3 || grants[2].ChannelNum != 0x4A7 {
		t.Errorf("unit-to-unit grant: %+v", grants[2])
	}
}

// TestMessagesWalkMotorolaRegroupUser: a Motorola regroup voice channel user
// (0x80) is 8 octets whatever its third octet holds. SDRTrunk reads service
// options there; with a call priority of 4 in them, a walk that took the
// octet for the message's length would cut the message to 4 octets and then
// read the middle of it as the next opcode.
func TestMessagesWalkMotorolaRegroupUser(t *testing.T) {
	for _, third := range []byte{0x08, 0x04, 0x00, 0x44} {
		pdu := make([]byte, 18)
		pdu[0] = byte(AirMACActive) << 5
		copy(pdu[1:], []byte{0x80, 0x90, third, 0x0C, 0x82, 0x04, 0xCF, 0x44}) // regroup user
		copy(pdu[9:], []byte{0x01, 0x00, 0x4E, 0xEA, 0x04, 0xCF, 0x43})        // group user
		msgs := AirMAC{Channel: AirFACCH, Type: AirMACActive, Bytes: pdu}.Messages()
		if len(msgs) != 2 {
			t.Fatalf("third octet %#02x: %d messages, want 2", third, len(msgs))
		}
		u, ok := msgs[0].AsMotorolaRegroupVoiceChannelUser()
		if !ok || u.SuperGroup != 3202 || u.SourceID != 315204 {
			t.Errorf("third octet %#02x: regroup user %+v ok=%v", third, u, ok)
		}
		if g, ok := msgs[1].AsGroupVoiceChannelUser(); !ok || g.SourceID != 315203 {
			t.Errorf("third octet %#02x: the message after it: %+v ok=%v", third, g, ok)
		}
	}
}
