package demod

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
// eye back inside the slicer's decision margins.
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
	for i, x := range buf {
		a.dc += a.beta * (float64(x) - a.dc)
		buf[i] = float32(float64(x) - a.dc)
	}
}

// Offset returns the current bias estimate in radians/sample; the
// residual carrier offset is Offset()·Fs/(2π) hertz.
func (a *CoarseAFC) Offset() float64 { return a.dc }

// Reset clears the estimate. Call on stream re-tune so a stale offset
// doesn't bleed across the discontinuity.
func (a *CoarseAFC) Reset() { a.dc = 0 }
