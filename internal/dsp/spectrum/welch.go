package spectrum

import (
	"math"
	"math/cmplx"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
)

// Welch returns the averaged power spectral density (dBFS) of in — the native
// Go equivalent of scipy.signal.welch. It splits in into overlapping segments
// of length len(win), windows and FFTs each, and averages their periodograms in
// linear power before a single conversion to dB. Averaging trades frequency
// resolution for a far lower-variance noise floor than the single-shot PowerDB
// periodogram, which is what makes a weak carrier legible against noise.
//
// win is the analysis window (length n, a power of two) and winSum is Σ(win²),
// precomputed once by the caller; plan must be fft.New(n). noverlap is the
// number of samples shared between consecutive segments (0 ≤ noverlap < n; a
// common choice is n/2). Bins are FFT-shifted so dst[0] is -Fs/2 and dst[n/2]
// is DC, identical to PowerDB / Frame. Normalization matches PowerDB
// (power = |X|²/(n·Σw²)), so a unit-amplitude tone reads ~0 dBFS.
//
// The returned slice is freshly allocated (length n). When in is shorter than
// one segment, no periodogram can be formed and every bin reads the clamp
// floor.
func Welch(plan fft.Plan, win []float64, winSum float64, in []complex64, noverlap int) []float32 {
	n := len(win)
	dst := make([]float32, n)
	floor := float32(10 * math.Log10(1e-30))
	if n == 0 {
		return dst
	}
	step := n - noverlap
	if step < 1 {
		step = n
	}

	bufIn := make([]complex128, n)
	bufOut := make([]complex128, n)
	acc := make([]float64, n) // linear power, already FFT-shifted
	norm := 1.0 / (float64(n) * winSum)
	half := n / 2

	segs := 0
	for off := 0; off+n <= len(in); off += step {
		seg := in[off : off+n]
		for i := 0; i < n; i++ {
			s := seg[i]
			w := win[i]
			bufIn[i] = complex(float64(real(s))*w, float64(imag(s))*w)
		}
		out := plan.Forward(bufOut, bufIn)
		for i := 0; i < n; i++ {
			dstIdx := (i + half) % n
			mag := cmplx.Abs(out[i])
			acc[dstIdx] += mag * mag * norm
		}
		segs++
	}

	if segs == 0 {
		for i := range dst {
			dst[i] = floor
		}
		return dst
	}
	inv := 1.0 / float64(segs)
	for i := 0; i < n; i++ {
		power := acc[i] * inv
		if power < 1e-30 {
			power = 1e-30
		}
		dst[i] = float32(10 * math.Log10(power))
	}
	return dst
}
