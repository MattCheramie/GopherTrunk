package demod

import "math"

// AdaptiveC4FMSlicer is a four-level C4FM slicer that tracks the
// *observed* per-symbol levels and places its decision thresholds at the
// midpoints between adjacent levels, so it follows an asymmetric or
// off-nominal eye that the fixed-threshold C4FM.Slice mis-decides
// (issue #402).
//
// The fixed slicer assumes a symmetric eye whose four levels sit at the
// nominal ±slicerScale (outer) and ±slicerScale/3 (inner), with
// thresholds frozen at 0 and ±2·slicerScale/3. On a real transmitter
// whose deviation is asymmetric — or whose TX pulse shaping doesn't quite
// match the receive matched filter — the demodulated eye is skewed (on
// the MMR Site 9 capture in #402 the +3 rail landed ~60 % high while the
// −3 rail was nominal), and the fixed thresholds slice it wrong: outer
// symbols leak into inner and the frame-sync word (mostly outer symbols)
// collapses. p25_survey / OP25 / SDRTrunk all handle this with adaptive
// level tracking; this is GopherTrunk's equivalent for the P25 Phase 1
// C4FM path.
//
// The whole upstream chain (atan2 FM discriminator, linear matched
// filter, scalar mean|x| AGC) is provably linear and symmetric, so it
// cannot create the skew — the skew is physical, in the received signal,
// and a data-driven slicer is the right place to absorb it.
//
// Safety. A data-driven slicer is a feedback loop, and #402's earlier
// decision-directed AFC showed how such a loop can false-lock. Three
// mechanisms keep this one in check, none of which need an external lock
// signal (the coupling that made the DDA fragile):
//
//   - A warmup window: it decides at the fixed nominal thresholds until
//     the upstream symbol-AGC has settled, so it never adapts to a
//     mis-scaled transient.
//   - A leak toward nominal on every update, so a level only departs
//     nominal as far as the data *persistently* supports it; an open,
//     well-separated eye is tracked, an ambiguous one stays near fixed.
//   - Per-level clamps and a strict-ordering invariant, so the thresholds
//     can't run away on noise / loss-of-signal.
//   - The outer decision thresholds are *inward-only capped*: they may move
//     toward zero freely (to follow a compressed or DC-shifted eye — the case
//     the fixed slicer genuinely can't handle) but their outward travel is
//     capped at a nominal inner→outer half-spacing (s/3) above the tracked
//     inner rail, i.e. at the fixed nominal threshold. On the #402 capture
//     every adaptive threshold that rose above the fixed nominal decoded
//     worse, so the cap keeps the slicer no worse than fixed there.
//   - The boundaries are *variance-aware*: each sits at the equal-σ-distance
//     point between adjacent rails (not the plain midpoint), so when an outer
//     rail's population is *spread* — the #402 +3 rail spreads low — the
//     boundary moves toward the tighter inner rail, recovering more of the
//     spread rail's symbols than a midpoint would. Inner-rail σ is clamped
//     tight so an overlapping outer tail can't inflate it and cancel this.
//   - The per-symbol level update is a soft-responsibility (Gaussian
//     mixture) step rather than a hard decided-rail EMA, so a true outer
//     symbol that landed below the threshold still contributes to the outer
//     level — removing the one-sided truncation bias that made the raw
//     outer estimate read high (~0.43 vs the true ~0.37 centroid on #402).
//
// On an open eye — symmetric or asymmetric, which is what a decodable
// signal presents (the #402 capture is open at the production sps) — the
// slicer tracks the eye and decodes at least as well as the fixed slicer,
// and on a clean symmetric eye reproduces its decisions exactly. On a
// pathologically *closed* eye (rails overlapping) decision-directed
// tracking is inherently unreliable for any slicer, fixed or adaptive; the
// three mechanisms above bound the damage rather than eliminate it, and
// such a signal does not decode under the fixed thresholds either.
type AdaptiveC4FMSlicer struct {
	// level[i] is the running EMA of the *signed* soft value for symbols
	// decided into rail i, indexed -3,-1,+1,+3 → 0,1,2,3. Signed (not
	// magnitude) so positive- and negative-rail asymmetry is tracked
	// independently.
	level   [4]float32
	nominal [4]float32 // seed + fallback: {-s, -s/3, +s/3, +s}
	loBound [4]float32 // per-level clamp (signed, ordered with nominal)
	hiBound [4]float32
	rate    float32 // single-pole EMA coefficient
	sigma   float32 // Gaussian-responsibility width for the soft level update
	// varia[i] is the running responsibility-weighted EMA of the squared
	// residual (soft − level[i])² for rail i — the per-rail spread. It lets
	// thresholds() place each boundary at the equal-σ-distance point between
	// adjacent rails (toward the tighter rail) rather than the plain
	// midpoint, so a rail whose population is spread (the #402 +3 outer rail,
	// spread low) yields a lower decision boundary that recovers more of its
	// symbols. Seeded equal across rails so the boundary starts at the
	// midpoint (== fixed nominal under the inward cap).
	varia  [4]float32
	sigMin float32    // σ floor (keeps the equal-σ-distance weighting stable)
	sigMax [4]float32 // per-rail σ ceiling: outer rails may be spread, inner rails are held tight
	warmup int        // symbols to slice at nominal thresholds before adapting
}

