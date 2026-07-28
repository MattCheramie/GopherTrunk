package trunking

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// waitForGrantLen spins until the tracker has recorded want entries or the
// deadline passes.
func waitForGrantLen(t *testing.T, tr *GrantTracker, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if tr.Len() == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for grant log len %d (have %d)", want, tr.Len())
}

func startGrantTracker(t *testing.T, opts GrantTrackerOptions) *GrantTracker {
	t.Helper()
	tr, err := NewGrantTracker(opts)
	if err != nil {
		t.Fatalf("NewGrantTracker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Run(ctx)
	t.Cleanup(func() { cancel(); tr.Close() })
	return tr
}

// TestGrantTrackerLogsSourceRID is the issue #915 guard for fix #2: a grant
// carrying a source RID off the control channel is retained and surfaced with
// that RID intact, so a consumer can read it off /api/v1/grants the way
// SDRtrunk's grant log does.
func TestGrantTrackerLogsSourceRID(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr := startGrantTracker(t, GrantTrackerOptions{Bus: bus})

	bus.Publish(events.Event{Kind: events.KindGrant, Payload: Grant{
		System: "MMR", Protocol: "p25", GroupID: 20001, SourceID: 202419,
		FrequencyHz: 422612500, Encrypted: true,
	}})
	waitForGrantLen(t, tr, 1)

	snap := tr.Snapshot(0)
	if len(snap) != 1 {
		t.Fatalf("snapshot len = %d, want 1", len(snap))
	}
	g := snap[0]
	if g.SourceID != 202419 || g.GroupID != 20001 || g.FrequencyHz != 422612500 {
		t.Fatalf("grant fields wrong: %+v", g)
	}
	if !g.Encrypted {
		t.Error("encrypted flag lost — must carry through to the grant log")
	}
	if g.At.IsZero() {
		t.Error("grant log entry has no timestamp")
	}
}

// TestGrantTrackerLogsSourcelessGrant confirms the log is coverage-truthful:
// a grant whose TSBK carried src=0 is retained as src=0, not dropped or
// backfilled — the log records what the control channel actually decoded.
func TestGrantTrackerLogsSourcelessGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr := startGrantTracker(t, GrantTrackerOptions{Bus: bus})

	bus.Publish(events.Event{Kind: events.KindGrant, Payload: Grant{
		System: "MMR", Protocol: "p25", GroupID: 20001, SourceID: 0, FrequencyHz: 1,
	}})
	waitForGrantLen(t, tr, 1)
	if snap := tr.Snapshot(0); len(snap) != 1 || snap[0].SourceID != 0 {
		t.Fatalf("source-less grant not logged as src=0: %+v", snap)
	}
}

// TestGrantTrackerNewestFirst checks ordering: the most recent grant leads.
func TestGrantTrackerNewestFirst(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	tr := startGrantTracker(t, GrantTrackerOptions{Bus: bus})

	for i := uint32(1); i <= 5; i++ {
		bus.Publish(events.Event{Kind: events.KindGrant, Payload: Grant{
			System: "S", Protocol: "p25", GroupID: 100, SourceID: i, FrequencyHz: 1,
		}})
	}
	waitForGrantLen(t, tr, 5)

	snap := tr.Snapshot(0)
	if len(snap) != 5 {
		t.Fatalf("len = %d, want 5", len(snap))
	}
	// Newest (SourceID 5) first, oldest (1) last.
	for i, want := range []uint32{5, 4, 3, 2, 1} {
		if snap[i].SourceID != want {
			t.Fatalf("snap[%d].SourceID = %d, want %d (order: %+v)", i, snap[i].SourceID, want, sourceIDs(snap))
		}
	}

	// A limit returns only the newest N.
	if lim := tr.Snapshot(2); len(lim) != 2 || lim[0].SourceID != 5 || lim[1].SourceID != 4 {
		t.Fatalf("Snapshot(2) = %+v, want newest two [5 4]", sourceIDs(lim))
	}
}

// TestGrantTrackerRingEviction confirms the fixed-capacity ring drops the
// oldest grants and never grows past Capacity.
func TestGrantTrackerRingEviction(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	tr := startGrantTracker(t, GrantTrackerOptions{Bus: bus, Capacity: 3})

	for i := uint32(1); i <= 7; i++ {
		bus.Publish(events.Event{Kind: events.KindGrant, Payload: Grant{
			System: "S", Protocol: "p25", GroupID: 1, SourceID: i, FrequencyHz: 1,
		}})
	}
	// Wait until all 7 are observed; the ring holds only the last 3.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && tr.Observed() < 7 {
		time.Sleep(2 * time.Millisecond)
	}
	if tr.Observed() != 7 {
		t.Fatalf("observed = %d, want 7", tr.Observed())
	}
	if tr.Len() != 3 {
		t.Fatalf("ring len = %d, want 3 (capacity)", tr.Len())
	}
	snap := tr.Snapshot(0)
	for i, want := range []uint32{7, 6, 5} {
		if snap[i].SourceID != want {
			t.Fatalf("after eviction snap[%d] = %d, want %d (%+v)", i, snap[i].SourceID, want, sourceIDs(snap))
		}
	}
}

// TestGrantTrackerIgnoresNonGrant confirms only KindGrant is logged.
func TestGrantTrackerIgnoresNonGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr := startGrantTracker(t, GrantTrackerOptions{Bus: bus})

	bus.Publish(events.Event{Kind: events.KindAffiliation, Payload: Affiliation{
		System: "S", SourceID: 1, GroupID: 2, Response: AffiliationAccepted,
	}})
	bus.Publish(events.Event{Kind: events.KindGrant, Payload: Grant{
		System: "S", Protocol: "p25", GroupID: 2, SourceID: 9, FrequencyHz: 1,
	}})
	waitForGrantLen(t, tr, 1)
	if snap := tr.Snapshot(0); len(snap) != 1 || snap[0].SourceID != 9 {
		t.Fatalf("only the grant should be logged, got %+v", sourceIDs(snap))
	}
}

func TestNewGrantTrackerRequiresBus(t *testing.T) {
	if _, err := NewGrantTracker(GrantTrackerOptions{}); err == nil {
		t.Fatal("expected error when Bus is nil")
	}
}

func sourceIDs(gs []Grant) []uint32 {
	out := make([]uint32, len(gs))
	for i, g := range gs {
		out[i] = g.SourceID
	}
	return out
}
