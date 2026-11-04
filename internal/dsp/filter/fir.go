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
	taps []float32
	// rtaps is taps reversed (rtaps[j] = taps[N-1-j]) so the dot product runs
	// over a contiguous, chronologically ordered history window.
	rtaps []float32
	// hist is a 2N mirrored history: every sample is written at histPos and
	// histPos+N, so hist[histPos:histPos+N] is always the last N samples in
	// arrival order with no wrap. That turns the per-tap modulo/branch of a
	// plain ring buffer into a straight two-slice dot product — the filter
	// is the largest single cost of a narrowband receiver (≈50% of a TETRA
	// wideband tap's CPU once the AACH decoder stopped re-encoding its
	// codebook), so the inner loop matters.
	hist    []complex64
	histPos int
	// folded selects the symmetric-tap path (EnableFolded).
	folded bool
}

// EnableFolded switches the filter to a folded dot product for symmetric
// (linear-phase) taps: y = Σ_{j<N/2} h[j]·(w[j] + w[N-1-j]) (+ h[mid]·w[mid]
// for odd N), which halves the multiplies. It reports false and leaves the
// filter unchanged when the taps are not bit-exactly symmetric, so it can
// never alter the response, only the float32 rounding of the sum. The
// default path stays bit-exact with the original ring-buffer loop
// (TestFIRMirroredWindowIsBitExact); callers opt in where the speed matters
// (the TETRA receiver's channel and matched filters, #1321).
func (f *FIR) EnableFolded() bool {
	n := len(f.taps)
	for k := 0; k < n/2; k++ {
		if f.taps[k] != f.taps[n-1-k] {
			return false
		}
	}
	f.folded = true
	return true
}

// Folded reports whether the folded symmetric path is active.
func (f *FIR) Folded() bool { return f.folded }

func NewFIR(taps []float32) *FIR {
	if len(taps) == 0 {
		panic("filter: NewFIR requires at least one tap")
	}
	cp := make([]float32, len(taps))
	copy(cp, taps)
	rt := make([]float32, len(taps))
	for j := range rt {
		rt[j] = taps[len(taps)-1-j]
	}
	return &FIR{taps: cp, rtaps: rt, hist: make([]complex64, 2*len(taps))}
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
	rt := f.rtaps[:N]
	if f.folded {
		f.processFolded(dst, src)
		return dst
	}
	for i, x := range src {
		f.hist[f.histPos] = x
		f.hist[f.histPos+N] = x
		f.histPos++
		if f.histPos == N {
			f.histPos = 0
		}
		// w holds the last N samples oldest-first; output is
		// sum_k h[k] * w[N-1-k] = sum_j rtaps[j] * w[j]. Accumulate newest
		// tap first (j descending) so the float32 summation order matches
		// the original ring-buffer loop exactly.
		w := f.hist[f.histPos : f.histPos+N]
		w = w[:len(rt)]
		var accI, accQ float32
		for j := len(rt) - 1; j >= 0; j-- {
			s := w[j]
			h := rt[j]
			accI += h * real(s)
			accQ += h * imag(s)
		}
		dst[i] = complex(accI, accQ)
	}
	return dst
}

// processFolded is Process for symmetric taps: each tap pair shares one
// multiply. The window and history handling are Process's.
func (f *FIR) processFolded(dst, src []complex64) {
	N := len(f.taps)
	half := N / 2
	h := f.rtaps[:half]
	var hMid float32
	odd := N%2 == 1
	if odd {
		hMid = f.rtaps[half]
	}
	for i, x := range src {
		f.hist[f.histPos] = x
		f.hist[f.histPos+N] = x
		f.histPos++
		if f.histPos == N {
			f.histPos = 0
		}
		w := f.hist[f.histPos : f.histPos+N]
		lo := w[:half]
		hi := w[N-half:]
		// Four independent accumulator pairs: a single running sum is one
		// dependent chain of float adds, so the loop would be bound by add
		// latency rather than by the (already halved) multiply count.
		var i0, i1, i2, i3, q0, q1, q2, q3 float32
		if odd {
			m := w[half]
			i0 = hMid * real(m)
			q0 = hMid * imag(m)
		}
		j := 0
		for ; j+4 <= half; j += 4 {
			// One bounds check per group of four, not one per element.
			cv := (*[4]float32)(h[j : j+4])
			av := (*[4]complex64)(lo[j : j+4])
			bv := (*[4]complex64)(hi[half-4-j : half-j])
			c0, c1, c2, c3 := cv[0], cv[1], cv[2], cv[3]
			a0, a1, a2, a3 := av[0], av[1], av[2], av[3]
			b0, b1, b2, b3 := bv[3], bv[2], bv[1], bv[0]
			i0 += c0 * (real(a0) + real(b0))
			q0 += c0 * (imag(a0) + imag(b0))
			i1 += c1 * (real(a1) + real(b1))
			q1 += c1 * (imag(a1) + imag(b1))
			i2 += c2 * (real(a2) + real(b2))
			q2 += c2 * (imag(a2) + imag(b2))
			i3 += c3 * (real(a3) + real(b3))
			q3 += c3 * (imag(a3) + imag(b3))
		}
		for ; j < half; j++ {
			c, a, b := h[j], lo[j], hi[half-1-j]
			i0 += c * (real(a) + real(b))
			q0 += c * (imag(a) + imag(b))
		}
		dst[i] = complex((i0+i1)+(i2+i3), (q0+q1)+(q2+q3))
	}
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

// Gaussian returns the impulse response of a Gaussian pulse-shaping
// / matched filter parameterised by samples-per-symbol, span (in
// symbols), and the BT product (3 dB bandwidth × symbol period).
// BT = 0.3 is the standard GFSK premod for EDACS / GE-Marc; BT = 0.5
// is typical for Bluetooth. Taps are normalised to unit DC gain so a
// sustained NRZ level passes through unchanged at the symbol centre.
func Gaussian(sps, nSymbols int, bt float64) []float32 {
	if sps <= 0 || nSymbols <= 0 || bt <= 0 {
		panic("filter: Gaussian requires positive sps, nSymbols, bt")
	}
	N := sps*nSymbols + 1 // odd length for symmetric linear-phase filter
	taps := make([]float32, N)
	mid := (N - 1) / 2
	// σ (in samples). From the standard Gaussian impulse + 3 dB
	// bandwidth relation B = sqrt(ln 2) / (2π σ) with B·T = BT and
	// T = sps samples: σ = sps · sqrt(ln 2) / (2π · BT).
	sigma := float64(sps) * math.Sqrt(math.Ln2) / (2 * math.Pi * bt)
	for i := 0; i < N; i++ {
		x := float64(i - mid)
		taps[i] = float32(math.Exp(-(x * x) / (2 * sigma * sigma)))
	}
	var sum float64
	for _, t := range taps {
		sum += float64(t)
	}
	if sum > 0 {
		k := float32(1.0 / sum)
		for i := range taps {
			taps[i] *= k
		}
	}
	return taps
}