// adaptiveSlicerRate is the EMA time constant for level tracking, in
// symbols. ~1/512 (≈107 ms at 4800 baud) is slow enough to ride out the
// symbol-distribution swings a control channel produces (an all-outer FSW
// preamble, an inner-leaning idle run) without chasing them, but fast
// enough to converge onto a site's eye well within the first second of a
// capture.
const adaptiveSlicerRate = 1.0 / 512.0

// adaptiveSlicerLeak is a regularization pull back toward the nominal eye,
// applied alongside the data-directed update. It makes "degrade gracefully
// to the fixed slicer" a real property rather than just a clamp: a level
// only departs nominal as far as the data persistently supports it. At
// equilibrium a level settles at rate/(rate+leak) of the way from nominal
// to the observed centroid — with leak = rate/4 that's 0.8, so a
// strongly-supported asymmetry (the #402 +3 rail at ~1.6× nominal) still
// tracks most of the way, while an ambiguous or partly-closed eye — where
// decision-directed tracking would otherwise run away and do *worse* than
// fixed — is held near nominal. Both the data pull and this leak are scaled
// by the per-symbol responsibility in update(), so each rail is regularized
// in proportion to the evidence it receives and the 0.8 mix is independent
// of the symbol prior.
const adaptiveSlicerLeak = adaptiveSlicerRate / 4

// adaptiveSlicerSigmaFrac sets the Gaussian-responsibility width used by the
// soft level update, as a fraction of the nominal outer level (slicerScale).
// At 1/4 the width (≈0.059 for the nominal eye) is a fraction of the nominal
// inner→outer gap (2·slicerScale/3 ≈ 0.157), so the four rails stay
// resolvable while an outer symbol that fell below the decision threshold —
// the lower tail that a hard decided-rail EMA never sees — still registers on
// the outer level. That is what removes the one-sided truncation bias (#402).
const adaptiveSlicerSigmaFrac = 1.0 / 4.0

// adaptiveSlicerSigmaMinFrac / adaptiveSlicerSigmaMax{Inner,Outer}Frac clamp
// the tracked per-rail σ (sqrt of varia[i]) used by the variance-aware
// boundary, as fractions of slicerScale. The floor keeps a near-zero-variance
// rail from making the equal-σ-distance weighting degenerate. The ceilings
// are per-rail-type: an OUTER rail may be genuinely spread (the #402 +3
// population is spread low), so it gets a generous ceiling; an INNER rail is
// held tight, because on an overlapping eye a spread outer rail's tail leaks
// partial responsibility onto the adjacent inner rail and would otherwise
// inflate the inner σ to match — cancelling the very boundary shift we want.
// Keeping the inner σ tight lets a spread outer rail actually pull the
// boundary toward the (tight) inner rail.
const (
	adaptiveSlicerSigmaMinFrac      = 0.05
	adaptiveSlicerSigmaMaxOuterFrac = 0.75
	adaptiveSlicerSigmaMaxInnerFrac = 0.15
)

