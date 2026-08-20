package soapyremote

import "math"

// branchAligner removes a CONSTANT inter-branch timing skew before the MRC
// combine. A scalar complex gain assumes the branches are time-aligned; a
// constant delay between them (start-of-stream skew, differing DDC group
// delay between daughterboards) turns the coherent sum into a comb filter —
// band-average power still looks fine, per-frequency coherence stays high,
// but broadband |rho| is diluted and the combined SYMBOLS are smeared. On the
// operator's 19 Aug 2026 X310 capture, branch 0 lagged branch 1 by a constant
// 2.60 samples (13 µs at 200 kS/s): every scalar-combined arm decoded ~22%
// FEWER CRC-clean BSCH than the best branch alone, while the same combine
// after delay alignment matched it exactly (see
// cmd/gophertrunk/diversity_replay_test.go's wb-aligned-static arm).
//
// Operation: while unmeasured, it accumulates a per-branch sample buffer;
// once full it scans integer lags ±alignMaxLag for the |rho|-maximising delay
// of branch 1 relative to branch 0, refines the fraction by parabolic
// interpolation of the neighbouring lags, and LATCHES the result — exactly
// once per measurement cycle, like the phase anchor, because changing a delay
// mid-stream is itself a discontinuity. From then on process() delays the
// EARLY branch by the measured amount through a small per-branch delay line
// (linear interpolation for the fraction — the channel occupies a few percent
// of the stream rate, where a linear interpolator is transparent). A
// measurement whose peak |rho| does not clear alignMinRho (no coherent signal
// yet) is discarded and re-accumulated. Stream-goroutine only.
type branchAligner struct {
	measured bool
	// delaySamples/delayFrac: the branch delayed and by how much. delayIdx is
	// -1 while nothing needs delaying (measured a zero skew, or unmeasured).
	delayIdx     int
	delaySamples int
	delayFrac    float64
	dline        []complex64 // last delaySamples+1 samples of the delayed branch

	buf0, buf1 []complex64 // measurement accumulation (nil once measured)
}

const (
	// alignMeasureSamples is the per-branch buffer the delay estimate is
	// computed from: 2^17 samples (0.65 s at 200 kS/s, 2 MB total) gives a
	// noise-only |rho| floor of ~0.0025 — alignMinRho sits 40× above it.
	alignMeasureSamples = 1 << 17
	// alignMaxLag bounds the scan; a skew beyond ±16 samples is not a subtle
	// DSP-chain difference but a broken stream, and combining is hopeless
	// anyway.
	alignMaxLag = 16
	// alignMinRho is the minimum peak coherence to latch a measurement: below
	// it there is no coherent signal to align on, and latching a noise-driven
	// ±16-sample delay would be worse than none. Discard and re-accumulate.
	alignMinRho = 0.1
)

func newBranchAligner() *branchAligner { return &branchAligner{delayIdx: -1} }

// reset drops the latched delay and measurement state; the next process()
// starts a fresh measurement. Call when the stream restarts (a start-of-
// stream skew is per-stream) or on a retune (cheap, and conservative about
// hardware that re-times its DSP chain on an LO change).
func (a *branchAligner) reset() {
	a.measured = false
	a.delayIdx = -1
	a.delaySamples = 0
	a.delayFrac = 0
	a.dline = nil
	a.buf0, a.buf1 = nil, nil
}

// process aligns one two-branch datagram in place (the delayed branch's slice
// is REPLACED, not mutated). Returns the measured total delay and true the
// one time a measurement latches, so the caller can log it.
func (a *branchAligner) process(branches [][]complex64) (latched bool, delay float64, peakRho float64) {
	if len(branches) != 2 {
		return false, 0, 0
	}
	if !a.measured {
		a.buf0 = append(a.buf0, branches[0]...)
		a.buf1 = append(a.buf1, branches[1]...)
		if len(a.buf0) < alignMeasureSamples || len(a.buf1) < alignMeasureSamples {
			return false, 0, 0
		}
		lag, frac, rho := estimateBranchDelay(a.buf0, a.buf1)
		a.buf0, a.buf1 = nil, nil
		if rho < alignMinRho {
			// No coherent signal yet: re-accumulate rather than latch noise.
			return false, 0, 0
		}
		a.measured = true
		total := float64(lag) + frac // delay of branch 1 relative to branch 0
		// Delay the EARLY branch so both see the wavefront at the same index.
		switch {
		case total > 0.01: // branch 1 lags ⇒ branch 0 is early
			a.configureDelay(0, total)
		case total < -0.01: // branch 0 lags ⇒ branch 1 is early
			a.configureDelay(1, -total)
		default:
			a.delayIdx = -1 // aligned already; nothing to do
		}
		latched, delay, peakRho = true, total, rho
	}
	if a.delayIdx >= 0 {
		branches[a.delayIdx] = a.applyDelay(branches[a.delayIdx])
	}
	return latched, delay, peakRho
}

