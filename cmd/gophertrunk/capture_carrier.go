package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
)

// captureProbeSamples is how many recorded (post-DDC) IQ samples each
// carrier-offset probe window holds. A power-of-two window keeps the FFT
// cheap; 32768 bins give sub-100 Hz resolution even at multi-MS/s rates and a
// fraction of a second at a narrowband slice rate.
const captureProbeSamples = 1 << 15

// captureProbeWindows is how many probe windows the capture spreads across the
// whole recording. One window at the very start of the stream (the original
// design) is exactly where front-end settling lives, and 11 ms at 3 MS/s is
// short enough for a momentarily-keyed carrier elsewhere in the span to win
// the probe — the #1143 "≈550.3 ppm" false warning. A real tuner ppm error is
// constant for the whole file, so requiring the same offset across windows
// spread over the capture separates it from transients.
const captureProbeWindows = 8

// carrierConsensusToleranceHz is how closely per-window offset estimates must
// agree to count as the same carrier. Window resolution is sub-100 Hz; the
// smoothing hump centre wobbles by a fraction of the channel width between
// windows, so 1 kHz is generous for "same channel" while safely separating
// 25 kHz-spaced neighbours.
const carrierConsensusToleranceHz = 1000.0

// carrierProbe collects captureProbeWindows disjoint windows of recorded
// (post-DDC) IQ, spread evenly across the expected length of the capture, for
// the carrier-offset consensus estimate. Feed sees the recorded stream in
// order; expected is the anticipated total recorded sample count (a capture
// cut short by Ctrl-C simply leaves later windows empty, and a too-short
// capture degenerates to a single window at the start — the old behaviour).
type carrierProbe struct {
	starts  []int64
	windows [][]complex64
	pos     int64
}

func newCarrierProbe(expected int64) *carrierProbe {
	k := int64(captureProbeWindows)
	if maxK := expected / captureProbeSamples; maxK < k {
		k = maxK
	}
	if k < 1 {
		k = 1
	}
	stride := int64(captureProbeSamples)
	if k > 1 {
		stride = expected / k
	}
	p := &carrierProbe{}
	for i := int64(0); i < k; i++ {
		p.starts = append(p.starts, i*stride)
		p.windows = append(p.windows, make([]complex64, 0, captureProbeSamples))
	}
	return p
}

// feed appends the overlap of this (in-order, contiguous) recorded chunk with
// each still-unfilled probe window.
func (p *carrierProbe) feed(samples []complex64) {
	base := p.pos
	p.pos += int64(len(samples))
	for i, start := range p.starts {
		w := p.windows[i]
		if len(w) >= captureProbeSamples {
			continue
		}
		lo := start + int64(len(w)) // next absolute index this window needs
		hi := start + captureProbeSamples
		s, e := lo-base, hi-base
		if s < 0 {
			s = 0
		}
		if e > int64(len(samples)) {
			e = int64(len(samples))
		}
		if s < e {
			p.windows[i] = append(w, samples[s:e]...)
		}
	}
}

// Windows returns the collected probe windows, dropping any too short to
// estimate from (an early-terminated capture leaves trailing windows empty).
func (p *carrierProbe) Windows() [][]complex64 {
	if p == nil {
		return nil
	}
	out := make([][]complex64, 0, len(p.windows))
	for _, w := range p.windows {
		if len(w) >= 1024 {
			out = append(out, w)
		}
	}
	return out
}

// carrierConsensus is the result of the multi-window carrier-offset estimate.
type carrierConsensus struct {
	OffsetHz float64
	MaxAbsHz float64 // largest |estimate| any single window produced
	Agree    int     // windows agreeing with the median estimate
	Est      int     // windows that produced any estimate
	Usable   int     // windows long enough to probe
	OK       bool    // a consensus offset was reached
}

