// Package dmrlcn implements DMR Tier III trunk autoconfiguration: it
// learns the LCN→downlink-frequency band plan of a trunked system from
// observed voice grants (which carry only a Logical Channel Number) and
// the RF carriers that key up in response, then hot-swaps the learned
// plan into the running control channel.
//
// This file holds the pure band-plan fitter. Given a set of confirmed
// (LCN, frequency) pairs it solves the regular base+spacing grid most
// DMR Tier III sites use, snapping spacing to a known channel grid and
// rejecting outliers; when the pairs don't fall on a regular grid it
// falls back to an explicit per-LCN table. It performs no I/O and owns
// no state, so it is exhaustively unit-testable in isolation.
package dmrlcn

import (
	"math"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Pair is one observed mapping from a Logical Channel Number to the
// downlink frequency the system used for it. Weight expresses how much
// the correlator trusts the observation — e.g. an embedded-LC match is
// weighted higher than a bare energy-onset coincidence — and biases the
// fitter's weighted-median estimates and confidence.
type Pair struct {
	LCN    uint16
	FreqHz uint32
	Weight float64
}

// FitOptions tunes FitBandPlan. The zero value is not valid; callers
// should start from DefaultFitOptions and adjust.
type FitOptions struct {
	// MinLCNs is the number of distinct LCNs that must be confirmed
	// before a plan (linear or table) is declared learned.
	MinLCNs int
	// SpacingGrids lists the channel spacings (Hz) the fitter will snap a
	// linear estimate to, in any order.
	SpacingGrids []uint32
	// SpacingTolHz is how close the estimated spacing must be to a grid
	// value to snap to it; beyond this the fit falls back to a table.
	SpacingTolHz uint32
	// ResidualTolHz is the largest |observed-predicted| frequency error a
	// linear fit may carry after outlier rejection.
	ResidualTolHz uint32
}

// fitOffset is the canonical LCN the emitted BaseHz is pinned to
// (freq = BaseHz + (LCN-fitOffset)*SpacingHz). Offset is purely a
// representation choice — the base absorbs any shift — so the fitter
// always emits the common LCN-1-indexed convention.
const fitOffset int8 = 1

// DefaultFitOptions returns sensible defaults: at least 4 confirmed
// LCNs, the standard DMR 6.25/12.5/25 kHz grids, ±600 Hz snap tolerance,
// and a 1.5 kHz residual ceiling.
func DefaultFitOptions() FitOptions {
	return FitOptions{
		MinLCNs:       4,
		SpacingGrids:  []uint32{6250, 12500, 25000},
		SpacingTolHz:  600,
		ResidualTolHz: 1500,
	}
}

// FitResult is the outcome of a fit. Exactly one of Linear / Table is
// populated when ok is true: Linear for a regular base+spacing grid,
// Table for an irregular layout. Confidence is in [0,1]; ResidualHz is
// the worst |observed-predicted| error of an accepted linear fit (0 for
// a table).
type FitResult struct {
	Linear     *trunking.DMRLinearBandPlan
	Table      []trunking.DMRBandPlanTableEntry
	Confidence float64
	NumPairs   int // distinct LCNs the result is built from
	ResidualHz uint32
}

// BandPlan renders the result as a trunking.DMRBandPlan, ready for
// tier3.ResolverFromPlan or config writeback. Returns nil for an empty
// result.
func (r FitResult) BandPlan() *trunking.DMRBandPlan {
	switch {
	case r.Linear != nil:
		lp := *r.Linear
		return &trunking.DMRBandPlan{Linear: &lp}
	case len(r.Table) > 0:
		tbl := make([]trunking.DMRBandPlanTableEntry, len(r.Table))
		copy(tbl, r.Table)
		return &trunking.DMRBandPlan{Table: tbl}
	default:
		return nil
	}
}

// point is one deduplicated LCN observation: the single best frequency
// chosen for an LCN plus the total weight backing it.
type point struct {
	lcn    uint16
	freqHz uint32
	weight float64
}

// FitBandPlan attempts to fit a band plan to the observed pairs. It
// returns ok=false (with a zero FitResult) until enough consistent
// observations accumulate to meet opts.MinLCNs. A regular grid yields a
// Linear plan; an irregular but well-populated layout yields a Table.
func FitBandPlan(pairs []Pair, opts FitOptions) (FitResult, bool) {
	opts = normalizeOptions(opts)

	pts := aggregate(pairs)
	if len(pts) < opts.MinLCNs || distinctFreqs(pts) < 2 {
		return FitResult{}, false
	}

	if lin, ok := fitLinear(pts, opts); ok {
		return lin, true
	}

	// Irregular layout: fall back to an explicit per-LCN table built from
	// every confirmed observation.
	table := make([]trunking.DMRBandPlanTableEntry, 0, len(pts))
	for _, p := range pts {
		table = append(table, trunking.DMRBandPlanTableEntry{LCN: p.lcn, FreqHz: p.freqHz})
	}
	return FitResult{
		Table:      table,
		Confidence: tableConfidence(pts),
		NumPairs:   len(pts),
	}, true
}

// normalizeOptions fills zero fields with DefaultFitOptions values so a
// partially-specified FitOptions still works.
func normalizeOptions(o FitOptions) FitOptions {
	d := DefaultFitOptions()
	if o.MinLCNs <= 0 {
		o.MinLCNs = d.MinLCNs
	}
	if len(o.SpacingGrids) == 0 {
		o.SpacingGrids = d.SpacingGrids
	}
	if o.SpacingTolHz == 0 {
		o.SpacingTolHz = d.SpacingTolHz
	}
	if o.ResidualTolHz == 0 {
		o.ResidualTolHz = d.ResidualTolHz
	}
	return o
}

// aggregate reduces raw pairs to one point per LCN, choosing the
// frequency with the greatest summed weight (ties broken by lowest
// frequency for determinism). The chosen frequency's total weight is
// carried forward. Result is sorted by LCN.
func aggregate(pairs []Pair) []point {
	type freqWeight map[uint32]float64
	byLCN := make(map[uint16]freqWeight)
	for _, p := range pairs {
		w := p.Weight
		if w <= 0 {
			w = 1
		}
		fw := byLCN[p.LCN]
		if fw == nil {
			fw = make(freqWeight)
			byLCN[p.LCN] = fw
		}
		fw[p.FreqHz] += w
	}

	pts := make([]point, 0, len(byLCN))
	for lcn, fw := range byLCN {
		var bestFreq uint32
		var bestWeight float64
		first := true
		for f, w := range fw {
			if first || w > bestWeight || (w == bestWeight && f < bestFreq) {
				bestFreq, bestWeight, first = f, w, false
			}
		}
		pts = append(pts, point{lcn: lcn, freqHz: bestFreq, weight: bestWeight})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].lcn < pts[j].lcn })
	return pts
}

