// Package filter implements the FIR/CIC/halfband primitives used by the DSP
// pipeline. All filters operate on complex64 IQ samples or real float32
// signals; coefficients are stored as float32.
package filter

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// FIR is a linear-phase finite-impulse-response filter for complex64 IQ.
// It maintains an internal sample history so consecutive Process calls
// produce a continuous output stream.
type FIR struct {
	taps    []float32
	hist    []complex64
	histPos int
}

func NewFIR(taps []float32) *FIR {
	if len(taps) == 0 {
		panic("filter: NewFIR requires at least one tap")
	}
	cp := make([]float32, len(taps))
	copy(cp, taps)
	return &FIR{taps: cp, hist: make([]complex64, len(taps))}
}

// Reset clears the internal history.
func (f *FIR) Reset() {
	for i := range f.hist {
		f.hist[i] = 0
	}
	f.histPos = 0
}

// Process consumes one input slice and returns an output slice of the same
// length. dst is reused if it has enough capacity.
func (f *FIR) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	N := len(f.taps)
	for i, x := range src {
		f.hist[f.histPos] = x
		f.histPos++
		if f.histPos == N {
			f.histPos = 0
		}
		// Convolve: output is sum_{k} h[k] * hist[(histPos - 1 - k) mod N].
		var accI, accQ float32
		idx := f.histPos - 1
		if idx < 0 {
			idx = N - 1
		}
		for k := 0; k < N; k++ {
			s := f.hist[idx]
			h := f.taps[k]
			accI += h * real(s)
			accQ += h * imag(s)
			idx--
			if idx < 0 {
				idx = N - 1
			}
		}
		dst[i] = complex(accI, accQ)
	}
	return dst
}

// LowpassKaiser designs a length-N (odd) lowpass FIR with cutoff fc
// (normalized; 0.5 = Nyquist) using a Kaiser window with shape beta.
func LowpassKaiser(n int, fc, beta float64) []float32 {
	if n%2 == 0 {
		n++ // force odd for symmetric linear-phase filter
	}
	w := window.Kaiser(n, beta)
	taps := make([]float32, n)
	mid := (n - 1) / 2
	for i := 0; i < n; i++ {
		k := i - mid
		var x float64
		if k == 0 {
			x = 2 * fc
		} else {
			x = math.Sin(2*math.Pi*fc*float64(k)) / (math.Pi * float64(k))
		}
		taps[i] = float32(x * w[i])
	}
	// Normalize DC gain to 1.
	var sum float64
	for _, t := range taps {
		sum += float64(t)
	}
	if sum != 0 {
		s := float32(1.0 / sum)
		for i := range taps {
			taps[i] *= s
		}
	}
	return taps
}

// RootRaisedCosine returns the impulse response of a root-raised-cosine
// pulse-shaping filter. sps = samples per symbol, nSymbols = total span,
// alpha = roll-off (0 < alpha ≤ 1). The filter is normalized to unit energy.
func RootRaisedCosine(sps, nSymbols int, alpha float64) []float32 {
	N := sps*nSymbols + 1 // odd length
	taps := make([]float32, N)
	mid := (N - 1) / 2
	T := 1.0 // symbol period
	Ts := T / float64(sps)
	for i := 0; i < N; i++ {
		t := float64(i-mid) * Ts
		taps[i] = float32(rrcSample(t, T, alpha))
	}
	// Unit-energy normalization.
	var energy float64
	for _, c := range taps {
		energy += float64(c) * float64(c)
	}
	if energy > 0 {
		k := float32(1.0 / math.Sqrt(energy))
		for i := range taps {
			taps[i] *= k
		}
	}
	return taps
}

func rrcSample(t, T, alpha float64) float64 {
	if t == 0 {
		return (1.0/T)*(1.0-alpha) + (4*alpha)/(math.Pi*T)
	}
	denomZero := math.Abs(t) - T/(4*alpha)
	if alpha != 0 && math.Abs(denomZero) < 1e-12 {
		return (alpha / (T * math.Sqrt2)) *
			((1+2/math.Pi)*math.Sin(math.Pi/(4*alpha)) +
				(1-2/math.Pi)*math.Cos(math.Pi/(4*alpha)))
	}
	num := math.Sin(math.Pi*t*(1-alpha)/T) +
		4*alpha*t/T*math.Cos(math.Pi*t*(1+alpha)/T)
	den := math.Pi * t / T * (1 - math.Pow(4*alpha*t/T, 2))
	return num / (T * den)
}
