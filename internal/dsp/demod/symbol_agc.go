package demod

// C4FMSymbolAGC tracks a running estimate of mean|x| over the per-symbol
// matched-filter output and scales each symbol so the estimate matches a
// target reference. The 4-level slicer's fixed thresholds are designed against
// a specific signal level; the AGC reconciles that with whatever level the
// upstream RRC matched filter actually produces.
//
// The loop is feed-forward — each output is the input scaled by the current
// estimate — so an occasional near-zero sample cannot drive the gain unbounded
// the way a feedback loop would. It seeds the EMA from the first non-trivial
// sample's |x| (not the first batch's mean) so the recovered stream is
// byte-identical regardless of how the IQ is chunked (issue #275).
//
// This is the shared implementation used by every 4800-baud C4FM-family
// receiver (P25 Phase 1, DMR, NXDN); it was originally three byte-identical
// copies pinned only by comment. Process returns the mean level the batch was
// scaled against, which the P25 Phase 1 DDA uses to un-normalise its residuals
// (issue #402); receivers that don't need it simply ignore the return value.
type C4FMSymbolAGC struct {
	Target float32 // desired mean|x| in the output stream; <=0 disables
	Rate   float32 // single-pole EMA coefficient (rate-of-tracking)
	level  float32 // current mean|x| estimate of the input
	seeded bool
}

// Process scales symbols in place so the running mean|x| matches Target, and
// returns the mean level the batch was scaled against (the mean of the
// per-sample EMA estimate), or 0 when calibration is disabled (Target<=0) or no
// sample seeded the loop. A no-op scaling when Target<=0 (the legacy
// pre-scaled-fixture path with DeviationHz=0).
func (a *C4FMSymbolAGC) Process(symbols []float32) float64 {
	if a.Target <= 0 {
		return 0 // calibration disabled (legacy DeviationHz=0 path)
	}
	var levelSum float64
	var levelN int
	for i, x := range symbols {
		ax := x
		if ax < 0 {
			ax = -ax
		}
		if !a.seeded {
			if ax > 1e-12 {
				a.level = ax
				a.seeded = true
			} else {
				continue
			}
		} else {
			a.level += a.Rate * (ax - a.level)
		}
		if a.level > 1e-12 {
			g := a.Target / a.level
			symbols[i] = x * g
			levelSum += float64(a.level)
			levelN++
		}
	}
	if levelN == 0 {
		return 0
	}
	return levelSum / float64(levelN)
}

// Reset clears the level estimate so a stream re-sync starts from a fresh seed.
func (a *C4FMSymbolAGC) Reset() {
	a.level = 0
	a.seeded = false
}

// Level reports the current mean|x| estimate of the input. The P25 Phase 1 DDA
// reads this to un-normalise its residuals (issue #402).
func (a *C4FMSymbolAGC) Level() float32 { return a.level }

// Seeded reports whether the EMA has been seeded by a non-trivial sample yet.
func (a *C4FMSymbolAGC) Seeded() bool { return a.seeded }

// C4FMAGCTarget is the mean|x| target a C4FM symbol AGC normalises to: two
// thirds of the outer-symbol slicer scale (a balanced ±1/±3 four-level stream
// sits at slicerScale·2/3), or 0 to disable calibration on the legacy
// pre-scaled-fixture path (deviationHz <= 0). Shared by the DMR and NXDN C4FM
// receivers, which previously each carried an identical agcTargetFor.
func C4FMAGCTarget(slicerScale, deviationHz float64) float64 {
	if deviationHz <= 0 {
		return 0
	}
	return slicerScale * 2.0 / 3.0
}
