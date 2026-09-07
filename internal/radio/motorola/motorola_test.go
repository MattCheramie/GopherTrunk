package motorola

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestOSWStatusFlagAccessors(t *testing.T) {
	// Talkgroup 0xB010 with encrypted flag (0x8) and emergency
	// option 2 in the low nibble: raw address 0xB01A.
	o := OSW{Address: 0xB01A, Group: true, Command: 0x8E}
	if o.Talkgroup() != 0xB010 {
		t.Errorf("Talkgroup = %#x, want 0xB010", o.Talkgroup())
	}
	if !o.Encrypted() {
		t.Error("Encrypted = false, want true (flag 0x8 set)")
	}
	if !o.Emergency() {
		t.Error("Emergency = false, want true (option 2)")
	}
	clear := OSW{Address: 0xB010, Group: true}
	if clear.Encrypted() || clear.Emergency() {
		t.Error("clean address reported encrypted/emergency")
	}
}

// TestSequencerLocksOnSystemIDBroadcast: the two-OSW 0x308 + 0x1F00
// channel pair is the lock signal, carrying the system ID.
func TestSequencerLocksOnSystemIDBroadcast(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500, Now: func() time.Time { return time.Unix(0, 0) }})

	c.Ingest(OSW{Address: 0x4567, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0x1F00, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	locked, _ := drainEvents(sub)
	if len(locked) != 1 {
		t.Fatalf("locked events = %d, want 1", len(locked))
	}
	if locked[0].SystemID != 0x4567 || locked[0].FrequencyHz != 854_562_500 {
		t.Errorf("LockState = %+v", locked[0])
	}
	if topo := c.Topology(); topo.SystemID != 0x4567 {
		t.Errorf("Topology.SystemID = %#x", topo.SystemID)
	}
}

// TestSequencerPublishesGroupGrantWithSource: 0x308 pair where the
// second OSW is a group-addressed channel → grant with source RID,
// masked talkgroup, band-plan frequency and status flags.
func TestSequencerPublishesGroupGrantWithSource(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500})

	c.Ingest(OSW{Address: 0x2E9A, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0xB01A, Group: true, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	_, grants := drainEvents(sub)
	if len(grants) != 1 {
		t.Fatalf("grants = %d, want 1", len(grants))
	}
	g := grants[0]
	if g.Protocol != "motorola" || g.System != "S" {
		t.Errorf("grant identity = %s/%s", g.Protocol, g.System)
	}
	if g.GroupID != 0xB010 {
		t.Errorf("GroupID = %#x, want 0xB010 (status nibble stripped)", g.GroupID)
	}
	if g.SourceID != 0x2E9A {
		t.Errorf("SourceID = %#x, want 0x2E9A", g.SourceID)
	}
	if g.FrequencyHz != 854_562_500 {
		t.Errorf("FrequencyHz = %d, want 854562500 (channel 0x8E)", g.FrequencyHz)
	}
	if !g.Encrypted || !g.Emergency {
		t.Errorf("flags = enc %v emg %v, want true/true (address 0xB01A)", g.Encrypted, g.Emergency)
	}
	if g.ChannelNum != 0x8E {
		t.Errorf("ChannelNum = %#x, want 0x8E", g.ChannelNum)
	}
}

// TestSequencerPublishesSingleOSWUpdate: a lone group-addressed
// channel OSW is a voice update (no source) that keeps a call alive.
func TestSequencerPublishesSingleOSWUpdate(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500})

	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	_, grants := drainEvents(sub)
	if len(grants) != 1 {
		t.Fatalf("grants = %d, want 1", len(grants))
	}
	if grants[0].SourceID != 0 || grants[0].GroupID != 0xB010 {
		t.Errorf("update grant = %+v", grants[0])
	}
}

// TestGrantSourceBackfilledOntoUpdates pins issue #1143: SmartNet carries the
// calling radio ID only on the two-OSW grant that starts a call; the single-OSW
// voice updates that keep it alive omit it. The ControlChannel remembers the
// initiating grant's source and backfills it onto those updates, so the Active
// Calls source no longer blanks after the first frame.
func TestGrantSourceBackfilledOntoUpdates(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	clk := time.Unix(1000, 0)
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500, Now: func() time.Time { return clk }})

	// Initiating group grant: source 0x2E9A, talkgroup 0xB010, channel 0x8E.
	c.Ingest(OSW{Address: 0x2E9A, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0xB01A, Group: true, Command: 0x8E})
	// Two source-less voice updates, spaced out (a weak CC sends them sparsely).
	clk = clk.Add(3 * time.Second)
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	clk = clk.Add(5 * time.Second)
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	// Flush the sequencer.
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	_, grants := drainEvents(sub)
	if len(grants) != 3 {
		t.Fatalf("grants = %d, want 3 (initiating grant + 2 updates)", len(grants))
	}
	for i, g := range grants {
		if g.GroupID != 0xB010 {
			t.Fatalf("grant %d GroupID = %#x, want 0xB010", i, g.GroupID)
		}
		if g.SourceID != 0x2E9A {
			t.Errorf("grant %d SourceID = %#x, want 0x2E9A backfilled from the initiating grant (issue #1143)", i, g.SourceID)
		}
	}
}

