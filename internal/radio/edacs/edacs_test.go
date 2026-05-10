package edacs

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestCCWAssembleParseRoundTrip(t *testing.T) {
	in := CCW{
		Command: CmdGroupVoiceGrant,
		Status:  0x3, // emergency + encrypted
		Address: 0x1234,
		LCN:     17,
		Aux:     0x055,
	}
	bytes := AssembleCCW(in)
	if len(bytes) != 5 {
		t.Fatalf("AssembleCCW = %d bytes", len(bytes))
	}
	got, err := ParseCCW(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestCCWFromBitsRoundTrip(t *testing.T) {
	in := CCW{Command: CmdSystemID, Status: 0xA, Address: 0xBEEF, LCN: 4, Aux: 0x123}
	bits := CCWBits(in)
	if len(bits) != 40 {
		t.Fatalf("CCWBits len = %d", len(bits))
	}
	got, err := CCWFromBits(bits)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v", got)
	}
}

func TestCCWFromBitsRejectsBadLength(t *testing.T) {
	if _, err := CCWFromBits(make([]byte, 39)); err == nil {
		t.Error("expected error for 39 bits")
	}
}

func TestCommandStringCoversKnownAndUnknown(t *testing.T) {
	cases := map[Command]string{
		CmdIdle:            "Idle",
		CmdGroupVoiceGrant: "GroupVoiceGrant",
		CmdProVoiceGrant:   "ProVoiceGrant",
		CmdSystemID:        "SystemID",
		CmdAdjacentSite:    "AdjacentSite",
		Command(0xC):       "Command(C)",
	}
	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("Command(%X).String() = %s, want %s", uint8(c), got, want)
		}
	}
}

func TestStatusFlags(t *testing.T) {
	c := CCW{Status: 0x3}
	if !c.IsEncrypted() || !c.IsEmergency() {
		t.Errorf("status 0x3 should set both flags: enc=%v emer=%v",
			c.IsEncrypted(), c.IsEmergency())
	}
	plain := CCW{Status: 0x0}
	if plain.IsEncrypted() || plain.IsEmergency() {
		t.Errorf("status 0x0 should clear both flags")
	}
}

func TestAsGroupVoiceGrant(t *testing.T) {
	c := CCW{Command: CmdGroupVoiceGrant, Status: 0x1, Address: 0x1234, LCN: 5}
	g, ok := c.AsGroupVoiceGrant()
	if !ok {
		t.Fatal("expected grant")
	}
	if g.GroupAddress != 0x1234 || g.LCN != 5 || !g.Encrypted || g.Emergency || g.ProVoice {
		t.Errorf("grant = %+v", g)
	}

	pv := CCW{Command: CmdProVoiceGrant, Status: 0x2, Address: 0x4321, LCN: 9}
	pg, ok := pv.AsGroupVoiceGrant()
	if !ok || !pg.ProVoice || !pg.Emergency || pg.Encrypted {
		t.Errorf("provoice grant = %+v ok=%v", pg, ok)
	}

	// Idle isn't a grant.
	if _, ok := (CCW{Command: CmdIdle}).AsGroupVoiceGrant(); ok {
		t.Error("idle CCW reported a grant")
	}
}

func TestAsSystemIDAndAdjacent(t *testing.T) {
	sys := CCW{Command: CmdSystemID, Address: 0xCAFE, Aux: 0x42}
	if s, ok := sys.AsSystemID(); !ok || s.ID != 0xCAFE || s.Aux != 0x42 {
		t.Errorf("system id = %+v ok=%v", s, ok)
	}

	adj := CCW{Command: CmdAdjacentSite, Address: 0x0017, LCN: 3}
	if a, ok := adj.AsAdjacentSite(); !ok || a.SiteID != 0x17 || a.LCN != 3 {
		t.Errorf("adjacent = %+v ok=%v", a, ok)
	}
}

func TestIsIdle(t *testing.T) {
	if !(CCW{Command: CmdIdle}).IsIdle() {
		t.Error("CmdIdle should be idle")
	}
	if (CCW{Command: CmdGroupVoiceGrant}).IsIdle() {
		t.Error("voice grant flagged as idle")
	}
}

func TestLinearBandPlan(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 851_000_000, SpacingHz: 25_000, Offset: 0}
	if hz, _ := bp.Frequency(0); hz != 851_000_000 {
		t.Errorf("LCN 0 → %d", hz)
	}
	if hz, _ := bp.Frequency(10); hz != 851_250_000 {
		t.Errorf("LCN 10 → %d", hz)
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 0}).Frequency(1); err == nil {
		t.Error("zero spacing should error")
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 25_000, Offset: -3}).Frequency(1); err == nil {
		t.Error("negative effective offset should error")
	}
}

