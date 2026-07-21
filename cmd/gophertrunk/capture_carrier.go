package main

import (
	"fmt"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
)

// captureProbeSamples is how many recorded (post-DDC) IQ samples the capture
// collects for the carrier-offset probe. A power-of-two window keeps the FFT
// cheap; 32768 bins give sub-100 Hz resolution even at multi-MS/s rates and a
// fraction of a second at a narrowband slice rate.
const captureProbeSamples = 1 << 15

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
	width := ((r - l) % n + n) % n
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
// capture's carrier sits far enough off centre to matter, or "" when it is
// close enough (or couldn't be estimated). freqHz is the tuned RF centre, used
// to translate the offset into an approximate ppm figure.
func carrierOffsetWarning(iq []complex64, rate float64, freqHz uint32) string {
	offset, ok := captureCarrierOffsetHz(iq, rate)
	if !ok || math.Abs(offset) < carrierOffsetWarnHz {
		return ""
	}
	ppm := 0.0
	if freqHz > 0 {
		ppm = offset / float64(freqHz) * 1e6
	}
	return formatCarrierOffsetWarning(offset, ppm)
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
