package phase2

import (
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

func TestControlChannelPublishesMotorolaPatch(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(MACPDU{
		Opcode:  OpVendorGroupRegroup,
		MFID:    MFIDMotorola,
		Payload: []byte{0x12, 0x34, 0x00, 0x0A, 0x00, 0x14, 0x00, 0x00},
	})

	p := nextPatch(t, sub)
	if p.SuperGroup != 0x1234 {
		t.Errorf("patch SuperGroup = %#x, want 0x1234", p.SuperGroup)
	}
	if p.Vendor != "motorola" || !p.Add {
		t.Errorf("patch Vendor=%q Add=%v, want motorola/true", p.Vendor, p.Add)
	}
	want := []uint32{10, 20}
	if len(p.Members) != len(want) || p.Members[0] != 10 || p.Members[1] != 20 {
		t.Errorf("patch Members = %v, want %v", p.Members, want)
	}
}

func TestControlChannelPublishesHarrisRegroup(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(MACPDU{
		Opcode:  OpVendorGroupRegroup,
		MFID:    MFIDHarris,
		Payload: []byte{0x05, 0x55, 0x00, 0xBE, 0xEF},
	})

	p := nextPatch(t, sub)
	if p.SuperGroup != 0x0555 || p.Vendor != "harris" || !p.Add {
		t.Errorf("Harris patch = %+v, want super 0x555 / harris / add", p)
	}
}
