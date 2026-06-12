package tier3

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// TestParseInboundCSBKRealVectors pins the C_AHOY and C_ACKD address
// offsets against the only real off-air vectors available (the same bursts
// csbk_realvectors_test.go validates for CRC/opcode). The addressing
// follows the grant-family convention (24-bit target at octets 2-4, source
// at octets 5-7); a regression of the offsets trips this test offline. No
// field is asserted that these two vectors don't confirm.
func TestParseInboundCSBKRealVectors(t *testing.T) {
	// C_AHOY (0x1C): payload {0x00,0x0e,0x00,0x02,0xd0,0xff,0xfe,0xca}.
	ahoyCSBK, err := ParseCSBK([]byte{0x9c, 0x00, 0x00, 0x0e, 0x00, 0x02, 0xd0, 0xff, 0xfe, 0xca, 0x94, 0x25})
	if err != nil {
		t.Fatalf("ParseCSBK rejected real C_AHOY: %v", err)
	}
	if a := ParseAhoy(ahoyCSBK.Payload); a.TargetID != 0x0002D0 || a.SourceID != 0xFFFECA {
		t.Errorf("C_AHOY target/source = %X/%X, want 0002D0/FFFECA", a.TargetID, a.SourceID)
	}

	// C_ACKD (0x20): payload {0x00,0xc4,0x00,0x03,0x45,0xff,0xfe,0xc6}.
	ackCSBK, err := ParseCSBK([]byte{0xa0, 0x00, 0x00, 0xc4, 0x00, 0x03, 0x45, 0xff, 0xfe, 0xc6, 0x4a, 0x9c})
	if err != nil {
		t.Fatalf("ParseCSBK rejected real C_ACKD: %v", err)
	}
	ack := ParseAck(ackCSBK.Opcode, ackCSBK.Payload)
	if ack.Opcode != OpAckD || ack.TargetID != 0x000345 || ack.SourceID != 0xFFFEC6 {
		t.Errorf("C_ACKD = %+v, want opcode C_ACKD target 000345 source FFFEC6", ack)
	}
}

func TestParseRandomAccessRoundTrip(t *testing.T) {
	// C_RAND has no real off-air vector; this only pins the parser to the
	// grant-family addressing convention via an AssembleCSBK round-trip.
	csbk := CSBK{LB: true, Opcode: OpRAND, Payload: [8]byte{0x07, 0x00, 0x00, 0x12, 0x34, 0x00, 0xAB, 0xCD}}
	parsed, err := ParseCSBK(AssembleCSBK(csbk))
	if err != nil {
		t.Fatalf("round-trip CRC: %v", err)
	}
	r := ParseRandomAccess(parsed.Payload)
	if r.Service != 0x07 || r.TargetID != 0x001234 || r.SourceID != 0x00ABCD {
		t.Errorf("C_RAND = %+v", r)
	}
}

// TestControlChannelInboundCSBKNoSideEffects confirms inbound / reverse-
// channel CSBKs (C_RAND, C_AHOY, C_ACKD) are parsed for visibility only —
// they never leak a voice grant or a control-channel lock.
func TestControlChannelInboundCSBKNoSideEffects(t *testing.T) {
	for _, op := range []CSBKOpcode{OpRAND, OpAhoy, OpAckD, OpNack} {
		bus := events.NewBus(8)
		sub := bus.Subscribe()
		cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1, Resolver: TableBandPlan{1: 1_000_000}})
		csbk := CSBK{LB: true, Opcode: op, Payload: [8]byte{0x00, 0x00, 0x00, 0x12, 0x34, 0x00, 0xAB, 0xCD}}
		cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

		select {
		case ev := <-sub.C:
			t.Errorf("opcode %s produced unexpected event %s", op, ev.Kind)
		case <-time.After(50 * time.Millisecond):
		}
		sub.Close()
		bus.Close()
	}
}
