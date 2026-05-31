package main

import (
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
)

// iqDiag accumulates the full dibit stream a replay produced and prints
// a structured demod-quality report at EOF — built for issue #275
// Phase B, where Phase A's NID-search widening (e60d3c5) ruled
// alignment out as the dominant failure mode on the Mt Anakie capture.
// The report surfaces what the existing nid-failure diag cannot:
//
//   - dibit-value histogram: a clean C4FM stream slices ~25% into each
//     of {0,1,2,3}; a strong skew tells the slicer thresholds are
//     mis-calibrated, an empty bin tells the slicer is collapsed.
//
//   - FSW-correlation histogram per rotation: for every dibit position
//     in the buffer, count the best Hamming distance to the 24-dibit
//     FrameSyncWord under each of the 4 cyclic rotations. The shape
//     tells whether the demod produces canonical dibits at all:
//
//   - tight low-mismatch peak under rot=0 ≈ clean C4FM
//
//   - peak under rot=2 only ≈ inverted discriminator polarity
//
//   - peak under rot=1 or rot=3 ≈ I/Q swap or 90° clock-recovery slip
//     (non-physical on a C4FM stream — strong evidence of a bug)
//
//   - no peak under any rotation ≈ SNR / demod-quality limited
//
//   - hit count per rotation: how many positions had a best-mismatch ≤
//     tolerance under each rotation. Asymmetry across rotations is the
//     same evidence as the peak shape but quantified.
//
// Designed to add nothing to the hot path: only allocates and runs
// when -diag is set.
type iqDiag struct {
	dibits []uint8
	// soft samples (pre-slicer matched-filter output) accumulated
	// from receiver.Options.SoftSink. Used to surface the slicer's
	// decision-region behaviour: if the matched-filter centres land
	// far outside the slicer's threshold, every sample collapses
	// to ±3 and the dibit histogram skews to {1, 3} only.
	soft []float32
	// iqStats accumulates the raw-IQ second-order moments (issue #402),
	// from which the report derives the front-end I/Q gain/phase imbalance
	// — the leading suspect for the asymmetric demodulated eye.
	iqStats rtlsdr.IQImbalanceStats
}

// distStats returns the mean, standard deviation, and 10th/50th/90th
// percentiles of v (sorts a copy; v is left unmodified). Zero values for an
// empty slice.
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

