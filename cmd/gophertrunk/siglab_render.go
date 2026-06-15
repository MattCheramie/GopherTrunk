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
	switch d := r.Detail.(type) {
	case *siglab.P25P1Detail:
		renderP25Detail(w, d)
	case *siglab.ProtocolDetail:
		renderProtocolDetail(w, d)
	}
	if r.Verdict != nil {
		renderVerdict(w, r.Verdict)
	}
}

// renderProtocolDetail prints the symbol-domain deep dive (histogram + sync
// landscape + FEC tally) for the non-P25-P1 protocols.
func renderProtocolDetail(w io.Writer, d *siglab.ProtocolDetail) {
	fmt.Fprintf(w, "siglab: %s detail  symbols=%d  cardinality=%d\n", d.Protocol, d.SymbolsBuffered, d.SymbolCardinality)
	for i, n := range d.SymbolHistogram {
		pct := 0.0
		if i < len(d.SymbolHistogramPct) {
			pct = d.SymbolHistogramPct[i]
		}
		fmt.Fprintf(w, "siglab:   symbol %d: %d (%.1f%%)\n", i, n, pct)
	}
	if s := d.Sync; s != nil {
		fmt.Fprintf(w, "siglab:   sync landscape (len=%d, tol=%d): winner=%s rot=%d hits=%d modal_spacing=%d\n",
			s.SyncLen, s.Tolerance, s.WinnerVariant, s.WinnerRotation, s.WinnerHits, s.ModalSpacing)
		for _, st := range s.Stats {
			if st.Hits > 0 || st.BestDist <= s.Tolerance {
				fmt.Fprintf(w, "siglab:     %s rot %d: best_dist=%d (@%d) hits=%d\n",
					st.Variant, st.Rotation, st.BestDist, st.BestPos, st.Hits)
			}
		}
	}
	for _, f := range d.FEC {
		fmt.Fprintf(w, "siglab:   fec %s: frames=%d clean=%d corrected=%d uncorrectable=%d",
			f.Stage, f.Frames, f.Clean, f.Corrected, f.Uncorrectable)
		if f.CRCPass+f.CRCFail > 0 {
			fmt.Fprintf(w, " crc_pass=%d crc_fail=%d", f.CRCPass, f.CRCFail)
		}
		fmt.Fprintln(w)
	}
	if d.Notes != "" {
		fmt.Fprintf(w, "siglab:   note: %s\n", d.Notes)
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
		fmt.Fprintf(w, "siglab: raw IQ power: %.1f dBFS (RMS)\n", s.RMSPowerDBFS)
		fmt.Fprintf(w, "siglab: raw IQ imbalance: gain=%+.3f dB  phase=%+.3f°  image_rejection=%.1f dB\n",
			s.IQGainImbalanceDB, s.IQPhaseImbalanceDeg, s.IQImageRejectionDB)
	}
	fmt.Fprintf(w, "siglab: decode-error rate: %.2f per 1000 symbols\n", s.DecodeErrorRate)
	if d := s.Demod; d != nil {
		fmt.Fprintf(w, "siglab: demod (%s): EVM=%.1f%%  SNR≈%.1f dB  (n=%d symbols)\n",
			d.Modulation, d.EVMPct, d.SNREstimateDB, d.SymbolsAnalyzed)
	}
}

func renderP25Detail(w io.Writer, d *siglab.P25P1Detail) {
	if cc := d.CCStats; cc != nil {
		nidAttempts := cc.NIDTrusted + cc.NIDMarginal + cc.NIDFailed
		tsbkAttempts := cc.TSBKDecoded + cc.TSBKTrellisFailed + cc.TSBKCRCFailed
		fmt.Fprintf(w, "siglab: nid   trusted=%d  marginal=%d  uncorrectable=%d  (of %d FSW-hit attempts)\n",
			cc.NIDTrusted, cc.NIDMarginal, cc.NIDFailed, nidAttempts)
		fmt.Fprintf(w, "siglab: tsbk  decoded=%d  trellis_failed=%d  crc_failed=%d  (of %d NID-passed frames)\n",
			cc.TSBKDecoded, cc.TSBKTrellisFailed, cc.TSBKCRCFailed, tsbkAttempts)
	}
	if d.DibitsBuffered == 0 && d.SoftEye == nil {
		// No -diag landscape collected; CCStats/state already printed above.
		renderReceiverStates(w, d.ReceiverStates)
		return
	}
	fmt.Fprintf(w, "siglab: p25 detail  dibits=%d  winning_rotation=%d  hits=%d\n",
		d.DibitsBuffered, d.WinningRotation, d.WinningHits)
	if d.DibitsBuffered > 0 {
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
	if e := d.SoftEye; e != nil {
		fmt.Fprintf(w, "siglab: soft eye  dc=%+.5f  std=%.5f  mean|x|=%.5f  max|x|=%.5f\n",
			e.SignedMeanDC, e.StdDev, e.MeanAbs, e.MaxAbs)
		for _, r := range e.PerDecidedSymbol {
			fmt.Fprintf(w, "siglab:   rail %s: n=%d centroid=%+.4f std=%.4f p10/50/90=%+.4f/%+.4f/%+.4f\n",
				r.Label, r.N, r.Mean, r.Std, r.P10, r.P50, r.P90)
		}
		for _, s := range e.OuterSpread {
			fmt.Fprintf(w, "siglab:   %s spread: steady=%.4f (n=%d) post-transition=%.4f (n=%d) ratio=%.2f\n",
				s.Label, s.SteadyStd, s.SteadyN, s.TransStd, s.TransN, s.Ratio)
		}
		if e.Verdict != "" {
			fmt.Fprintf(w, "siglab:   → %s\n", e.Verdict)
		}
	}
	renderReceiverStates(w, d.ReceiverStates)
}

// renderReceiverStates prints the per-second receiver-state series (deep P25
// path). Kept compact — one line per snapshot — to mirror replay's old log.
func renderReceiverStates(w io.Writer, states []siglab.ReceiverState) {
	for _, s := range states {
		if s.CQPSK {
			fmt.Fprintf(w, "siglab: rx state  t=%.2fs  carrier_hz=%.3f  gardner_mu=%.4f  gardner_sps=%.2f  agc_gain=%.4g  cma_err=%.4g\n",
				s.TimeSec, s.CarrierHzEst, s.GardnerMu, s.GardnerSPS, s.CQPSKAGCGain, s.CMAError)
			continue
		}
		fmt.Fprintf(w, "siglab: rx state  t=%.2fs  afc_hz=%.3f  agc_level=%.6g  agc_target=%.6g  mm_mu=%.4f  mm_sps=%.2f  dda=%v\n",
			s.TimeSec, s.AFCHzEst, s.AGCLevel, s.AGCTarget, s.MMMu, s.MMSPS, s.DDAActive)
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