// carrierOffsetConsensus runs the single-window estimator over every probe
// window and accepts an offset only when the estimates corroborate each
// other: the transient that fooled the old first-11-ms probe (#1143) lights
// one window; a real ppm error lights every window the signal is up in. With
// a single usable window (a short capture) the lone estimate stands, as
// before. With several, at least two must agree within
// carrierConsensusToleranceHz of their median, and the agreeing set must be a
// majority of the windows that produced an estimate at all.
func carrierOffsetConsensus(windows [][]complex64, rate float64) carrierConsensus {
	var c carrierConsensus
	var ests []float64
	for _, w := range windows {
		if len(w) < 1024 {
			continue
		}
		c.Usable++
		if off, ok := captureCarrierOffsetHz(w, rate); ok {
			ests = append(ests, off)
			if a := math.Abs(off); a > c.MaxAbsHz {
				c.MaxAbsHz = a
			}
		}
	}
	c.Est = len(ests)
	if c.Est == 0 {
		return c
	}
	sorted := append([]float64(nil), ests...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]
	var agreeing []float64
	for _, e := range ests {
		if math.Abs(e-median) <= carrierConsensusToleranceHz {
			agreeing = append(agreeing, e)
		}
	}
	c.Agree = len(agreeing)
	if c.Usable > 1 && (c.Agree < 2 || 2*c.Agree < c.Est) {
		return c
	}
	var sum float64
	for _, e := range agreeing {
		sum += e
	}
	c.OffsetHz = sum / float64(len(agreeing))
	c.OK = true
	return c
}

// carrierOffsetWarnHz is the |offset| above which the capture warns. Small
// offsets are pulled in by every decoder's AFC; a multi-kHz offset (a real
// ppm error) is what prevents lock, which is what we want to flag.
const carrierOffsetWarnHz = 2000.0

// captureCarrierOffsetHz estimates the dominant signal's frequency offset from
// baseband centre (DC) over iq, sampled at rate Hz. A ppm-mistuned capture
// puts the whole channel off centre by a fixed number of Hz; this recovers
// that.
//
// A modulated channel has no sharp carrier line — its power is spread across
// the occupied band and (being noisy and roughly symmetric) a simple centroid
// is pulled toward DC by the noise. Instead this smooths the power spectrum
// with a moving average about a channel-fraction wide, collapsing the
// modulation ripple into a single hump, and returns the peak of that hump —
// the channel's centre. It returns ok=false when the window is too short or no
// hump stands clearly above the mean (a noise-only capture), so a featureless
// capture never produces a spurious warning.
func captureCarrierOffsetHz(iq []complex64, rate float64) (float64, bool) {
	// Largest power of two we have, capped at the probe size.
	n := 1
	for n*2 <= len(iq) && n*2 <= captureProbeSamples {
		n *= 2
	}
	if n < 1024 || rate <= 0 {
		return 0, false
	}
	in := fft.Cmplx64ToCmplx128(nil, iq[:n])
	spec := fft.New(n).Forward(nil, in)

	power := make([]float64, n)
	var total float64
	for k := 0; k < n; k++ {
		re, im := real(spec[k]), imag(spec[k])
		power[k] = re*re + im*im
		total += power[k]
	}
	if total <= 0 {
		return 0, false
	}

	// Circular moving-average sum over a window ~n/64 bins wide. The spectrum
	// wraps at DC (bin n-1 is adjacent to bin 0), so the window is circular.
	win := n / 64
	if win < 8 {
		win = 8
	}
	half := win / 2
	win = 2*half + 1 // actual window width

	smoothed := make([]float64, n)
	var sum float64
	for j := -half; j <= half; j++ {
		sum += power[((j%n)+n)%n]
	}
	smoothed[0] = sum
	best, bestK := sum, 0
	for k := 1; k < n; k++ {
		sum -= power[(((k-1-half)%n)+n)%n]
		sum += power[(((k+half)%n)+n)%n]
		smoothed[k] = sum
		if sum > best {
			best, bestK = sum, k
		}
	}

	// Require the peak window to hold clearly more energy than an average
	// window — otherwise there is no channel, just noise.
	meanWindow := total / float64(n) * float64(win)
	const peakMargin = 3.0
	if best < meanWindow*peakMargin {
		return 0, false
	}

	// The argmax is the FIRST bin reaching the peak, which for a flat-topped
	// channel (or a sharp tone) sits at the leading edge of the plateau —
	// biased low by ~half the smoothing window. Expand the plateau (circularly)
	// and take its centre so the estimate lands on the channel's middle.
	thr := best * 0.999
	l, r := bestK, bestK
	for i := 0; i < n; i++ {
		p := ((l-1)%n + n) % n
		if smoothed[p] < thr {
			break
		}
		l = p
	}
	for i := 0; i < n; i++ {
		q := (r + 1) % n
		if smoothed[q] < thr {
			break
		}
		r = q
	}
	width := ((r-l)%n + n) % n
	centerK := (l + width/2) % n

	// Map the centre bin to a signed baseband frequency (fftshift): the upper
	// half of the spectrum is the negative frequencies.
	f := float64(centerK)
	if centerK >= n/2 {
		f -= float64(n)
	}
	return f * rate / float64(n), true
}

