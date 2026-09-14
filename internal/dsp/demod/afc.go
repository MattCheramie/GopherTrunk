package demod

import "math"

// CoarseAFC removes the residual carrier-frequency offset from an
// FM-discriminator output stream.
//
// A constant carrier offset Δf leaves the FM discriminator as a
// constant DC bias of 2π·Δf/Fs radians/sample. The C4FM 4-level slicer
// thresholds are fixed, so an un-removed bias shifts every symbol off
// its decision region and the recovered dibit stream is systematically
// wrong — on a real RTL-SDR the P25 control channel never locks (issue
// #275: a ≥500 Hz tuner offset stopped the Frame Sync Word correlating
// at all).
//
// CoarseAFC tracks the bias with a slow single-pole IIR average and
// subtracts it, recentring the 4-level eye on the slicer thresholds.
// The averaging pole sits far below the symbol rate, so the estimate
// follows the carrier offset rather than the data modulation — a
// balanced 4-level stream has zero mean. On a signal with no offset the
// estimate converges to ~0 and the stage is a near-noop.
//
// It is a coarse correction: it removes the static frequency error a
// real tuner carries, not per-symbol phase jitter — enough to put the
// eye back inside the slicer's decision margins. On a sustained
// unbalanced symbol distribution (an FSW preamble, an idle TDU run)
// the open-loop integrator drifts onto the data mean — issue #402
// observed afc_hz_est swinging 2 kHz / 20 kHz / 2 kHz on a locked CC.
// Pair with a DecisionDirectedAFC after the slicer to take over once
// decisions are trustworthy and freeze CoarseAFC at its handoff value.
type CoarseAFC struct {
	beta float64 // single-pole smoothing coefficient
	dc   float64 // current bias estimate
}

// coarseAFCSymbols is the CoarseAFC averaging time constant in symbol
// periods. ~64 symbols is slow enough to ignore the 4800-baud data
// modulation yet converges well inside a typical control-channel
// warmup preamble.
const coarseAFCSymbols = 64

// NewCoarseAFC builds a coarse-AFC DC tracker for an FM-discriminator
// output sampled at sps samples per symbol. Panics if sps is not
// positive.
func NewCoarseAFC(sps float64) *CoarseAFC {
	if sps <= 0 {
		panic("demod: CoarseAFC sps must be positive")
	}
	return &CoarseAFC{beta: 1.0 / (coarseAFCSymbols * sps)}
}

// Process updates the bias estimate from buf and subtracts it in place.
// buf is a real FM-discriminator (or matched-filter) output; the
// estimate carries across calls so a chunked stream converges once.
func (a *CoarseAFC) Process(buf []float32) {
	a.ProcessGated(buf, nil)
}

// ProcessGated is Process with a per-sample carrier-presence gate: a sample
// whose present flag is false has the current estimate subtracted but does
// not update it (a nil present is all-true, byte-identical to Process). On a
// burst-mode carrier the inter-burst noise is zero-mean, so an ungated
// tracker decays toward zero in every gap and re-converges from scratch on
// every burst — holding it carries the settled offset across (issue #836).
func (a *CoarseAFC) ProcessGated(buf []float32, present []bool) {
	for i, x := range buf {
		if present == nil || present[i] {
			a.dc += a.beta * (float64(x) - a.dc)
		}
		buf[i] = float32(float64(x) - a.dc)
	}
}

// Subtract removes the current bias estimate from buf in place without
// updating it. Use this after a DecisionDirectedAFC handoff freezes
// the open-loop tracker at the value it carried at handoff time —
// CoarseAFC keeps subtracting that value, and the DDA carries any
// further drift on its own integrator.
func (a *CoarseAFC) Subtract(buf []float32) {
	for i, x := range buf {
		buf[i] = float32(float64(x) - a.dc)
	}
}

// Offset returns the current bias estimate in radians/sample; the
// residual carrier offset is Offset()·Fs/(2π) hertz.
func (a *CoarseAFC) Offset() float64 { return a.dc }