// printTrueSymbolEye measures the TRUE outer-rail eye from the FSW hits and
// localizes the mechanism of the #402 outer-rail spread.
//
// Every symbol of the P25 frame-sync word is an outer (±3) symbol and is
// known, so at each FSW hit the 24 aligned soft samples are attributed to
// their *known* symbol regardless of how the slicer decided them — an
// uncontaminated outer-rail distribution (the per-decided-symbol block above
// conflates true spread with slicer misclassification). Each sample is then
// split by transition context: whether its true symbol equals the previous
// FSW symbol (steady) or is a ±6 swing (post-transition). The comparison
// names the cause:
//
//   - post-transition std ≫ steady std → ISI / symbol-timing (steep
//     transitions are where off-centre sampling and channel ISI bite),
//   - roughly equal, symmetric → amplitude-intrinsic (FM clicks / SNR, or a
//     TX nonlinearity) — not an ISI/equalizer fix,
//   - post-transition skewed (mean ≫ median) → overshoot / ringing.
//
// winner is the rotation that aligns the demod's dibits to the canonical
// FrameSyncWord; the expected natural-frame symbol therefore un-rotates it.
// Hits with up to `tolerance` mis-sliced dibits are included on purpose —
// those mis-sliced outer symbols are exactly the heavy-tail outliers we want
// to see.
func (d *iqDiag) printTrueSymbolEye(w io.Writer, winner int, positions []int) {
	all, steady, trans, ok := d.trueSymbolEye(winner, positions)
	if !ok {
		return
	}

	labels := [2]string{"+3", "-3"}
	fmt.Fprintf(w, "diag: true-symbol outer-rail eye (from %d FSW hits; symbols known, slicer-independent):\n", len(positions))
	var skew [2]float64
	for b := 0; b < 2; b++ {
		mean, std, p10, p50, p90 := distStats(all[b])
		skew[b] = mean - p50
		fmt.Fprintf(w, "diag:   true %s: n=%d  mean=%+.4f  std=%.4f  p10/50/90=%+.4f/%+.4f/%+.4f  skew(mean−median)=%+.4f\n",
			labels[b], len(all[b]), mean, std, p10, p50, p90, skew[b])
	}
	var ratio [2]float64
	for b := 0; b < 2; b++ {
		_, ss, _, _, _ := distStats(steady[b])
		_, ts, _, _, _ := distStats(trans[b])
		ratio[b] = ratioOrInf(ts, ss)
		fmt.Fprintf(w, "diag:   %s spread: steady std=%.4f (n=%d)  post-transition std=%.4f (n=%d)  ratio=%.2f\n",
			labels[b], ss, len(steady[b]), ts, len(trans[b]), ratio[b])
	}

	// Advisory verdict. Thresholds are coarse — the numbers above are the
	// evidence; this just points at the next move.
	maxRatio := ratio[0]
	if ratio[1] > maxRatio {
		maxRatio = ratio[1]
	}
	maxSkew := skew[0]
	if math.Abs(skew[1]) > math.Abs(maxSkew) {
		maxSkew = skew[1]
	}
	switch {
	case maxRatio >= 1.5:
		fmt.Fprintln(w, "diag:   → outer spread is transition-driven (post-transition ≫ steady): ISI / symbol-timing — chase the Mueller-Müller clock or an equalizer.")
	case math.Abs(maxSkew) >= 0.05:
		fmt.Fprintln(w, "diag:   → outer spread is skewed but transition-independent: overshoot / TX nonlinearity — not a timing/ISI fix.")
	default:
		fmt.Fprintln(w, "diag:   → outer spread is symmetric and transition-independent: amplitude-intrinsic (FM clicks / SNR) — not fixable in the symbol domain.")
	}
}

