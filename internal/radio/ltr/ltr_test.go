package ltr

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestStatusAssembleParseRoundTrip(t *testing.T) {
	in := Status{
		Sync: true, Area: 7, Group: true,
		Channel: 5, Home: 5, GroupID: 42, Free: 3, FCS: 0xABC,
	}
	bytes := AssembleStatus(in)
	if len(bytes) != 6 {
		t.Fatalf("AssembleStatus = %d bytes", len(bytes))
	}
	got, err := ParseStatus(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestStatusFromBitsRoundTrip(t *testing.T) {
	in := Status{
		Sync: true, Area: 1, Group: false,
		Channel: 12, Home: 12, GroupID: 200, Free: 7, FCS: 0x123,
	}
	bits := StatusBits(in)
	if len(bits) != 41 {
		t.Fatalf("StatusBits len = %d", len(bits))
	}
	got, err := StatusFromBits(bits)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestStatusFromBitsRejectsBadLength(t *testing.T) {
	if _, err := StatusFromBits(make([]byte, 40)); err == nil {
		t.Error("expected error for 40 bits")
	}
}

func TestIsActive(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want bool
	}{
		{"group + non-zero id", Status{Group: true, GroupID: 42}, true},
		{"group set + zero id", Status{Group: true, GroupID: 0}, false},
		{"group clear", Status{Group: false, GroupID: 42}, false},
		{"both clear", Status{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.IsActive(); got != c.want {
				t.Errorf("IsActive() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestLinearBandPlan(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 451_000_000, SpacingHz: 12_500, Offset: -1}
	// Channel 1 with offset -1 → idx 0 → BaseHz exactly.
	if hz, _ := bp.Frequency(1); hz != 451_000_000 {
		t.Errorf("ch 1 → %d, want 451M", hz)
	}
	if hz, _ := bp.Frequency(20); hz != 451_237_500 {
		t.Errorf("ch 20 → %d", hz)
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 0}).Frequency(1); err == nil {
		t.Error("zero spacing should error")
	}
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 12_500, Offset: -10}).Frequency(1); err == nil {
		t.Error("negative effective offset should error")
	}
}

func TestTableBandPlan(t *testing.T) {
	bp := TableBandPlan{1: 461_000_000, 5: 461_062_500, 12: 461_175_000}
	if hz, _ := bp.Frequency(5); hz != 461_062_500 {
		t.Errorf("ch 5 → %d", hz)
	}
	if _, err := bp.Frequency(99); err == nil {
		t.Error("missing channel should error")
	}
}

// activeStatus returns a status word that announces a call for the
// supplied group on the supplied channel.
func activeStatus(group uint16, channel, home, area uint8) Status {
	return Status{
		Sync: true, Area: area, Group: true,
		Channel: channel, Home: home, GroupID: group,
		Free: 0, FCS: 0,
	}
}

func TestControlChannelEmitsLockOnFirstStatus(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestLTR",
		FrequencyHz: 461_000_000,
	})
	cc.Ingest(activeStatus(0, 1, 1, 3))

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok || ls.FrequencyHz != 461_000_000 || ls.Area != 3 || ls.Repeater != 1 {
			t.Errorf("lock state = %+v", ev.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelDropsBadSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(Status{Sync: false, Group: true, GroupID: 42, Channel: 1, Home: 1})

	select {
	case ev := <-sub.C:
		t.Errorf("unsynced status produced an event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelFiltersByArea(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus: bus, SystemName: "X", FrequencyHz: 1,
		Area: 5, // accept only area 5
	})
	// Wrong area: dropped.
	cc.Ingest(activeStatus(42, 3, 3, 7))
	// Right area: emits.
	cc.Ingest(activeStatus(42, 3, 3, 5))

	first := <-sub.C
	if first.Kind != events.KindCCLocked {
		t.Errorf("kind = %s, want cc.locked", first.Kind)
	}
	if ls, ok := first.Payload.(LockState); !ok || ls.Area != 5 {
		t.Errorf("lock area = %+v", first.Payload)
	}
}

func TestControlChannelPublishesGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestLTR",
		FrequencyHz: 461_000_000,
		Resolver: LinearBandPlan{
			BaseHz: 461_000_000, SpacingHz: 12_500, Offset: -1,
		},
	})
	cc.Ingest(activeStatus(42, 3, 3, 1))
	// First event is the cc.locked.
	if ev := <-sub.C; ev.Kind != events.KindCCLocked {
		t.Fatalf("first kind = %s, want cc.locked", ev.Kind)
	}
	// Second event is the grant.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("second kind = %s", ev.Kind)
		}
		g, ok := ev.Payload.(trunking.Grant)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if g.System != "TestLTR" || g.Protocol != "ltr" {
			t.Errorf("identity = %+v", g)
		}
		if g.GroupID != 42 || g.ChannelNum != 3 {
			t.Errorf("group/channel = %+v", g)
		}
		// Channel 3 with offset -1, spacing 12.5 kHz → +25 kHz from base.
		if g.FrequencyHz != 461_025_000 {
			t.Errorf("freq = %d, want 461_025_000", g.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant")
	}
}

func TestControlChannelDoesNotRepublishSameGroup(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(activeStatus(42, 1, 1, 0))
	<-sub.C // cc.locked
	<-sub.C // grant for tg 42

	// Several more identical status words during the same call: no
	// new grant events.
	for i := 0; i < 5; i++ {
		cc.Ingest(activeStatus(42, 1, 1, 0))
	}
	select {
	case ev := <-sub.C:
		t.Errorf("repeat status republished: %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}

	// A different group → new grant.
	cc.Ingest(activeStatus(99, 2, 1, 0))
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Errorf("kind = %s, want grant", ev.Kind)
		}
		if g, ok := ev.Payload.(trunking.Grant); ok && g.GroupID != 99 {
			t.Errorf("group = %d, want 99", g.GroupID)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant for new group")
	}
}

func TestControlChannelGrantWithoutResolverHasZeroFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(activeStatus(42, 1, 1, 0))
	<-sub.C // cc.locked
	ev := <-sub.C
	g := ev.Payload.(trunking.Grant)
	if g.FrequencyHz != 0 {
		t.Errorf("freq = %d, want 0 (no resolver)", g.FrequencyHz)
	}
	if g.ChannelNum != 1 {
		t.Errorf("channel = %d, want 1", g.ChannelNum)
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "X", FrequencyHz: 1})
	cc.Ingest(activeStatus(0, 1, 1, 0))
	<-sub.C // cc.locked

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
