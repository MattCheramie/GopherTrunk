package trunking

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestAffiliationTrackerObservesGrants(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr, err := NewAffiliationTracker(AffiliationTrackerOptions{Bus: bus})
	if err != nil {
		t.Fatalf("NewAffiliationTracker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Run(ctx)
	t.Cleanup(func() { cancel(); tr.Close() })

	bus.Publish(events.Event{
		Kind:    events.KindGrant,
		Payload: Grant{System: "Metro", Protocol: "dmr-tier3", GroupID: 100, SourceID: 4242, FrequencyHz: 1},
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if tr.Len() == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	snap := tr.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot has %d units, want 1", len(snap))
	}
	u := snap[0]
	if u.RadioID != 4242 || u.Talkgroup != 100 || u.System != "Metro" {
		t.Fatalf("unit activity wrong: %+v", u)
	}
	if u.Explicit {
		t.Error("a grant-observed association must not be marked Explicit")
	}
	if got := tr.UnitsOnTalkgroup(100); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("UnitsOnTalkgroup(100) = %v, want [4242]", got)
	}
}

func TestAffiliationTrackerExplicitAffiliation(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr, _ := NewAffiliationTracker(AffiliationTrackerOptions{Bus: bus})
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Run(ctx)
	t.Cleanup(func() { cancel(); tr.Close() })

	bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: Affiliation{
			System: "Metro", Protocol: "p25", SourceID: 7, GroupID: 50,
			Response: AffiliationAccepted,
		},
	})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && tr.Len() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	snap := tr.Snapshot()
	if len(snap) != 1 || !snap[0].Explicit {
		t.Fatalf("explicit affiliation not recorded: %+v", snap)
	}
}

func TestAffiliationTrackerExpiry(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	clock := time.Now()
	tr, _ := NewAffiliationTracker(AffiliationTrackerOptions{
		Bus: bus,
		TTL: 10 * time.Minute,
		Now: func() time.Time { return clock },
	})
	defer tr.Close()

	// observe is the internal record path; drive it directly so the
	// test controls the clock without racing the event loop.
	tr.observe(1, 100, "S", "p25", false, false)
	if tr.Len() != 1 {
		t.Fatalf("unit not recorded")
	}
	// Advance past the TTL and sweep.
	clock = clock.Add(11 * time.Minute)
	tr.sweep()
	if tr.Len() != 0 {
		t.Fatalf("expired unit not swept: %d remain", tr.Len())
	}
}

func TestAffiliationTrackerIgnoresDeniedAffiliation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tr, _ := NewAffiliationTracker(AffiliationTrackerOptions{Bus: bus})
	defer tr.Close()
	tr.handle(events.Event{
		Kind: events.KindAffiliation,
		Payload: Affiliation{
			SourceID: 9, GroupID: 1, Response: AffiliationDenied,
		},
	})
	if tr.Len() != 0 {
		t.Fatal("a denied affiliation must not enter the table")
	}
}

func TestAffiliationTrackerRequiresBus(t *testing.T) {
	if _, err := NewAffiliationTracker(AffiliationTrackerOptions{}); err == nil {
		t.Fatal("NewAffiliationTracker without a bus should error")
	}
}
