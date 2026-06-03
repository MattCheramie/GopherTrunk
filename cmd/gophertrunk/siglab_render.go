package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// renderResultText writes a human-readable summary of a siglab.Result to w —
// the shared text rendering for `analyze`, `test`, and `replay`'s
// non-native-protocol path. It mirrors the layout of the historical replay
// EOF summary so operator muscle memory carries over, while covering every
// protocol uniformly off the structured model.
func renderResultText(w io.Writer, r *siglab.Result) {
	fmt.Fprintln(w, "----")
	fmt.Fprintf(w, "siglab: %s — %s — %d samples (%.2fs at %.0f Hz), %d symbols emitted\n",
		r.Source, r.Protocol, r.TotalSamples, r.DurationSec, r.SampleRateHz, r.Symbols)
	if r.PipelineRateHz != r.SampleRateHz {
		fmt.Fprintf(w, "siglab: ddc enabled  pipeline_rate_hz=%.0f  tune_hz=%.1f\n", r.PipelineRateHz, r.TuneHz)
	}

	if r.ExpectedBaud > 0 && r.EffectiveBaud > 0 {
		warning := ""
		if abs(r.BaudDeviationPct) > 2 {
			warning = "  (>2% — capture sample rate may not match -sample-rate)"
		}
		fmt.Fprintf(w, "siglab: effective baud %.1f (expected %.0f, deviation %+.1f%%)%s\n",
			r.EffectiveBaud, r.ExpectedBaud, r.BaudDeviationPct, warning)
	}

	if r.Locked {
		fmt.Fprintf(w, "siglab: LOCKED  freq=%d  latency=%.2fs", lockFreq(r), r.LockLatencySec)
		for _, k := range sortedKeys(lockFields(r)) {
			fmt.Fprintf(w, "  %s=%v", k, lockFields(r)[k])
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintln(w, "siglab: did NOT lock the control channel")
	}

	if len(r.Grants) > 0 {
		freqs := map[uint32]struct{}{}
		for _, g := range r.Grants {
			freqs[g.FrequencyHz] = struct{}{}
		}
		fmt.Fprintf(w, "siglab: %d grant(s) across %d frequencies\n", len(r.Grants), len(freqs))
	}

	if len(r.DecodeErrors) > 0 {
		fmt.Fprint(w, "siglab: decode errors:")
		for _, stage := range siglab.SortedDecodeErrors(r.DecodeErrors) {
			fmt.Fprintf(w, " %s=%d", stage, r.DecodeErrors[stage])
		}
		fmt.Fprintln(w)
	}

	if r.Signal != nil {
		renderSignal(w, r.Signal)
	}
	if r.P25P1 != nil {
		renderP25Detail(w, r.P25P1)
	}
	if r.Verdict != nil {
		renderVerdict(w, r.Verdict)
	}
}

func renderSignal(w io.Writer, s *siglab.SignalQuality) {
	fmt.Fprintf(w, "siglab: signal  symbols=%d  cardinality=%d\n", sumInt64(s.SymbolHistogram), s.SymbolCardinality)
	for i, n := range s.SymbolHistogram {
		pct := 0.0
		if i < len(s.SymbolHistogramPct) {
			pct = s.SymbolHistogramPct[i]
		}
		fmt.Fprintf(w, "siglab:   symbol %d: %d (%.1f%%)\n", i, n, pct)
	}
	if s.IQObserved {
		fmt.Fprintf(w, "siglab: raw IQ imbalance: gain=%+.3f dB  phase=%+.3f°  image_rejection=%.1f dB\n",
			s.IQGainImbalanceDB, s.IQPhaseImbalanceDeg, s.IQImageRejectionDB)
	}
	fmt.Fprintf(w, "siglab: decode-error rate: %.2f per 1000 symbols\n", s.DecodeErrorRate)
}

func renderP25Detail(w io.Writer, d *siglab.P25P1Detail) {
	fmt.Fprintf(w, "siglab: p25 detail  dibits=%d  winning_rotation=%d  hits=%d\n",
		d.DibitsBuffered, d.WinningRotation, d.WinningHits)
	fmt.Fprintf(w, "siglab:   dibit histogram: %d/%d/%d/%d\n",
		d.DibitHistogram[0], d.DibitHistogram[1], d.DibitHistogram[2], d.DibitHistogram[3])
	for rot := 0; rot < 4; rot++ {
		rs := d.Rotations[rot]
		fmt.Fprintf(w, "siglab:   rot %d: best_dist=%d (@%d) hits≤4=%d\n", rot, rs.BestDist, rs.BestPos, rs.Hits)
	}
	for _, nd := range d.NIDDecodes {
		fmt.Fprintf(w, "siglab:   nid@%d errs=%d nac=%#x duid=%d ok=%v\n", nd.Pos, nd.Errs, nd.NAC, nd.DUID, nd.OK)
	}
}

func renderVerdict(w io.Writer, v *siglab.Verdict) {
	status := "PASS"
	if !v.Pass {
		status = "FAIL"
	}
	fmt.Fprintf(w, "siglab: verdict = %s\n", status)
	for _, c := range v.Checks {
		fmt.Fprintf(w, "siglab:   %s\n", c)
	}
}

func lockFreq(r *siglab.Result) uint32 {
	if r.Lock == nil {
		return 0
	}
	return r.Lock.FrequencyHz
}

func lockFields(r *siglab.Result) map[string]any {
	if r.Lock == nil {
		return nil
	}
	return r.Lock.Fields
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sumInt64(v []int64) int64 {
	var t int64
	for _, n := range v {
		t += n
	}
	return t
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