// trueSymbolEye attributes the soft samples at each FSW hit to their known
// outer symbol (+3 → bucket 0, −3 → bucket 1), overall and split by transition
// context (steady = true symbol equals the previous FSW symbol; trans = a ±6
// swing). Returns ok=false when the soft/dibit buffers aren't aligned or there
// are no hits. Factored out of printTrueSymbolEye for testing.
func (d *iqDiag) trueSymbolEye(winner int, positions []int) (all, steady, trans [2][]float64, ok bool) {
	if len(d.soft) != len(d.dibits) || len(positions) == 0 {
		return all, steady, trans, false
	}
	const fswLen = 24
	dibitToSymbol := [4]int8{1, 3, -1, -3} // dibit 0,1,2,3 → +1,+3,-1,-3
	// expected[kk] is the natural-frame symbol of FSW position kk: undo the
	// winning rotation applied to the demod's dibits.
	var expected [fswLen]int8
	for kk := 0; kk < fswLen; kk++ {
		nd := (p25phase1.FrameSyncWord[kk] + 4 - uint8(winner&3)) & 3
		expected[kk] = dibitToSymbol[nd]
	}
	bucket := func(sym int8) int { // outer +3 → 0, outer −3 → 1
		if sym > 0 {
			return 0
		}
		return 1
	}
	for _, pos := range positions {
		if pos < 0 || pos+fswLen > len(d.soft) {
			continue
		}
		for kk := 0; kk < fswLen; kk++ {
			b := bucket(expected[kk])
			v := float64(d.soft[pos+kk])
			all[b] = append(all[b], v)
			if kk > 0 {
				// Transition is rotation-invariant, so compare canonical FSW.
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

// ratioOrInf returns a/b, or 0 when b≈0 (avoids a divide-by-zero in the
// transition-spread comparison when one group is empty).
func ratioOrInf(a, b float64) float64 {
	if b < 1e-9 {
		return 0
	}
	return a / b
}

// observe appends a chunk of dibits to the rolling buffer the EOF
// report walks. Called from the DibitSink in replay.go when -diag
// is set.
func (d *iqDiag) observe(dibits []uint8) {
	d.dibits = append(d.dibits, dibits...)
}

// observeIQ folds a chunk of raw (pre-DDC) IQ into the I/Q-imbalance
// moments. Called from the read loop in replay.go when -diag is set.
func (d *iqDiag) observeIQ(raw []complex64) {
	d.iqStats.Observe(raw)
}

// observeSoft appends pre-slicer per-symbol soft samples to the
// rolling soft buffer.
func (d *iqDiag) observeSoft(soft []float32) {
	d.soft = append(d.soft, soft...)
}

// printReport walks the accumulated dibit stream and emits the full
// demod-quality report to w. Idempotent.
func (d *iqDiag) printReport(w io.Writer) {
	if len(d.dibits) == 0 {
		fmt.Fprintln(w, "diag: no dibits emitted — receiver produced nothing")
		return
	}

	fmt.Fprintln(w, "---- diag ----")
	fmt.Fprintf(w, "diag: %d dibits buffered\n", len(d.dibits))

	// Pre-slicer soft-sample distribution (C4FM path: matched-filter
	// output at symbol times, in rad/sample). For a clean P25 Phase 1
	// signal at 1800 Hz peak deviation, outer centres sit at
	// ±2π·1800/SampleRateHz and inner at ±(1/3) of that, with the
	// slicer threshold at ±(2/3)·outer. If outer-bin counts ≫
	// inner-bin counts, the matched-filter output is bigger than the
	// slicer expects and inner symbols are being misclassified as
	// outer (issue #275 Phase B).
	if len(d.soft) > 0 {
		var sumAbs, maxAbs float64
		var negPosInner, negPosOuter, signZero int
		for _, s := range d.soft {
			a := float64(s)
			if a < 0 {
				a = -a
			}
			sumAbs += a
			if a > maxAbs {
				maxAbs = a
			}
			if s == 0 {
				signZero++
			}
			// Bin into rough quartiles around the abs-mean to spot
			// inner-vs-outer balance.
			_, _ = negPosInner, negPosOuter
		}
		meanAbs := sumAbs / float64(len(d.soft))
		// Threshold at the inner-vs-outer slicer boundary the C4FM
		// stage uses (2/3 of the deviation calibration). Without
		// access to the receiver's deviation here we use meanAbs as
		// a proxy mid-point — most useful as a relative comparison.
		var nInner, nOuter int
		for _, s := range d.soft {
			a := float64(s)
			if a < 0 {
				a = -a
			}
			if a < meanAbs {
				nInner++
			} else {
				nOuter++
			}
		}
		fmt.Fprintf(w, "diag: pre-slicer soft samples: n=%d  mean|x|=%.6f  max|x|=%.6f  <mean|x|: %d (%.1f%%)  ≥mean|x|: %d (%.1f%%)\n",
			len(d.soft), meanAbs, maxAbs,
			nInner, 100*float64(nInner)/float64(len(d.soft)),
			nOuter, 100*float64(nOuter)/float64(len(d.soft)))

		// Coarse histogram of soft sample magnitudes in deciles
		// of maxAbs — surfaces whether the distribution is bimodal
		// (clean 4-level eye: two peaks near ±inner and ±outer
		// centres) or unimodal saturated (everything piled near
		// ±max).
		const bins = 10
		var mhist [bins]int
		for _, s := range d.soft {
			a := float64(s)
			if a < 0 {
				a = -a
			}
			b := int(a / maxAbs * float64(bins))
			if b >= bins {
				b = bins - 1
			}
			mhist[b]++
		}
		fmt.Fprintln(w, "diag: |soft| magnitude histogram (10 bins of max|x|):")
		for b := 0; b < bins; b++ {
			fmt.Fprintf(w, "diag:   [%.3f,%.3f): %7d  (%5.2f%%)\n",
				float64(b)/bins*maxAbs, float64(b+1)/bins*maxAbs,
				mhist[b], 100*float64(mhist[b])/float64(len(d.soft)))
		}

		// Signed-eye summary (issue #402). The magnitude histogram above
		// folds sign away, so it cannot see the eye-skew the reporter is
		// hitting — a heavily negative dibit distribution can sit under a
		// "sane-looking" |x| spread. These two views keep the sign:
		//
		//   - Overall signed mean (DC): the AFC should have driven this to
		//     ~0. A non-zero value is residual carrier DC shifting the
		//     whole eye off the slicer's symmetric thresholds — outer-
		//     positive symbols then fall short of the +threshold while
		//     outer-negative overshoot, exactly the inner/negative-heavy
		//     skew #402 shows.
		//
		//   - Per-decided-symbol centroids: the mean soft value of the
		//     samples the slicer assigned to each of +3/+1/-1/-3 (decided
		//     dibit 1/0/2/3 — see phase1.SymbolToDibit). On a clean eye
		//     these are symmetric (|+3|≈|-3|, |+1|≈|-1|) and the outer/
		//     inner ratio is ~3. Asymmetry localises the fault: a uniform
		//     offset = DC shift, compressed outer centroids = scale error
		//     (outer symbols not reaching the eye corners), one-sided =
		//     gain/deviation-calibration asymmetry.
		var signedSum float64
		for _, s := range d.soft {
			signedSum += float64(s)
		}
		signedMean := signedSum / float64(len(d.soft))
		var sqdev float64
		for _, s := range d.soft {
			dv := float64(s) - signedMean
			sqdev += dv * dv
		}
		std := math.Sqrt(sqdev / float64(len(d.soft)))
		fmt.Fprintf(w, "diag: signed soft samples: mean=%+.6f (DC; want ~0)  stddev=%.6f\n", signedMean, std)

		// Group soft samples by the slicer's decision. d.dibits and
		// d.soft are appended in lockstep (one dibit and one soft sample
		// per recovered symbol, same order), so they align index-for-
		// index when their lengths match. Skip if they don't (e.g. a
		// future caller wires only one sink).
		if len(d.dibits) == len(d.soft) {
			// dibit → label and slice index. Mapping per
			// phase1.SymbolToDibit: 0:+1, 1:+3, 2:-1, 3:-3.
			labels := [4]string{"+1", "+3", "-1", "-3"}
			var vals [4][]float64
			for i, s := range d.soft {
				db := d.dibits[i] & 3
				vals[db] = append(vals[db], float64(s))
			}
			// Per-rail centroid AND spread (std + 10/50/90 percentiles). The
			// spread is the #402 discriminator: a rail whose centroid looks
			// fine but whose p10..p90 is very wide is *spread* (e.g. the +3
			// outer population landing low), which is why a threshold derived
			// from the centroid mis-slices it — distinct from a tight cluster.
			fmt.Fprintln(w, "diag: per-decided-symbol soft distribution (clean eye: ±outer ≈ 3×±inner, symmetric, tight):")
			for _, db := range [4]uint8{3, 2, 0, 1} { // eye order -3,-1,+1,+3
				v := vals[db]
				mean, std, p10, p50, p90 := distStats(v)
				fmt.Fprintf(w, "diag:   %s (dibit %d): n=%7d (%5.2f%%)  centroid=%+.6f  std=%.6f  p10/50/90=%+.4f/%+.4f/%+.4f\n",
					labels[db], db, len(v), 100*float64(len(v))/float64(len(d.soft)), mean, std, p10, p50, p90)
			}
		}
	}

	// I/Q imbalance on the raw (pre-DDC) IQ (issue #402). The RX chain is
	// provably symmetric, so an asymmetric eye must come from the IQ samples;
	// an uncorrected front-end I/Q imbalance (worst at the on-channel DC the
	// DDC sits on) is the leading suspect. A clean front-end is balanced
	// (≈0 dB gain, ≈0° phase) with image rejection ≳ 40 dB. Re-run with
	// -iq-correct to A/B.
	if d.iqStats.Count() > 0 {
		bal := d.iqStats.Balancer()
		fmt.Fprintf(w, "diag: raw IQ imbalance: gain=%+.3f dB  phase=%+.3f°  image_rejection=%.1f dB  (correction GainQ=%.4f Phase=%+.4f rad)\n",
			d.iqStats.GainImbalanceDB(), d.iqStats.PhaseImbalanceDeg(), d.iqStats.ImageRejectionDB(), bal.GainQ, bal.Phase)
	}

	// Dibit value histogram — should be ~25% per bin on a clean C4FM
	// control channel carrying TSDU traffic. A bin near zero says the
	// 4-level slicer collapsed; a single dominant bin says the signal
	// is below the slicer thresholds and everything quantizes to one
	// symbol.
	var hist [4]int
	for _, db := range d.dibits {
		hist[db&3]++
	}
	fmt.Fprintln(w, "diag: dibit value histogram (expect ~25% per bin on clean C4FM):")
	total := float64(len(d.dibits))
	for v := 0; v < 4; v++ {
		fmt.Fprintf(w, "diag:   %d: %7d  (%5.2f%%)\n", v, hist[v], 100*float64(hist[v])/total)
	}

	// FSW correlation landscape per rotation. At every dibit position,
	// compute the Hamming distance from the next 24 dibits to the
	// canonical FrameSyncWord under each cyclic rotation, and
	// histogram the best distance per rotation. The shape of the
	// distribution at low distances tells whether the demod produces
	// canonical dibits and which rotation aligns the stream.
	type rotStats struct {
		hist     [25]int // distance 0..24
		hits     int     // positions with mismatch ≤ tolerance
		bestDist int     // global minimum across all positions
		bestPos  int
	}
	const fswLen = 24
	const tolerance = 4
	var stats [4]rotStats
	for i := range stats {
		stats[i].bestDist = fswLen + 1
	}
	if len(d.dibits) >= fswLen {
		for pos := 0; pos+fswLen <= len(d.dibits); pos++ {
			for rot := uint8(0); rot < 4; rot++ {
				mismatch := 0
				for kk := 0; kk < fswLen; kk++ {
					if ((d.dibits[pos+kk] + rot) & 3) != p25phase1.FrameSyncWord[kk] {
						mismatch++
					}
				}
				stats[rot].hist[mismatch]++
				if mismatch <= tolerance {
					stats[rot].hits++
				}
				if mismatch < stats[rot].bestDist {
					stats[rot].bestDist = mismatch
					stats[rot].bestPos = pos
				}
			}
		}
	}

	fmt.Fprintln(w, "diag: FSW correlation per rotation (Hamming distance histogram, tolerance=4):")
	fmt.Fprintln(w, "diag:   rot  best_dist  hits≤4   dist=0..6 counts")
	for rot := 0; rot < 4; rot++ {
		s := stats[rot]
		fmt.Fprintf(w, "diag:   %d    %3d (@pos %d)  %6d   %d/%d/%d/%d/%d/%d/%d\n",
			rot, s.bestDist, s.bestPos, s.hits,
			s.hist[0], s.hist[1], s.hist[2], s.hist[3], s.hist[4], s.hist[5], s.hist[6])
	}

	// Pick the rotation with the most low-distance hits as the
	// "winning" rotation, and show the first few hit positions so the
	// reader can spot whether they cluster at regular frame intervals
	// (good) or are scattered (likely false positives).
	winner := 0
	for rot := 1; rot < 4; rot++ {
		if stats[rot].hits > stats[winner].hits {
			winner = rot
		}
	}
	fmt.Fprintf(w, "diag: winning rotation = %d (%d hits ≤ tolerance)\n", winner, stats[winner].hits)
	if stats[winner].hits > 0 {
		positions := make([]int, 0, stats[winner].hits)
		for pos := 0; pos+fswLen <= len(d.dibits); pos++ {
			mismatch := 0
			for kk := 0; kk < fswLen; kk++ {
				if ((d.dibits[pos+kk] + uint8(winner)) & 3) != p25phase1.FrameSyncWord[kk] {
					mismatch++
				}
			}
			if mismatch <= tolerance {
				positions = append(positions, pos)
			}
		}
		// Inter-hit deltas: a real control channel emits FSWs at
		// frame intervals (~163 on-air dibits including status
		// symbols for a TSBK frame, ~864 + status for an LDU). A
		// modal delta of one of those values says we're detecting
		// real frames; broadly scattered deltas say most hits are
		// false positives.
		var deltas []int
		for i := 1; i < len(positions); i++ {
			deltas = append(deltas, positions[i]-positions[i-1])
		}
		sort.Ints(deltas)
		fmt.Fprintf(w, "diag: first %d hit positions (rot=%d):", min(8, len(positions)), winner)
		for i := 0; i < min(8, len(positions)); i++ {
			fmt.Fprintf(w, " %d", positions[i])
		}
		fmt.Fprintln(w)
		if len(deltas) > 0 {
			modal := modalInt(deltas)
			fmt.Fprintf(w, "diag: inter-hit dibit-deltas: min=%d  median=%d  max=%d  modal=%d  (P25 TSBK frame ≈ 163 on-air dibits, LDU ≈ 864)\n",
				deltas[0], deltas[len(deltas)/2], deltas[len(deltas)-1], modal)
		}

		// True-symbol outer-rail eye: the FSW symbols are all known (and all
		// outer), so this measures the real outer-rail spread free of slicer
		// misclassification, and localizes its mechanism (issue #402).
		d.printTrueSymbolEye(w, winner, positions)

		// For each of the first few perfect-distance FSW hits, dump
		// the FSW + next 32 dibits raw AND run them through the
		// real BCH NID decoder under both strip modes. If any
		// decode, the issue is signal-quality (some NIDs corrupt,
		// some clean); if none decode despite perfect FSW
		// alignment, framing (status-symbol offset, dibit-bit
		// ordering) is broken.
		printed := 0
		for _, pos := range positions {
			if printed >= 5 {
				break
			}
			mismatch := 0
			for kk := 0; kk < fswLen; kk++ {
				if ((d.dibits[pos+kk] + uint8(winner)) & 3) != p25phase1.FrameSyncWord[kk] {
					mismatch++
				}
			}
			if mismatch != 0 || pos+fswLen+33 > len(d.dibits) {
				continue
			}
			printed++
			fmt.Fprintf(w, "diag: FSW@%d (dist=0, rot=%d) — payload[0..31] then BCH:\n", pos, winner)
			rawNID := make([]uint8, 33) // 32 NID + 1 for strip variant
			for j := 0; j < 33; j++ {
				rawNID[j] = (d.dibits[pos+fswLen+j] + uint8(winner)) & 3
			}
			fmt.Fprint(w, "diag:   raw  ")
			for j := 0; j < 32; j++ {
				fmt.Fprintf(w, "%d", rawNID[j])
				if (j+1)%4 == 0 {
					fmt.Fprint(w, " ")
				}
			}
			fmt.Fprintln(w)
			// Strip=false: take payload[0..31] directly
			n0, e0, _, err0 := p25phase1.NIDFromDibitsWithErrors(rawNID[:32])
			// Strip=true: skip payload[11] (the spec status-symbol position
			// when FSW is at frame_pos 0..23 and status falls at 35)
			stripped := make([]uint8, 0, 32)
			stripped = append(stripped, rawNID[:11]...)
			stripped = append(stripped, rawNID[12:33]...)
			n1, e1, _, err1 := p25phase1.NIDFromDibitsWithErrors(stripped)
			fmt.Fprintf(w, "diag:   strip=false: errs=%d nac=%#x duid=%d err=%v\n", e0, n0.NAC, n0.DUID, err0)
			fmt.Fprintf(w, "diag:   strip=true:  errs=%d nac=%#x duid=%d err=%v\n", e1, n1.NAC, n1.DUID, err1)
			// TSBK[0..23] dump — if this is a real control channel the
			// TSBK content varies across frames; a fixed pattern at this
			// 360-dibit cadence is a periodic broadcast (probably not a
			// TSDU at all). Comparing TSBKs across the first few FSWs
			// distinguishes the two without running the trellis decoder.
			tsbkLen := 24
			if pos+fswLen+32+tsbkLen <= len(d.dibits) {
				fmt.Fprint(w, "diag:   tsbk[0..23] ")
				for j := 0; j < tsbkLen; j++ {
					tb := (d.dibits[pos+fswLen+32+j] + uint8(winner)) & 3
					fmt.Fprintf(w, "%d", tb)
					if (j+1)%4 == 0 {
						fmt.Fprint(w, " ")
					}
				}
				fmt.Fprintln(w)
			}
			// Trellis-decode the 98 channel dibits after the NID under
			// the on-air status-symbol layout (status at frame pos 35,
			// 71, 107, 143 — NID positions 11, 47, 83, 119) and dump
			// the resulting 12-byte info block plus the augmented-CRC
			// check that validates the trailer. A successful TSBK
			// decode produces metric=0 (clean Viterbi path) and
			// crc=0x0000 (trailer matches under TIA-102.AABF augmented
			// CRC). Issue #275 Phase B part 3 surfaced the wrong CRC
			// convention in the production code; this dump confirmed
			// on-air trailers verify cleanly under the augmented
			// variant.
			needed := fswLen + 32 + 98 + 4 // pad for status symbols
			if pos+needed <= len(d.dibits) {
				channel := make([]uint8, 0, 98)
				fswStart := pos
				// First TSBK data dibit follows FSW(24) + NID(32 data + 1
				// status at frame_pos 35) = 57 on-air positions in.
				start := pos + fswLen + 33
				i := start
				for len(channel) < 98 && i < len(d.dibits) {
					if (i-fswStart)%36 != 35 {
						channel = append(channel, (d.dibits[i]+uint8(winner))&3)
					}
					i++
				}
				if len(channel) == 98 {
					decoded := p25phase1.DeinterleaveTSBK(channel)
					info, metric := p25phase1.DecodeTrellis(decoded)
					bytes := make([]byte, 12)
					for k := 0; k < 12; k++ {
						bytes[k] = (info[4*k+0] << 6) | (info[4*k+1] << 4) | (info[4*k+2] << 2) | info[4*k+3]
					}
					fmt.Fprintf(w, "diag:   tsbk info (metric=%d): % 02X  crc=0x%04X (0 means valid)\n",
						metric, bytes, framing.CRCCCITTAugmented(bytes))
				}
			}
		}
	}
}

// modalInt returns the most-frequent value in a sorted int slice. Ties
// resolve to the smaller value.
func modalInt(sorted []int) int {
	if len(sorted) == 0 {
		return 0
	}
	bestVal := sorted[0]
	bestCount := 1
	curVal := sorted[0]
	curCount := 1
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == curVal {
			curCount++
		} else {
			if curCount > bestCount {
				bestVal, bestCount = curVal, curCount
			}
			curVal, curCount = sorted[i], 1
		}
	}
	if curCount > bestCount {
		bestVal = curVal
	}
	return bestVal
}