// adaptiveSlicerWarmup is the number of symbols the slicer decides at its
// nominal (fixed-threshold) eye before it starts adapting. Upstream of the
// slicer, the receiver's symbol-AGC is a 1/256 EMA that takes ~1300 symbols
// to settle; until it has, the soft levels are mis-scaled, and adapting to
// that transient drives a tracked level into its clamp where the slow EMA
// leaves it stuck (it slices like the fixed slicer during warmup, so this
// is purely a safe deferral, not a behaviour change). 2048 symbols clears
// the AGC settle with margin and is a tiny fraction of any real capture.
const adaptiveSlicerWarmup = 2048

// NewAdaptiveC4FMSlicer builds a slicer seeded to the nominal symmetric
// eye for the given slicerScale (the post-AGC outer-symbol level, i.e.
// 2π·deviation/sampleRate on the C4FM path — the same value C4FM.Slice
// uses as its deviation). The tracked levels are clamped to a band around
// these nominals so the slicer degrades gracefully to ~fixed-threshold
// behaviour rather than running away.
func NewAdaptiveC4FMSlicer(slicerScale float64) *AdaptiveC4FMSlicer {
	s := float32(slicerScale)
	nominal := [4]float32{-s, -s / 3, s / 3, s}
	a := &AdaptiveC4FMSlicer{
		level:   nominal,
		nominal: nominal,
		rate:    adaptiveSlicerRate,
		sigma:   s * adaptiveSlicerSigmaFrac,
		sigMin:  s * adaptiveSlicerSigmaMinFrac,
		sigMax: [4]float32{
			s * adaptiveSlicerSigmaMaxOuterFrac, // -3 outer
			s * adaptiveSlicerSigmaMaxInnerFrac, // -1 inner
			s * adaptiveSlicerSigmaMaxInnerFrac, // +1 inner
			s * adaptiveSlicerSigmaMaxOuterFrac, // +3 outer
		},
		warmup: adaptiveSlicerWarmup,
	}
	// Seed every rail's tracked spread equal, so before any data the
	// equal-σ-distance boundaries reduce to the plain midpoints (== the fixed
	// nominal thresholds under the inward cap).
	seedVar := a.sigma * a.sigma
	a.varia = [4]float32{seedVar, seedVar, seedVar, seedVar}
	// Per-rail clamp bands. Outer rails may stretch generously (the #402
	// site ran +3 at ~1.6× nominal); inner rails are held tighter since a
	// collapsed inner level is the dangerous case (it would pull the zero
	// threshold or the outer threshold onto the wrong side). The bands are
	// chosen so the four clamped ranges cannot overlap into a different
	// rail's nominal, keeping the ordering invariant satisfiable.
	for i := 0; i < 4; i++ {
		mag := a.nominal[i]
		if mag < 0 {
			mag = -mag
		}
		var lo, hi float32
		if i == 0 || i == 3 { // outer
			lo, hi = 0.4*mag, 2.5*mag
		} else { // inner
			lo, hi = 0.2*mag, 1.3*mag
		}
		if a.nominal[i] < 0 {
			a.loBound[i], a.hiBound[i] = -hi, -lo
		} else {
			a.loBound[i], a.hiBound[i] = lo, hi
		}
	}
	return a
}

// boundary returns the decision threshold between two adjacent rails (lo
// below hi) at the equal-σ-distance point: the value x where
// (x−μlo)/σlo = (μhi−x)/σhi, i.e. x = (μlo·σhi + μhi·σlo)/(σlo+σhi). This is
// the plain midpoint when the two spreads are equal, but biases toward the
// *tighter* rail when one is more spread — so a rail whose population is
// spread (the #402 +3 outer rail) yields a boundary pulled toward the tight
// inner rail, recovering more of the spread rail's symbols than a midpoint.
func (a *AdaptiveC4FMSlicer) boundary(lo, hi int) float32 {
	slo := a.clampSigma(lo)
	shi := a.clampSigma(hi)
	return (a.level[lo]*shi + a.level[hi]*slo) / (slo + shi)
}

