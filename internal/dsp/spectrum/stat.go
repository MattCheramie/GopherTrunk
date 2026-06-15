package spectrum

import (
	"math"
	"sort"
)

// Smooth returns bins convolved with a centered boxcar of half-width w (total
// window 2w+1), the discrete uniform filter used to tame per-bin noise before
// peak picking (np.convolve with a normalized rectangular kernel). w<=0 returns
// a copy. Windows are clipped at the array edges so the output keeps the input
// length without wrap-around.
func Smooth(bins []float32, w int) []float32 {
	out := make([]float32, len(bins))
	if w <= 0 {
		copy(out, bins)
		return out
	}
	for i := range bins {
		lo, hi := i-w, i+w
		if lo < 0 {
			lo = 0
		}
		if hi >= len(bins) {
			hi = len(bins) - 1
		}
		var s float32
		for j := lo; j <= hi; j++ {
			s += bins[j]
		}
		out[i] = s / float32(hi-lo+1)
	}
	return out
}

// Percentile returns the p-quantile (p in 0..1) of bins via copy+sort, the
// robust noise-floor estimate (np.percentile with nearest-rank interpolation).
// An empty slice returns 0; p is clamped into range.
func Percentile(bins []float32, p float64) float32 {
	if len(bins) == 0 {
		return 0
	}
	cp := make([]float32, len(bins))
	copy(cp, bins)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(p * float64(len(cp)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

// WeightedCentroidHz returns the power-weighted mean frequency of the carrier
// hump around peakBin: the contiguous band of bins within ±halfWidth, each
// weighted by its linear power (dB → power via DBToLinearPower). This finds the
// true center of a wide carrier whose single strongest bin is noisy
// (np.average of the bin frequencies weighted by power). baseHz is the absolute
// frequency of bin 0 and binHz the bin spacing. Bins at or below floorDb are
// excluded; when no bin clears the floor the peak bin's frequency is returned.
func WeightedCentroidHz(binsDb []float32, peakBin, halfWidth int, floorDb float32, baseHz, binHz float64) float64 {
	lo, hi := peakBin-halfWidth, peakBin+halfWidth
	if lo < 0 {
		lo = 0
	}
	if hi >= len(binsDb) {
		hi = len(binsDb) - 1
	}
	var wsum, psum float64
	for j := lo; j <= hi; j++ {
		if binsDb[j] <= floorDb {
			continue
		}
		p := dbToLinearPower(float64(binsDb[j]))
		freq := baseHz + float64(j)*binHz
		wsum += p * freq
		psum += p
	}
	if psum == 0 {
		return baseHz + float64(peakBin)*binHz
	}
	return wsum / psum
}

// dbToLinearPower mirrors dsp.DBToLinearPower; spectrum sits below dsp's
// top-level package so the conversion is duplicated here to avoid an import
// cycle. Keep the two in sync.
func dbToLinearPower(db float64) float64 { return math.Pow(10, db/10) }
