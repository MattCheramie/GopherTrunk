package loudness

import "math"

// absGateLUFS is the BS.1770 absolute gating threshold: blocks quieter
// than this contribute neither to the measurement nor to the relative
// gate's reference level.
const absGateLUFS = -70.0

// relGateLU is the BS.1770 relative gating offset below the (absolutely
// gated) mean loudness.
const relGateLU = 10.0

// IntegratedLUFS measures the integrated (gated) loudness of mono PCM in
// [-1, 1] at sampleRate, in LUFS, following ITU-R BS.1770-4.
//
// ok is false when the signal is shorter than one 400 ms gating block or
// when every block falls below the absolute gate (i.e. silence). Callers
// should skip normalization in that case rather than treat the returned
// value as meaningful.
func IntegratedLUFS(samples []float64, sampleRate int) (lufs float64, ok bool) {
	if sampleRate <= 0 {
		return 0, false
	}
	blockLen := (sampleRate * 4) / 10 // 400 ms
	hop := sampleRate / 10            // 100 ms (75% overlap)
	if blockLen <= 0 || hop <= 0 || len(samples) < blockLen {
		return 0, false
	}

	// K-weight a private copy so the caller's slice is untouched.
	work := make([]float64, len(samples))
	copy(work, samples)
	shelf, hpf := kWeighting(sampleRate)
	shelf.process(work)
	hpf.process(work)

	// Per-block mean square ("z"), with the matching block loudness used
	// only for gating decisions.
	var blockZ []float64
	for start := 0; start+blockLen <= len(work); start += hop {
		var sum float64
		for _, x := range work[start : start+blockLen] {
			sum += x * x
		}
		z := sum / float64(blockLen)
		blockZ = append(blockZ, z)
	}
	if len(blockZ) == 0 {
		return 0, false
	}

	// Absolute gate.
	var absSum float64
	var absN int
	for _, z := range blockZ {
		if blockLoudness(z) >= absGateLUFS {
			absSum += z
			absN++
		}
	}
	if absN == 0 {
		return 0, false
	}
	relThreshold := blockLoudness(absSum/float64(absN)) - relGateLU

	// Relative gate (applied to the absolutely-gated set).
	var relSum float64
	var relN int
	for _, z := range blockZ {
		l := blockLoudness(z)
		if l >= absGateLUFS && l >= relThreshold {
			relSum += z
			relN++
		}
	}
	if relN == 0 {
		return 0, false
	}
	return blockLoudness(relSum / float64(relN)), true
}

// blockLoudness converts a mean-square value to loudness in LUFS for a
// single mono channel (channel weight 1.0).
func blockLoudness(z float64) float64 {
	if z <= 0 {
		return math.Inf(-1)
	}
	return -0.691 + 10*math.Log10(z)
}
