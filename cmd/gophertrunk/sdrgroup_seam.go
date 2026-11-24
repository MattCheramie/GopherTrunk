package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/diversity"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// Seam analysis: the feasibility instrument for coherent SDR-group stitching
// (docs/design/coherent-sdr-stitching.md). Two adjacent group members both
// see the band where their windows overlap. Cut that band out of each
// member's recording, line the two up, and watch how their relative delay
// and phase move over time:
//
//   - the DELAY drifting means the members' sample clocks run at different
//     rates (independent crystals): no fixed alignment exists, and coherent
//     stitching is impossible on that hardware;
//   - the PHASE drifting with the delay fixed means the LOs differ but the
//     sample clocks are locked: stitching needs per-seam phase tracking;
//   - both constant means a fixed delay + phase per seam aligns the members.
//
// The verdict needs a common signal in the overlap. Noise alone has no
// coherence between two receivers, so a quiet overlap reports that instead of
// guessing.

// seamWindow is one analysis window's alignment of the upper member to the
// lower one.
type seamWindow struct {
	T          float64 // seconds into the lower member's recording
	LagSamples float64 // the upper member's lag at the seam rate (fractional)
	PhaseDeg   float64 // arg of the cross-correlation at the lag
	Coherence  float64 // |rho| at the integer lag
}

// seamReport is one seam's result.
type seamReport struct {
	Lower, Upper      string
	FreqHz            float64 // the seam band's centre
	RateHz            float64 // the rate the seam band was analysed at
	Windows           []seamWindow
	MedianCoherence   float64
	LagDriftPPM       float64 // relative sample-clock rate error, ppm
	PhaseDriftDegPerS float64
	Verdict           string
}

// Seam-analysis constants.
const (
	seamTargetRateHz   = 50_000 // the overlap band is cut out at about this rate
	seamCoarseLen      = 1 << 17
	seamWindowSeconds  = 0.25
	seamMinCoherence   = 0.2  // below: no common signal to measure
	seamLockedPPM      = 0.05 // |clock error| below this reads as locked clocks
	seamPhaseStableDeg = 5.0  // °/s below this reads as a fixed phase
)

// analyzeSeam aligns member b (centred cb) to member a (centred ca) in the
// band f0 ± bw/2, both recorded at rate.
func analyzeSeam(a, b []complex64, rate, ca, cb, f0 float64) seamReport {
	rep := seamReport{FreqHz: f0}
	da := ccdecoder.NewDownconverterWithOffset(rate, seamTargetRateHz, f0-ca)
	db := ccdecoder.NewDownconverterWithOffset(rate, seamTargetRateHz, f0-cb)
	sa := da.Process(nil, a)
	sb := db.Process(nil, b)
	rep.RateHz = da.OutRateHz()
	if len(sa) < 1024 || len(sb) < 1024 {
		rep.Verdict = "recordings too short to analyse"
		return rep
	}

	lag := coarseLag(envelope(sa), envelope(sb))
	// A residual LO offset between the members rotates b against a; over
	// one window it would cancel the correlation. Measure it from the
	// aligned cross-product and take it out first (it is reported back in
	// the phase drift).
	dfHz := residualOffsetHz(sa, sb, lag, rep.RateHz)
	if dfHz != 0 {
		rot := cmplx.Rect(1, -2*math.Pi*dfHz/rep.RateHz)
		ph := complex128(1)
		for i := range sb {
			sb[i] = complex64(complex128(sb[i]) * ph)
			ph *= rot
			if i%4096 == 0 {
				ph /= complex(cmplx.Abs(ph), 0)
			}
		}
	}
	win := int(seamWindowSeconds * rep.RateHz)
	for start := 0; start+win <= len(sa); start += win {
		best, frac, phase, ok := fineLag(sa[start:start+win], sb, start, lag)
		if !ok {
			continue
		}
		lag = best // track a drifting lag window to window
		var cs diversity.CrossStats
		cs.Accumulate(sa[start:start+win], sb[start+best:start+best+win])
		rep.Windows = append(rep.Windows, seamWindow{
			T: float64(start) / rep.RateHz, LagSamples: float64(best) + frac,
			PhaseDeg: phase, Coherence: cs.Coherence(),
		})
	}
	if len(rep.Windows) < 2 {
		rep.Verdict = "the two recordings overlap too little in time to analyse"
		return rep
	}

	cohs := make([]float64, len(rep.Windows))
	for i, w := range rep.Windows {
		cohs[i] = w.Coherence
	}
	sort.Float64s(cohs)
	rep.MedianCoherence = cohs[len(cohs)/2]

	// Fit only the windows that saw the common signal.
	var ts, lags, phases []float64
	prev := 0.0
	for _, w := range rep.Windows {
		if w.Coherence < seamMinCoherence {
			continue
		}
		p := w.PhaseDeg
		if len(phases) > 0 {
			for p-prev > 180 {
				p -= 360
			}
			for p-prev < -180 {
				p += 360
			}
		}
		prev = p
		ts, lags, phases = append(ts, w.T), append(lags, w.LagSamples), append(phases, p)
	}
	if len(ts) < 2 {
		rep.Verdict = fmt.Sprintf("no common signal in the overlap (median coherence %.2f): put a steady carrier (or a noise source) in %.4f MHz ± %.0f kHz and record again",
			rep.MedianCoherence, f0/1e6, rep.RateHz/2e3)
		return rep
	}
	lagSlope := slope(ts, lags) // samples per second
	rep.LagDriftPPM = lagSlope / rep.RateHz * 1e6
	rep.PhaseDriftDegPerS = slope(ts, phases) + 360*dfHz
	switch {
	case math.Abs(rep.LagDriftPPM) > seamLockedPPM:
		rep.Verdict = fmt.Sprintf("independent sample clocks (%.2f ppm apart): no fixed alignment exists, so coherent stitching is impossible on this pair; the virtual group (sdr.groups) is the right mode", rep.LagDriftPPM)
	case math.Abs(rep.PhaseDriftDegPerS) > seamPhaseStableDeg:
		rep.Verdict = fmt.Sprintf("sample clocks locked but the phase walks %.1f °/s (LOs differ by %.2f Hz): coherent stitching needs per-seam phase tracking", rep.PhaseDriftDegPerS, rep.PhaseDriftDegPerS/360)
	default:
		rep.Verdict = fmt.Sprintf("coherent stitching feasible on this seam: fixed delay %.2f samples at %.0f S/s, fixed phase %.1f°",
			lags[len(lags)/2], rep.RateHz, phases[len(phases)/2])
	}
	return rep
}