// clampSigma returns rail i's tracked σ (sqrt of varia[i]) clamped to
// [sigMin, sigMax[i]] (the per-rail-type ceiling).
func (a *AdaptiveC4FMSlicer) clampSigma(i int) float32 {
	s := float32(math.Sqrt(float64(a.varia[i])))
	if s < a.sigMin {
		return a.sigMin
	}
	if s > a.sigMax[i] {
		return a.sigMax[i]
	}
	return s
}

// thresholds returns the three live decision boundaries: the zero (−1/+1)
// crossing and the two outer (−3/−1 and +1/+3) boundaries.
//
// Each is the variance-aware equal-σ-distance point (see boundary), so a
// spread rail pulls its boundary toward the tighter neighbour. The two outer
// boundaries are additionally capped *inward-only*: each may move toward zero
// freely (to follow a compressed / DC-shifted eye, which the fixed slicer
// can't) but its outward travel is capped at one nominal inner→outer spacing
// (s/3) beyond the tracked inner rail — i.e. it can never sit farther out
// than the fixed nominal threshold. On the #402 stretched/closed eye every
// adaptive threshold that rose above the fixed nominal decoded worse, so the
// cap keeps the slicer no worse than fixed there while the variance-aware
// term lets it do *better* on a genuinely spread eye by moving the boundary
// inward. The cap references the *tracked* inner level, so it bounds the
// inner→outer spacing, not an absolute threshold (it follows a shifted eye).
func (a *AdaptiveC4FMSlicer) thresholds() (tNegOuter, tZero, tPosOuter float32) {
	// half the nominal inner→outer spacing = (s − s/3)/2 = s/3. The fixed
	// +1/+3 threshold is nominal_inner + s/3 (= 2s/3), so capping the outer
	// boundary at tracked_inner + s/3 pins it at the fixed nominal on a
	// nominal-inner eye and never lets it travel farther outward.
	half := (a.nominal[3] - a.nominal[2]) / 2

	tPosOuter = a.boundary(2, 3)
	if capPos := a.level[2] + half; tPosOuter > capPos {
		tPosOuter = capPos
	}
	tNegOuter = a.boundary(0, 1)
	if capNeg := a.level[1] - half; tNegOuter < capNeg {
		tNegOuter = capNeg
	}
	tZero = a.boundary(1, 2)
	return tNegOuter, tZero, tPosOuter
}

// Thresholds returns the three live decision boundaries (negative-outer,
// zero, positive-outer). Diagnostic: lets replay print the actual thresholds
// the slicer is deciding on, not just the tracked levels (issue #402).
func (a *AdaptiveC4FMSlicer) Thresholds() [3]float32 {
	tNeg, tZero, tPos := a.thresholds()
	return [3]float32{tNeg, tZero, tPos}
}

// slice maps one soft sample to a symbol in {-3,-1,+1,+3} using the live
// thresholds.
func (a *AdaptiveC4FMSlicer) slice(soft float32) int8 {
	tNegOuter, tZero, tPosOuter := a.thresholds()
	switch {
	case soft >= tPosOuter:
		return 3
	case soft >= tZero:
		return 1
	case soft >= tNegOuter:
		return -1
	default:
		return -3
	}
}

