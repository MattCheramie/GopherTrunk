package metrics

import "math"

// SNRFromEVM converts an error-vector-magnitude percentage to an SNR in dB
// under the standard assumption that the residual error vector is the noise:
//
//	SNR ≈ 1 / EVM²  ⇒  SNR(dB) = −20·log10(EVM_fraction) = −20·log10(EVM%/100).
//
// It is the quick cross-check on the residual-variance estimators below — they
// should agree to within a fraction of a dB on a clean AWGN stream. Returns
// +Inf for a zero EVM (a noise-free constellation).
func SNRFromEVM(evmPct float64) float64 {
	if evmPct <= 0 {
		return math.Inf(1)
	}
	return -20 * math.Log10(evmPct/100)
}

// SNRResidualC4FM estimates the symbol SNR of a C4FM soft stream in dB from
// the residual of each sample about its nearest ideal rail. Signal power is
// the mean square of the rails the samples were assigned to; noise power is
// the mean square residual:
//
//	SNR(dB) = 10·log10( meanSquare(assignedRail) / meanSquare(x − assignedRail) ).
//
// outer is the outer-rail reference (see EstimateOuterRailC4FM when unknown).
// This is modulation-aware (it knows the 4-level structure), so unlike a
// blind moment estimator it stays unbiased on the non-constant-modulus C4FM
// eye. Returns +Inf when the residual is zero and 0 for an empty input.
func SNRResidualC4FM(soft []float32, outer float64) float64 {
	if len(soft) == 0 || outer <= 0 {
		return 0
	}
	inner := outer * c4fmInnerRatio
	rails := [4]float64{-outer, -inner, inner, outer}
	var sigSq, noiseSq float64
	for _, s := range soft {
		x := float64(s)
		bestRail := rails[0]
		best := math.Inf(1)
		for _, r := range rails {
			d := x - r
			if d*d < best {
				best = d * d
				bestRail = r
			}
		}
		sigSq += bestRail * bestRail
		noiseSq += best
	}
	if noiseSq <= 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(sigSq/noiseSq)
}

// SNRResidualConstellation estimates the symbol SNR of a complex constellation
// in dB from the residual about the nearest ideal QPSK point. Signal power is
// the reference radius squared (the RMS modulus); noise power is the mean
// square distance to the assigned ideal point, in the same normalized units.
// Returns +Inf for a zero residual and 0 for an empty input.
func SNRResidualConstellation(points []complex64) float64 {
	evm := EVMConstellation(points)
	if evm <= 0 {
		if len(points) == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return SNRFromEVM(evm)
}

// SNRM2M4Constellation estimates the SNR of a constant-modulus complex
// constellation (QPSK / π/4-DQPSK) in dB using the modulation-agnostic
// second/fourth-moment (M2M4) estimator. For a constant-modulus signal in
// AWGN with M2 = E[|r|²] and M4 = E[|r|⁴]:
//
//	S = √(2·M2² − M4),   N = M2 − S,   SNR = S/N.
//
// It needs no decisions, so it cross-checks the decision-directed residual
// estimator: the two should agree on a QPSK stream. It is *not* valid on the
// non-constant-modulus C4FM soft axis (the kurtosis assumption breaks), so
// there is deliberately no C4FM M2M4 variant. Returns 0 for an empty input or
// when the moments fall outside the estimator's valid region.
func SNRM2M4Constellation(points []complex64) float64 {
	n := len(points)
	if n == 0 {
		return 0
	}
	var m2, m4 float64
	for _, p := range points {
		pwr := float64(real(p))*float64(real(p)) + float64(imag(p))*float64(imag(p))
		m2 += pwr
		m4 += pwr * pwr
	}
	m2 /= float64(n)
	m4 /= float64(n)
	disc := 2*m2*m2 - m4
	if disc <= 0 {
		return math.Inf(1) // essentially noise-free (M4 → M2²·... region)
	}
	s := math.Sqrt(disc)
	noise := m2 - s
	if noise <= 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(s/noise)
}
