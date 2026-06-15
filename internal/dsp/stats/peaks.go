package stats

import "sort"

// PeakOpts constrains FindPeaks, mirroring a minimal subset of
// scipy.signal.find_peaks. A zero value finds every interior local maximum.
type PeakOpts struct {
	// Height is an absolute minimum: peaks with y < Height are dropped. 0
	// disables the absolute floor.
	Height float64
	// RelHeight keeps only peaks at least RelHeight·max(peak) tall (a relative
	// floor, e.g. 0.01 ≈ 20 dB down). 0 disables it. When both Height and
	// RelHeight are set the stricter threshold wins.
	RelHeight float64
	// MinDistance is the minimum index spacing between kept peaks. When peaks
	// are closer, the taller one wins (greedy, strongest first). 0 disables
	// spacing suppression.
	MinDistance int
	// IncludeEnds treats the first and last samples as candidate maxima (a peak
	// at a band edge still qualifies). When false they are never peaks, matching
	// scipy's default.
	IncludeEnds bool
}

// FindPeaks returns the indices of the local maxima in y that satisfy opts,
// sorted by descending height (strongest first). A sample is a local maximum
// when it is strictly greater than its left neighbour and greater than or equal
// to its right neighbour (the ">=" right rule keeps the left-most index of a
// flat top, matching the carrier-search convention this generalises). Returns
// nil for empty input or when no peak clears the thresholds.
func FindPeaks(y []float64, opts PeakOpts) []int {
	n := len(y)
	if n == 0 {
		return nil
	}
	var idx []int
	for i := 0; i < n; i++ {
		if (i == 0 || i == n-1) && !opts.IncludeEnds {
			continue
		}
		left := i == 0 || y[i] > y[i-1]
		right := i == n-1 || y[i] >= y[i+1]
		if left && right {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return nil
	}

	// Threshold: the stricter of the absolute and relative floors.
	threshold := opts.Height
	if opts.RelHeight > 0 {
		var maxPeak float64
		for j, i := range idx {
			if j == 0 || y[i] > maxPeak {
				maxPeak = y[i]
			}
		}
		if rt := opts.RelHeight * maxPeak; rt > threshold {
			threshold = rt
		}
	}
	kept := make([]int, 0, len(idx))
	for _, i := range idx {
		if y[i] >= threshold {
			kept = append(kept, i)
		}
	}
	sort.SliceStable(kept, func(a, b int) bool { return y[kept[a]] > y[kept[b]] })

	if opts.MinDistance > 0 {
		var out []int
		for _, i := range kept {
			ok := true
			for _, j := range out {
				d := i - j
				if d < 0 {
					d = -d
				}
				if d < opts.MinDistance {
					ok = false
					break
				}
			}
			if ok {
				out = append(out, i)
			}
		}
		kept = out
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}
