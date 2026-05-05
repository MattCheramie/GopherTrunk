// Package window provides standard window functions for FIR design and FFT
// pre-processing. All windows are symmetric and length-N.
package window

import "math"

func Rect(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 1
	}
	return w
}

func Hann(n int) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	for i := 0; i < n; i++ {
		w[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
	}
	return w
}

func Hamming(n int) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	for i := 0; i < n; i++ {
		w[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(n-1))
	}
	return w
}

func Blackman(n int) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	for i := 0; i < n; i++ {
		x := 2 * math.Pi * float64(i) / float64(n-1)
		w[i] = 0.42 - 0.5*math.Cos(x) + 0.08*math.Cos(2*x)
	}
	return w
}

// Kaiser returns a length-N Kaiser window with shape parameter beta. Larger
// beta → narrower main lobe + lower side lobes. Common values: 5 (~-37 dB),
// 8.6 (~-65 dB), 14 (~-110 dB).
func Kaiser(n int, beta float64) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	denom := i0(beta)
	half := float64(n-1) / 2
	for i := 0; i < n; i++ {
		x := (float64(i) - half) / half
		w[i] = i0(beta*math.Sqrt(1-x*x)) / denom
	}
	return w
}

// i0 evaluates the modified Bessel function of the first kind, order 0.
// Series converges quickly for beta < ~50.
func i0(x float64) float64 {
	const eps = 1e-12
	t := x / 2
	sum := 1.0
	term := 1.0
	for k := 1; k < 100; k++ {
		term *= (t / float64(k)) * (t / float64(k))
		sum += term
		if term < eps*sum {
			break
		}
	}
	return sum
}