// envelope returns the mean-removed power of x: a frequency offset between
// two receivers leaves it unchanged, so it aligns them in time when their
// LOs differ.
func envelope(x []complex64) []complex64 {
	out := make([]complex64, len(x))
	var mean float64
	for _, v := range x {
		mean += float64(real(v)*real(v) + imag(v)*imag(v))
	}
	mean /= float64(len(x))
	for i, v := range x {
		out[i] = complex(float32(float64(real(v)*real(v)+imag(v)*imag(v))-mean), 0)
	}
	return out
}

// residualOffsetHz estimates the frequency offset of b against a at the
// given lag: the peak of the spectrum of b[n+lag]·conj(a[n]), refined
// parabolically. 0 when the product has no clear line.
func residualOffsetHz(a, b []complex64, lag int, rate float64) float64 {
	size := 1
	for size < seamCoarseLen {
		size <<= 1
	}
	prod := make([]complex128, size)
	n := 0
	for i := 0; i < len(a) && n < size; i++ {
		j := i + lag
		if j < 0 || j >= len(b) {
			continue
		}
		prod[n] = complex128(b[j]) * cmplx.Conj(complex128(a[i]))
		n++
	}
	if n < 1024 {
		return 0
	}
	spec := fft.New(size).Forward(nil, prod)
	mags := make([]float64, size)
	best := 0
	var sum float64
	for i, v := range spec {
		mags[i] = cmplx.Abs(v)
		sum += mags[i]
		if mags[i] > mags[best] {
			best = i
		}
	}
	if mags[best] < 8*sum/float64(size) {
		return 0 // no line stands out: no common signal, or none to correct
	}
	frac := 0.0
	l, r := (best-1+size)%size, (best+1)%size
	if d := mags[l] - 2*mags[best] + mags[r]; d != 0 {
		frac = 0.5 * (mags[l] - mags[r]) / d
	}
	k := float64(best) + frac
	if k > float64(size)/2 {
		k -= float64(size)
	}
	return k * rate / float64(size)
}

// coarseLag finds the integer lag L maximising |Σ b[n+L]·conj(a[n])| over the
// first seamCoarseLen samples, by FFT cross-correlation. L may be negative.
func coarseLag(a, b []complex64) int {
	n := min(len(a), len(b), seamCoarseLen)
	size := 1
	for size < 2*n {
		size <<= 1
	}
	plan := fft.New(size)
	pa := make([]complex128, size)
	pb := make([]complex128, size)
	for i := 0; i < n; i++ {
		pa[i], pb[i] = complex128(a[i]), complex128(b[i])
	}
	fa := plan.Forward(nil, pa)
	fb := plan.Forward(nil, pb)
	for i := range fb {
		fb[i] *= cmplx.Conj(fa[i])
	}
	r := plan.Inverse(nil, fb)
	best, bestMag := 0, -1.0
	for i, v := range r {
		if m := cmplx.Abs(v); m > bestMag {
			best, bestMag = i, m
		}
	}
	if best > size/2 {
		best -= size
	}
	return best
}

// fineLag searches lags lag-2..lag+2 for window wa (starting at index start
// in the a stream) against b, returning the best integer lag, its parabolic
// fractional refinement, and the phase there. ok is false when the window
// does not fit inside b at those lags.
func fineLag(wa, b []complex64, start, lag int) (best int, frac, phaseDeg float64, ok bool) {
	const span = 2
	mags := make([]float64, 2*span+1)
	sums := make([]complex128, 2*span+1)
	for k := -span; k <= span; k++ {
		off := start + lag + k
		if off < 0 || off+len(wa) > len(b) {
			return 0, 0, 0, false
		}
		var s complex128
		for i, x := range wa {
			s += complex128(b[off+i]) * cmplx.Conj(complex128(x))
		}
		sums[k+span], mags[k+span] = s, cmplx.Abs(s)
	}
	bi := 0
	for i := range mags {
		if mags[i] > mags[bi] {
			bi = i
		}
	}
	if bi > 0 && bi < len(mags)-1 {
		y0, y1, y2 := mags[bi-1], mags[bi], mags[bi+1]
		if d := y0 - 2*y1 + y2; d != 0 {
			frac = 0.5 * (y0 - y2) / d
		}
	}
	return lag + bi - span, frac, cmplx.Phase(sums[bi]) * 180 / math.Pi, true
}

// slope is the least-squares slope of y against x.
func slope(x, y []float64) float64 {
	n := float64(len(x))
	var sx, sy, sxx, sxy float64
	for i := range x {
		sx, sy, sxx, sxy = sx+x[i], sy+y[i], sxx+x[i]*x[i], sxy+x[i]*y[i]
	}
	d := n*sxx - sx*sx
	if d == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / d
}
