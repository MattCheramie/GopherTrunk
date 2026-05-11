package ysf

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// streamWithFSWAt returns a 480-dibit YSF frame skeleton with the
// FSW placed at offset and the rest of the buffer zeroed.
func streamWithFSWAt(offset int) []uint8 {
	out := make([]uint8, FrameDibits)
	copy(out[offset:], FSWPattern)
	return out
}

func TestControlChannelEmitsLockOnSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 444_525_000)
	cc.Process(streamWithFSWAt(FSWOffset), 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload type = %T, want LockState", ev.Payload)
		}
		if ls.FrequencyHz != 444_525_000 {
			t.Errorf("freq = %d, want 444525000", ls.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no event published")
	}
}

func TestControlChannelDoesNotRelock(t *testing.T) {
	// Two FSW hits in the same stream should produce exactly one
	// cc.locked event (no relock until MarkLost).
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 1)
	stream := make([]uint8, FrameDibits*2)
	copy(stream[10:], FSWPattern)
	copy(stream[FrameDibits+10:], FSWPattern)
	cc.Process(stream, 0)

	count := 0
	deadline := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				count++
			}
		case <-deadline:
			if count != 1 {
				t.Errorf("got %d cc.locked events, want 1 (no relock)", count)
			}
			return
		}
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 145_550_000)
	cc.Process(streamWithFSWAt(0), 0)
	<-sub.C // CCLocked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Errorf("kind = %s, want cc.lost", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if ls.FrequencyHz != 145_550_000 {
			t.Errorf("freq = %d, want 145550000", ls.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost event")
	}
}

func TestControlChannelMarkLostNoOpWhenUnlocked(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 1)
	cc.MarkLost()

	select {
	case ev := <-sub.C:
		t.Errorf("MarkLost on never-locked channel emitted %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestLockStateSatisfiesTrunkingLockedPayload(t *testing.T) {
	// Compile-time check: the hunter consumes lock events through
	// the trunking.LockedPayload interface, so YSF's LockState must
	// satisfy it.
	var lp trunking.LockedPayload = LockState{FrequencyHz: 100}
	if lp.LockedFrequencyHz() != 100 {
		t.Errorf("LockedFrequencyHz = %d, want 100", lp.LockedFrequencyHz())
	}
	if lp.LockedNAC() != 0 {
		t.Errorf("LockedNAC = %d, want 0 (YSF has no NAC)", lp.LockedNAC())
	}
}

func TestControlChannelHeaderFICHEmitsGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "demo-repeater",
		FrequencyHz: 444_525_000,
	})
	cc.ProcessFICH(FICH{
		FrameType:   FrameTypeHeader,
		CallType:    CallTypeGroup,
		DataType:    DataTypeVDMode2,
		SquelchMode: true,
		SquelchCode: 23,
	})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g, ok := ev.Payload.(trunking.Grant)
			if !ok {
				t.Fatalf("payload type = %T, want trunking.Grant", ev.Payload)
			}
			if g.Protocol != "ysf" {
				t.Errorf("protocol = %q, want ysf", g.Protocol)
			}
			if g.System != "demo-repeater" {
				t.Errorf("system = %q, want demo-repeater", g.System)
			}
			if g.GroupID != 23 {
				t.Errorf("group_id = %d, want 23 (DG-ID)", g.GroupID)
			}
			if g.FrequencyHz != 444_525_000 {
				t.Errorf("freq = %d, want 444525000", g.FrequencyHz)
			}
			return
		case <-deadline:
			t.Fatal("no grant event published")
		}
	}
}

func TestControlChannelDuplicateHeaderSuppressed(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	header := FICH{FrameType: FrameTypeHeader, CallType: CallTypeGroup, SquelchMode: true, SquelchCode: 5}
	cc.ProcessFICH(header)
	cc.ProcessFICH(header)

	grants := 0
	deadline := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				grants++
			}
		case <-deadline:
			if grants != 1 {
				t.Errorf("got %d grants for duplicate Header, want 1", grants)
			}
			return
		}
	}
}

func TestControlChannelTerminatorClearsGrantState(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	header := FICH{FrameType: FrameTypeHeader, CallType: CallTypeGroup, SquelchMode: true, SquelchCode: 5}
	cc.ProcessFICH(header)
	cc.ProcessFICH(FICH{FrameType: FrameTypeTerminator, CallType: CallTypeGroup})
	cc.ProcessFICH(header)

	grants := 0
	deadline := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				grants++
			}
		case <-deadline:
			if grants != 2 {
				t.Errorf("got %d grants (Header → Terminator → Header), want 2", grants)
			}
			return
		}
	}
}

func TestControlChannelPrivateCallSkipsGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	cc.ProcessFICH(FICH{
		FrameType:   FrameTypeHeader,
		CallType:    CallTypeRadioID,
		SquelchMode: true,
		SquelchCode: 5,
	})

	select {
	case ev := <-sub.C:
		t.Errorf("private call should not fire grant; got %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}
