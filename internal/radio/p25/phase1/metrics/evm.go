package metrics

import "math"

// C4FM ideal rail ratios. The 4-level eye sits at ±outer and ±outer/3 (the
// dibit centres +3,+1,−1,−3 scaled to the soft-sample units). innerRatio is
// the inner rail as a fraction of the outer.
const c4fmInnerRatio = 1.0 / 3.0

// EVMC4FM returns the RMS error-vector magnitude of a C4FM soft-symbol stream
// as a percentage, given the outer-rail reference level (the soft value an
// ideal +3 symbol lands on — AGCTarget()·3/2, or the known modulator level).
//
// Each soft sample is assigned to the nearest of the four ideal rails
// {−outer, −outer/3, +outer/3, +outer}; EVM is the RMS distance to that rail,
// normalized to the outer rail and expressed in percent:
//
//	EVM% = 100 · √(mean((x − nearestRail)²)) / outer.
//
// A clean eye sits near a few percent; as the eye closes toward the slicer
// thresholds EVM climbs past ~33% (the half-spacing between rails). Returns 0
// for an empty input or a non-positive reference.
func EVMC4FM(soft []float32, outer float64) float64 {
	if len(soft) == 0 || outer <= 0 {
		return 0
	}
	inner := outer * c4fmInnerRatio
	rails := [4]float64{-outer, -inner, inner, outer}
	var sumSq float64
	for _, s := range soft {
		x := float64(s)
		best := math.Inf(1)
		for _, r := range rails {
			d := x - r
			if d*d < best {
				best = d * d
			}
		}
		sumSq += best
	}
	return 100 * math.Sqrt(sumSq/float64(len(soft))) / outer
}

// EstimateOuterRailC4FM recovers the outer-rail reference from a C4FM soft
// stream when the caller does not know it (e.g. the synthetic sweep, where the
// soft level depends on the modulator and AGC). It runs a few fixed-point
// iterations of "assign each sample to the nearest of {±a/3, ±a}, then set a to
// the mean magnitude of the outer-assigned samples," seeded from the mean
// magnitude. Returns 0 for an empty input.
func EstimateOuterRailC4FM(soft []float32) float64 {
	if len(soft) == 0 {
		return 0
	}
	// Seed: mean|x| on a balanced 4-level stream is (outer + inner)·... ≈
	// (1 + 1/3)/2·outer = 2/3·outer, so outer ≈ 1.5·mean|x|.
	var meanAbs float64
	for _, s := range soft {
		meanAbs += math.Abs(float64(s))
	}
	meanAbs /= float64(len(soft))
	a := 1.5 * meanAbs
	if a <= 0 {
		return 0
	}
	for iter := 0; iter < 8; iter++ {
		var outerSum float64
		var outerN int
		mid := a * (1 + c4fmInnerRatio) / 2 // boundary between inner and outer rail
		for _, s := range soft {
			ax := math.Abs(float64(s))
			if ax >= mid {
				outerSum += ax
				outerN++
			}
		}
		if outerN == 0 {
			break
		}
		a = outerSum / float64(outerN)
	}
	return a
}

// dqpskPhases is the number of distinct phase positions a π/4-DQPSK symbol
// stream visits: the modulation alternates between two QPSK constellations
// offset by 45°, so its pre-differential-decode points land on one of eight
// evenly-spaced phases on the unit circle (0, 45, …, 315°). Assigning EVM
// against only the four QPSK positions would charge a clean π/4-DQPSK signal
// ~40% EVM for the symbols sitting on the *other* ring.
const dqpskPhases = 8

// EVMConstellation returns the RMS error-vector magnitude of a π/4-DQPSK
// complex symbol constellation (the CQPSK path's post-carrier-recovery points,
// pre differential decode) as a percentage. Each point is normalized by the
// stream's RMS modulus to unit reference, assigned to the nearest of the eight
// ideal π/4-DQPSK phases on the unit circle, and EVM is the RMS distance to
// that position in percent:
//
//	EVM% = 100 · √(mean(|r̂ − ideal|²)),
//
// where r̂ is the point scaled so the reference radius is 1. Returns 0 for an
// empty input. The reference is the RMS radius rather than a fixed scale, so
// the estimate is independent of the AGC gain the points arrive at. (A plain
// 4-point QPSK stream is a subset of the eight phases, so this stays correct
// for it too.)
func EVMConstellation(points []complex64) float64 {
	if len(points) == 0 {
		return 0
	}
	var sumSq float64
	// 8th-power phase estimator: Σ exp(j·8·θ) concentrates the eight-phase
	// modulation onto one tone, so its argument /8 is the common carrier-phase
	// offset the recovery loop happened to lock at. Removing it makes the EVM
	// invariant to that offset — without it a loop locked a few degrees off the
	// grid would be charged sin(offset) of bogus dispersion (the clean signal
	// reading tens of percent EVM that *fell* when noise was added).
	var pwrReal, pwrImag float64
	for _, p := range points {
		i, q := float64(real(p)), float64(imag(p))
		sumSq += i*i + q*q
		ang := dqpskPhases * math.Atan2(q, i)
		pwrReal += math.Cos(ang)
		pwrImag += math.Sin(ang)
	}
	rms := math.Sqrt(sumSq / float64(len(points)))
	if rms <= 0 {
		return 0
	}
	offset := math.Atan2(pwrImag, pwrReal) / dqpskPhases
	cosO, sinO := math.Cos(-offset), math.Sin(-offset)

	// Ideal π/4-DQPSK phase positions on the unit circle.
	var ideal [dqpskPhases][2]float64
	for k := range ideal {
		ang := float64(k) * 2 * math.Pi / dqpskPhases
		ideal[k] = [2]float64{math.Cos(ang), math.Sin(ang)}
	}
	var errSq float64
	for _, p := range points {
		// Normalize to unit reference and derotate by the recovered offset.
		ri := float64(real(p)) / rms
		rq := float64(imag(p)) / rms
		i := ri*cosO - rq*sinO
		q := ri*sinO + rq*cosO
		best := math.Inf(1)
		for _, id := range ideal {
			di := i - id[0]
			dq := q - id[1]
			d := di*di + dq*dq
			if d < best {
				best = d
			}
		}
		errSq += best
	}
	return 100 * math.Sqrt(errSq/float64(len(points)))
}
