package dmrlcn

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// resolve evaluates a linear plan at an LCN the way tier3.LinearBandPlan does.
func resolve(lp *trunking.DMRLinearBandPlan, lcn uint16) uint32 {
	return uint32(int64(lp.BaseHz) + (int64(lcn)-int64(lp.Offset))*int64(lp.SpacingHz))
}

// linearPairs builds confirmed pairs on a regular grid:
// freq = base + (lcn-offset)*spacing, weight 1.
func linearPairs(base uint32, spacing uint32, offset int8, lcns ...uint16) []Pair {
	out := make([]Pair, 0, len(lcns))
	for _, l := range lcns {
		f := int64(base) + (int64(l)-int64(offset))*int64(spacing)
		out = append(out, Pair{LCN: l, FreqHz: uint32(f), Weight: 1})
	}
	return out
}

func TestFitLinearCleanGrids(t *testing.T) {
	cases := []struct {
		name    string
		base    uint32
		spacing uint32
		offset  int8
	}{
		{"12.5kHz offset1", 851_037_500, 12_500, 1},
		{"25kHz offset1", 866_000_000, 25_000, 1},
		{"6.25kHz offset0", 460_000_000, 6_250, 0},
		{"12.5kHz offset0", 451_000_000, 12_500, 0},
	}
	lcns := []uint16{1, 2, 3, 5, 8, 12}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pairs := linearPairs(tc.base, tc.spacing, tc.offset, lcns...)
			res, ok := FitBandPlan(pairs, FitOptions{})
			if !ok {
				t.Fatal("expected a learned plan")
			}
			if res.Linear == nil {
				t.Fatalf("expected linear plan, got table %+v", res.Table)
			}
			if res.Linear.SpacingHz != tc.spacing {
				t.Errorf("spacing = %d, want %d", res.Linear.SpacingHz, tc.spacing)
			}
			// Offset/base are a representation choice; verify the plan
			// resolves every input LCN back to its exact input frequency.
			for _, l := range lcns {
				want := uint32(int64(tc.base) + (int64(l)-int64(tc.offset))*int64(tc.spacing))
				if got := resolve(res.Linear, l); got != want {
					t.Errorf("resolve(lcn=%d) = %d, want %d", l, got, want)
				}
			}
			if res.ResidualHz != 0 {
				t.Errorf("residual = %d, want 0 for an exact grid", res.ResidualHz)
			}
			if res.Confidence <= 0 {
				t.Errorf("confidence = %f, want > 0", res.Confidence)
			}
		})
	}
}

func TestFitLinearTolerATesSubBinJitter(t *testing.T) {
	// Energy-onset frequency estimates land on FFT bins, so observed
	// frequencies carry a few hundred Hz of jitter. The fit should still
	// snap spacing and recover a base within the residual ceiling.
	base, spacing := uint32(851_037_500), uint32(12_500)
	jitter := []int32{+200, -150, +300, -250, +100, -300}
	pairs := make([]Pair, 0, len(jitter))
	for i, l := range []uint16{1, 2, 3, 4, 6, 9} {
		f := int64(base) + (int64(l)-1)*int64(spacing) + int64(jitter[i])
		pairs = append(pairs, Pair{LCN: l, FreqHz: uint32(f), Weight: 1})
	}
	res, ok := FitBandPlan(pairs, FitOptions{})
	if !ok || res.Linear == nil {
		t.Fatalf("expected linear plan, ok=%v res=%+v", ok, res)
	}
	if res.Linear.SpacingHz != spacing {
		t.Errorf("spacing = %d, want %d", res.Linear.SpacingHz, spacing)
	}
	if res.ResidualHz > 1500 {
		t.Errorf("residual = %d, want <= 1500", res.ResidualHz)
	}
}

func TestFitLinearRejectsOutlier(t *testing.T) {
	pairs := linearPairs(851_037_500, 12_500, 1, 1, 2, 3, 4, 5)
	// Corrupt LCN 3 with a wildly wrong frequency (a mis-paired carrier).
	for i := range pairs {
		if pairs[i].LCN == 3 {
			pairs[i].FreqHz += 5_000_000
		}
	}
	res, ok := FitBandPlan(pairs, FitOptions{})
	if !ok || res.Linear == nil {
		t.Fatalf("expected linear plan after outlier rejection, ok=%v res=%+v", ok, res)
	}
	if res.Linear.BaseHz != 851_037_500 || res.Linear.SpacingHz != 12_500 {
		t.Errorf("plan = base %d spacing %d, want 851037500/12500", res.Linear.BaseHz, res.Linear.SpacingHz)
	}
	if res.NumPairs != 4 {
		t.Errorf("NumPairs = %d, want 4 (outlier dropped)", res.NumPairs)
	}
}

