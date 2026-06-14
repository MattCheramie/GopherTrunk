package carriers

import "math"

// standardChannelSteps are the trunking channel rasters InferGrid tests, finest
// first. They are the channel steps used by real P25/DMR/MPT/analog land-mobile
// systems worldwide; 11.6 kHz and friends are measurement artifacts, not real
// channel plans.
var standardChannelSteps = []uint32{6_250, 12_500, 25_000}

// gridFitThreshold is the minimum mean-resultant-length (circular concentration,
// 0..1) a candidate raster must reach for its residuals to count as "clustered
// on a grid". 0.8 keeps a genuine raster (residuals piled at one phase) while
// rejecting frequencies scattered across the step.
const gridFitThreshold = 0.8

// minGridCandidates is the fewest candidates InferGrid needs before it trusts a
// detected raster; below it there isn't enough evidence to pick a step, so the
// caller falls back to the finest standard step (which still corrects sub-step
// bin jitter).
const minGridCandidates = 3

// InferGrid picks the channel raster (step, phase in Hz) that best explains the
// candidate frequencies. For each standard step it treats the residuals
// (f mod step) as points on a circle and measures their concentration (mean
// resultant length); the raster fits when the residuals pile up at one phase.
// It returns the coarsest step whose residuals are tightly clustered — the
// coarsest raster the channels genuinely sit on — and that cluster's phase.
//
// With too few candidates, or no step clustering tightly, it falls back to the
// finest standard step at phase 0: a lone carrier (the 440.125 MHz repro) then
// still snaps to the nearest 6.25 kHz point, correcting the FFT-bin offset.
func InferGrid(freqs []uint32) (stepHz, phaseHz uint32) {
	if len(freqs) < minGridCandidates {
		return standardChannelSteps[0], 0
	}
	bestStep, bestPhase := uint32(0), uint32(0)
	for _, step := range standardChannelSteps {
		phase, r := gridFit(freqs, step)
		if r >= gridFitThreshold {
			// Coarsest qualifying step wins: standardChannelSteps is finest-first,
			// so keep overwriting as we walk to coarser steps that still fit.
			bestStep, bestPhase = step, phase
		}
	}
	if bestStep == 0 {
		return standardChannelSteps[0], 0
	}
	return bestStep, bestPhase
}

// gridFit returns the cluster phase (Hz) and the mean resultant length (0..1) of
// the candidate frequencies' residuals modulo step. A resultant length near 1
// means the residuals pile up at one phase (the carriers sit on this raster);
// near 0 means they are spread across the step (this raster doesn't explain them).
func gridFit(freqs []uint32, step uint32) (phaseHz uint32, resultant float64) {
	var sumSin, sumCos float64
	for _, f := range freqs {
		theta := 2 * math.Pi * float64(f%step) / float64(step)
		sumSin += math.Sin(theta)
		sumCos += math.Cos(theta)
	}
	n := float64(len(freqs))
	resultant = math.Hypot(sumCos, sumSin) / n
	mean := math.Atan2(sumSin, sumCos)
	if mean < 0 {
		mean += 2 * math.Pi
	}
	phaseHz = uint32(math.Round(mean/(2*math.Pi)*float64(step))) % step
	return phaseHz, resultant
}

// SnapHz snaps f to the nearest point on the (step, phase) grid, but only when
// that point is within maxCorrectionHz of f — so a small FFT-bin quantization
// error is corrected to the true channel centre while a carrier that genuinely
// sits off the raster (e.g. an analog channel on a different plan) is left where
// it is. maxCorrectionHz == 0, or step == 0, disables snapping.
func SnapHz(f, step, phase, maxCorrectionHz uint32) uint32 {
	if step == 0 || maxCorrectionHz == 0 {
		return f
	}
	k := math.Round((float64(f) - float64(phase)) / float64(step))
	nearest := int64(phase) + int64(k)*int64(step)
	if nearest < 0 {
		return f
	}
	d := int64(f) - nearest
	if d < 0 {
		d = -d
	}
	if d <= int64(maxCorrectionHz) {
		return uint32(nearest)
	}
	return f
}
