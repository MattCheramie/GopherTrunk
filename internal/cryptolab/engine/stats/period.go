package stats

import (
	"math"
	"sort"
)

// Autocorrelation returns, for each lag 1..maxLag, the coincidence rate: the
// fraction of positions i where data[i] == data[i+lag]. For a uniform random
// byte stream this sits near 1/256 ≈ 0.0039 at every lag; a repeating-key
// cipher, a periodic scrambler, or a reused keystream produces sharp peaks at
// multiples of its period. This is the byte-level analogue of CrypTool's
// autocorrelation view, and the prerequisite measurement before choosing a
// repeating-XOR key length or a rolling-descramble frame size.
func Autocorrelation(data []byte, maxLag int) []float64 {
	n := len(data)
	if maxLag < 1 || n < 2 {
		return nil
	}
	if maxLag > n-1 {
		maxLag = n - 1
	}
	out := make([]float64, maxLag+1) // index by lag; out[0] unused
	for lag := 1; lag <= maxLag; lag++ {
		matches := 0
		count := n - lag
		for i := 0; i < count; i++ {
			if data[i] == data[i+lag] {
				matches++
			}
		}
		out[lag] = float64(matches) / float64(count)
	}
	return out
}

// PeriodCandidate is one lag whose coincidence rate stands out, the likely
// period (or a multiple of it) of a repeating structure.
type PeriodCandidate struct {
	Lag         int
	Coincidence float64
	// ZScore is how many baseline standard deviations the coincidence sits
	// above the mean coincidence across all lags — a scale-free strength.
	ZScore float64
}

// DetectPeriod ranks the lags of Autocorrelation by how far their coincidence
// rate exceeds the across-lag baseline, returning the strongest candidates
// first. An empty result means no lag stood out (no detectable period within
// maxLag). topN <= 0 returns all peaks above the threshold.
func DetectPeriod(data []byte, maxLag, topN int) []PeriodCandidate {
	ac := Autocorrelation(data, maxLag)
	if len(ac) == 0 {
		return nil
	}
	// Baseline mean and standard deviation over lags 1..maxLag.
	var sum, sumSq float64
	k := 0
	for lag := 1; lag < len(ac); lag++ {
		sum += ac[lag]
		sumSq += ac[lag] * ac[lag]
		k++
	}
	mean := sum / float64(k)
	variance := sumSq/float64(k) - mean*mean
	if variance < 0 {
		variance = 0
	}
	std := math.Sqrt(variance)
	var out []PeriodCandidate
	for lag := 1; lag < len(ac); lag++ {
		z := 0.0
		if std > 0 {
			z = (ac[lag] - mean) / std
		}
		// Keep lags that are a real, positive excursion above baseline.
		if z >= 3 && ac[lag] > mean {
			out = append(out, PeriodCandidate{Lag: lag, Coincidence: ac[lag], ZScore: z})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ZScore > out[j].ZScore })
	if topN > 0 && len(out) > topN {
		out = out[:topN]
	}
	return out
}

// NGram is one byte n-gram and how often it occurs.
type NGram struct {
	Bytes []byte
	Count int
}

// TopNGrams returns the most frequent length-n byte sequences, highest count
// first, capped at topN. It is the histogram CrypTool exposes for spotting
// repeated blocks (ECB-style structure, repeated headers, padding).
func TopNGrams(data []byte, n, topN int) []NGram {
	if n < 1 || len(data) < n {
		return nil
	}
	counts := make(map[string]int)
	for i := 0; i+n <= len(data); i++ {
		counts[string(data[i:i+n])]++
	}
	out := make([]NGram, 0, len(counts))
	for k, c := range counts {
		out = append(out, NGram{Bytes: []byte(k), Count: c})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return string(out[i].Bytes) < string(out[j].Bytes)
	})
	if topN > 0 && len(out) > topN {
		out = out[:topN]
	}
	return out
}