func TestFitFallsBackToTableForIrregularGrid(t *testing.T) {
	// Frequencies not on any 6.25/12.5/25 kHz grid relative to LCN.
	pairs := []Pair{
		{LCN: 1, FreqHz: 451_012_500, Weight: 1},
		{LCN: 2, FreqHz: 451_031_300, Weight: 1}, // +18.8 kHz
		{LCN: 3, FreqHz: 451_077_900, Weight: 1}, // +46.6 kHz
		{LCN: 4, FreqHz: 451_093_700, Weight: 1}, // +15.8 kHz
	}
	res, ok := FitBandPlan(pairs, FitOptions{})
	if !ok {
		t.Fatal("expected a table plan")
	}
	if res.Linear != nil {
		t.Fatalf("expected table fallback, got linear %+v", res.Linear)
	}
	if len(res.Table) != 4 {
		t.Errorf("table len = %d, want 4", len(res.Table))
	}
}

func TestFitFallsBackToTableForUnsnappableSpacing(t *testing.T) {
	// Regular 9 kHz spacing: not a DMR grid, so linear must be rejected
	// even though the layout is perfectly linear.
	pairs := linearPairs(451_000_000, 9_000, 1, 1, 2, 3, 4, 5)
	res, ok := FitBandPlan(pairs, FitOptions{})
	if !ok {
		t.Fatal("expected a table plan")
	}
	if res.Linear != nil {
		t.Errorf("9 kHz spacing should not snap to a DMR grid; got linear %+v", res.Linear)
	}
}

func TestFitNotLearnedBelowMinLCNs(t *testing.T) {
	pairs := linearPairs(851_037_500, 12_500, 1, 1, 2, 3) // only 3, default MinLCNs=4
	if _, ok := FitBandPlan(pairs, FitOptions{}); ok {
		t.Fatal("expected ok=false below MinLCNs")
	}
}

func TestFitNotLearnedSingleFrequency(t *testing.T) {
	// Many observations but all the same frequency → cannot be a plan.
	pairs := []Pair{
		{LCN: 1, FreqHz: 851_000_000, Weight: 1},
		{LCN: 2, FreqHz: 851_000_000, Weight: 1},
		{LCN: 3, FreqHz: 851_000_000, Weight: 1},
		{LCN: 4, FreqHz: 851_000_000, Weight: 1},
		{LCN: 5, FreqHz: 851_000_000, Weight: 1},
	}
	if _, ok := FitBandPlan(pairs, FitOptions{}); ok {
		t.Fatal("expected ok=false with a single distinct frequency")
	}
}

func TestFitAggregatesConflictingObservationsByWeight(t *testing.T) {
	// LCN 3 was paired to a wrong frequency once (low weight) and the
	// right one twice (high weight). The weighted mode wins.
	pairs := linearPairs(851_037_500, 12_500, 1, 1, 2, 4, 5)
	pairs = append(pairs,
		Pair{LCN: 3, FreqHz: 851_037_500 + 2*12_500, Weight: 2}, // correct, weight 2
		Pair{LCN: 3, FreqHz: 999_000_000, Weight: 0.4},          // wrong, low weight
	)
	res, ok := FitBandPlan(pairs, FitOptions{})
	if !ok || res.Linear == nil {
		t.Fatalf("expected linear plan, ok=%v res=%+v", ok, res)
	}
	if res.Linear.BaseHz != 851_037_500 || res.Linear.SpacingHz != 12_500 {
		t.Errorf("plan = base %d spacing %d", res.Linear.BaseHz, res.Linear.SpacingHz)
	}
}

func TestFitResultBandPlanRoundTrip(t *testing.T) {
	res, ok := FitBandPlan(linearPairs(851_037_500, 12_500, 1, 1, 2, 3, 4, 5), FitOptions{})
	if !ok {
		t.Fatal("expected learned plan")
	}
	bp := res.BandPlan()
	if bp == nil || bp.Linear == nil {
		t.Fatalf("BandPlan() = %+v, want linear", bp)
	}
	if bp.Linear.BaseHz != 851_037_500 || bp.Linear.SpacingHz != 12_500 || bp.Linear.Offset != 1 {
		t.Errorf("round-tripped plan = %+v", bp.Linear)
	}
}
