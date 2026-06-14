package dsp

import (
	"math"
	"math/cmplx"
	"sort"
)

// CarrierCandidate is one detected narrowband carrier: its offset from DC
// (Hz, positive = above centre) and its averaged power relative to the
// strongest carrier in the band (dB, ≤ 0; the dominant carrier is 0 dB).
type CarrierCandidate struct {
	OffsetHz   float64
	RelPowerDb float64
}

const (
	// defaultCarrierMinSpacingHz keeps adjacent coarse bins of one carrier
	// from each becoming a separate candidate. One P25/DMR channel step.
	defaultCarrierMinSpacingHz = 12_500.0
	// carrierCandidateFloorDb drops candidates more than this far below the
	// strongest carrier — a control channel buried 6–15 dB under a louder
	// voice carrier still qualifies, but noise bins (typically ≥ 30 dB down
	// after windowed averaging) do not.
	carrierCandidateFloorDb = 20.0
	// defaultMaxCarrierCandidates caps how many carriers the search returns.
	defaultMaxCarrierCandidates = 6
)

// EstimateCarrierOffsetHz finds the dominant narrowband carrier offset in a
// complex baseband capture: the frequency (in Hz, positive = above centre)
// carrying the most averaged power within ±searchHz of DC. It is the
// single-carrier wrapper over EstimateCarrierCandidatesHz, preserving the
// original two-stage coarse+fine behaviour. Returns 0 for an empty input or
// zero searchHz.
func EstimateCarrierOffsetHz(iq []complex64, sampleRateHz, searchHz float64) float64 {
	c := EstimateCarrierCandidatesHz(iq, sampleRateHz, searchHz, defaultCarrierMinSpacingHz, 1)
	if len(c) == 0 {
		return 0
	}
	return c[0].OffsetHz
}

