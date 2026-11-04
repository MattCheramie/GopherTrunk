package filter

import (
	"math"
	"math/rand"
	"testing"
)

// The folded path must compute the same filter as the direct one: only the
// float32 rounding of the sum may differ. Checked on the TETRA receiver's
// two filters (#1321) plus an even-length symmetric set, fed in uneven
// chunks so the history wrap is exercised.
func TestFIRFoldedMatchesDirect(t *testing.T) {
	sets := map[string][]float32{
		"kaiser-145":  LowpassKaiser(145, 12500.0/144000, 8.6),
		"rrc-8x10":    RootRaisedCosine(8, 10, 0.35),
		"even-sym-16": {1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1},
		"single":      {0.5},
	}
	rng := rand.New(rand.NewSource(1321))
	src := make([]complex64, 20000)
	for i := range src {
		src[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
	}
	for name, taps := range sets {
		direct, folded := NewFIR(taps), NewFIR(taps)
		if !folded.EnableFolded() {
			t.Fatalf("%s: EnableFolded refused symmetric taps", name)
		}
		var peak, worst float64
		for off := 0; off < len(src); {
			n := 1 + rng.Intn(700)
			if off+n > len(src) {
				n = len(src) - off
			}
			a := direct.Process(nil, src[off:off+n])
			b := folded.Process(nil, src[off:off+n])
			for i := range a {
				peak = math.Max(peak, math.Abs(float64(real(a[i]))))
				worst = math.Max(worst, math.Abs(float64(real(a[i]-b[i]))))
				worst = math.Max(worst, math.Abs(float64(imag(a[i]-b[i]))))
			}
			off += n
		}
		if worst > 1e-5*peak {
			t.Errorf("%s: folded differs from direct by %g (peak %g)", name, worst, peak)
		}
	}
}

func TestFIREnableFoldedRefusesAsymmetricTaps(t *testing.T) {
	f := NewFIR([]float32{1, 2, 3, 2.0000002, 1})
	if f.EnableFolded() || f.Folded() {
		t.Fatal("EnableFolded accepted taps that are not bit-exactly symmetric")
	}
	src := []complex64{1, 0, 0, 0, 0, 0}
	got := f.Process(nil, src)
	if real(got[3]) != 2.0000002 {
		t.Errorf("refused filter no longer computes its own response: %v", got)
	}
}

func BenchmarkFIRKaiser145Direct(b *testing.B) { benchFIR(b, false) }
func BenchmarkFIRKaiser145Folded(b *testing.B) { benchFIR(b, true) }

func benchFIR(b *testing.B, fold bool) {
	f := NewFIR(LowpassKaiser(145, 12500.0/144000, 8.6))
	if fold {
		f.EnableFolded()
	}
	src := make([]complex64, 4096)
	for i := range src {
		src[i] = complex(float32(i%7), float32(i%5))
	}
	dst := make([]complex64, len(src))
	b.SetBytes(int64(len(src) * 8))
	for i := 0; i < b.N; i++ {
		dst = f.Process(dst, src)
	}
}