// TestGrantSourceMemoryAgesOut: once a call's updates stop for longer than
// sourceMemoryTTL (the call ended), a later source-less update on the same
// talkgroup must NOT inherit the previous talker.
func TestGrantSourceMemoryAgesOut(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	clk := time.Unix(1000, 0)
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500, Now: func() time.Time { return clk }})

	// A call from source 0x2E9A on TG 0xB010. Flush the sequencer so the
	// initiating grant is interpreted now (interpretation lags the queue), then
	// advance the clock — otherwise the grant and the later update would be
	// interpreted back-to-back at the same instant.
	c.Ingest(OSW{Address: 0x2E9A, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	// Long silence past the TTL — the call ended.
	clk = clk.Add(sourceMemoryTTL + time.Second)
	// A lone source-less update on the same talkgroup (e.g. a new call whose
	// initiating grant was missed on a weak CC).
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	_, grants := drainEvents(sub)
	if len(grants) != 2 {
		t.Fatalf("grants = %d, want 2", len(grants))
	}
	if grants[0].SourceID != 0x2E9A {
		t.Errorf("first grant SourceID = %#x, want 0x2E9A", grants[0].SourceID)
	}
	if grants[1].SourceID != 0 {
		t.Errorf("stale update SourceID = %#x, want 0 (previous talker aged out after %s)", grants[1].SourceID, sourceMemoryTTL)
	}
}

// TestGrantSourceMemoryFollowsNewTalker: a fresh grant with a different source
// on the same talkgroup replaces the remembered talker, so its updates carry
// the new source, not the old one.
func TestGrantSourceMemoryFollowsNewTalker(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	clk := time.Unix(1000, 0)
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500, Now: func() time.Time { return clk }})

	// First talker.
	c.Ingest(OSW{Address: 0x1111, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	// Second talker keys up on the same talkgroup a moment later.
	clk = clk.Add(2 * time.Second)
	c.Ingest(OSW{Address: 0x2222, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	// A following update should carry the second talker.
	c.Ingest(OSW{Address: 0xB010, Group: true, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	_, grants := drainEvents(sub)
	if len(grants) != 3 {
		t.Fatalf("grants = %d, want 3", len(grants))
	}
	if grants[0].SourceID != 0x1111 {
		t.Errorf("grant 0 SourceID = %#x, want 0x1111", grants[0].SourceID)
	}
	if grants[1].SourceID != 0x2222 || grants[2].SourceID != 0x2222 {
		t.Errorf("after re-grant SourceIDs = %#x,%#x, want 0x2222 (new talker replaces old)", grants[1].SourceID, grants[2].SourceID)
	}
}

// TestSequencerRecordsAlternateCC: the 0x30B + 0x6000-masked pair
// advertises an alternate / adjacent control channel for the hunt
// topology, and its system ID locks.
func TestSequencerRecordsAlternateCC(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500})

	c.Ingest(OSW{Address: 0x4567, Command: CmdFirstAlternate})
	c.Ingest(OSW{Address: 0x6000 | 0x090, Command: 0x2F8}) // alt CC on channel 0x090
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})

	locked, _ := drainEvents(sub)
	if len(locked) != 1 || locked[0].SystemID != 0x4567 {
		t.Fatalf("alternate-CC broadcast did not lock: %+v", locked)
	}
	topo := c.Topology()
	if len(topo.Neighbors) != 1 || topo.Neighbors[0].LCN != 0x090 {
		t.Fatalf("neighbors = %+v, want one with LCN 0x090", topo.Neighbors)
	}
	if hz, ok := c.NeighborFrequency(0x090); !ok || hz != 854_612_500 {
		t.Errorf("NeighborFrequency(0x090) = %d, %v; want 854612500", hz, ok)
	}
}

// TestMarkLostPublishesCCLost mirrors the other protocols' lost-path
// contract.
func TestMarkLostPublishesCCLost(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 854_562_500})

	c.Ingest(OSW{Address: 0x4567, Command: CmdFirstNormal})
	c.Ingest(OSW{Address: 0x1F00, Command: 0x8E})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.Ingest(OSW{Address: 0x02F8, Command: CmdIdle})
	c.MarkLost()

	var lost bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLost {
				lost = true
			}
		default:
			if !lost {
				t.Fatal("MarkLost published no cc.lost")
			}
			return
		}
	}
}

// TestLockStateSatisfiesLockedPayload keeps the cchunt supervisor
// type-assertion contract.
func TestLockStateSatisfiesLockedPayload(t *testing.T) {
	var p trunking.LockedPayload = LockState{FrequencyHz: 854_562_500, SystemID: 0x4567}
	if p.LockedFrequencyHz() != 854_562_500 || p.LockedNAC() != 0x4567 {
		t.Errorf("LockedPayload = %d/%#x", p.LockedFrequencyHz(), p.LockedNAC())
	}
}
