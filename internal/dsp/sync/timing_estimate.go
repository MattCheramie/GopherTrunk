package sync

import "math"

// EstimateSymbolPhase is a non-data-aided, feed-forward symbol-timing
// estimate for a shaped multi-level PAM signal: an eye-opening search. For
// each of the sps candidate sampling phases it decimates the window and
// measures the kurtosis E[x⁴]/E[x²]² of the samples; at the ISI-free decision
// instant a 4-level signal takes four discrete values (kurtosis 1.64 for
// equiprobable ±1/±3), while between instants the samples are ISI mixtures
// that smear toward Gaussian (kurtosis → 3). The phase with the lowest
// kurtosis is the symbol instant; a parabolic fit across its neighbours
// refines it below one sample. The statistic is scale-free (no AGC needed)
// and, unlike the square-law spectral-line estimator, does not depend on the
// pulse's excess bandwidth — on the 20 % roll-off of the C4FM family the
// symbol-rate line is often weaker than the noise floor, while the eye stays
// open.
//
// tau is the position, in samples relative to samples[0], of the symbol
// instants (samples[tau + k·sps] are the decision points), in [0, sps). It
// needs ~64–128 symbols and no prior phase, which is what a feedback loop
// lacks: a Mueller-Müller loop pulls in at gain·error per symbol and, started
// near a half-symbol offset (its unstable equilibrium), can take seconds to
// converge — measured on the #836 reporter's direct-mode DMR capture, three
// of ten sub-symbol start phases cost 1.3–2.8 s before the first sync word
// while the other seven decoded the first burst. Seeding the loop from this
// estimate (MuellerMuller.SetPhase) makes acquisition phase-independent.
//
// ok is false when the window is too short (fewer than minPhaseSymbols per
// phase) or shows no eye: the best phase's kurtosis is not below
// maxEyeKurtosis (noise, or an unshaped/continuous-phase input), or the
// spread between the best and worst phase is under minEyeContrast (a flat
// statistic carries no timing information). The window should contain only
// signal: callers on a burst carrier start it inside the burst, past the
// matched filter's transient.
func EstimateSymbolPhase(samples []float32, sps float64) (tau float64, ok bool) {
	n := int(sps + 0.5)
	if n < 2 || len(samples) < minPhaseSymbols*n {
		return 0, false
	}
	var kurt [64]float64
	if n > len(kurt) {
		return 0, false
	}
	best, bestK, worstK := -1, math.Inf(1), 0.0
	for p := 0; p < n; p++ {
		// Central moments: an uncorrected carrier offset is a DC bias on
		// the discriminator output (before the coarse acquirer engages, and
		// the residual after), which would skew raw moments toward a single
		// lump; the eye's shape is in the spread about the mean.
		var mean float64
		cnt := 0
		for i := p; i < len(samples); i += n {
			mean += float64(samples[i])
			cnt++
		}
		if cnt < minPhaseSymbols {
			return 0, false
		}
		mean /= float64(cnt)
		var m2, m4 float64
		for i := p; i < len(samples); i += n {
			d := float64(samples[i]) - mean
			d2 := d * d
			m2 += d2
			m4 += d2 * d2
		}
		if m2 <= 0 {
			return 0, false
		}
		m2 /= float64(cnt)
		m4 /= float64(cnt)
		kurt[p] = m4 / (m2 * m2)
		if kurt[p] < bestK {
			bestK, best = kurt[p], p
		}
		if kurt[p] > worstK {
			worstK = kurt[p]
		}
	}
	if bestK > maxEyeKurtosis || worstK < bestK*minEyeContrast {
		return 0, false
	}
	return refinePhase(kurt[:n], best, sps), true
}

// refinePhase fits a parabola through stat's minimum at best and its two
// (circular) neighbours and returns the vertex as a phase in [0, sps). The
// vertex offset is clamped to ±0.5 so a lopsided eye cannot push the
// estimate past the neighbouring integer phase.
func refinePhase(stat []float64, best int, sps float64) float64 {
	n := len(stat)
	km := stat[(best+n-1)%n]
	kp := stat[(best+1)%n]
	denom := km - 2*stat[best] + kp
	frac := 0.0
	if denom > 0 {
		frac = 0.5 * (km - kp) / denom
		if frac > 0.5 {
			frac = 0.5
		} else if frac < -0.5 {
			frac = -0.5
		}
	}
	return math.Mod(float64(best)+frac+sps, sps)
}

