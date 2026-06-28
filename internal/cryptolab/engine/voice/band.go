package voice

import (
	"math"
	"math/cmplx"
)

// This file adds frequency-domain descramblers beyond the full-band
// SpectralInvert in voice.go: split-band inversion (invert sub-bands
// independently) and rolling inversion (a per-frame schedule), plus a
// best-effort detector for which frames are inverted. All operate offline on
// a whole clip via an internal radix-2 FFT; at power-of-two lengths they are
// exact involutions (run twice to undo), and approximately so otherwise.

func nextPow2(n int) int {
	m := 1
	for m < n {
		m <<= 1
	}
	return m
}

// fft does an in-place iterative radix-2 transform; inverse normalizes by n.
func fft(x []complex128, inverse bool) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		ang := 2 * math.Pi / float64(length)
		if !inverse {
			ang = -ang
		}
		wlen := cmplx.Exp(complex(0, ang))
		for i := 0; i < n; i += length {
			w := complex(1, 0)
			half := length >> 1
			for k := 0; k < half; k++ {
				u := x[i+k]
				v := x[i+k+half] * w
				x[i+k] = u + v
				x[i+k+half] = u - v
				w *= wlen
			}
		}
	}
	if inverse {
		nc := complex(float64(n), 0)
		for i := range x {
			x[i] /= nc
		}
	}
}

// invertBinBands mirrors (spectrally inverts) each given positive-half bin
// range within itself and returns the real result. Bands are integer bin
// ranges [lo,hi] over 1..half-1 and MUST be disjoint, which makes the
// operation an exact involution at power-of-two lengths. DC and the Nyquist
// bin are never moved, so the output stays real.
func invertBinBands(samples []float64, bands [][2]int) []float64 {
	n := len(samples)
	if n == 0 {
		return nil
	}
	m := nextPow2(n)
	x := make([]complex128, m)
	for i, s := range samples {
		x[i] = complex(s, 0)
	}
	fft(x, false)

	half := m / 2
	y := make([]complex128, m)
	copy(y, x)
	for _, b := range bands {
		lo, hi := b[0], b[1]
		for k := lo; k <= hi; k++ {
			y[k] = x[lo+hi-k] // mirror within the band
		}
	}
	// Rebuild the negative half from conjugate symmetry so the IFFT is real.
	for k := 1; k < half; k++ {
		y[m-k] = cmplx.Conj(y[k])
	}
	fft(y, true)
	out := make([]float64, n)
	for i := range out {
		out[i] = real(y[i])
	}
	return out
}

func clampBin(b, half int) int {
	if b < 1 {
		return 1
	}
	if b > half-1 {
		return half - 1
	}
	return b
}

// BandInvert spectrally inverts a single normalized band [loFrac,hiFrac] of
// Nyquist. BandInvert(s,0,1) is full-band inversion (cf. SpectralInvert).
func BandInvert(samples []float64, loFrac, hiFrac float64) []float64 {
	half := nextPow2(len(samples)) / 2
	if half < 2 {
		return append([]float64(nil), samples...)
	}
	lo := clampBin(int(math.Round(loFrac*float64(half))), half)
	hi := clampBin(int(math.Round(hiFrac*float64(half))), half)
	if lo > hi {
		return append([]float64(nil), samples...)
	}
	return invertBinBands(samples, [][2]int{{lo, hi}})
}

// SplitBandInvert inverts the low sub-band [0,split] and the high sub-band
// [split,1] independently — the classic split-band (variable-split) inverter.
// The two sub-bands are disjoint by construction, so it is its own inverse.
func SplitBandInvert(samples []float64, split float64) []float64 {
	half := nextPow2(len(samples)) / 2
	if half < 2 {
		return append([]float64(nil), samples...)
	}
	sb := int(math.Round(split * float64(half)))
	if sb < 1 {
		sb = 1
	}
	if sb > half-1 {
		sb = half - 1
	}
	var bands [][2]int
	bands = append(bands, [2]int{1, sb}) // low sub-band [1, sb]
	if sb+1 <= half-1 {
		bands = append(bands, [2]int{sb + 1, half - 1}) // high sub-band, disjoint
	}
	return invertBinBands(samples, bands)
}

// RollingInvert applies a per-frame split-band inversion: frame i is inverted
// with splits[i % len(splits)]. Re-applying the same schedule descrambles a
// rolling/hopping inverter whose schedule is known. A split of 1.0 is a
// full-band inversion of that frame.
func RollingInvert(samples []float64, frameLen int, splits []float64) []float64 {
	if frameLen <= 0 || len(splits) == 0 {
		return append([]float64(nil), samples...)
	}
	out := make([]float64, len(samples))
	copy(out, samples)
	for f, start := 0, 0; start < len(samples); f, start = f+1, start+frameLen {
		end := start + frameLen
		if end > len(samples) {
			end = len(samples)
		}
		inv := SplitBandInvert(out[start:end], splits[f%len(splits)])
		copy(out[start:end], inv)
	}
	return out
}

// DetectInversion is a best-effort per-frame inversion detector for full-band
// rolling inverters: voiced speech concentrates energy in the lower half of
// the band, so a frame with more high-band than low-band energy is flagged as
// inverted. It is a heuristic sync aid, not exact recovery.
func DetectInversion(samples []float64, frameLen int) []bool {
	if frameLen <= 0 {
		return nil
	}
	var flags []bool
	for start := 0; start < len(samples); start += frameLen {
		end := start + frameLen
		if end > len(samples) {
			end = len(samples)
		}
		frame := samples[start:end]
		lo := BandEnergy(frame, 0, 0.5)
		hi := BandEnergy(frame, 0.5, 1.0)
		flags = append(flags, hi > lo)
	}
	return flags
}

// RollingAuto detects full-band-inverted frames and undoes them, returning the
// descrambled signal and the per-frame flags it acted on.
func RollingAuto(samples []float64, frameLen int) ([]float64, []bool) {
	flags := DetectInversion(samples, frameLen)
	out := make([]float64, len(samples))
	copy(out, samples)
	for i, inv := range flags {
		if !inv {
			continue
		}
		start := i * frameLen
		end := start + frameLen
		if end > len(samples) {
			end = len(samples)
		}
		copy(out[start:end], BandInvert(out[start:end], 0, 1))
	}
	return out, flags
}