func distinctFreqs(pts []point) int {
	seen := make(map[uint32]struct{}, len(pts))
	for _, p := range pts {
		seen[p.freqHz] = struct{}{}
	}
	return len(seen)
}

// fitLinear tries to fit pts to BaseHz + (LCN-Offset)*SpacingHz. It
// estimates spacing from the median per-LCN slope, snaps it to a known
// grid, solves the base by weighted median, then iteratively rejects the
// worst-residual point until all residuals fall within tolerance (or too
// few points remain to meet MinLCNs).
func fitLinear(pts []point, opts FitOptions) (FitResult, bool) {
	spacing, ok := estimateSpacing(pts, opts)
	if !ok {
		return FitResult{}, false
	}

	work := append([]point(nil), pts...)
	for {
		base, okBase := solveBase(work, spacing, fitOffset)
		if !okBase {
			return FitResult{}, false
		}
		worst, maxRes := worstResidual(work, base, spacing, fitOffset)
		if maxRes <= opts.ResidualTolHz {
			if len(work) < opts.MinLCNs {
				return FitResult{}, false
			}
			return FitResult{
				Linear: &trunking.DMRLinearBandPlan{
					BaseHz:    base,
					SpacingHz: spacing,
					Offset:    fitOffset,
				},
				Confidence: linearConfidence(work, maxRes, opts),
				NumPairs:   len(work),
				ResidualHz: maxRes,
			}, true
		}
		// Drop the worst outlier and refit; bail once we can't meet MinLCNs.
		if len(work)-1 < opts.MinLCNs {
			return FitResult{}, false
		}
		work = append(work[:worst], work[worst+1:]...)
	}
}

