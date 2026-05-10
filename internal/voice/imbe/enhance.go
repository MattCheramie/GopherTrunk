package imbe

import "math"

// IMBE 4400 spectral-amplitude enhancement — TIA-102.BABA §6.2.
//
// After §6.1 cross-frame log-amplitude recovery (synth.go) and the
// log2(Ml) → linear Ml conversion (amps.go), the synthesizer's
// per-harmonic amplitudes carry the spec's quantized envelope but
// not the spec's "compensated" envelope. §6.2 boosts harmonics that
// the model under-represents (typically mid-band harmonics whose
// energy gets averaged into adjacent bands by the PRBA + HOC
// pipeline) so the spectral envelope tilts more naturally on
// playback.
//
// The enhancement is a per-harmonic multiplier W_l that depends on
// the spectral moments R_M0 = Σ Ml² and R_M1 = Σ Ml² · cos(ω₀·l)
// and the harmonic position. Low-frequency harmonics (l ≤ L/8) and
// the very-high-frequency band are left at W = 1; mid-band harmonics
// get a weight derived from the closed-form below, clamped to a
// safe range so the enhancement doesn't blow up on degenerate
// frames.
//
// After the per-harmonic multiply, total frame energy is renormalized
// so R_M0 (the integrated power across all L harmonics) is preserved.
// That keeps the absolute amplitude of the synthesized output stable
// across frames where the enhancement happens to redistribute energy
// significantly.
//
// Algorithmic reference: TIA-102.BABA §6.2 + szechyjs/mbelib's
// mbe_spectralAmpEnhance equivalent (ISC-licensed; attribution in
// tables.go). Exact constants in the closed form (the 0.96 scale, the
// W ∈ [0.5, 1.2] clamp, the L/8 lower-band cutoff) follow common
// public references; spec-tuning to bit-match mbelib output is part
// of step 5c (gain calibration).

// EnhanceWMin / EnhanceWMax bound the per-harmonic enhancement
// weight so a frame whose R_M0² − R_M1² approaches zero (a
// near-pure-tone spectrum) doesn't produce a runaway W_l. The
// [0.5, 1.2] band is the standard published range.
const (
	EnhanceWMin = 0.5
	EnhanceWMax = 1.2
)

// EnhanceAmplitudes applies the §6.2 spectral-amplitude enhancement
// to M[1..L] in place: each Ml gets multiplied by a per-harmonic
// weight W_l, then the entire frame is rescaled so total energy
// (Σ Ml²) is preserved.
//
// The per-harmonic weight is:
//
//	if 8·l ≤ L:        W_l = 1.0          (low band: untouched)
//	else:
//	    c = cos(ω₀ · l)
//	    num = R_M0² + R_M1² − 2·R_M0·R_M1·c
//	    den = R_M0 · (R_M0² − R_M1²)
//	    if den ≤ 0:    W_l = 1.0          (degenerate frame: skip)
//	    else:
//	        ξ = 0.96 · num / den
//	        W_l = ξ^0.25
//	        clamped to [EnhanceWMin, EnhanceWMax]
//
// Silent + zero-L frames + degenerate (R_M0 ≤ 0) frames are no-ops
// so callers can invoke unconditionally on the synthesis path.
func EnhanceAmplitudes(p Params, M *[57]float64) {
	if p.Silent || p.L == 0 {
		return
	}
	L := p.L
	if L > 56 {
		L = 56
	}

	rm0 := FrameEnergy(M, L)
	if rm0 <= 0 {
		// Degenerate frame (e.g., all-zero amplitudes): nothing to
		// enhance, nothing to rescale.
		return
	}
	rm1 := SpectralCosineSum(M, L, p.W0)

	// Pre-compute the closed-form denominator. R_M0² − R_M1² ≥ 0 by
	// Cauchy-Schwarz when M_l² are non-negative weights, but the
	// strict inequality fails on a pure tone (one nonzero harmonic)
	// where R_M1² = R_M0². Guard against the sign + zero edge.
	rm0Sq := rm0 * rm0
	rm1Sq := rm1 * rm1
	den := rm0 * (rm0Sq - rm1Sq)

	for l := 1; l <= L; l++ {
		if 8*l <= L {
			// Low-band harmonic: leave M[l] alone (W_l = 1).
			continue
		}
		w := 1.0
		if den > 0 {
			c := math.Cos(p.W0 * float64(l))
			num := rm0Sq + rm1Sq - 2*rm0*rm1*c
			xi := 0.96 * num / den
			if xi > 0 {
				w = math.Pow(xi, 0.25)
				if w < EnhanceWMin {
					w = EnhanceWMin
				} else if w > EnhanceWMax {
					w = EnhanceWMax
				}
			}
		}
		M[l] *= w
	}

	// Energy preservation: rescale so Σ M_enhanced² = R_M0_orig.
	enhSq := FrameEnergy(M, L)
	if enhSq <= 0 {
		return
	}
	scale := math.Sqrt(rm0 / enhSq)
	for l := 1; l <= L; l++ {
		M[l] *= scale
	}
}