// SetOffset overrides the bias estimate. Used at DDA handoff to zero
// out CoarseAFC's contribution after folding it into the DDA's
// running estimate, so the two stages don't double-subtract.
func (a *CoarseAFC) SetOffset(v float64) { a.dc = v }

// Reset clears the estimate. Call on stream re-tune so a stale offset
// doesn't bleed across the discontinuity.
func (a *CoarseAFC) Reset() { a.dc = 0 }

// DecisionDirectedAFC refines a coarse carrier-offset estimate using
// post-slicer symbol decisions, decoupling the AFC from the data
// modulation that an open-loop tracker (CoarseAFC) confuses for
// carrier drift on sustained-unbalanced sequences — the issue #402
// failure mode.
//
// Each Update incorporates the per-symbol residual (soft − sliced
// nominal), which has zero mean for any symbol distribution as long
// as the decisions are correct. The result tracks only true carrier
// offset (and noise), not the symbol-stream mean.
//
// Inputs to Update are in the post-AGC normalised space (where the
// slicer's nominal symbol values are ±slicerScale outer and
// ±slicerScale/3 inner). The DDA un-normalises the residual via the
// caller-supplied AGC gain ratio (level/target) so its Offset() is
// reported in the same units as CoarseAFC.Offset() — radians/sample
// at the matched-filter input rate — and the receiver can subtract
// them additively from the matched-filter buffer.
//
// Updates whose normalised residual exceeds gateNorm (~one third of
// a slicer cell, derived from slicerScale at construction) are
// skipped: such a residual indicates the slicer probably crossed the
// wrong threshold, and integrating it would let bad decisions teach
// the loop bad offsets.
type DecisionDirectedAFC struct {
	beta              float64 // single-pole smoothing coefficient
	dc                float64 // current bias estimate (rad/sample at input)
	clampRadPerSample float64 // |dc| safety bound
	gateNorm          float64 // skip updates with |residual_norm| > gateNorm

	residMeanBeta float64 // smoothing coefficient for residMean
	residMean     float64 // EMA of accepted residuals (normalised, signed)
}

// ddaSymbols is the DDA averaging time constant in symbol periods.
// ~1024 symbols (~213 ms at 4800 baud) rides out occasional decision
// errors and short bursts of slicer overshoot without losing real
// thermal-drift tracking on an RTL-SDR.
const ddaSymbols = 1024

// ddaResidMeanSymbols is the time constant (in accepted symbols) of the
// running mean the receiver reads via AcceptedResidualMean to tell a
// genuinely open eye (mean ≈ 0) from a uniformly-biased false lock
// (sustained non-zero mean). Shorter than ddaSymbols so the estimate is
// trustworthy by the time the handoff fires (~256 accepted updates).
// Issue #402.
const ddaResidMeanSymbols = 128

// NewDecisionDirectedAFC builds a decision-directed tracker calibrated
// for a slicer whose outer-symbol nominal value (in the post-AGC
// space) is slicerScaleNorm. The gate threshold is slicerScaleNorm/3
// — the distance from any symbol centre to its nearest slicer
// threshold, so residuals that land outside that radius are by
// definition past the wrong-decision boundary.
//
// maxOffsetHz clamps the integrator as a safety net (an RTL-SDR at
// 420 MHz with 50 ppm tuner accuracy is bounded by ~21 kHz; pass a
// little above so the clamp catches anomalies, not normal operation).
// sampleRateHz is the matched-filter input rate (== the SDR's post-
// DDC rate, typically 48 kHz).
//
// Panics if any argument is non-positive.
func NewDecisionDirectedAFC(maxOffsetHz, sampleRateHz, slicerScaleNorm float64) *DecisionDirectedAFC {
	if maxOffsetHz <= 0 || sampleRateHz <= 0 || slicerScaleNorm <= 0 {
		panic("demod: DecisionDirectedAFC requires positive maxOffsetHz, sampleRateHz, slicerScaleNorm")
	}
	return &DecisionDirectedAFC{
		beta:              1.0 / float64(ddaSymbols),
		clampRadPerSample: 2.0 * math.Pi * maxOffsetHz / sampleRateHz,
		gateNorm:          slicerScaleNorm / 3.0,
		residMeanBeta:     1.0 / float64(ddaResidMeanSymbols),
	}
}

