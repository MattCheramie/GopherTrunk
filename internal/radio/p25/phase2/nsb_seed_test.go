package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// assembleNSBPayload builds an 18-byte MAC PDU info window for an explicit
// Network Status Broadcast with the supplied identity fields, in the
// SDRTrunk NetworkStatusBroadcastExplicit layout (receive channel = the
// transmit channel + 1, service class 0x70).
func assembleNSBPayload(lra uint8, wacn uint32, sysID, cc uint16, chanID uint8, chanNum uint16) []byte {
	pdu := make([]byte, 18)
	pdu[0] = byte(OpNetworkStatusBroadcastUpdate)
	pdu[1] = lra
	pdu[2] = byte(wacn >> 12)
	pdu[3] = byte(wacn >> 4)
	pdu[4] = byte((wacn&0x0F)<<4) | byte(sysID>>8)&0x0F
	pdu[5] = byte(sysID & 0xFF)
	pdu[6] = (chanID << 4) | byte(chanNum>>8)&0x0F
	pdu[7] = byte(chanNum)
	pdu[8] = (chanID << 4) | byte((chanNum+1)>>8)&0x0F
	pdu[9] = byte(chanNum + 1)
	pdu[10] = 0x70
	pdu[11] = byte(cc>>8) & 0x0F
	pdu[12] = byte(cc)
	return pdu
}

// TestAsNetworkStatusBroadcastMatchesReferenceLayout pins both NSB forms
// against literal messages laid out from the reference offsets (SDRTrunk
// NetworkStatusBroadcastExplicit / Implicit; OP25 tk_p25.py op 0xfb reads
// wacn = msg[2:5]>>4, syid = msg[4:6]&0xfff, ch_t = msg[6:8]). The Color
// Code is the last 12 bits, NOT bytes 6..7 — which GopherTrunk used to
// read, so a site's NSB installed a wrong descramble seed.
func TestAsNetworkStatusBroadcastMatchesReferenceLayout(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  []byte
	}{
		{"explicit", []byte{0xFB, 0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x10, 0x05, 0x10, 0x06, 0x70, 0x02, 0x93}},
		{"implicit", []byte{0x7B, 0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x10, 0x05, 0x70, 0x02, 0x93}},
	} {
		pdu, err := ParseMACPDU(tc.msg)
		if err != nil {
			t.Fatal(err)
		}
		nsb, ok := pdu.AsNetworkStatusBroadcast()
		if !ok {
			t.Fatalf("%s: not parsed", tc.name)
		}
		want := NetworkStatusBroadcast{LRA: 0x07, WACN: 0xABCDE, SystemID: 0x123, ColorCode: 0x293,
			ChannelID: 1, ChannelNumber: 5, ServiceClass: 0x70}
		if nsb != want {
			t.Errorf("%s: got %+v, want %+v", tc.name, nsb, want)
		}
		if l := int(airMACMsgLen[tc.msg[0]]); l != len(tc.msg) {
			t.Errorf("%s: OP25 mac_msg_len = %d, literal is %d bytes", tc.name, l, len(tc.msg))
		}
	}
}

// TestAsNetworkStatusBroadcastExtractsIdentity verifies the parser
// recovers the (WACN, SystemID, ColorCode) triple from a hand-built
// NSB payload.
func TestAsNetworkStatusBroadcastExtractsIdentity(t *testing.T) {
	const (
		wacn  uint32 = 0xABCDE
		sysID uint16 = 0x123
		cc    uint16 = 0xF7
		lra   uint8  = 0x42
	)
	info := assembleNSBPayload(lra, wacn, sysID, cc, 0x3, 0x456)
	pdu, err := ParseMACPDU(info)
	if err != nil {
		t.Fatalf("ParseMACPDU: %v", err)
	}
	nsb, ok := pdu.AsNetworkStatusBroadcast()
	if !ok {
		t.Fatalf("AsNetworkStatusBroadcast returned !ok")
	}
	if nsb.WACN != wacn {
		t.Errorf("WACN = %#x, want %#x", nsb.WACN, wacn)
	}
	if nsb.SystemID != sysID {
		t.Errorf("SystemID = %#x, want %#x", nsb.SystemID, sysID)
	}
	if nsb.ColorCode != cc {
		t.Errorf("ColorCode = %#x, want %#x", nsb.ColorCode, cc)
	}
	if nsb.LRA != lra {
		t.Errorf("LRA = %#x, want %#x", nsb.LRA, lra)
	}
	if nsb.ChannelID != 0x3 {
		t.Errorf("ChannelID = %#x, want 0x3", nsb.ChannelID)
	}
	if nsb.ChannelNumber != 0x456 {
		t.Errorf("ChannelNumber = %#x, want 0x456", nsb.ChannelNumber)
	}
}

