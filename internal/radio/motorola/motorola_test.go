package motorola

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestOSWAssembleParseRoundTrip(t *testing.T) {
	in := OSW{Address: 0x1234, Command: 0x308A}
	bytes := AssembleOSW(in)
	if len(bytes) != 4 {
		t.Fatalf("AssembleOSW = %d bytes", len(bytes))
	}
	got, err := ParseOSW(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestOSWFromBitsRoundTrip(t *testing.T) {
	in := OSW{Address: 0xABCD, Command: 0x080F}
	bits := OSWBits(in)
	if len(bits) != 32 {
		t.Fatalf("OSWBits len = %d", len(bits))
	}
	got, err := OSWFromBits(bits)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v", got)
	}
}

func TestOpcodeAndLCN(t *testing.T) {
	// Command = 0x308A → opcode 0x308 (Group Voice Channel Grant), LCN 0xA
	o := OSW{Address: 0x1234, Command: 0x308A}
	if o.Opcode() != OpGroupVoiceChannelGrant {
		t.Errorf("Opcode = %s, want GroupVoiceChannelGrant", o.Opcode())
	}
	if o.LCN() != 0xA {
		t.Errorf("LCN = %X, want A", o.LCN())
	}
}

func TestAsGroupVoiceChannelGrant(t *testing.T) {
	o := OSW{Address: 0x1234, Command: 0x308A}
	g, ok := o.AsGroupVoiceChannelGrant()
	if !ok {
		t.Fatal("expected grant")
	}
	if g.GroupAddress != 0x1234 || g.LCN != 0xA {
		t.Errorf("grant = %+v", g)
	}

	// Idle isn't a grant.
	if _, ok := (OSW{Command: 0x28D0}).AsGroupVoiceChannelGrant(); ok {
		t.Error("idle OSW reported a grant")
	}
}

func TestAsSystemIDAndAdjacent(t *testing.T) {
	sysOSW := OSW{Address: 0xCAFE, Command: 0x0805}
	if s, ok := sysOSW.AsSystemID(); !ok || s.ID != 0xCAFE || s.Class != 5 {
		t.Errorf("system id = %+v ok=%v", s, ok)
	}

	adjOSW := OSW{Address: 0x0042, Command: 0x31B7}
	if a, ok := adjOSW.AsAdjacentSite(); !ok || a.SiteID != 0x42 || a.LCN != 7 {
		t.Errorf("adjacent = %+v ok=%v", a, ok)
	}
}

func TestIsIdle(t *testing.T) {
	for _, c := range []uint16{0x28D0, 0x290F} {
		if !(OSW{Command: c}).IsIdle() {
			t.Errorf("Command %04X should be idle", c)
		}
	}
	if (OSW{Command: 0x308A}).IsIdle() {
		t.Error("voice grant flagged as idle")
	}
}

func TestOpcodeString(t *testing.T) {
	cases := map[Opcode]string{
		OpGroupVoiceChannelGrant: "GroupVoiceChannelGrant",
		OpAdjacentSiteStatus:     "AdjacentSiteStatus",
		OpSystemIDExtended:       "SystemIDExtended",
		OpIdle1:                  "Idle",
		Opcode(0x999):            "Opcode(999)",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("Opcode(%X).String() = %s, want %s", uint16(op), got, want)
		}
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
	bp2 := LinearBandPlan{BaseHz: 851_000_000, SpacingHz: 25_000, Offset: -3}
	if _, err := bp2.Frequency(1); err == nil {
		t.Error("negative effective offset should error")
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 0}).Frequency(1); err == nil {
		t.Error("zero spacing should error")
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
	copy(stream[20:], OutboundSyncBits())
	hits, _ := det.Process(nil, stream, 0)
	if len(hits) != 1 || hits[0] != 20+SyncBits-1 {
		t.Errorf("hits = %v, want [%d]", hits, 20+SyncBits-1)
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

	det2 := NewSyncDetector(OutboundSyncBits(), 0)
	hits2, _ := det2.Process(nil, stream, 0)
	if len(hits2) != 0 {
		t.Errorf("hits = %v, want 0 with tolerance 0", hits2)
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
	cc.Ingest(OSW{Address: 0xCAFE, Command: 0x0800})

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
	cc.Ingest(OSW{Address: 0x1234, Command: 0x3088}) // grant; LCN=8

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("kind = %s", ev.Kind)
		}
		g, ok := ev.Payload.(trunking.Grant)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if g.System != "TestSys" || g.Protocol != "motorola" {
			t.Errorf("grant identity = %+v", g)
		}
		if g.GroupID != 0x1234 || g.ChannelNum != 8 {
			t.Errorf("grant group/lcn = %+v", g)
		}
		if g.FrequencyHz != 866_200_000 {
			t.Errorf("grant freq = %d, want 866_200_000", g.FrequencyHz)
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
	cc.Ingest(OSW{Address: 0x100, Command: 0x3081})
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
	cc.Ingest(OSW{Command: 0x28D0}) // idle
	cc.Ingest(OSW{Command: 0x290F}) // idle

	select {
	case ev := <-sub.C:
		t.Errorf("idle OSW produced an event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(OSW{Address: 0x42, Command: 0x0800})
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
