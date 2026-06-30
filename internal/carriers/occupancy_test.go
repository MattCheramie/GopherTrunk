package carriers

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
)

// flatBins returns n bins at the given floor dBFS.
func flatBins(n int, floorDb float32) []float32 {
	b := make([]float32, n)
	for i := range b {
		b[i] = floorDb
	}
	return b
}

func TestDetectOccupancy_MidbandPlateau(t *testing.T) {
	const n, center, rate = 1024, 751_000_000, 2_048_000 // binHz = 2000
	bins := flatBins(n, -85)
	for i := 400; i <= 600; i++ { // 201 bins ≈ 402 kHz, well inside the frame
		bins[i] = -30
	}
	frame := spectrum.Frame{CenterHz: center, SampleRate: rate, Bins: bins}

	occ := DetectOccupancy(frame, OccupancyOptions{})
	if len(occ) != 1 {
		t.Fatalf("len(occ) = %d, want 1: %+v", len(occ), occ)
	}
	o := occ[0]
	if o.EdgeClippedLow || o.EdgeClippedHigh {
		t.Errorf("midband plateau marked edge-clipped: %+v", o)
	}
	// 201 bins × 2000 Hz/bin = 402 kHz (±1 bin rounding).
	if o.OccupiedBwHz < 400_000 || o.OccupiedBwHz > 404_000 {
		t.Errorf("OccupiedBwHz = %d, want ≈402000", o.OccupiedBwHz)
	}
	// Plateau bins 400..600 center on bin 500; binToHz(500) with
	// base = 751e6 - 1.024e6 and 2000 Hz/bin = 750.976 MHz.
	wantCenter := uint32(750_976_000)
	if absDiff(o.CenterHz, wantCenter) > 4_000 {
		t.Errorf("CenterHz = %d, want ≈%d", o.CenterHz, wantCenter)
	}
	if o.SNRDb < 40 {
		t.Errorf("SNRDb = %.1f, want a strong plateau (>40 dB over floor)", o.SNRDb)
	}
}

func TestDetectOccupancy_EdgeClippedFillsFrame(t *testing.T) {
	const n, center, rate = 1024, 2_600_000_000, 2_048_000
	bins := flatBins(n, -30) // the whole frame is occupied — a signal wider than the tune
	frame := spectrum.Frame{CenterHz: center, SampleRate: rate, Bins: bins}

	// A fully-occupied step has no quiet region, so its own percentile floor
	// sits inside the signal and the per-frame estimate sees nothing — this is
	// exactly why the sweeper supplies a sweep-wide floor.
	if occ := DetectOccupancy(frame, OccupancyOptions{}); len(occ) != 0 {
		t.Errorf("per-frame floor on a flat frame should detect nothing, got %+v", occ)
	}

	occ := DetectOccupancy(frame, OccupancyOptions{FloorDbFS: -85})
	if len(occ) != 1 {
		t.Fatalf("with sweep-wide floor: len(occ) = %d, want 1: %+v", len(occ), occ)
	}
	if !occ[0].EdgeClippedLow || !occ[0].EdgeClippedHigh {
		t.Errorf("frame-filling span not flagged edge-clipped on both sides: %+v", occ[0])
	}
}

func TestDetectOccupancy_NarrowCarrierIgnored(t *testing.T) {
	const n, center, rate = 1024, 460_000_000, 2_048_000
	bins := flatBins(n, -85)
	bins[500] = -20 // a single strong narrowband carrier — DetectPeaks territory
	frame := spectrum.Frame{CenterHz: center, SampleRate: rate, Bins: bins}

	if occ := DetectOccupancy(frame, OccupancyOptions{}); len(occ) != 0 {
		t.Errorf("narrow carrier produced occupancy spans: %+v", occ)
	}
}

func TestDetectOccupancy_TwoSpansSortedByLowEdge(t *testing.T) {
	const n, center, rate = 1024, 915_000_000, 2_048_000
	bins := flatBins(n, -85)
	for i := 200; i <= 350; i++ { // 151 bins ≈ 302 kHz
		bins[i] = -35
	}
	for i := 700; i <= 850; i++ {
		bins[i] = -35
	}
	frame := spectrum.Frame{CenterHz: center, SampleRate: rate, Bins: bins}

	occ := DetectOccupancy(frame, OccupancyOptions{})
	if len(occ) != 2 {
		t.Fatalf("len(occ) = %d, want 2: %+v", len(occ), occ)
	}
	if occ[0].LowHz >= occ[1].LowHz {
		t.Errorf("spans not sorted by ascending low edge: %d then %d", occ[0].LowHz, occ[1].LowHz)
	}
}