// update folds one soft sample into the tracked levels, then re-applies the
// clamp + ordering invariant.
//
// Rather than a hard decided-rail EMA (which only ever updates the single
// rail the sample was sliced into, so an outer level averages only the
// samples *above* its threshold — a lower-truncated, biased-high estimate),
// this is a soft-responsibility step: a four-component equal-variance,
// equal-prior Gaussian mixture. Every level is pulled toward the sample
// weighted by that rail's responsibility, so an outer symbol whose soft
// value fell below the decision threshold still contributes most of its
// weight to the outer level. That removes the one-sided truncation bias and
// lets the outer estimate converge to the true cluster centroid (#402).
//
// The update is a pure function of (soft, level) with no cross-call state, so
// SliceMany stays deterministic across chunk boundaries.
func (a *AdaptiveC4FMSlicer) update(soft float32, sliced int8) {
	_ = sliced // the soft responsibilities, not the hard decision, drive the update

	inv2s2 := 1.0 / (2 * a.sigma * a.sigma)
	var w, d [4]float32
	var sum float32
	for k := 0; k < 4; k++ {
		d[k] = soft - a.level[k]
		e := float32(math.Exp(float64(-d[k] * d[k] * inv2s2)))
		w[k] = e
		sum += e
	}
	if sum <= 0 { // numerical floor: sample absurdly far from every rail
		return
	}
	for k := int8(0); k < 4; k++ {
		rk := w[k] / sum
		// Responsibility-weighted pull toward the sample AND a
		// responsibility-weighted leak back toward nominal. Both terms scale
		// by rk so each rail is updated at the same cadence the hard
		// decided-rail EMA used (its symbol prior), preserving the intended
		// equilibrium mix level = rate/(rate+leak) ≈ 0.8 of the way from
		// nominal to the observed centroid. (Leaking on every sample while
		// pulling only by rk halved that mix and made the outer rail
		// under-track — issue #402 follow-up to #439.)
		a.level[k] += rk * (a.rate*(soft-a.level[k]) + adaptiveSlicerLeak*(a.nominal[k]-a.level[k]))
		// Track the rail's spread the same way (responsibility-weighted EMA of
		// the squared residual against the current level), so thresholds() can
		// place each boundary at the equal-σ-distance point.
		dk := d[k]
		a.varia[k] += rk * a.rate * (dk*dk - a.varia[k])
		a.clampLevel(k)
	}
	a.enforceOrder()
}

// clampLevel holds a single tracked level inside its nominal band.
func (a *AdaptiveC4FMSlicer) clampLevel(i int8) {
	if a.level[i] < a.loBound[i] {
		a.level[i] = a.loBound[i]
	} else if a.level[i] > a.hiBound[i] {
		a.level[i] = a.hiBound[i]
	}
}

// enforceOrder restores the invariant level[0] < level[1] < 0 < level[2]
// < level[3]. A degenerate stream (e.g. all symbols decided into one
// rail during loss-of-signal) can push levels out of order; when that
// happens the offending level is snapped back to its nominal, which —
// together with the per-level clamp — keeps the thresholds within a
// bounded band of the fixed ones.
func (a *AdaptiveC4FMSlicer) enforceOrder() {
	// Outer rails must sit beyond their inner neighbour; inner rails must
	// keep the correct sign. Reset any violator to nominal.
	if !(a.level[1] < 0) || a.level[1] <= a.level[0] {
		a.level[1] = a.nominal[1]
	}
	if !(a.level[2] > 0) || a.level[2] >= a.level[3] {
		a.level[2] = a.nominal[2]
	}
	if a.level[0] >= a.level[1] {
		a.level[0] = a.nominal[0]
	}
	if a.level[3] <= a.level[2] {
		a.level[3] = a.nominal[3]
	}
}

// SliceMany slices a batch of soft samples, adapting after each decision.
// Mirrors C4FM.SliceMany so the receiver can swap it in for slicing while
// keeping the C4FM matched filter. The adaptation is per-sample and
// deterministic given the input, so the output is identical regardless of
// how the stream is chunked across calls.
func (a *AdaptiveC4FMSlicer) SliceMany(dst []int8, src []float32) []int8 {
	if cap(dst) < len(src) {
		dst = make([]int8, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		sym := a.slice(s)
		dst[i] = sym
		if a.warmup > 0 {
			a.warmup-- // hold at the nominal eye until the AGC has settled
			continue
		}
		a.update(s, sym)
	}
	return dst
}

// Levels returns the four tracked levels (−3,−1,+1,+3 order). Diagnostic:
// lets the replay state log show the slicer converging onto a site's eye.
func (a *AdaptiveC4FMSlicer) Levels() [4]float32 { return a.level }

// Reset restores the tracked levels to the nominal symmetric eye. Call on
// stream re-sync so a stale eye estimate doesn't bleed across the
// discontinuity.
func (a *AdaptiveC4FMSlicer) Reset() {
	a.level = a.nominal
	seedVar := a.sigma * a.sigma
	a.varia = [4]float32{seedVar, seedVar, seedVar, seedVar}
	a.warmup = adaptiveSlicerWarmup
}
