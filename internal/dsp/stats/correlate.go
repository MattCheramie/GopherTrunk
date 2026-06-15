package stats

import "math"

// Correlate returns the full normalized cross-correlation of a and b
// (numpy.correlate / scipy.signal.correlate, mode="full"), together with the
// lag of its peak. Each output sample is divided by sqrt(Σa²·Σb²), so a
// sequence correlated against itself peaks at exactly 1.0.
//
// The output has length len(a)+len(b)-1. Index k corresponds to lag
// s = k-(len(b)-1): the value is Σ a[i]·b[i-s] over the overlapping region.
// peakLag is the lag s maximizing the correlation — equivalently, the offset
// into a at which b best aligns (so a pattern b embedded in a at index p peaks
// at peakLag == p). Returns (nil, 0) when either input is empty.
func Correlate(a, b []float64) (corr []float64, peakLag int) {
	na, nb := len(a), len(b)
	if na == 0 || nb == 0 {
		return nil, 0
	}
	var ea, eb float64
	for _, v := range a {
		ea += v * v
	}
	for _, v := range b {
		eb += v * v
	}
	norm := math.Sqrt(ea * eb)

	corr = make([]float64, na+nb-1)
	bestIdx, bestVal := 0, math.Inf(-1)
	for k := range corr {
		s := k - (nb - 1)
		// i ranges over indices where both a[i] and b[i-s] exist.
		iLo := 0
		if s > 0 {
			iLo = s
		}
		iHi := na
		if s+nb < iHi {
			iHi = s + nb
		}
		var acc float64
		for i := iLo; i < iHi; i++ {
			acc += a[i] * b[i-s]
		}
		if norm > 0 {
			acc /= norm
		}
		corr[k] = acc
		if acc > bestVal {
			bestVal, bestIdx = acc, k
		}
	}
	return corr, bestIdx - (nb - 1)
}
