package loudness

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// truePeakOversample is the upsampling factor used to estimate
// inter-sample peaks. BS.1770-4 Annex 2 specifies at least 4x for
// content up to 48 kHz.
const truePeakOversample = 4

// TruePeak returns the estimated true (inter-sample) peak magnitude of
// mono PCM in [-1, 1], by 4x polyphase oversampling. The result is a
// linear magnitude (1.0 == full scale, i.e. 0 dBTP); convert to dBTP via
// 20*log10(peak). It is always >= the raw sample peak.
func TruePeak(samples []float64, sampleRate int) float64 {
	peak := 0.0
	for _, x := range samples {
		if a := math.Abs(x); a > peak {
			peak = a
		}
	}
	if len(samples) < 2 {
		return peak
	}

	// Interpolation lowpass: cutoff at the original Nyquist (0.5/L of the
	// oversampled band), scaled to gain L so amplitude is preserved across
	// the inserted phases.
	const L = truePeakOversample
	taps := filter.LowpassKaiser(L*12, 0.5/L, 8.0)
	h := make([]float64, len(taps))
	for i, t := range taps {
		h[i] = float64(t) * L
	}
	n := len(h)

	for i := range samples {
		// Phase 0 reproduces the (delayed) input and is already covered by
		// the sample-peak scan above; phases 1..L-1 are the inter-sample
		// estimates.
		for p := 1; p < L; p++ {
			var acc float64
			for j := p; j < n; j += L {
				k := (j - p) / L
				si := i - k
				if si < 0 {
					break
				}
				acc += h[j] * samples[si]
			}
			if a := math.Abs(acc); a > peak {
				peak = a
			}
		}
	}
	return peak
}
