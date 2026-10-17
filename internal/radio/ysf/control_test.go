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

// grantsFrom drains the bus for a short window and returns the grants seen.
func grantsFrom(t *testing.T, sub *events.Subscription) []trunking.Grant {
	t.Helper()
	var out []trunking.Grant
	deadline := time.After(150 * time.Millisecond)
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
			out = append(out, g)
		case <-deadline:
			return out
		}
	}
}

// TestControlChannelGrantsFromOnAirFrames is the #1261 regression: a dibit
// stream of real frames (MMDVMHost's encoded FICHs) through the live Process
// path publishes one grant per transmission. Before, Process ran only the
// sync detector and the FICH model was fabricated, so a YSF control channel
// locked but never granted.
func TestControlChannelGrantsFromOnAirFrames(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "demo-repeater", FrequencyHz: 444_525_000})
	body := make([]uint8, PayloadDibits)
	var stream []uint8
	// Two transmissions: header, communications, terminator.
	for n := 0; n < 2; n++ {
		stream = append(stream, buildAirFrame(t, airFICHVectors[0].onair, body)...)
		for i := 0; i < 3; i++ {
			stream = append(stream, buildAirFrame(t, airFICHVectors[1].onair, body)...)
		}
		stream = append(stream, buildAirFrame(t, airFICHVectors[2].onair, body)...)
	}
	stream = append(stream, make([]uint8, FrameDibits)...) // flush the last frame
	for i := 0; i < len(stream); i += 97 {
		end := min(i+97, len(stream))
		cc.Process(stream[i:end], i)
	}

	gs := grantsFrom(t, sub)
	if len(gs) != 2 {
		t.Fatalf("got %d grants, want 2 (one per transmission)", len(gs))
	}
	for _, g := range gs {
		if g.Protocol != "ysf" || g.System != "demo-repeater" || g.FrequencyHz != 444_525_000 || g.GroupID != 0 {
			t.Errorf("grant = %+v, want ysf / demo-repeater / 444525000 / DG-ID 0", g)
		}
	}
}

// TestControlChannelLateEntryOnDGID: a call joined mid-transmission (no
// header) grants from its first communications frame, on the FICH's DG-ID.
func TestControlChannelLateEntryOnDGID(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	body := make([]uint8, PayloadDibits)
	stream := buildAirFrame(t, airFICHVectors[3].onair, body) // comms, DG-ID 11
	stream = append(stream, make([]uint8, FrameDibits)...)
	cc.Process(stream, 0)

	gs := grantsFrom(t, sub)
	if len(gs) != 1 || gs[0].GroupID != 11 {
		t.Fatalf("grants = %+v, want one on DG-ID 11", gs)
	}
}

func TestControlChannelDuplicateHeaderSuppressed(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	header := AirFICH{FI: AirFIHeader, DT: AirDTVDMode2, DGID: 5}
	cc.ProcessAirFICH(header)
	cc.ProcessAirFICH(header)
	cc.ProcessAirFICH(AirFICH{FI: AirFIComms, DT: AirDTVDMode2, DGID: 5})

	if gs := grantsFrom(t, sub); len(gs) != 1 {
		t.Errorf("got %d grants for one call, want 1", len(gs))
	}
}

func TestControlChannelTerminatorClearsGrantState(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	header := AirFICH{FI: AirFIHeader, DT: AirDTVDMode2, DGID: 5}
	cc.ProcessAirFICH(header)
	cc.ProcessAirFICH(AirFICH{FI: AirFITerminator, DT: AirDTVDMode2, DGID: 5})
	cc.ProcessAirFICH(header)

	if gs := grantsFrom(t, sub); len(gs) != 2 {
		t.Errorf("got %d grants (Header → Terminator → Header), want 2", len(gs))
	}
}

func TestControlChannelSkipsIndividualAndDataCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "x", FrequencyHz: 1})
	cc.ProcessAirFICH(AirFICH{FI: AirFIHeader, CM: airCMIndividual, DT: AirDTVDMode2, DGID: 5})
	cc.ProcessAirFICH(AirFICH{FI: AirFIHeader, DT: AirDTDataFR, DGID: 6})

	if gs := grantsFrom(t, sub); len(gs) != 0 {
		t.Errorf("individual / data-FR calls granted: %+v", gs)
	}
}