// TestAsNetworkStatusBroadcastRejectsWrongOpcode confirms the
// accessor rejects PDUs whose opcode is not NSB-Update.
func TestAsNetworkStatusBroadcastRejectsWrongOpcode(t *testing.T) {
	pdu := MACPDU{
		Opcode:  OpGroupVoiceChannelGrant,
		Payload: make([]byte, 17),
	}
	if _, ok := pdu.AsNetworkStatusBroadcast(); ok {
		t.Errorf("AsNetworkStatusBroadcast accepted non-NSB opcode")
	}
}

// TestAsNetworkStatusBroadcastRejectsShortPayload confirms the
// accessor rejects a payload too short to carry the identity
// fields.
func TestAsNetworkStatusBroadcastRejectsShortPayload(t *testing.T) {
	pdu := MACPDU{
		Opcode:  OpNetworkStatusBroadcastUpdate,
		Payload: make([]byte, 11), // need 12
	}
	if _, ok := pdu.AsNetworkStatusBroadcast(); ok {
		t.Errorf("AsNetworkStatusBroadcast accepted 11-byte payload")
	}
}

// TestIngestNSBInstallsScramblerSeed exercises the full auto-seed
// path: hand the state machine an NSB-Update PDU and assert the
// ControlChannel updates its scrambler seed to the
// PN44SeedFromIdentity(WACN, SysID, CC) value.
//
// This is the key behaviour change that closes the Phase 2
// NSB-driven-seed follow-up.
func TestIngestNSBInstallsScramblerSeed(t *testing.T) {
	const (
		wacn  uint32 = 0xABCDE
		sysID uint16 = 0x123
		cc    uint16 = 0xF7
	)
	wantSeed := framing.PN44SeedFromIdentity(wacn, sysID, cc)

	bus := events.NewBus(8)
	defer bus.Close()
	cc1 := New(Options{
		Bus:         bus,
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	if cc1.ScramblerSeed() == wantSeed {
		t.Fatalf("scrambler seed already matches target before NSB ingest; test is meaningless")
	}

	info := assembleNSBPayload(0, wacn, sysID, cc, 0, 0)
	pdu, err := ParseMACPDU(info)
	if err != nil {
		t.Fatalf("ParseMACPDU: %v", err)
	}
	cc1.Ingest(pdu)

	if got := cc1.ScramblerSeed(); got != wantSeed {
		t.Errorf("ScramblerSeed after NSB Ingest = %#x, want %#x", got, wantSeed)
	}
}

// TestIngestNSBOverwritesInitialSeed confirms an NSB-derived seed
// replaces a previously-installed config seed when the two
// disagree. Initial seed installed via SetScramblerSeed; NSB
// arrives with different identity values; final seed reflects the
// NSB.
func TestIngestNSBOverwritesInitialSeed(t *testing.T) {
	const (
		initialSeed = uint64(0xDEADBEEF)
		wacn        = uint32(0x12345)
		sysID       = uint16(0x678)
		cc          = uint16(0x9AB)
	)
	wantSeed := framing.PN44SeedFromIdentity(wacn, sysID, cc)

	bus := events.NewBus(8)
	defer bus.Close()
	cc1 := New(Options{
		Bus:         bus,
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	cc1.SetScramblerSeed(initialSeed)
	if cc1.ScramblerSeed() != initialSeed {
		t.Fatalf("initial seed setup failed; got %#x", cc1.ScramblerSeed())
	}

	info := assembleNSBPayload(0, wacn, sysID, cc, 0, 0)
	pdu, _ := ParseMACPDU(info)
	cc1.Ingest(pdu)

	if got := cc1.ScramblerSeed(); got != wantSeed {
		t.Errorf("ScramblerSeed after NSB overwrite = %#x, want %#x (initial was %#x)",
			got, wantSeed, initialSeed)
	}
}
