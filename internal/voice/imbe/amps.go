package imbe

import "math"

// IMBE 4400 amplitude prep — TIA-102.BABA §6.2 entry path.
//
// Step 4a (synth.go) recovers log2(Ml)[1..L]. Before §6.2's
// spectral enhancement and §6.3 / §6.4's voiced + unvoiced
// excitation can run, those log-amplitudes need to be exponentiated
// to linear Ml, and the synthesizer needs cheap derived quantities
// (R_M0 = Σ Ml² for energy + enhancement, Vl voicing fraction for
// AGC + voice-activity hints).
//
// This file ships those utility primitives. The voiced harmonic
// generator (§6.3) and unvoiced FFT excitation (§6.4) land in
// follow-up PRs and consume what's here.

// AmplitudesFromLog2Ml fills dst[1..L] with the linear spectral
// amplitudes Ml[l] = 2^log2Ml[l]. Indices outside [1..L] are zeroed
// so that downstream loops over dst don't pick up stale values from
// a previous frame's longer L. dst[0] is unused per the project's
// 1-indexed convention.
//
// log2Ml[l] is the eq. 77 output (recovered log-amplitude). When
// l falls outside [1..L] the corresponding index in dst is forced
// to 0 — this matters because synthesis loops sometimes iterate
// over a fixed [1..56] range and rely on out-of-range entries
// being inert.
func AmplitudesFromLog2Ml(log2Ml *[57]float64, L int, dst *[57]float64) {
	for l := 0; l <= 56; l++ {
		dst[l] = 0
	}
	if L < 1 {
		return
	}
	if L > 56 {
		L = 56
	}
	for l := 1; l <= L; l++ {
		dst[l] = math.Exp2(log2Ml[l])
	}
}

// FrameEnergy returns R_M0 = Σ_{l=1..L} Ml[l]². This is the
// numerator of the §6.2 spectral-amplitude enhancement weight and
// — more practically for now — a per-frame energy estimate that
// lets the recorder + AGC distinguish voice from silence without
// waiting for the full synthesizer to produce PCM.
//
// Returns 0 for L < 1 so callers can pass a Silent frame's L
// without an extra guard.
func FrameEnergy(M *[57]float64, L int) float64 {
	if L < 1 {
		return 0
	}
	if L > 56 {
		L = 56
	}
	var sum float64
	for l := 1; l <= L; l++ {
		sum += M[l] * M[l]
	}
	return sum
}

// VoicingFraction returns the fraction of voiced harmonics in
// Vl[1..L] — Σ Vl[l] / L. Used as a coarse voiced/unvoiced hint
// for the upcoming synthesis combiner: pure-tone speech reports
// near 1.0, fricatives / silence near 0.0, ordinary voiced speech
// somewhere in between.
//
// Returns 0 for L < 1 so callers can pass a Silent frame without
// an extra guard. Vl entries outside {0, 1} count as zero — the
// upstream unpacker only writes 0 or 1, so any other value is a
// programming error and falls through to the safe default.
func VoicingFraction(Vl *[57]int, L int) float64 {
	if L < 1 {
		return 0
	}
	if L > 56 {
		L = 56
	}
	var voiced int
	for l := 1; l <= L; l++ {
		if Vl[l] == 1 {
			voiced++
		}
	}
	return float64(voiced) / float64(L)
}

// SpectralCosineSum returns R_M1 = Σ_{l=1..L} Ml[l]² · cos(ω₀ · l).
// This is the second moment used by the §6.2 enhancement weight
// (alongside R_M0 from FrameEnergy). Lives here rather than being
// inlined into the enhancer so tests can pin its closed-form
// value at known inputs and the enhancer stays a one-line
// plumbing layer over R_M0 / R_M1 / ω₀ / l when it lands.
//
// Returns 0 for L < 1 so Silent frames pass through without a
// guard.
func SpectralCosineSum(M *[57]float64, L int, w0 float64) float64 {
	if L < 1 {
		return 0
	}
	if L > 56 {
		L = 56
	}
	var sum float64
	for l := 1; l <= L; l++ {
		sum += M[l] * M[l] * math.Cos(w0*float64(l))
	}
	return sum
}
