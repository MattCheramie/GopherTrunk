package dsp

import "math"

// RMS returns the root-mean-square amplitude of a complex baseband buffer:
// sqrt(mean(|x|^2)). For a full-scale complex sinusoid (|x| = 1) this is 1.0,
// so RMS in dBFS is 20·log10(RMS) (0 dBFS at full scale). It is the
// numpy `sqrt(mean(abs(x)**2))` capture-level power measure that no exported
// helper provided. Returns 0 for an empty buffer.
func RMS(iq []complex64) float64 {
	if len(iq) == 0 {
		return 0
	}
	var sum float64
	for _, v := range iq {
		re, im := float64(real(v)), float64(imag(v))
		sum += re*re + im*im
	}
	return math.Sqrt(sum / float64(len(iq)))
}

// Power2dB converts a linear power ratio to decibels (10·log10). Values at or
// below zero clamp to a floor of -300 dB rather than returning -Inf/NaN.
func Power2dB(power float64) float64 {
	if power <= 0 {
		return -300
	}
	return 10 * math.Log10(power)
}

// Mag2dB converts a linear amplitude (magnitude) ratio to decibels (20·log10).
// Use it for an RMS or voltage figure; use Power2dB for a power figure. Values
// at or below zero clamp to -300 dB.
func Mag2dB(mag float64) float64 {
	if mag <= 0 {
		return -300
	}
	return 20 * math.Log10(mag)
}