func (a *branchAligner) configureDelay(idx int, total float64) {
	a.delayIdx = idx
	a.delaySamples = int(total)
	a.delayFrac = total - float64(a.delaySamples)
	a.dline = make([]complex64, a.delaySamples+1)
}

// applyDelay emits x delayed by delaySamples+delayFrac, carrying the tail
// across datagrams: with dline = the previous (delaySamples+1) samples and
// ext = dline ++ x, y[i] = (1-f)·ext[i+1] + f·ext[i].
func (a *branchAligner) applyDelay(x []complex64) []complex64 {
	ext := make([]complex64, 0, len(a.dline)+len(x))
	ext = append(ext, a.dline...)
	ext = append(ext, x...)
	f := float32(a.delayFrac)
	y := make([]complex64, len(x))
	for i := range y {
		y[i] = ext[i+1]*complex(1-f, 0) + ext[i]*complex(f, 0)
	}
	copy(a.dline, ext[len(ext)-len(a.dline):])
	return y
}

// estimateBranchDelay returns the integer lag (positive = branch 1 lags
// branch 0), a parabolic fractional refinement in (-1, 1), and the peak
// normalised |rho| at the integer lag. Inputs are DC-removed before
// correlating — two branches sharing LO leakage would otherwise report a
// confident lag-0 on nothing (the CrossStats lesson).
func estimateBranchDelay(b0, b1 []complex64) (lag int, frac float64, rho float64) {
	n := len(b0)
	if len(b1) < n {
		n = len(b1)
	}
	if n <= 2*alignMaxLag+2 {
		return 0, 0, 0
	}
	x0 := removeDC(b0[:n])
	x1 := removeDC(b1[:n])

	var p0, p1 float64
	for i := 0; i < n; i++ {
		p0 += float64(real(x0[i]))*float64(real(x0[i])) + float64(imag(x0[i]))*float64(imag(x0[i]))
		p1 += float64(real(x1[i]))*float64(real(x1[i])) + float64(imag(x1[i]))*float64(imag(x1[i]))
	}
	norm := math.Sqrt(p0 * p1)
	if norm == 0 {
		return 0, 0, 0
	}

	mags := make([]float64, 2*alignMaxLag+1)
	best := 0.0
	for l := -alignMaxLag; l <= alignMaxLag; l++ {
		var cr, ci float64
		for i := 0; i < n-alignMaxLag; i++ {
			var a, b complex64
			if l >= 0 {
				a, b = x0[i], x1[i+l]
			} else {
				a, b = x0[i-l], x1[i]
			}
			// conj(a)·b
			cr += float64(real(a))*float64(real(b)) + float64(imag(a))*float64(imag(b))
			ci += float64(real(a))*float64(imag(b)) - float64(imag(a))*float64(real(b))
		}
		m := math.Hypot(cr, ci)
		mags[l+alignMaxLag] = m
		if r := m / norm; r > best {
			best, lag = r, l
		}
	}
	rho = best

	// Parabolic refinement on the correlation magnitudes around the peak.
	k := lag + alignMaxLag
	if k > 0 && k < len(mags)-1 {
		ym, y0, yp := mags[k-1], mags[k], mags[k+1]
		den := ym - 2*y0 + yp
		if den != 0 {
			f := 0.5 * (ym - yp) / den
			if f > -0.99 && f < 0.99 && !math.IsNaN(f) {
				frac = f
			}
		}
	}
	return lag, frac, rho
}

func removeDC(x []complex64) []complex64 {
	var sr, si float64
	for _, z := range x {
		sr += float64(real(z))
		si += float64(imag(z))
	}
	m := complex(float32(sr/float64(len(x))), float32(si/float64(len(x))))
	out := make([]complex64, len(x))
	for i, z := range x {
		out[i] = z - m
	}
	return out
}
