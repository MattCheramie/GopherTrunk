package siglab

import (
	"math"
	"sort"

	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// SoftEye is the structured pre-slicer soft-sample analysis for the P25 deep
// C4FM path — the part of the historical iqdiag report that worked on the
// matched-filter output. It exposes the eye-skew the dibit histogram folds
// away (issue #402): the DC offset, the per-decided-symbol rail distribution,
// and the slicer-independent true-outer-rail eye with a mechanism verdict.
type SoftEye struct {
	SoftSamples  int     `json:"soft_samples" yaml:"soft_samples"`
	SignedMeanDC float64 `json:"signed_mean_dc" yaml:"signed_mean_dc"`
	StdDev       float64 `json:"std_dev" yaml:"std_dev"`
	MeanAbs      float64 `json:"mean_abs" yaml:"mean_abs"`
	MaxAbs       float64 `json:"max_abs" yaml:"max_abs"`
	// MagnitudeHistogram bins |soft| into 10 deciles of MaxAbs — bimodal ⇒ a
	// clean 4-level eye, unimodal-saturated ⇒ everything piled near ±max.
	MagnitudeHistogram [10]int64 `json:"magnitude_histogram" yaml:"magnitude_histogram"`
	// PerDecidedSymbol is the soft distribution grouped by the slicer's
	// decision (rails -3,-1,+1,+3). Clean ⇒ symmetric, |outer| ≈ 3·|inner|.
	PerDecidedSymbol []RailStat `json:"per_decided_symbol" yaml:"per_decided_symbol"`
	// TrueOuterRail is the slicer-independent outer-rail (+3,-3) distribution
	// from the known FSW symbols.
	TrueOuterRail []RailStat `json:"true_outer_rail" yaml:"true_outer_rail"`
	// OuterSpread splits each true outer rail into steady vs post-transition
	// standard deviation — the ISI / timing discriminator.
	OuterSpread []SpreadStat `json:"outer_spread" yaml:"outer_spread"`
	// Verdict points at the dominant outer-spread mechanism.
	Verdict string `json:"verdict" yaml:"verdict"`
}

// RailStat is the distribution of soft samples on one decision rail.
type RailStat struct {
	Label string  `json:"label" yaml:"label"`
	N     int     `json:"n" yaml:"n"`
	Mean  float64 `json:"mean" yaml:"mean"`
	Std   float64 `json:"std" yaml:"std"`
	P10   float64 `json:"p10" yaml:"p10"`
	P50   float64 `json:"p50" yaml:"p50"`
	P90   float64 `json:"p90" yaml:"p90"`
}

// SpreadStat compares steady-state vs post-transition spread on one outer rail.
type SpreadStat struct {
	Label     string  `json:"label" yaml:"label"`
	SteadyStd float64 `json:"steady_std" yaml:"steady_std"`
	SteadyN   int     `json:"steady_n" yaml:"steady_n"`
	TransStd  float64 `json:"trans_std" yaml:"trans_std"`
	TransN    int     `json:"trans_n" yaml:"trans_n"`
	Ratio     float64 `json:"ratio" yaml:"ratio"`
}

// distStats returns the mean, standard deviation, and 10th/50th/90th
// percentiles of v (sorts a copy; v is unmodified). Zero for an empty slice.
// Ported from cmd/gophertrunk/iqdiag.go.
func distStats(v []float64) (mean, std, p10, p50, p90 float64) {
	if len(v) == 0 {
		return 0, 0, 0, 0, 0
	}
	var sum float64
	for _, x := range v {
		sum += x
	}
	mean = sum / float64(len(v))
	var sq float64
	for _, x := range v {
		d := x - mean
		sq += d * d
	}
	std = math.Sqrt(sq / float64(len(v)))
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	pct := func(p float64) float64 {
		idx := int(p * float64(len(s)-1))
		return s[idx]
	}
	return mean, std, pct(0.10), pct(0.50), pct(0.90)
}

// railStat builds a RailStat from a label + sample slice.
func railStat(label string, v []float64) RailStat {
	mean, std, p10, p50, p90 := distStats(v)
	return RailStat{Label: label, N: len(v), Mean: mean, Std: std, P10: p10, P50: p50, P90: p90}
}

// buildSoftEye produces the SoftEye from the aligned soft + dibit streams and
// the winning rotation's FSW hit positions. Ports the soft-sample +
// true-symbol-eye sections of cmd/gophertrunk/iqdiag.go into structured form.
func buildSoftEye(soft []float32, dibits []uint8, winner int, positions []int) *SoftEye {
	if len(soft) == 0 || len(soft) != len(dibits) {
		return nil
	}
	e := &SoftEye{SoftSamples: len(soft)}

	// DC, std, mean|x|, max|x|.
	var signedSum, sumAbs, maxAbs float64
	for _, s := range soft {
		x := float64(s)
		signedSum += x
		a := math.Abs(x)
		sumAbs += a
		if a > maxAbs {
			maxAbs = a
		}
	}
	n := float64(len(soft))
	e.SignedMeanDC = signedSum / n
	e.MeanAbs = sumAbs / n
	e.MaxAbs = maxAbs
	var sq float64
	for _, s := range soft {
		d := float64(s) - e.SignedMeanDC
		sq += d * d
	}
	e.StdDev = math.Sqrt(sq / n)

	// Magnitude histogram (10 deciles of maxAbs).
	if maxAbs > 0 {
		for _, s := range soft {
			b := int(math.Abs(float64(s)) / maxAbs * 10)
			if b >= 10 {
				b = 9
			}
			e.MagnitudeHistogram[b]++
		}
	}

	// Per-decided-symbol distribution (dibit 0:+1, 1:+3, 2:-1, 3:-3).
	labels := [4]string{"+1", "+3", "-1", "-3"}
	var vals [4][]float64
	for i, s := range soft {
		vals[dibits[i]&3] = append(vals[dibits[i]&3], float64(s))
	}
	for _, db := range [4]uint8{3, 2, 0, 1} { // eye order -3,-1,+1,+3
		e.PerDecidedSymbol = append(e.PerDecidedSymbol, railStat(labels[db], vals[db]))
	}

	// True-symbol outer-rail eye from the known FSW symbols.
	all, steady, trans, ok := trueSymbolEye(soft, winner, positions)
	if ok {
		outerLabels := [2]string{"+3", "-3"}
		for b := 0; b < 2; b++ {
			e.TrueOuterRail = append(e.TrueOuterRail, railStat(outerLabels[b], all[b]))
			_, ss, _, _, _ := distStats(steady[b])
			_, ts, _, _, _ := distStats(trans[b])
			e.OuterSpread = append(e.OuterSpread, SpreadStat{
				Label:     outerLabels[b],
				SteadyStd: ss, SteadyN: len(steady[b]),
				TransStd: ts, TransN: len(trans[b]),
				Ratio: ratioOrInf(ts, ss),
			})
		}
		e.Verdict = eyeVerdict(e.OuterSpread, e.TrueOuterRail)
	}
	return e
}

// trueSymbolEye attributes each FSW hit's aligned soft samples to their known
// outer symbol (+3 → bucket 0, −3 → bucket 1), overall and split by transition
// context. Ported from cmd/gophertrunk/iqdiag.go.
func trueSymbolEye(soft []float32, winner int, positions []int) (all, steady, trans [2][]float64, ok bool) {
	if len(positions) == 0 {
		return all, steady, trans, false
	}
	dibitToSymbol := [4]int8{1, 3, -1, -3}
	var expected [p25FSWLen]int8
	for kk := 0; kk < p25FSWLen; kk++ {
		nd := (p25phase1.FrameSyncWord[kk] + 4 - uint8(winner&3)) & 3
		expected[kk] = dibitToSymbol[nd]
	}
	bucket := func(sym int8) int {
		if sym > 0 {
			return 0
		}
		return 1
	}
	for _, pos := range positions {
		if pos < 0 || pos+p25FSWLen > len(soft) {
			continue
		}
		for kk := 0; kk < p25FSWLen; kk++ {
			b := bucket(expected[kk])
			v := float64(soft[pos+kk])
			all[b] = append(all[b], v)
			if kk > 0 {
				if p25phase1.FrameSyncWord[kk] == p25phase1.FrameSyncWord[kk-1] {
					steady[b] = append(steady[b], v)
				} else {
					trans[b] = append(trans[b], v)
				}
			}
		}
	}
	return all, steady, trans, true
}

// eyeVerdict mirrors iqdiag's advisory verdict: transition-driven spread ⇒
// ISI / timing; skewed but transition-independent ⇒ overshoot / TX
// nonlinearity; symmetric + transition-independent ⇒ amplitude-intrinsic.
func eyeVerdict(spread []SpreadStat, outer []RailStat) string {
	var maxRatio float64
	for _, s := range spread {
		if s.Ratio > maxRatio {
			maxRatio = s.Ratio
		}
	}
	var maxSkew float64
	for _, r := range outer {
		skew := r.Mean - r.P50
		if math.Abs(skew) > math.Abs(maxSkew) {
			maxSkew = skew
		}
	}
	switch {
	case maxRatio >= 1.5:
		return "transition-driven (post-transition ≫ steady): ISI / symbol-timing — chase the clock or an equalizer"
	case math.Abs(maxSkew) >= 0.05:
		return "skewed but transition-independent: overshoot / TX nonlinearity — not a timing/ISI fix"
	default:
		return "symmetric and transition-independent: amplitude-intrinsic (FM clicks / SNR) — not fixable in the symbol domain"
	}
}

// ratioOrInf returns a/b, or 0 when b≈0 (avoids divide-by-zero when one group
// is empty). Ported from cmd/gophertrunk/iqdiag.go.
func ratioOrInf(a, b float64) float64 {
	if b < 1e-9 {
		return 0
	}
	return a / b
}
