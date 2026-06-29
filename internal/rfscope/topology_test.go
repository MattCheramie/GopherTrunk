package rfscope

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func emitterByLabelContains(es []Emitter, sub string) (Emitter, bool) {
	for _, e := range es {
		if containsStr(e.Label, sub) {
			return e, true
		}
	}
	return Emitter{}, false
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestClusterEmittersTwoChannels(t *testing.T) {
	t.Parallel()
	sc := &Scene{
		SpanHz: 1_000_000, WindowSec: 10,
		Bursts: []Burst{
			{ID: 1, FreqHz: 450_100_000, StartSec: 0, EndSec: 1, PeakDbFS: -40, Class: survey.ClassNBFM, OccupiedBwHz: 12_500},
			{ID: 2, FreqHz: 450_100_000, StartSec: 2, EndSec: 3, PeakDbFS: -41, Class: survey.ClassNBFM, OccupiedBwHz: 12_500},
			{ID: 3, FreqHz: 450_300_000, StartSec: 0, EndSec: 5, PeakDbFS: -30, Class: survey.ClassC4FM, OccupiedBwHz: 12_500},
		},
	}
	clusterEmitters(sc)
	if len(sc.Emitters) != 2 {
		t.Fatalf("want 2 emitters, got %d: %+v", len(sc.Emitters), sc.Emitters)
	}
	// Every burst is assigned to an emitter.
	for _, b := range sc.Bursts {
		if b.EmitterID == 0 {
			t.Fatalf("burst %d unassigned", b.ID)
		}
	}
	// The two NBFM bursts share one emitter; the C4FM burst is its own.
	if sc.Bursts[0].EmitterID != sc.Bursts[1].EmitterID {
		t.Fatalf("co-channel NBFM bursts should share an emitter")
	}
	if sc.Bursts[2].EmitterID == sc.Bursts[0].EmitterID {
		t.Fatalf("C4FM burst must be a distinct emitter")
	}
}

func TestClusterEmittersMergesHopper(t *testing.T) {
	t.Parallel()
	// Same family/power, three distinct freqs, time-disjoint ⇒ one hopper.
	sc := &Scene{
		SpanHz: 2_000_000, WindowSec: 10,
		Bursts: []Burst{
			{ID: 1, FreqHz: 450_100_000, StartSec: 0.0, EndSec: 0.5, PeakDbFS: -40, Class: survey.ClassFSK, OccupiedBwHz: 12_500},
			{ID: 2, FreqHz: 450_500_000, StartSec: 1.0, EndSec: 1.5, PeakDbFS: -41, Class: survey.ClassFSK, OccupiedBwHz: 12_500},
			{ID: 3, FreqHz: 450_900_000, StartSec: 2.0, EndSec: 2.5, PeakDbFS: -39, Class: survey.ClassFSK, OccupiedBwHz: 12_500},
		},
	}
	clusterEmitters(sc)
	if len(sc.Emitters) != 1 {
		t.Fatalf("want 1 hopper emitter, got %d: %+v", len(sc.Emitters), sc.Emitters)
	}
	e := sc.Emitters[0]
	if len(e.HopSet) != 3 {
		t.Fatalf("want HopSet of 3 freqs, got %v", e.HopSet)
	}
	if _, ok := emitterByLabelContains([]Emitter{e}, "hopper"); !ok {
		t.Fatalf("hopper label expected, got %q", e.Label)
	}
}

func TestClusterEmittersNoMergeWhenOverlapping(t *testing.T) {
	t.Parallel()
	// Same family/power but two freqs active at the SAME time ⇒ two radios, no merge.
	sc := &Scene{
		SpanHz: 2_000_000, WindowSec: 10,
		Bursts: []Burst{
			{ID: 1, FreqHz: 450_100_000, StartSec: 0.0, EndSec: 2.0, PeakDbFS: -40, Class: survey.ClassFSK, OccupiedBwHz: 12_500},
			{ID: 2, FreqHz: 450_500_000, StartSec: 1.0, EndSec: 3.0, PeakDbFS: -40, Class: survey.ClassFSK, OccupiedBwHz: 12_500},
		},
	}
	clusterEmitters(sc)
	if len(sc.Emitters) != 2 {
		t.Fatalf("overlapping co-band emitters must not merge: got %d", len(sc.Emitters))
	}
}

func TestTopologyViaRunner(t *testing.T) {
	t.Parallel()
	sc := &Scene{
		SpanHz: 1_000_000, WindowSec: 10,
		Bursts: []Burst{{ID: 1, FreqHz: 450_100_000, StartSec: 0, EndSec: 1, PeakDbFS: -40, Class: survey.ClassNBFM, OccupiedBwHz: 12_500}},
	}
	if err := Run(context.Background(), sc, &Input{}, "topology"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(sc.Emitters) != 1 || sc.Emitters[0].ID == 0 {
		t.Fatalf("topology runner did not build emitters: %+v", sc.Emitters)
	}
}
