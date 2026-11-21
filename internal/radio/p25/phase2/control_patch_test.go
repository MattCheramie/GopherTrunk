package phase2

import (
	"encoding/hex"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func nextPatch(t *testing.T, sub *events.Subscription) trunking.Patch {
	t.Helper()
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindPatch {
				return ev.Payload.(trunking.Patch)
			}
		default:
			t.Fatal("no KindPatch event published")
			return trunking.Patch{}
		}
	}
}

// TestControlChannelPublishesMotorolaPatch feeds the control channel the
// group regroup message a real MMR SACCH carries: supergroup 3202 patching
// talkgroup 201.
func TestControlChannelPublishesMotorolaPatch(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	raw, _ := hex.DecodeString("81900b0c8200c900c900c9")
	pdu, err := ParseMACPDU(raw)
	if err != nil {
		t.Fatal(err)
	}
	cc.Ingest(pdu)

	p := nextPatch(t, sub)
	if p.SuperGroup != 3202 {
		t.Errorf("patch SuperGroup = %d, want 3202", p.SuperGroup)
	}
	if p.Vendor != "motorola" || !p.Add {
		t.Errorf("patch Vendor=%q Add=%v, want motorola/true", p.Vendor, p.Add)
	}
	if len(p.Members) != 1 || p.Members[0] != 201 {
		t.Errorf("patch Members = %v, want [201]", p.Members)
	}
}

// TestControlChannelIgnoresUnidentifiedVendorMessages: opcode 0x81 under the
// Harris MFID and 0x82 under any are messages no reference decoder
// identifies. This package used to publish a "Harris regroup" patch from the
// first and assemble a plain-text talker alias from the second, both from
// layouts of its own.
func TestControlChannelIgnoresUnidentifiedVendorMessages(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(MACPDU{Opcode: 0x81, MFID: MFIDHarris, Payload: []byte{0x05, 0x55, 0x00, 0xBE, 0xEF}})
	cc.Ingest(MACPDU{Opcode: 0x82, MFID: MFIDMotorola, Payload: append([]byte{0x00, 0x12, 0x34, 0, 1}, "UNIT-7"...)})
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindPatch || ev.Kind == events.KindTalkerAlias {
				t.Fatalf("published %v from a vendor message no reference identifies: %+v", ev.Kind, ev.Payload)
			}
			continue
		default:
		}
		break
	}
}