// EstimateCarrierCandidatesHz returns up to maxCandidates carrier offsets in
// ±searchHz of DC, strongest first and each fine-refined. It generalises the
// single-peak EstimateCarrierOffsetHz so a control channel that is NOT the
// dominant carrier in a wideband capture can still be found and tried: a
// louder adjacent voice carrier no longer hides it. Coarse local maxima
// within minSpacingHz of an already-kept (stronger) candidate are suppressed,
// and candidates more than carrierCandidateFloorDb below the peak are dropped.
//
// The search is the same bounded averaged periodogram as the single-carrier
// path (a Goertzel-style DFT, no full FFT): a coarse pass scores every bin
// across the band, local maxima are kept (spacing- and floor-gated), then each
// is refined with a fine pass around its coarse bin.
//
// minSpacingHz ≤ 0 uses defaultCarrierMinSpacingHz; maxCandidates ≤ 0 uses
// defaultMaxCarrierCandidates. Returns nil for an empty input or zero searchHz.
func EstimateCarrierCandidatesHz(iq []complex64, sampleRateHz, searchHz, minSpacingHz float64, maxCandidates int) []CarrierCandidate {
	if len(iq) == 0 || searchHz <= 0 || sampleRateHz <= 0 {
		return nil
	}
	if minSpacingHz <= 0 {
		minSpacingHz = defaultCarrierMinSpacingHz
	}
	if maxCandidates <= 0 {
		maxCandidates = defaultMaxCarrierCandidates
	}
	x := make([]complex128, len(iq))
	for i, v := range iq {
		x[i] = complex(float64(real(v)), float64(imag(v)))
	}
	loN := -searchHz / sampleRateHz
	hiN := searchHz / sampleRateHz

	const coarseN = 257
	// Match the coarse DFT window to the grid step so the bin width ≈ the
	// spacing between grid points. A narrower window (the single-carrier path's
	// 4096) leaves gaps: an off-grid carrier falls in the DFT sidelobes and
	// reads tens of dB low, which is fine when you only need the peak bin's
	// LOCATION (the fine pass fixes it) but wrong when ranking carriers against
	// each other — an off-grid strong carrier would lose to an on-grid weak one.
	gridStepN := (hiN - loN) / float64(coarseN-1)
	coarseWin := int(math.Round(1.0 / gridStepN))
	if coarseWin < 64 {
		coarseWin = 64
	}
	if coarseWin > 4096 {
		coarseWin = 4096
	}
	freqs := make([]float64, coarseN)
	powers := make([]float64, coarseN)
	for i := 0; i < coarseN; i++ {
		freqs[i] = loN + (hiN-loN)*float64(i)/float64(coarseN-1)
		powers[i] = avgPowerNorm(x, freqs[i], coarseWin, 256)
	}

	// Collect local maxima (endpoints included, so a carrier at the band edge
	// — and always the global max — is a candidate).
	type pk struct{ f, p float64 }
	var peaks []pk
	for i := 0; i < coarseN; i++ {
		left := i == 0 || powers[i] > powers[i-1]
		right := i == coarseN-1 || powers[i] >= powers[i+1]
		if left && right {
			peaks = append(peaks, pk{freqs[i], powers[i]})
		}
	}
	if len(peaks) == 0 {
		return nil
	}
	sort.Slice(peaks, func(a, b int) bool { return peaks[a].p > peaks[b].p })

	// Min-spacing suppression + relative-power floor.
	minSpacingN := minSpacingHz / sampleRateHz
	floorRatio := math.Pow(10, -carrierCandidateFloorDb/10)
	pMax := peaks[0].p
	var kept []pk
	for _, c := range peaks {
		if c.p < pMax*floorRatio {
			continue
		}
		tooClose := false
		for _, k := range kept {
			if math.Abs(c.f-k.f) < minSpacingN {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
		kept = append(kept, c)
		if len(kept) >= maxCandidates {
			break
		}
	}

	// Fine-refine each kept coarse bin, then rank by refined power.
	stepN := (hiN - loN) / float64(coarseN-1)
	type ref struct{ f, p float64 }
	refined := make([]ref, len(kept))
	var refMax float64
	for i, c := range kept {
		ff := zoomPeakNorm(x, c.f-stepN, c.f+stepN, 201, 16384, 60)
		pp := avgPowerNorm(x, ff, 16384, 60)
		refined[i] = ref{ff, pp}
		if pp > refMax {
			refMax = pp
		}
	}
	sort.Slice(refined, func(a, b int) bool { return refined[a].p > refined[b].p })

	out := make([]CarrierCandidate, 0, len(refined))
	for _, r := range refined {
		rel := 0.0
		if refMax > 0 && r.p > 0 {
			rel = 10 * math.Log10(r.p/refMax)
		}
		out = append(out, CarrierCandidate{OffsetHz: r.f * sampleRateHz, RelPowerDb: rel})
	}
	return out
}

// zoomPeakNorm returns the normalised frequency (cycles/sample) in
// [f0, f1] with the greatest averaged |X(f)|² over up to maxWindows blocks
// of length win. Frequencies are evaluated by direct summation so the grid
// is independent of any FFT bin size.
func zoomPeakNorm(x []complex128, f0, f1 float64, nf, win, maxWindows int) float64 {
	if len(x) == 0 || win <= 0 {
		return (f0 + f1) / 2
	}
	bestF, bestP := (f0+f1)/2, -1.0
	for i := 0; i < nf; i++ {
		f := f0
		if nf > 1 {
			f = f0 + (f1-f0)*float64(i)/float64(nf-1)
		}
		if acc := avgPowerNorm(x, f, win, maxWindows); acc > bestP {
			bestP, bestF = acc, f
		}
	}
	return bestF
}

// avgPowerNorm is the averaged |X(f)|² at one normalised frequency f
// (cycles/sample), summed over up to maxWindows blocks of length win via a
// recursive phasor (one cmplx.Exp per call, then one complex multiply per
// sample). It is the shared inner kernel of the coarse grid scan and the fine
// zoom refinement.
func avgPowerNorm(x []complex128, f float64, win, maxWindows int) float64 {
	if win > len(x) {
		win = len(x)
	}
	if win <= 0 {
		return 0
	}
	step := cmplx.Exp(complex(0, -2*math.Pi*f))
	var acc float64
	nw := 0
	for off := 0; off+win <= len(x) && nw < maxWindows; off += win {
		ph := complex(1, 0)
		var s complex128
		for n := 0; n < win; n++ {
			s += x[off+n] * ph
			ph *= step
		}
		acc += real(s * cmplx.Conj(s))
		nw++
	}
	return acc
}