func TestTableBandPlan(t *testing.T) {
	bp := TableBandPlan{1: 154_115_000, 2: 154_205_000}
	if hz, _ := bp.Frequency(1); hz != 154_115_000 {
		t.Errorf("LCN 1 → %d", hz)
	}
	if _, err := bp.Frequency(99); err == nil {
		t.Error("missing LCN should error")
	}
}

func TestSyncDetectorMatchesCleanSync(t *testing.T) {
	det := NewSyncDetector(OutboundSyncBits(), 0)
	stream := make([]byte, 80)
	copy(stream[15:], OutboundSyncBits())
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 || hits[0] != 15+SyncBits-1 {
		t.Errorf("hits = %v, want [%d]", hits, 15+SyncBits-1)
	}
}

func TestSyncDetectorTolerance(t *testing.T) {
	stream := make([]byte, 80)
	copy(stream[5:], OutboundSyncBits())
	stream[7] ^= 1
	stream[15] ^= 1

	det := NewSyncDetector(OutboundSyncBits(), 2)
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 {
		t.Errorf("hits = %v, want 1 with tolerance 2", hits)
	}

	det0 := NewSyncDetector(OutboundSyncBits(), 0)
	hits0, _ := det0.Process(nil, stream, 0)
	if len(hits0) != 0 {
		t.Errorf("hits = %v, want 0 with tolerance 0", hits0)
	}
}

func TestControlChannelEmitsLockOnSystemID(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
	})
	cc.Ingest(CCW{Command: CmdSystemID, Address: 0xCAFE, Aux: 0x12})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok || ls.SystemID != 0xCAFE || ls.FrequencyHz != 851_000_000 {
			t.Errorf("lock state = %+v", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelPublishesGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Resolver: LinearBandPlan{
			BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 0,
		},
	})
	cc.Ingest(CCW{
		Command: CmdGroupVoiceGrant, Status: 0x3,
		Address: 0x1234, LCN: 8,
	})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("kind = %s", ev.Kind)
		}
		g, ok := ev.Payload.(trunking.Grant)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if g.System != "TestSys" || g.Protocol != "edacs" {
			t.Errorf("grant identity = %+v", g)
		}
		if g.GroupID != 0x1234 || g.ChannelNum != 8 {
			t.Errorf("grant group/lcn = %+v", g)
		}
		if g.FrequencyHz != 866_200_000 {
			t.Errorf("grant freq = %d, want 866_200_000", g.FrequencyHz)
		}
		if !g.Encrypted || !g.Emergency {
			t.Errorf("expected encrypted+emergency flags: %+v", g)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event")
	}
}

func TestControlChannelPropagatesProVoiceFlag(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Resolver:    LinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000},
	})
	cc.Ingest(CCW{Command: CmdProVoiceGrant, Address: 0x4321, LCN: 9})

	select {
	case ev := <-sub.C:
		g, ok := ev.Payload.(trunking.Grant)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if !g.ProVoice {
			t.Errorf("ProVoice flag not set on trunking.Grant: %+v", g)
		}
		if g.GroupID != 0x4321 || g.ChannelNum != 9 {
			t.Errorf("identity wrong: %+v", g)
		}
		// Plain GroupVoiceGrants must not have ProVoice set; verify the
		// flag is grant-type-specific, not always-on.
		cc.Ingest(CCW{Command: CmdGroupVoiceGrant, Address: 0x100, LCN: 1})
		ev2 := <-sub.C
		if g2 := ev2.Payload.(trunking.Grant); g2.ProVoice {
			t.Errorf("regular GroupVoiceGrant flagged ProVoice: %+v", g2)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant event")
	}
}

func TestControlChannelGrantWithoutResolverHasZeroFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(CCW{Command: CmdGroupVoiceGrant, Address: 0x100, LCN: 1})
	ev := <-sub.C
	g := ev.Payload.(trunking.Grant)
	if g.FrequencyHz != 0 {
		t.Errorf("freq = %d, want 0 (no resolver)", g.FrequencyHz)
	}
	if g.ChannelNum != 1 {
		t.Errorf("LCN = %d, want 1", g.ChannelNum)
	}
}

func TestControlChannelIgnoresIdle(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(CCW{Command: CmdIdle, Aux: 0x42})

	select {
	case ev := <-sub.C:
		t.Errorf("idle CCW produced an event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(CCW{Command: CmdSystemID, Address: 0x42})
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
