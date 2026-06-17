package trunking

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestSiteTrackerObservesSiteUpdates(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr, err := NewSiteTracker(SiteTrackerOptions{Bus: bus})
	if err != nil {
		t.Fatalf("NewSiteTracker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Run(ctx)
	t.Cleanup(func() { cancel(); tr.Close() })

	bus.Publish(events.Event{
		Kind: events.KindSiteUpdate,
		Payload: SiteUpdate{
			System: "MMR", RFSSID: 1, SiteID: 1,
			ControlChannelHz: 420012500, WACN: 0xBEE00, SystemID: 0x123,
		},
	})

	waitFor(t, func() bool { return tr.Len() == 1 })
	snap := tr.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot has %d sites, want 1", len(snap))
	}
	s := snap[0]
	if s.System != "MMR" || s.RFSSID != 1 || s.SiteID != 1 {
		t.Fatalf("site identity wrong: %+v", s)
	}
	if s.ControlChannelHz != 420012500 || s.WACN != 0xBEE00 || s.SystemID != 0x123 {
		t.Fatalf("site network fields wrong: %+v", s)
	}
}

func TestSiteTrackerDedupesAndUpdates(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr, _ := NewSiteTracker(SiteTrackerOptions{Bus: bus})
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Run(ctx)
	t.Cleanup(func() { cancel(); tr.Close() })

	// Two distinct sites plus a refresh of the first on a different CC.
	bus.Publish(events.Event{Kind: events.KindSiteUpdate, Payload: SiteUpdate{System: "MMR", RFSSID: 1, SiteID: 1, ControlChannelHz: 420012500}})
	bus.Publish(events.Event{Kind: events.KindSiteUpdate, Payload: SiteUpdate{System: "MMR", RFSSID: 2, SiteID: 9, ControlChannelHz: 423012500}})
	bus.Publish(events.Event{Kind: events.KindSiteUpdate, Payload: SiteUpdate{System: "MMR", RFSSID: 1, SiteID: 1, ControlChannelHz: 420112500}})

	waitFor(t, func() bool { return tr.Len() == 2 })
	snap := tr.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("want 2 sites, got %d: %+v", len(snap), snap)
	}
	// Snapshot is sorted by (system, rfss, site); the (1,1) site is first
	// and must reflect the most-recent control-channel frequency.
	if snap[0].RFSSID != 1 || snap[0].SiteID != 1 || snap[0].ControlChannelHz != 420112500 {
		t.Fatalf("first site not the refreshed (1,1): %+v", snap[0])
	}
	if snap[1].RFSSID != 2 || snap[1].SiteID != 9 {
		t.Fatalf("second site wrong: %+v", snap[1])
	}
}
