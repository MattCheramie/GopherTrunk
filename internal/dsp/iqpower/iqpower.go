// Package iqpower holds the small, reusable IQ signal-level helpers
// shared by the decode engines for operator-facing diagnostics: a
// window-averaged dBFS power measure plus the "looks dead" threshold.
//
// The dBFS computation mirrors the control-channel decoder's
// observeIQPower (internal/scanner/ccdecoder) bit for bit — accumulate
// |x|² over a window, divide by the sample count, clamp at a small
// epsilon so a silent window reads "very low" instead of -Inf, and take
// 10·log10. Extracting it here lets the wideband Tier II engine surface
// the same per-channel signal level the ccdecoder already exposes,
// without either package importing the other.
package iqpower

import (
	"math"
	"time"
)

// Window is the interval over which power is aggregated before a fresh
// mean is reported. Mirrors ccdecoder's iqPowerWindow.
const Window = time.Second

// LowPowerThresholdDbFS is the level below which the IQ stream looks
// dead — gain at 0, antenna disconnected, or USB stuck. Mirrors
// ccdecoder's iqLowPowerThresholdDbFS: -55 dBFS sits well below the
// idle-noise floor of an R820T2 at moderate gain (~-45 dBFS), so
// legitimate weak signal doesn't trip it.
const LowPowerThresholdDbFS = -55.0

// powerEpsilon clamps the mean power so 10·log10 of a silent window
// yields a finite "very low" reading instead of -Inf.
const powerEpsilon = 1e-12

// MeanPowerDbFS returns 10·log10(mean(|x|²)) over iq, clamped at
// powerEpsilon. Returns the clamped floor for an empty slice.
func MeanPowerDbFS(iq []complex64) float64 {
	if len(iq) == 0 {
		return 10 * math.Log10(powerEpsilon)
	}
	var sumSq float64
	for _, c := range iq {
		r := float64(real(c))
		i := float64(imag(c))
		sumSq += r*r + i*i
	}
	return dbFS(sumSq / float64(len(iq)))
}

// Accumulator folds successive sample windows so a caller pumping IQ in
// chunks can report one mean per Window without buffering the samples.
// It is not safe for concurrent use; the wideband engine drives it
// inline on its single pump goroutine.
type Accumulator struct {
	sumSq float64
	n     int
}

// Add folds one chunk of IQ samples into the running window.
func (a *Accumulator) Add(iq []complex64) {
	for _, c := range iq {
		r := float64(real(c))
		i := float64(imag(c))
		a.sumSq += r*r + i*i
	}
	a.n += len(iq)
}

// Samples reports how many samples have accumulated since the last
// reset. Zero means nothing has been Added — callers should skip the
// flush rather than report the epsilon floor as a real measurement.
func (a *Accumulator) Samples() int { return a.n }

// MeanDbFSAndReset returns the window's mean power in dBFS and the
// sample count it was computed over, then resets the accumulator for
// the next window. With no samples it reports the clamped floor and a
// count of 0.
func (a *Accumulator) MeanDbFSAndReset() (dbfs float64, samples int) {
	if a.n == 0 {
		return dbFS(0), 0
	}
	dbfs = dbFS(a.sumSq / float64(a.n))
	samples = a.n
	a.sumSq = 0
	a.n = 0
	return dbfs, samples
}

func dbFS(meanPower float64) float64 {
	if meanPower < powerEpsilon {
		meanPower = powerEpsilon
	}
	return 10 * math.Log10(meanPower)
}
