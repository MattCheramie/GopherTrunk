package tier3

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestVendorFromFID(t *testing.T) {
	cases := []struct {
		fid  uint8
		want Vendor
	}{
		{FIDStandard, VendorStandard},
		{FIDMotorola, VendorMotorola},
		{FIDConnectPlus, VendorConnectPlus},
		{FIDHytera, VendorHytera},
		{FIDHyteraAlt, VendorHytera},
		{0x7F, VendorStandard},
	}
	for _, c := range cases {
		if got := VendorFromFID(c.fid); got != c.want {
			t.Errorf("VendorFromFID(%#x) = %v, want %v", c.fid, got, c.want)
		}
	}
}

// tvGrantPayload builds the 8-octet TVGrant CSBK payload.
func tvGrantPayload(group, source uint32, lcn uint8) [8]byte {
	return [8]byte{
		0x00,
		byte(group >> 16), byte(group >> 8), byte(group),
		byte(source >> 16), byte(source >> 8), byte(source),
		lcn & 0x7F,
	}
}

func TestMotorolaVendorGrantEmitted(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		FrequencyHz: 851_000_000,
		Resolver:    TableBandPlan{4: 462_550_000},
	})
	// Motorola Capacity Plus/Max voice grant: FID 0x10, ETSI-shaped
	// 8-octet payload.
	csbk := CSBK{
		LB:      true,
		Opcode:  OpTVGrant,
		FID:     FIDMotorola,
		Payload: tvGrantPayload(0x1234, 0x5678, 4),
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("kind = %s, want grant", ev.Kind)
		}
		g := ev.Payload.(trunking.Grant)
		if g.GroupID != 0x1234 || g.SourceID != 0x5678 {
			t.Errorf("grant ids = tg %d src %d", g.GroupID, g.SourceID)
		}
		if g.FrequencyHz != 462_550_000 {
			t.Errorf("grant freq = %d, want 462550000", g.FrequencyHz)
		}
		if g.Protocol != "dmr-tier3" {
			t.Errorf("grant protocol = %q", g.Protocol)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event from a Motorola vendor CSBK")
	}
}

// TestVendorCSBKNotMisparsedAsStandard is the regression guard for the
// FID-aware dispatch: a Connect Plus CSBK whose opcode collides with
// the standard TalkGroup Voice Channel Grant must NOT be force-decoded
// into a (garbage) standard grant.
func TestVendorCSBKNotMisparsedAsStandard(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		FrequencyHz: 851_000_000,
		Resolver:    TableBandPlan{4: 462_550_000}, // would resolve if misparsed
	})
	csbk := CSBK{
		LB:      true,
		Opcode:  OpTVGrant, // 0x30 — collides with the standard opcode
		FID:     FIDConnectPlus,
		Payload: tvGrantPayload(0x1234, 0x5678, 4),
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		t.Fatalf("vendor CSBK produced an event %s — should have been routed to the vendor handler", ev.Kind)
	case <-time.After(150 * time.Millisecond):
		// No event: the FID-aware dispatch correctly withheld a
		// standard grant for an unvalidated vendor payload.
	}
}

func TestMotorolaVendorSysInfoLocks(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, FrequencyHz: 851_000_000})
	csbk := CSBK{
		LB:      true,
		Opcode:  OpSysInfo,
		FID:     FIDMotorola,
		Payload: [8]byte{0xCA, 0xFE, 0x01, 0x07, 0xFF, 0, 0, 0},
	}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 3, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		if ls := ev.Payload.(LockState); ls.SystemID != 0xCAFE {
			t.Errorf("system id = %#x, want 0xCAFE", ls.SystemID)
		}
	case <-time.After(time.Second):
		t.Fatal("no lock event from a Motorola vendor system-info CSBK")
	}
	// The rest channel is taken from the system-info SiteID octet.
	if cc.restChannel != 0x07 {
		t.Errorf("rest channel = %d, want 7", cc.restChannel)
	}
}