// estimateSpacing computes the median of the per-adjacent-LCN slopes and
// snaps it to the nearest configured grid within tolerance. Returns
// ok=false when the layout is descending/degenerate or no grid is close
// enough.
func estimateSpacing(pts []point, opts FitOptions) (uint32, bool) {
	slopes := make([]float64, 0, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		dl := int(pts[i].lcn) - int(pts[i-1].lcn)
		if dl == 0 {
			continue
		}
		df := float64(pts[i].freqHz) - float64(pts[i-1].freqHz)
		slopes = append(slopes, df/float64(dl))
	}
	if len(slopes) == 0 {
		return 0, false
	}
	est := medianFloat(slopes)
	if est <= 0 {
		return 0, false
	}
	bestGrid := uint32(0)
	bestErr := math.MaxFloat64
	for _, g := range opts.SpacingGrids {
		if e := math.Abs(est - float64(g)); e < bestErr {
			bestErr, bestGrid = e, g
		}
	}
	if bestErr > float64(opts.SpacingTolHz) {
		return 0, false
	}
	return bestGrid, true
}

// solveBase picks the BaseHz that, for the given spacing and offset, best
// fits the points — the weighted median of each point's implied base.
// Returns ok=false if the implied base is negative or overflows uint32.
func solveBase(pts []point, spacing uint32, offset int8) (uint32, bool) {
	type bw struct {
		base   int64
		weight float64
	}
	bws := make([]bw, 0, len(pts))
	for _, p := range pts {
		idx := int64(p.lcn) - int64(offset)
		base := int64(p.freqHz) - idx*int64(spacing)
		bws = append(bws, bw{base: base, weight: p.weight})
	}
	sort.Slice(bws, func(i, j int) bool { return bws[i].base < bws[j].base })
	var total float64
	for _, b := range bws {
		total += b.weight
	}
	half := total / 2
	var cum float64
	chosen := bws[len(bws)-1].base
	for _, b := range bws {
		cum += b.weight
		if cum >= half {
			chosen = b.base
			break
		}
	}
	if chosen < 0 || chosen > math.MaxUint32 {
		return 0, false
	}
	return uint32(chosen), true
}

// worstResidual returns the index and magnitude of the point whose
// frequency is furthest from the prediction.
func worstResidual(pts []point, base, spacing uint32, offset int8) (idx int, maxRes uint32) {
	for i, p := range pts {
		predicted := int64(base) + (int64(p.lcn)-int64(offset))*int64(spacing)
		res := int64(p.freqHz) - predicted
		if res < 0 {
			res = -res
		}
		if uint32(res) > maxRes {
			maxRes, idx = uint32(res), i
		}
	}
	return idx, maxRes
}

// linearConfidence blends residual tightness, LCN coverage, and the mean
// observation weight into a [0,1] score.
func linearConfidence(pts []point, maxRes uint32, opts FitOptions) float64 {
	residualScore := 1 - float64(maxRes)/float64(opts.ResidualTolHz)
	coverageScore := math.Min(1, float64(len(pts))/float64(opts.MinLCNs))
	weightScore := meanWeightScore(pts)
	return clamp01(residualScore * coverageScore * weightScore)
}

// tableConfidence scores an irregular fit: residual is exact by
// construction, so it depends only on coverage and weight.
func tableConfidence(pts []point) float64 {
	coverageScore := math.Min(1, float64(len(pts))/float64(DefaultFitOptions().MinLCNs))
	return clamp01(coverageScore * meanWeightScore(pts))
}

func meanWeightScore(pts []point) float64 {
	if len(pts) == 0 {
		return 0
	}
	var sum float64
	for _, p := range pts {
		sum += clamp01(p.weight)
	}
	return sum / float64(len(pts))
}

func medianFloat(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
