package tetra

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestGrantCarriesTrafficLMS pins the tetra_traffic_lms plumbing from
// Options to the published grant: the flag the ccdecoder/widebandt2
// factories parse from per-system config must reach the voice composer via
// Grant.TETRATrafficLMS, and stays false by default (the opt-in posture —
// the GT_TETRA_LMS capture A/B gates any default-on).
func TestGrantCarriesTrafficLMS(t *testing.T) {
	for _, on := range []bool{false, true} {
		bus := events.NewBus(16)
		sub := bus.Subscribe()
		clock := time.Unix(1_785_000_000, 0)
		cc := New(Options{
			Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 467_912_500,
			Now:        func() time.Time { return clock },
			TrafficLMS: on,
		})
		cc.learnMainCarrier(2716)
		cc.publishGrant(VoiceGrant{DestSSI: 1020543, CarrierNumber: 2716, Timeslot: 1})

		grants := drainGrants(sub)
		if len(grants) != 1 {
			t.Fatalf("TrafficLMS=%v: published %d grants, want 1", on, len(grants))
		}
		if grants[0].TETRATrafficLMS != on {
			t.Errorf("TrafficLMS=%v: grant.TETRATrafficLMS = %v", on, grants[0].TETRATrafficLMS)
		}
		sub.Close()
		bus.Close()
	}
}
