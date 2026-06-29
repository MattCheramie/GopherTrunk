package rfscope

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func timelineFixture() *Scene {
	return &Scene{
		SpanHz:         1_000_000,
		WindowSec:      10,
		NoiseFloorDbFS: -90,
		Bursts: []Burst{
			{ID: 1, FreqHz: 450_100_000, StartSec: 0, EndSec: 1, PeakDbFS: -40, Class: survey.ClassC4FM, OccupiedBwHz: 12_500},
			{ID: 2, FreqHz: 450_100_000, StartSec: 2, EndSec: 3, PeakDbFS: -42, Class: survey.ClassC4FM, OccupiedBwHz: 12_500},
			{ID: 3, FreqHz: 450_300_000, StartSec: 0, EndSec: 5, PeakDbFS: -30, Class: survey.ClassNBFM, OccupiedBwHz: 12_500},
		},
	}
}

func channelAt(chs []Channel, f uint32) (Channel, bool) {
	for _, c := range chs {
		if c.FreqHz == f {
			return c, true
		}
	}
	return Channel{}, false
}

func TestTimelineDutyAndRate(t *testing.T) {
	t.Parallel()
	sc := timelineFixture()
	if err := (timelineAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(sc.Channels) != 2 {
		t.Fatalf("want 2 channels, got %d", len(sc.Channels))
	}
	// Sorted ascending by frequency.
	if sc.Channels[0].FreqHz > sc.Channels[1].FreqHz {
		t.Fatalf("channels not sorted by frequency")
	}

	a, ok := channelAt(sc.Channels, 450_100_000)
	if !ok {
		t.Fatalf("missing channel A")
	}
	if a.BurstCount != 2 || a.AirtimeSec != 2 {
		t.Fatalf("channel A counts: %+v", a)
	}
	if a.DutyCycle < 0.19 || a.DutyCycle > 0.21 { // 2s / 10s
		t.Fatalf("channel A duty: %v", a.DutyCycle)
	}
	if a.BurstRatePerMin < 11.9 || a.BurstRatePerMin > 12.1 { // 2 bursts / (10/60) min
		t.Fatalf("channel A burst rate: %v", a.BurstRatePerMin)
	}
	if a.DominantClass != survey.ClassC4FM {
		t.Fatalf("channel A dominant class: %v", a.DominantClass)
	}
	if len(a.Timeline) != timelineBins {
		t.Fatalf("timeline length: got %d want %d", len(a.Timeline), timelineBins)
	}
	// 2s occupied over a 10s/100-bin window ⇒ 20 occupied bins ⇒ 20%.
	if a.OccupancyPct < 19 || a.OccupancyPct > 21 {
		t.Fatalf("channel A occupancy pct: %v", a.OccupancyPct)
	}

	b, _ := channelAt(sc.Channels, 450_300_000)
	if b.DutyCycle < 0.49 || b.DutyCycle > 0.51 { // 5s / 10s
		t.Fatalf("channel B duty: %v", b.DutyCycle)
	}
	// An occupied bin in the first half must carry the burst peak, not the floor.
	if b.Timeline[0].PowerDbFS != -30 || !b.Timeline[0].Occupied {
		t.Fatalf("channel B first bin: %+v", b.Timeline[0])
	}
	// A bin past the burst end falls back to the noise floor.
	if b.Timeline[len(b.Timeline)-1].PowerDbFS != sc.NoiseFloorDbFS {
		t.Fatalf("channel B last bin should be at noise floor: %+v", b.Timeline[len(b.Timeline)-1])
	}
}

func TestTimelineNoWindow(t *testing.T) {
	t.Parallel()
	sc := &Scene{Bursts: []Burst{{ID: 1, FreqHz: 1}}}
	if err := (timelineAnalyzer{}).Analyze(context.Background(), sc, nil); err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(sc.Channels) != 0 {
		t.Fatalf("no window ⇒ no channels")
	}
}
