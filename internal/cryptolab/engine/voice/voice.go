// Package voice provides analog voice-privacy descramblers — the keyless /
// weak-key spectral schemes still found on RF voice channels. The core
// operation is full-band spectral inversion (the classic "frequency
// inverter" scrambler), which is its own inverse, so the same call both
// scrambles and descrambles.
package voice

import "math"

// SpectralInvert inverts the spectrum of a real signal by multiplying every
// other sample by -1 (mixing with a Nyquist-rate carrier). This maps
// frequency f to fs/2 − f, which is exactly the full-band "frequency
// inverter" analog scramble. Applying it twice returns the original signal.
func SpectralInvert(samples []float64) []float64 {
	out := make([]float64, len(samples))
	for i, s := range samples {
		if i&1 == 1 {
			out[i] = -s
		} else {
			out[i] = s
		}
	}
	return out
}

// SpectralInvertInt16 is SpectralInvert over 16-bit PCM, clamping on the
// rare overflow at full scale.
func SpectralInvertInt16(samples []int16) []int16 {
	out := make([]int16, len(samples))
	for i, s := range samples {
		v := int32(s)
		if i&1 == 1 {
			v = -v
		}
		if v > math.MaxInt16 {
			v = math.MaxInt16
		} else if v < math.MinInt16 {
			v = math.MinInt16
		}
		out[i] = int16(v)
	}
	return out
}

// BandEnergy returns the energy of a real signal in the normalized
// frequency band [loFrac, hiFrac] of Nyquist (each in 0..1), via a direct
// DFT-bin sum. Useful for confirming an inversion moved energy from the low
// band to the high band (and for tests).
func BandEnergy(samples []float64, loFrac, hiFrac float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	loBin := int(loFrac * float64(n) / 2)
	hiBin := int(hiFrac * float64(n) / 2)
	if hiBin > n/2 {
		hiBin = n / 2
	}
	var energy float64
	for k := loBin; k <= hiBin; k++ {
		var re, im float64
		for t, s := range samples {
			ang := -2 * math.Pi * float64(k) * float64(t) / float64(n)
			re += s * math.Cos(ang)
			im += s * math.Sin(ang)
		}
		energy += re*re + im*im
	}
	return energy
}
