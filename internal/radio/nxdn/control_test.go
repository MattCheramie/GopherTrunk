package nxdn

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestParseVCallAssign(t *testing.T) {
	// ServiceOptions=0x01, group=0x00C8, source=0x1234, channel=0x0007.
	p := [8]byte{0x01, 0x00, 0xC8, 0x12, 0x34, 0x00, 0x07, 0x00}
	got := ParseVCallAssign(p)
	if got.ServiceOptions != 0x01 || got.GroupAddress != 0x00C8 ||
		got.SourceID != 0x1234 || got.Channel != 0x0007 {
		t.Errorf("ParseVCallAssign = %+v", got)
	}
}

func TestControlChannelEmitsLockOnSiteInfo(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 467_000_000, Rate9600)
	lich := ParseLICH(AssembleLICH(LICH{RFCh: RFChControl, FCT: FCTNSACCH, Direction: DirectionOutbound}))
	cac := &CACMessage{Type: RCCHSITEINFO, Payload: [8]byte{0x12, 0x34, 0x05, 0x06, 0xCA, 0xFE, 0, 0}}
	cc.IngestFrame(lich, cac)

	select {
	case ev := <-sub.C:
		ls := ev.Payload.(LockState)
		if ls.SystemID != 0xCAFE || ls.SiteID != 0x0506 {
			t.Errorf("payload = %+v", ls)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelEmitsGrantOnVCallAssign(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 467_000_000, Rate9600)
	cc.SetSystemName("Metro")
	// Channel-1-indexed linear plan: channel 3 → 461_000_000 + 2*12_500.
	cc.SetBandPlan(LinearBandPlan{BaseHz: 461_000_000, SpacingHz: 12_500, Offset: 1})

	lich := ParseLICH(AssembleLICH(LICH{RFCh: RFChControl, FCT: FCTNSACCH, Direction: DirectionOutbound}))
	// VCALL_ASSGN: ServiceOptions, group=0x0064, source=0x1092, channel=3.
	cac := &CACMessage{Type: RCCHVCALLASGN, Payload: [8]byte{0x00, 0x00, 0x64, 0x10, 0x92, 0x00, 0x03, 0x00}}
	cc.IngestFrame(lich, cac)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("event kind = %s, want grant", ev.Kind)
		}
		g := ev.Payload.(trunking.Grant)
		if g.Protocol != "nxdn" || g.System != "Metro" {
			t.Errorf("grant identity = %+v", g)
		}
		if g.GroupID != 0x0064 || g.SourceID != 0x1092 || g.ChannelNum != 3 {
			t.Errorf("grant ids = group %d src %d chan %d", g.GroupID, g.SourceID, g.ChannelNum)
		}
		if g.FrequencyHz != 461_025_000 {
			t.Errorf("grant freq = %d, want 461025000", g.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event")
	}
}

func TestControlChannelDropsGrantWithoutBandPlan(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 467_000_000, Rate9600)
	// No SetBandPlan → grant has no frequency to tune → dropped.
	lich := ParseLICH(AssembleLICH(LICH{RFCh: RFChControl, FCT: FCTNSACCH, Direction: DirectionOutbound}))
	cac := &CACMessage{Type: RCCHVCALLASGN, Payload: [8]byte{0x00, 0x00, 0x64, 0x10, 0x92, 0x00, 0x03, 0x00}}
	cc.IngestFrame(lich, cac)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelIgnoresTrafficLICH(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 467_000_000, Rate9600)
	lich := ParseLICH(AssembleLICH(LICH{RFCh: RFChTraffic, FCT: FCTNUDCH}))
	cac := &CACMessage{Type: RCCHSITEINFO, Payload: [8]byte{}}
	cc.IngestFrame(lich, cac)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelIgnoresBadParity(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 467_000_000, Rate9600)
	lich := LICH{RFCh: RFChControl, ParityOK: false}
	cac := &CACMessage{Type: RCCHSITEINFO}
	cc.IngestFrame(lich, cac)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 467_000_000, Rate4800)
	lich := ParseLICH(AssembleLICH(LICH{RFCh: RFChControl}))
	cc.IngestFrame(lich, &CACMessage{Type: RCCHCCH})
	<-sub.C // CCLocked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Errorf("kind = %s, want cc.lost", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost")
	}
}
