package trunking

import (
	"context"
	"strings"
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

// TestSiteTrackerTSBKQualityGuard checks that a per-site TSBK error rate is
// recorded from an update that carries stats, and that a later zero-stat update
// (a fresh lock, or a non-TSBK Phase 2 site-update on the same key) does not
// clobber it (issue #858).
func TestSiteTrackerTSBKQualityGuard(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr, err := NewSiteTracker(SiteTrackerOptions{Bus: bus})
	if err != nil {
		t.Fatalf("NewSiteTracker: %v", err)
	}
	t.Cleanup(func() { tr.Close() })

	// Observe directly to keep the ordering deterministic.
	tr.observe(SiteUpdate{System: "MMR", RFSSID: 1, SiteID: 1,
		ControlChannelHz: 420012500, ControlChannelTSBKErrorRate: 8.0, ControlChannelTSBKCount: 250})
	got := tr.Snapshot()[0]
	if got.ControlChannelTSBKErrorRate != 8.0 || got.ControlChannelTSBKCount != 250 {
		t.Fatalf("after stats update: rate=%v count=%d, want 8.0 / 250",
			got.ControlChannelTSBKErrorRate, got.ControlChannelTSBKCount)
	}

	// A zero-stat update must leave the recorded rate untouched.
	tr.observe(SiteUpdate{System: "MMR", RFSSID: 1, SiteID: 1, ControlChannelHz: 420012500})
	got = tr.Snapshot()[0]
	if got.ControlChannelTSBKErrorRate != 8.0 || got.ControlChannelTSBKCount != 250 {
		t.Fatalf("zero-stat update clobbered quality: rate=%v count=%d, want 8.0 / 250",
			got.ControlChannelTSBKErrorRate, got.ControlChannelTSBKCount)
	}
}

func TestSiteTrackerReportFromTopology(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	tr, err := NewSiteTracker(SiteTrackerOptions{Bus: bus})
	if err != nil {
		t.Fatalf("NewSiteTracker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go tr.Run(ctx)
	t.Cleanup(func() { cancel(); tr.Close() })

	// No topology observed yet ⇒ no report.
	if _, ok := tr.Report("MMR"); ok {
		t.Fatal("Report should be unavailable before any topology")
	}

	bus.Publish(events.Event{
		Kind: events.KindSiteUpdate,
		Payload: SiteUpdate{
			System: "MMR", RFSSID: 1, SiteID: 1, ControlChannelHz: 450125000,
			WACN: 0xBEE00, SystemID: 0x2C2,
			Topology: &TopologySnapshot{
				SystemName: "MMR", Protocol: "p25", WACN: 0xBEE00, SystemID: 0x2C2, NAC: 0x2C1,
				RFSS: 1, Site: 1,
				PrimaryCC: &TopoChannelRef{ChannelID: 2, ChannelNumber: 1620, FrequencyHz: 450125000},
			},
		},
	})

	var report string
	waitFor(t, func() bool {
		r, ok := tr.Report("MMR")
		report = r
		return ok
	})
	if !strings.Contains(report, "WACN:BEE00[781824]") || !strings.Contains(report, "PRI CONTROL CHANNEL:2-1620") {
		t.Errorf("report missing expected tokens:\n%s", report)
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

	// Wait for the THIRD event (the refresh), not just Len()==2 — the count
	// reaches 2 as soon as the second event is processed, and asserting the
	// refreshed frequency in that window raced the tracker goroutine (seen as
	// a -race CI flake: snapshot still carried the first event's 420012500).
	waitFor(t, func() bool {
		s := tr.Snapshot()
		return len(s) == 2 && s[0].ControlChannelHz == 420112500
	})
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
