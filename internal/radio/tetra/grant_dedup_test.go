package tetra

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// drainGrants collects every KindGrant currently queued on sub.
func drainGrants(sub *events.Subscription) []trunking.Grant {
	var grants []trunking.Grant
	for {
		select {
		case ev := <-sub.C:
			if g, ok := ev.Payload.(trunking.Grant); ok && ev.Kind == events.KindGrant {
				grants = append(grants, g)
			}
		default:
			return grants
		}
	}
}

// TestGrantRebroadcastDeduped pins the "CC spam" fix: TETRA re-announces an
// active call's voice grant every multiframe, so publishGrant must emit exactly
// one KindGrant per call within grantDedupWindow — not one per rebroadcast — and
// must re-announce a genuinely new call for the same target after a release.
// Before the dedup a single call produced ~3-4 grant events (field-reported 173
// grants for 47 calls).
func TestGrantRebroadcastDeduped(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	clock := time.Unix(1_785_000_000, 0)
	cc := New(Options{
		Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 467_912_500,
		Now: func() time.Time { return clock },
	})
	cc.learnMainCarrier(2716)

	grant := VoiceGrant{DestSSI: 1020543, CarrierNumber: 2716, Timeslot: 1}

	// Four rebroadcasts ~1 s apart, all inside grantDedupWindow → one event.
	for i := 0; i < 4; i++ {
		cc.publishGrant(grant)
		clock = clock.Add(1 * time.Second)
	}
	if g := drainGrants(sub); len(g) != 1 {
		t.Fatalf("published %d grants for one call's rebroadcasts, want 1", len(g))
	}

	// A distinct concurrent call on another slot is NOT a duplicate.
	cc.publishGrant(VoiceGrant{DestSSI: 1020544, CarrierNumber: 2716, Timeslot: 2})
	if g := drainGrants(sub); len(g) != 1 {
		t.Fatalf("distinct slot grant published %d, want 1", len(g))
	}

	// After the call is released, a new grant to the same target re-announces
	// even though it is still within grantDedupWindow of the last rebroadcast.
	cc.publishRelease(grant.DestSSI, 0)
	drainGrants(sub) // discard the KindCallRelease-adjacent traffic, if any
	cc.publishGrant(grant)
	if g := drainGrants(sub); len(g) != 1 {
		t.Fatalf("grant after release published %d, want 1 (release must clear the dedup)", len(g))
	}

	// Past the window (no release), the rebroadcast is announced again.
	clock = clock.Add(grantDedupWindow + time.Second)
	cc.publishGrant(grant)
	if g := drainGrants(sub); len(g) != 1 {
		t.Fatalf("grant past the dedup window published %d, want 1", len(g))
	}
}
