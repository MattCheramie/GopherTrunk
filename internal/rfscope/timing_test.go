package rfscope

import (
	"context"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// periodicScene builds one channel with a burst every periodSec across the
// window, each of length lenSec.
func periodicScene(window, periodSec, lenSec float64) *Scene {
	sc := &Scene{SpanHz: 1_000_000, WindowSec: window, NoiseFloorDbFS: -90}
	var id uint64
	for t := 0.0; t+lenSec <= window; t += periodSec {
		id++
		sc.Bursts = append(sc.Bursts, Burst{
			ID: id, FreqHz: 450_000_000, StartSec: t, EndSec: t + lenSec,
			PeakDbFS: -40, Class: survey.ClassFSK, OccupiedBwHz: 12_500,
		})
	}
	return sc
}

func TestTimingRecoversPeriod(t *testing.T) {
	t.Parallel()
	const (
		window = 10.0
		period = 1.0
		length = 0.2
	)
	sc := periodicScene(window, period, length)

	// Run via the registry so the timeline dependency is pulled in first.
	if err := Run(context.Background(), sc, &Input{}, "timing"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(sc.Channels) != 1 {
		t.Fatalf("want 1 channel, got %d", len(sc.Channels))
	}
	ch := sc.Channels[0]

	binDur := window / float64(timelineBins) // 0.1s
	if math.Abs(ch.PeriodSec-period) > binDur*1.5 {
		t.Fatalf("period %.3fs not near %.3fs (binDur=%.3f)", ch.PeriodSec, period, binDur)
	}
	if ch.PeriodConfidence < 0.3 {
		t.Fatalf("period confidence too low: %v", ch.PeriodConfidence)
	}

	// Inter-arrival gaps are all ~1.0s, so the histogram has entries.
	if len(ch.InterArrivalHist.Counts) == 0 {
		t.Fatalf("inter-arrival histogram empty")
	}
	total := 0
	for _, c := range ch.InterArrivalHist.Counts {
		total += c
	}
	if total != len(sc.Bursts)-1 {
		t.Fatalf("inter-arrival count %d, want %d", total, len(sc.Bursts)-1)
	}
	if len(ch.BurstLenHist.Counts) == 0 {
		t.Fatalf("burst-length histogram empty")
	}
}

func TestTimingNonPeriodicHasNoStrongPeriod(t *testing.T) {
	t.Parallel()
	// A single long burst has no repeating structure.
	sc := &Scene{
		SpanHz: 1_000_000, WindowSec: 10, NoiseFloorDbFS: -90,
		Bursts: []Burst{{ID: 1, FreqHz: 1_000_000, StartSec: 1, EndSec: 9, PeakDbFS: -40, Class: survey.ClassNBFM, OccupiedBwHz: 12_500}},
	}
	if err := Run(context.Background(), sc, &Input{}, "timing"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sc.Channels[0].PeriodConfidence >= 0.9 {
		t.Fatalf("single burst should not look strongly periodic: conf=%v", sc.Channels[0].PeriodConfidence)
	}
}

func TestDetectPeriodShortSignal(t *testing.T) {
	t.Parallel()
	if p, c := detectPeriod([]float64{1, 0}, 0.1); p != 0 || c != 0 {
		t.Fatalf("short signal should yield no period, got %v %v", p, c)
	}
}
