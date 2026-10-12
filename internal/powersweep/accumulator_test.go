package powersweep

import (
	"math"
	"math/rand"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// The accumulator runs on fft.Radix2 in float32 for speed (#1230). It must
// read the same spectrum as the float64 gonum periodogram it replaced: same
// Hann window, normalisation and FFT shift, to well under what a dB display
// shows, at every bin of a tone-plus-noise input and at several sizes.
func TestAccumulatorMatchesFloat64Reference(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for _, n := range []int{8, 256, 1024, 8192} {
		const frames = 6
		iq := make([]complex64, frames*n)
		for i := range iq {
			ph := 2 * math.Pi * 0.1234 * float64(i)
			iq[i] = complex(float32(0.3*math.Cos(ph)+r.NormFloat64()*0.01), float32(0.3*math.Sin(ph)+r.NormFloat64()*0.01))
		}
		a := newAccumulator(n)
		a.add(iq)
		got := a.dB(0, n)

		// Reference: the previous float64 path, written out.
		win := window.Hann(n)
		var ws float64
		for _, w := range win {
			ws += w * w
		}
		plan := fft.New(n)
		in, sum := make([]complex128, n), make([]float64, n)
		for f := 0; f < frames; f++ {
			for i, s := range iq[f*n : (f+1)*n] {
				in[i] = complex(float64(real(s))*win[i], float64(imag(s))*win[i])
			}
			for i, v := range plan.Forward(nil, in) {
				sum[(i+n/2)%n] += (real(v)*real(v) + imag(v)*imag(v)) / (float64(n) * ws)
			}
		}
		for k := range got {
			want := 10 * math.Log10(max(sum[k]/frames, 1e-30))
			if d := math.Abs(float64(got[k]) - want); d > 0.01 {
				t.Fatalf("n=%d bin %d: %.3f dBFS, float64 reference %.3f (Δ %.3f dB)", n, k, got[k], want, d)
			}
		}
	}
}

// BenchmarkAccumulatorAdd measures the sweep's per-sample cost (MB/s here is
// mega-samples per second). The #1230 phones need it above the dongle rate
// (2.4 MS/s) on their CPU; compare GOARCH=386 against main for a 32-bit view.
func BenchmarkAccumulatorAdd(b *testing.B) {
	for _, n := range []int{256, 1024, 8192} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			a := newAccumulator(n)
			iq := make([]complex64, 1<<16)
			r := rand.New(rand.NewSource(1))
			for i := range iq {
				iq[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
			}
			b.SetBytes(int64(len(iq)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				a.add(iq)
			}
		})
	}
}
