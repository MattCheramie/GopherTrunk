package hunt

import (
	"math"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
)

// Peak is a candidate carrier found in a spectrum frame: its absolute
// frequency, its power, and how far it stands above the estimated noise floor.
type Peak struct {
	FreqHz    uint32
	PowerDbFS float32
	SNRDb     float32
}

// PeakOptions tune the carrier detector.
type PeakOptions struct {
	// ThresholdDb is the minimum power above the estimated noise floor for a
	// local maximum to count as a carrier. 0 ⇒ a sensible default (10 dB).
	ThresholdDb float32
	// MinSpacingHz is the minimum separation between reported peaks; when two
	// maxima fall closer than this the stronger one wins. 0 ⇒ 6.25 kHz (the
	// tightest trunking channel step).
	MinSpacingHz uint32
	// GuardBins drops this many bins at each band edge (rolloff) from
	// consideration. 0 ⇒ a default proportional to the FFT size.
	GuardBins int
}

// noiseFloorPercentile is the percentile of bin powers used as the noise-floor
// estimate. The low quartile is robust to strong carriers occupying the upper
// tail, unlike a mean.
const noiseFloorPercentile = 0.25

// DetectPeaks finds candidate carriers in a spectrum frame. It estimates a
// robust noise floor (low-quartile of the bin powers), keeps local maxima that
// stand ThresholdDb above it, drops the DC bin and the band-edge guard bins,
// and enforces a minimum carrier spacing (stronger peak wins). Bin indices are
// mapped to absolute Hz using the frame's center and sample rate. Returned
// peaks are sorted by descending SNR.
func DetectPeaks(frame spectrum.Frame, opts PeakOptions) []Peak {
	n := len(frame.Bins)
	if n < 8 || frame.SampleRate == 0 {
		return nil
	}
	threshold := opts.ThresholdDb
	if threshold <= 0 {
		threshold = 10
	}
	minSpacing := opts.MinSpacingHz
	if minSpacing == 0 {
		minSpacing = 6_250
	}
	guard := opts.GuardBins
	if guard <= 0 {
		guard = n / 64 // ~1.5% of the band at each edge
		if guard < 1 {
			guard = 1
		}
	}

	floor := percentile(frame.Bins, noiseFloorPercentile)
	binHz := float64(frame.SampleRate) / float64(n)
	dcBin := n / 2 // FFT-shifted: DC sits in the middle (see spectrum.frame)

	var peaks []Peak
	for i := guard; i < n-guard; i++ {
		if i == dcBin {
			continue
		}
		v := frame.Bins[i]
		if v-floor < threshold {
			continue
		}
		// Strict local maximum against immediate neighbors.
		if v <= frame.Bins[i-1] || v < frame.Bins[i+1] {
			continue
		}
		peaks = append(peaks, Peak{
			FreqHz:    binToHz(frame, i, binHz),
			PowerDbFS: v,
			SNRDb:     v - floor,
		})
	}

	// Strongest-first, then suppress weaker peaks within MinSpacingHz.
	sort.Slice(peaks, func(a, b int) bool { return peaks[a].SNRDb > peaks[b].SNRDb })
	var kept []Peak
	for _, p := range peaks {
		ok := true
		for _, k := range kept {
			if absDiff(p.FreqHz, k.FreqHz) < minSpacing {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, p)
		}
	}
	return kept
}

// binToHz maps an FFT-shifted bin index to its absolute center frequency.
// Bins[0] = CenterHz - SampleRate/2; step = SampleRate/N.
func binToHz(frame spectrum.Frame, bin int, binHz float64) uint32 {
	base := float64(frame.CenterHz) - float64(frame.SampleRate)/2
	return uint32(math.Round(base + float64(bin)*binHz))
}

// percentile returns the p-quantile (0..1) of bins using a copy+sort. Used for
// the robust noise-floor estimate.
func percentile(bins []float32, p float64) float32 {
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

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