// carrierOffsetWarning returns an operator-facing warning when a recorded
// capture's carrier sits far enough off centre to matter — corroborated
// across the probe windows — or "" when it is close enough, transient, or
// couldn't be estimated. freqHz is the tuned RF centre, used to translate the
// offset into an approximate ppm figure.
func carrierOffsetWarning(windows [][]complex64, rate float64, freqHz uint32) string {
	c := carrierOffsetConsensus(windows, rate)
	if !c.OK || math.Abs(c.OffsetHz) < carrierOffsetWarnHz {
		return ""
	}
	return formatCarrierOffsetWarning(c.OffsetHz, offsetPPM(c.OffsetHz, freqHz))
}

func offsetPPM(offsetHz float64, freqHz uint32) float64 {
	if freqHz == 0 {
		return 0
	}
	return offsetHz / float64(freqHz) * 1e6
}

// carrierOffsetMeasurement renders the always-on measured-offset line for a
// consensus estimate — the capture command doubling as a ppm-measurement
// instrument (issue #836: kalibrate-rtl needs GSM towers, which no longer
// exist in much of the world; any strong known-frequency carrier plus this
// line does the same job). Returns "" when no consensus was reached; the
// companion carrierOffsetInconclusiveNote covers the transient case.
func carrierOffsetMeasurement(c carrierConsensus, freqHz uint32) string {
	if !c.OK {
		return ""
	}
	msg := fmt.Sprintf("capture: measured carrier offset %+.0f Hz from centre", c.OffsetHz)
	if freqHz > 0 {
		msg += fmt.Sprintf(" (≈%.1f ppm at %.4f MHz)", math.Abs(offsetPPM(c.OffsetHz, freqHz)), float64(freqHz)/1e6)
	}
	if c.Usable > 1 {
		msg += fmt.Sprintf(", consistent across %d of %d probe windows spread over the capture", c.Agree, c.Usable)
	}
	msg += ". If the tuned frequency is a known-accurate carrier, this measures the tuner's ppm error for -ppm / sdr.ppm."
	return msg
}

// carrierOffsetInconclusiveNote flags the case where the probe saw a strong
// carrier in some window but the windows do not corroborate one another — a
// keyed-up transient or front-end settling, not a tuner offset. The old
// single-window probe turned exactly this into a confident wrong ppm warning
// (#1143's "≈550.3 ppm"). Returns "" when there was consensus or nothing to
// see.
func carrierOffsetInconclusiveNote(c carrierConsensus) string {
	if c.OK || c.Est == 0 || c.Usable <= 1 || c.MaxAbsHz < carrierOffsetWarnHz {
		return ""
	}
	return fmt.Sprintf("capture: carrier-offset probe inconclusive — a strong off-centre signal appeared in "+
		"%d of %d probe windows without agreeing across the capture; likely a transient/burst carrier, not a "+
		"tuner ppm error, so no offset is reported.", c.Est, c.Usable)
}

// formatCarrierOffsetWarning renders the warning string. Split out so the
// wording is unit-testable without synthesising IQ.
func formatCarrierOffsetWarning(offsetHz, ppm float64) string {
	// The sign convention of -ppm is driver-specific, so quote the magnitude
	// and tell the operator which way to nudge rather than risk a wrong sign.
	dir := "below"
	if offsetHz > 0 {
		dir = "above"
	}
	msg := fmt.Sprintf("capture: WARNING — the recorded carrier sits %.1f kHz %s centre",
		math.Abs(offsetHz)/1000, dir)
	if ppm != 0 {
		msg += fmt.Sprintf(" (≈%.1f ppm at this frequency)", math.Abs(ppm))
	}
	msg += fmt.Sprintf(". If unintended this is an uncorrected tuner error: set -ppm to "+
		"pull the carrier back to centre (try ±%.0f and keep the sign that shrinks the "+
		"offset). If you offset -center on purpose, ignore this.", math.Abs(ppm))
	return msg
}
