package hunt

import (
	"context"
	"math/rand"
	"testing"
)

// TestStitchWideband_JoinsAdjacentSpansAcrossSteps feeds the stitcher a wide
// signal seen across three steps — partial (edge-clipped) spans at the two
// boundary steps and a fully-occupied interior step (elevated floor, no
// per-frame span) — and asserts it collapses to one wideband candidate whose
// width is the signal's true extent.
func TestStitchWideband_JoinsAdjacentSpansAcrossSteps(t *testing.T) {
	const floor = float32(-90)
	steps := []wbStep{
		{floorDb: floor}, // a quiet step: establishes the sweep-wide floor
		{ // signal enters from the high edge of this step
			floorDb: floor,
			lowHz:   900_000, highHz: 1_100_000,
			spans: []Occupancy{{LowHz: 1_000_000, HighHz: 1_200_000, PowerDbFS: -40, EdgeClippedHigh: true}},
		},
		{ // interior step entirely inside the signal: floor elevated, no per-frame span
			floorDb: -40,
			lowHz:   1_200_000, highHz: 1_400_000,
		},
		{ // signal leaves through the low edge of this step
			floorDb: floor,
			lowHz:   1_400_000, highHz: 1_600_000,
			spans: []Occupancy{{LowHz: 1_400_000, HighHz: 1_600_000, PowerDbFS: -42, EdgeClippedLow: true}},
		},
		{floorDb: floor}, // another quiet step
	}

	got := stitchWideband(steps, 0, 1_000)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 stitched span: %+v", len(got), got)
	}
	c := got[0]
	if !c.IsWideband {
		t.Errorf("IsWideband = false, want true")
	}
	// [1.0,1.2] ∪ [1.2,1.4] ∪ [1.4,1.6] MHz = 600 kHz wide, centered at 1.3 MHz.
	if c.BwHz != 600_000 {
		t.Errorf("BwHz = %d, want 600000", c.BwHz)
	}
	if c.FreqHz != 1_300_000 {
		t.Errorf("FreqHz = %d, want 1300000 (span center)", c.FreqHz)
	}
	if c.SNRDb < 45 { // -40 power over a -90 floor
		t.Errorf("SNRDb = %.1f, want ≈50", c.SNRDb)
	}
}

// TestStitchWideband_SeparateSignalsStayDistinct verifies two wide signals
// with a clear gap do not merge.
func TestStitchWideband_SeparateSignalsStayDistinct(t *testing.T) {
	steps := []wbStep{
		{floorDb: -90},
		{floorDb: -90, spans: []Occupancy{{LowHz: 700_000_000, HighHz: 710_000_000, PowerDbFS: -40}}},
		{floorDb: -90, spans: []Occupancy{{LowHz: 740_000_000, HighHz: 750_000_000, PowerDbFS: -40}}},
	}
	got := stitchWideband(steps, 0, 50_000)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 distinct spans: %+v", len(got), got)
	}
}

// TestStitchWideband_NoOccupancyNoCandidates verifies a sweep with only quiet
// steps (or only narrowband carriers) yields no wideband candidates.
func TestStitchWideband_NoOccupancyNoCandidates(t *testing.T) {
	if got := stitchWideband(nil, 0, 1000); got != nil {
		t.Errorf("nil steps: got %+v, want nil", got)
	}
	quiet := []wbStep{{floorDb: -90}, {floorDb: -89}, {floorDb: -91}}
	if got := stitchWideband(quiet, 0, 1000); len(got) != 0 {
		t.Errorf("quiet sweep produced wideband candidates: %+v", got)
	}
}

// bandNoiseSource is an IQSource that returns loud broadband noise when the
// tuned frame overlaps a target band and quiet noise otherwise — a synthetic
// "signal wider than the tune" for the sweeper integration test.
type bandNoiseSource struct {
	rate   uint32
	sigLo  uint32
	sigHi  uint32
	center uint32
	rng    *rand.Rand
}

func (s *bandNoiseSource) Tune(c uint32) error  { s.center = c; return nil }
func (s *bandNoiseSource) SampleRateHz() uint32 { return s.rate }
func (s *bandNoiseSource) Capture(ctx context.Context, n int) ([]complex64, error) {
	fLo := s.center - s.rate/2
	fHi := s.center + s.rate/2
	amp := float32(0.002) // quiet noise floor
	if fLo < s.sigHi && s.sigLo < fHi {
		amp = 0.4 // the tuned frame overlaps the wide signal
	}
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(amp*float32(s.rng.NormFloat64()), amp*float32(s.rng.NormFloat64()))
	}
	return out, nil
}

// TestSweeperDetectsWidebandSpan runs the real sweeper with DetectWideband over
// a band containing a synthetic broadband signal and asserts it returns a
// stitched IsWideband candidate near the signal.
func TestSweeperDetectsWidebandSpan(t *testing.T) {
	const rate = 2_048_000
	src := &bandNoiseSource{
		rate:  rate,
		sigLo: 700_000_000,
		sigHi: 706_000_000, // ~6 MHz wide — far wider than the 2.048 MHz tune
		rng:   rand.New(rand.NewSource(0x5151)),
	}
	sw, err := NewSweeper(SweepOptions{
		Source:         src,
		Bands:          []Band{{LowHz: 695_000_000, HighHz: 712_000_000}},
		FFTSize:        1024,
		DetectWideband: true,
	})
	if err != nil {
		t.Fatalf("NewSweeper: %v", err)
	}
	cands, err := sw.Sweep(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	var wb []Candidate
	for _, c := range cands {
		if c.IsWideband {
			wb = append(wb, c)
		}
	}
	if len(wb) == 0 {
		t.Fatalf("no wideband candidate found; candidates=%+v", cands)
	}
	// The stitched span should be at least a couple MHz wide (it spans every
	// step whose frame overlapped the signal) and centered in the signal band.
	w := wb[0]
	if w.BwHz < 2_000_000 {
		t.Errorf("widest span BwHz = %d, want a multi-MHz stitched span", w.BwHz)
	}
	if w.FreqHz < 698_000_000 || w.FreqHz > 708_000_000 {
		t.Errorf("span center %d Hz outside the signal neighbourhood", w.FreqHz)
	}
}
