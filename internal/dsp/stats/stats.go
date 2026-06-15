// Package stats provides small, dependency-free numerical helpers — the
// descriptive statistics, histogram, mode, argmax, normalized
// cross-correlation and peak-finding that the signal-analysis paths reach for.
// They are the native Go equivalents of the handful of numpy/scipy routines
// the demod and sync-landscape analyses lean on (numpy.mean/std/histogram,
// numpy.argmax, scipy.signal.find_peaks, scipy.signal.correlate), so those
// operations live in one tested place instead of being re-derived inline.
//
// Everything is float64/int and stdlib-only, matching the sibling leaf
// packages (window, fft) — no gonum, no CGO.
package stats

import (
	"math"
	"sort"
)

// Mean returns the arithmetic mean of x (numpy.mean), or 0 for empty input.
func Mean(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	var sum float64
	for _, v := range x {
		sum += v
	}
	return sum / float64(len(x))
}

// Variance returns the variance of x. When sample is true it applies Bessel's
// correction (divide by n-1, ddof=1, numpy.var(ddof=1)); otherwise it is the
// population variance (divide by n). Returns 0 when there are too few points
// for the requested estimator.
func Variance(x []float64, sample bool) float64 {
	n := len(x)
	if n == 0 || (sample && n < 2) {
		return 0
	}
	m := Mean(x)
	var ss float64
	for _, v := range x {
		d := v - m
		ss += d * d
	}
	denom := float64(n)
	if sample {
		denom = float64(n - 1)
	}
	return ss / denom
}

// StdDev returns the standard deviation of x (sqrt of Variance), honouring the
// same sample/population choice as Variance.
func StdDev(x []float64, sample bool) float64 {
	return math.Sqrt(Variance(x, sample))
}

// Min returns the smallest value in x and its index, or (0, -1) for empty input.
func Min(x []float64) (val float64, idx int) {
	if len(x) == 0 {
		return 0, -1
	}
	val, idx = x[0], 0
	for i, v := range x {
		if v < val {
			val, idx = v, i
		}
	}
	return val, idx
}

// Max returns the largest value in x and its index, or (0, -1) for empty input.
func Max(x []float64) (val float64, idx int) {
	if len(x) == 0 {
		return 0, -1
	}
	val, idx = x[0], 0
	for i, v := range x {
		if v > val {
			val, idx = v, i
		}
	}
	return val, idx
}

// ArgMax returns the index of the largest value in x (numpy.argmax), or -1 for
// empty input. Ties resolve to the first (lowest) index.
func ArgMax(x []float64) int {
	_, idx := Max(x)
	return idx
}

// ArgMin returns the index of the smallest value in x (numpy.argmin), or -1 for
// empty input. Ties resolve to the first (lowest) index.
func ArgMin(x []float64) int {
	_, idx := Min(x)
	return idx
}

// ArgMaxInt returns the index of the largest value in an integer slice, or -1
// for empty input. Ties resolve to the first (lowest) index. It generalises the
// "pick the winner by hit count" selection in the sync-landscape analysis.
func ArgMaxInt(x []int) int {
	if len(x) == 0 {
		return -1
	}
	best := 0
	for i, v := range x {
		if v > x[best] {
			best = i
		}
	}
	return best
}

// Histogram bins x into nbins equal-width bins spanning [min(x), max(x)]
// (numpy.histogram). It returns the per-bin counts (length nbins) and the
// nbins+1 bin edges. Values equal to the right-most edge fall in the last bin,
// matching numpy. Returns (nil, nil) when nbins < 1 or x is empty.
func Histogram(x []float64, nbins int) (counts []int, edges []float64) {
	if nbins < 1 || len(x) == 0 {
		return nil, nil
	}
	lo, _ := Min(x)
	hi, _ := Max(x)
	edges = make([]float64, nbins+1)
	counts = make([]int, nbins)
	if lo == hi {
		// Degenerate range: numpy widens it to [lo-0.5, hi+0.5].
		lo, hi = lo-0.5, hi+0.5
	}
	width := (hi - lo) / float64(nbins)
	for i := 0; i <= nbins; i++ {
		edges[i] = lo + width*float64(i)
	}
	for _, v := range x {
		b := int((v - lo) / width)
		if b == nbins { // right edge folds into the last bin
			b = nbins - 1
		}
		if b >= 0 && b < nbins {
			counts[b]++
		}
	}
	return counts, edges
}

// ModeInt returns the most frequent value in x. Ties resolve toward the
// smaller value, matching the sync-landscape modal-spacing convention it
// generalises. Returns 0 for empty input.
func ModeInt(x []int) int {
	if len(x) == 0 {
		return 0
	}
	s := append([]int(nil), x...)
	sort.Ints(s)
	bestVal, bestCount := s[0], 1
	curVal, curCount := s[0], 1
	for i := 1; i < len(s); i++ {
		if s[i] == curVal {
			curCount++
		} else {
			if curCount > bestCount {
				bestVal, bestCount = curVal, curCount
			}
			curVal, curCount = s[i], 1
		}
	}
	if curCount > bestCount {
		bestVal = curVal
	}
	return bestVal
}
