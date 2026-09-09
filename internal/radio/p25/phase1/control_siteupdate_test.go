package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestSiteUpdateIsEdgeTriggered is the failing-first regression for the P25
// dashboard "site.update spam": a control channel rebroadcasts its RFSS Status
// Broadcast many times a second, but the KindSiteUpdate bus event (which ships
// the whole topology block to every SSE client) must fire only when the site's
// material content changes — plus a slow heartbeat — not once per broadcast.
// Before the edge-trigger this published one event per RFSS TSBK.
func TestSiteUpdateIsEdgeTriggered(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	now := time.Unix(1_700_000_000, 0).UTC()
	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Now:         func() time.Time { return now },
	})

	// RFSS Status Broadcast: LRA=9, SystemID=0x123, RFSS=1, Site=1.
	rfss := TSBK{LB: true, Opcode: OpRFSSStatusBroadcast, Payload: [8]byte{0x09, 0x01, 0x23, 0x01, 0x01, 0x00, 0x00, 0x00}}

	// Feed the SAME broadcast 20 times at the same instant. Exactly one
	// site.update should reach the bus (the first); the other 19 are steady-
	// state repeats with identical content and no heartbeat elapsed.
	const repeats = 20
	for i := 0; i < repeats; i++ {
		cc.Process(buildLockedStreamWithTSBK(10, 0x111, DUIDTrunkingSignaling, rfss), i<<20)
	}

	got := drainSiteUpdates(sub)
	if len(got) != 1 {
		t.Fatalf("steady RFSS broadcasts published %d site.update events, want 1 (edge-triggered)", len(got))
	}
	if got[0].RFSSID != 1 || got[0].SiteID != 1 {
		t.Errorf("first update site = RFSS %d / Site %d, want 1/1", got[0].RFSSID, got[0].SiteID)
	}

	// A material topology change (a newly-advertised neighbor site) must publish
	// again. Neighbors surface on first observation (not majority-voted), so one
	// adjacent-site broadcast changes the topology fingerprint. Feed it a few
	// times; the neighbor is added once, so only the first changes content.
	adj := TSBK{Opcode: OpAdjacentSiteStatusBroadcast, Payload: [8]byte{0x00, 0x31, 0x64, 0x04, 0x2E, 0xF0, 0x65, 0x70}}
	for i := 0; i < 5; i++ {
		cc.Process(buildLockedStreamWithTSBK(10, 0x111, DUIDTrunkingSignaling, adj), (repeats+i)<<20)
	}
	if got := drainSiteUpdates(sub); len(got) != 1 {
		t.Fatalf("a topology change (new neighbor) published %d updates, want 1: %+v", len(got), got)
	}

	// After the heartbeat interval elapses, an unchanged broadcast republishes
	// so the sites table's live carrier offset / TSBK error-rate stay fresh.
	now = now.Add(siteUpdateHeartbeat + time.Second)
	cc.Process(buildLockedStreamWithTSBK(10, 0x111, DUIDTrunkingSignaling, rfss), (repeats+9)<<20)
	if got := drainSiteUpdates(sub); len(got) != 1 {
		t.Fatalf("heartbeat republished %d updates, want 1", len(got))
	}
}

// drainSiteUpdates collects every KindSiteUpdate currently queued on the
// subscription, returning after a short idle gap.
func drainSiteUpdates(sub *events.Subscription) []trunking.SiteUpdate {
	var out []trunking.SiteUpdate
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindSiteUpdate {
				if u, ok := ev.Payload.(trunking.SiteUpdate); ok {
					out = append(out, u)
				}
			}
		case <-time.After(100 * time.Millisecond):
			return out
		}
	}
}
