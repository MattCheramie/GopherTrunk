package channelizer

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// TestOversampledMatchesDirectFilterbank pins the polyphase structure against
// the textbook definition it implements: bin k at output step s is the input
// demodulated by e^{-j2πk t/M} and filtered by the prototype, sampled every
// M/2 input samples.
func TestOversampledMatchesDirectFilterbank(t *testing.T) {
	const M = 8
	proto := filter.LowpassKaiser(M*6+1, 0.75/M, 7)
	ch := NewOversampledWithPrototype(M, proto)
	rng := rand.New(rand.NewSource(1))
	const N = 512
	in := make([]complex64, N)
	for i := range in {
		in[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
	}
	// Feed in uneven chunks to prove Process is chunk-invariant.
	var out [][]complex64
	for i := 0; i < N; {
		e := i + rng.Intn(40) + 1
		if e > N {
			e = N
		}
		part := ch.Process(nil, in[i:e])
		if out == nil {
			out = make([][]complex64, M)
		}
		for k := range part {
			out[k] = append(out[k], part[k]...)
		}
		i = e
	}
	step := M / 2
	for k := 0; k < M; k++ {
		if got, want := len(out[k]), N/step; got != want {
			t.Fatalf("bin %d: %d outputs, want %d", k, got, want)
		}
		for s := 0; s < N/step; s++ {
			ts := (s+1)*step - 1
			var want complex128
			for n := 0; n < len(proto); n++ {
				idx := ts - n
				if idx < 0 {
					break
				}
				x := complex128(in[idx])
				want += complex(float64(proto[n]), 0) * x * cmplx.Exp(complex(0, -2*math.Pi*float64(k)*float64(idx)/M))
			}
			got := complex128(out[k][s])
			if cmplx.Abs(got-want) > 1e-3*(1+cmplx.Abs(want)) {
				t.Fatalf("bin %d step %d: got %v want %v", k, s, got, want)
			}
		}
	}
}

// TestOversampledBinEdgeToneIsFlat is the property the critically-sampled
// channelizer lacks: a tone anywhere within ±0.5 bin of a bin centre leaves
// that bin at (almost) unit amplitude and at exactly its residual frequency —
// the bin edge is inside the flat passband, not on the −6 dB roll-off and
// not folded.
func TestOversampledBinEdgeToneIsFlat(t *testing.T) {
	const M = 16
	const Fs = 16.0 * 48_000
	const binRate = Fs / M
	ch := NewOversampled(M, 16, 9.0)
	for _, residual := range []float64{0, 0.25, 0.48, -0.48, 0.5} {
		const k0 = 5
		f := (k0 + residual) * binRate
		const N = 1 << 14
		in := make([]complex64, N)
		for i := range in {
			th := 2 * math.Pi * f * float64(i) / Fs
			in[i] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
		}
		out := ch.Process(nil, in)
		bin := out[k0][len(out[k0])/2:] // skip the filter transient
		outRate := 2 * binRate
		// Locate the tone in the bin output by DFT peak.
		L := len(bin)
		bestP, bestF := 0.0, 0.0
		for q := -L / 2; q < L/2; q++ {
			var acc complex128
			for i, s := range bin {
				acc += complex128(s) * cmplx.Exp(complex(0, -2*math.Pi*float64(q)*float64(i)/float64(L)))
			}
			if p := cmplx.Abs(acc); p > bestP {
				bestP, bestF = p, float64(q)*outRate/float64(L)
			}
		}
		amp := bestP / float64(L)
		if math.Abs(bestF-residual*binRate) > outRate/float64(L)*1.5 {
			t.Errorf("residual %.2f: tone left bin %d at %.0f Hz, want %.0f Hz", residual, k0, bestF, residual*binRate)
		}
		if db := 20 * math.Log10(amp); db < -1.0 || db > 0.5 {
			t.Errorf("residual %.2f: bin %d amplitude %.2f dB, want within [-1, +0.5] dB (flat passband)", residual, k0, db)
		}
		ch.Reset()
	}
}

// TestOversampledOutputRate pins 2 output samples per bin per M input samples.
func TestOversampledOutputRate(t *testing.T) {
	const M = 4
	ch := NewOversampled(M, 8, 8.6)
	out := ch.Process(nil, make([]complex64, 1024))
	for c := 0; c < M; c++ {
		if got, want := len(out[c]), 1024*2/M; got != want {
			t.Errorf("bin %d: len = %d, want %d", c, got, want)
		}
	}
}
