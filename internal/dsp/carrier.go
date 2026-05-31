package dsp

import (
	"math"
	"math/cmplx"
)

// EstimateCarrierOffsetHz finds the dominant narrowband carrier offset in a
// complex baseband capture: the frequency (in Hz, positive = above centre)
// carrying the most averaged power within ±searchHz of DC. It is intended
// for tuning a recorded wideband capture's control channel down to 0 Hz
// before a receiver that expects a channelised, centred stream (the SDR
// tuner does this in the live pipeline; a file replay has to do it here).
//
// The search is a two-stage averaged periodogram evaluated directly (a
// bounded Goertzel-style DFT, no full FFT): a coarse pass locates the peak
// bin across the band, a fine pass refines it around that bin. This costs
// O(nFreq · window · nWindows) — cheap for the small bounded grids used —
// and avoids a transform-size/zero-pad dependency.
//
// For a suppressed-carrier linear modulation (π/4-DQPSK / LSM) the power
// spectral density is symmetric about the carrier, so its peak coincides
// with the carrier offset. Returns 0 for an empty input or zero searchHz.
func EstimateCarrierOffsetHz(iq []complex64, sampleRateHz, searchHz float64) float64 {
	if len(iq) == 0 || searchHz <= 0 || sampleRateHz <= 0 {
		return 0
	}
	x := make([]complex128, len(iq))
	for i, v := range iq {
		x[i] = complex(float64(real(v)), float64(imag(v)))
	}
	loN := -searchHz / sampleRateHz
	hiN := searchHz / sampleRateHz
	const coarseN = 257
	coarse := zoomPeakNorm(x, loN, hiN, coarseN, 4096, 100)
	stepN := (hiN - loN) / float64(coarseN-1)
	fine := zoomPeakNorm(x, coarse-stepN, coarse+stepN, 201, 16384, 60)
	return fine * sampleRateHz
}

// zoomPeakNorm returns the normalised frequency (cycles/sample) in
// [f0, f1] with the greatest averaged |X(f)|² over up to maxWindows blocks
// of length win. Frequencies are evaluated by direct summation so the grid
// is independent of any FFT bin size.
func zoomPeakNorm(x []complex128, f0, f1 float64, nf, win, maxWindows int) float64 {
	if win > len(x) {
		win = len(x)
	}
	if win <= 0 {
		return (f0 + f1) / 2
	}
	bestF, bestP := (f0+f1)/2, -1.0
	for i := 0; i < nf; i++ {
		f := f0
		if nf > 1 {
			f = f0 + (f1-f0)*float64(i)/float64(nf-1)
		}
		// Recursive phasor: one cmplx.Exp per frequency, then a single
		// complex multiply per sample (instead of cmplx.Exp per sample).
		step := cmplx.Exp(complex(0, -2*math.Pi*f))
		var acc float64
		nw := 0
		for off := 0; off+win <= len(x) && nw < maxWindows; off += win {
			ph := complex(1, 0)
			var s complex128
			for n := 0; n < win; n++ {
				s += x[off+n] * ph
				ph *= step
			}
			acc += real(s * cmplx.Conj(s))
			nw++
		}
		if acc > bestP {
			bestP, bestF = acc, f
		}
	}
	return bestF
}
