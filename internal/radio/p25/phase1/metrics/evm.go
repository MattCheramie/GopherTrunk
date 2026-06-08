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

// EVMConstellation returns the RMS error-vector magnitude of a complex symbol
// constellation (the CQPSK / π/4-DQPSK path's post-carrier-recovery points) as
// a percentage. Each point is normalized by the stream's RMS modulus to unit
// reference, assigned to the nearest ideal QPSK position (±45°, ±135° on the
// unit circle), and EVM is the RMS distance to that position in percent:
//
//	EVM% = 100 · √(mean(|r̂ − ideal|²)),
//
// where r̂ is the point scaled so the reference radius is 1. Returns 0 for an
// empty input. The reference is the RMS radius rather than a fixed scale, so
// the estimate is independent of the AGC gain the points arrive at.
func EVMConstellation(points []complex64) float64 {
	if len(points) == 0 {
		return 0
	}
	var sumSq float64
	for _, p := range points {
		sumSq += float64(real(p))*float64(real(p)) + float64(imag(p))*float64(imag(p))
	}
	rms := math.Sqrt(sumSq / float64(len(points)))
	if rms <= 0 {
		return 0
	}
	// Ideal π/4-DQPSK / QPSK positions on the unit circle.
	const s = math.Sqrt2 / 2 // sin(45°)=cos(45°)
	ideal := [4][2]float64{{s, s}, {-s, s}, {-s, -s}, {s, -s}}
	var errSq float64
	for _, p := range points {
		i := float64(real(p)) / rms
		q := float64(imag(p)) / rms
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