// Update incorporates one symbol's residual into the bias estimate.
// Returns true if the update was accepted (residual within gate),
// false if skipped. The receiver counts accepted updates to decide
// when to hand CoarseAFC off to the DDA.
//
//	softNorm     - AGC-normalised soft sample at the symbol-decision instant
//	expectedNorm - slicer's nominal value for the decision
//	               (±slicerScaleNorm outer, ±slicerScaleNorm/3 inner)
//	agcUnscale   - AGC's level/target gain ratio; converts the
//	               normalised residual back to rad/sample at the
//	               matched-filter input
//
// The loop is a pure integrator: dc += β · residualRaw. The receiver
// subtracts dc from the matched-filter buffer before slicing, so the
// residual fed back at the next sample is (true_offset − dc); the
// integrator's zero-error fixed point is dc = true_offset. (An EMA
// loop, dc += β·(r − dc), converges to dc = (true_offset)/2 in this
// feedback configuration — half the right value, because the EMA
// already assumes "the input is the value I'm trying to track" but
// our input is "how far off am I", a derivative.)
func (a *DecisionDirectedAFC) Update(softNorm, expectedNorm, agcUnscale float32) bool {
	residualNorm := float64(softNorm) - float64(expectedNorm)
	if math.Abs(residualNorm) > a.gateNorm {
		return false
	}
	residualRaw := residualNorm * float64(agcUnscale)
	a.dc += a.beta * residualRaw
	if a.dc > a.clampRadPerSample {
		a.dc = a.clampRadPerSample
	} else if a.dc < -a.clampRadPerSample {
		a.dc = -a.clampRadPerSample
	}
	// Track the running mean of accepted residuals (in normalised
	// units). On correct decisions this is ~0 for any symbol
	// distribution; a uniformly-biased eye that still slips inside the
	// gate leaves a sustained non-zero mean — the signal a count-only
	// handoff gate can't see. Issue #402.
	a.residMean += a.residMeanBeta * (residualNorm - a.residMean)
	return true
}

// AcceptedResidualMean returns the EMA of accepted residuals in the
// post-AGC normalised space (signed). It is ~0 when decisions are
// correct (the DDA's data-immunity premise) and trends toward the
// slicer bias when the eye is uniformly off-centre — the receiver uses
// it to refuse a handoff onto a biased false lock. Issue #402.
func (a *DecisionDirectedAFC) AcceptedResidualMean() float64 { return a.residMean }

// Apply subtracts the current bias estimate from buf in place. Called
// once per matched-filter buffer alongside CoarseAFC's correction.
func (a *DecisionDirectedAFC) Apply(buf []float32) {
	if a.dc == 0 {
		return
	}
	for i, x := range buf {
		buf[i] = float32(float64(x) - a.dc)
	}
}

// Offset returns the current bias estimate in radians/sample.
func (a *DecisionDirectedAFC) Offset() float64 { return a.dc }

// AddOffset folds an external estimate into the DDA's tracked value.
// Used at handoff: the receiver moves CoarseAFC.Offset() into the DDA
// and zeroes CoarseAFC, so the matched-filter buffer stops seeing two
// independent estimates fighting each other.
func (a *DecisionDirectedAFC) AddOffset(v float64) {
	a.dc += v
	if a.dc > a.clampRadPerSample {
		a.dc = a.clampRadPerSample
	} else if a.dc < -a.clampRadPerSample {
		a.dc = -a.clampRadPerSample
	}
}

// Reset clears the estimate. Call on stream re-tune.
func (a *DecisionDirectedAFC) Reset() {
	a.dc = 0
	a.residMean = 0
}