// EstimateSymbolPhaseCM is EstimateSymbolPhase for a linearly modulated PSK
// signal after its matched filter (P25 Phase 2 H-DQPSK, the π/4-DQPSK
// family): a constant-modulus eye search. At the ISI-free decision instant
// every sample of an RRC-matched PSK signal has the same magnitude, so the
// normalised fourth moment E[|y|⁴]/E[|y|²]² is 1; between instants the
// samples are ISI mixtures whose magnitude varies, and complex Gaussian
// noise sits at 2. The phase with the lowest ratio is the symbol instant,
// refined below one sample like EstimateSymbolPhase. The statistic depends
// only on |y|, so it needs no carrier lock, ignores any residual frequency
// offset, and is scale-free (no AGC needed).
//
// Same contract as EstimateSymbolPhase: tau in [0, sps), samples[tau + k·sps]
// are the decision points; ok is false for a window too short, or one that
// shows no eye (best ratio above maxCMRatio, or less than minCMContrast
// between the best and worst phase).
func EstimateSymbolPhaseCM(samples []complex64, sps float64) (tau float64, ok bool) {
	n := int(sps + 0.5)
	if n < 2 || len(samples) < minPhaseSymbols*n {
		return 0, false
	}
	var ratio [64]float64
	if n > len(ratio) {
		return 0, false
	}
	best, bestR, worstR := -1, math.Inf(1), 0.0
	for p := 0; p < n; p++ {
		var m2, m4 float64
		cnt := 0
		for i := p; i < len(samples); i += n {
			re, im := float64(real(samples[i])), float64(imag(samples[i]))
			a := re*re + im*im
			m2 += a
			m4 += a * a
			cnt++
		}
		if cnt < minPhaseSymbols || m2 <= 0 {
			return 0, false
		}
		m2 /= float64(cnt)
		m4 /= float64(cnt)
		ratio[p] = m4 / (m2 * m2)
		if ratio[p] < bestR {
			bestR, best = ratio[p], p
		}
		if ratio[p] > worstR {
			worstR = ratio[p]
		}
	}
	if bestR > maxCMRatio || worstR < bestR*minCMContrast {
		return 0, false
	}
	return refinePhase(ratio[:n], best, sps), true
}

const (
	// minPhaseSymbols is the fewest decimated samples per candidate phase the
	// kurtosis is trusted on (its estimator's standard deviation is about
	// sqrt(24/N)).
	minPhaseSymbols = 32
	// maxEyeKurtosis is the best-phase kurtosis above which the window is
	// treated as showing no eye. A 4-level PAM at the instant sits at 1.6–2.0
	// (ISI residue and noise raise it from the ideal 1.64); Gaussian noise
	// decimated at any phase sits at 3 ± sqrt(24/N) — at N = 96, 2.5 is ~1
	// standard deviation below it and 2.2 is ~1.6 below.
	maxEyeKurtosis = 2.2
	// minEyeContrast is the minimum worst/best kurtosis ratio across phases.
	// A shaped 4-level signal spans ~1.7 → ~2.4 (ratio ≥ 1.3); noise varies
	// only by estimation error.
	minEyeContrast = 1.12

	// maxCMRatio is the best-phase E[|y|⁴]/E[|y|²]² above which
	// EstimateSymbolPhaseCM treats the window as showing no eye. An
	// RRC-matched PSK signal at its instant sits near 1 (noise and ISI
	// residue raise it); complex Gaussian noise sits at 2 at every phase.
	maxCMRatio = 1.5
	// minCMContrast is the minimum worst/best ratio across phases for
	// EstimateSymbolPhaseCM: off the instant the PSK magnitude varies with
	// the neighbouring symbols, while noise varies only by estimation error.
	minCMContrast = 1.1
)

// SetPhase seeds the loop so its next symbol instant lands mu samples after
// the last sample it processed (0 < mu ≤ sps; values outside are wrapped
// into that range), and forgets the previous symbol so the first error term
// after the seed is formed between two symbols of the new phase rather than
// against a stale one. Intended for a feed-forward estimate at a burst onset
// (EstimateSymbolPhase); the nominal sps and gain are untouched.
func (m *MuellerMuller) SetPhase(mu float64) {
	mu = math.Mod(mu, m.sps)
	if mu <= 0 {
		mu += m.sps
	}
	m.mu = mu
	m.prevSym = 0
	m.have = false
}
