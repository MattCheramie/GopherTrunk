package dsp

import (
	"math/rand"
	"testing"
)

// refResampler is the pre-#1321 phase-by-phase commutator walk with a plain
// ring-buffer history, kept verbatim as the reference the fast path must match
// bit for bit.
type refResampler struct {
	L, M     int
	branches [][]float32
	hist     []complex64
	histPos  int
	branchN  int
	idx      int
	mCount   int
}

func newRefResampler(r *Resampler) *refResampler {
	return &refResampler{L: r.L, M: r.M, branches: r.branches, branchN: r.branchN, hist: make([]complex64, r.branchN)}
}

func (r *refResampler) process(dst, src []complex64) []complex64 {
	dst = dst[:0]
	for _, x := range src {
		r.hist[r.histPos] = x
		r.histPos++
		if r.histPos == r.branchN {
			r.histPos = 0
		}
		for p := 0; p < r.L; p++ {
			if r.mCount == 0 {
				dst = append(dst, r.computeBranch(r.idx))
			}
			r.idx++
			if r.idx == r.L {
				r.idx = 0
			}
			r.mCount++
			if r.mCount == r.M {
				r.mCount = 0
			}
		}
	}
	return dst
}

func (r *refResampler) computeBranch(b int) complex64 {
	row := r.branches[b]
	var accI, accQ float32
	idx := r.histPos - 1
	if idx < 0 {
		idx = r.branchN - 1
	}
	for k := 0; k < r.branchN; k++ {
		s := r.hist[idx]
		h := row[k]
		accI += h * real(s)
		accQ += h * imag(s)
		idx--
		if idx < 0 {
			idx = r.branchN - 1
		}
	}
	return complex(accI, accQ)
}

// TestResamplerFastPathMatchesReference pins the #1321 resampler fast path
// (visit only the phases that emit, mirrored history) to the phase-by-phase
// walk it replaced: identical outputs, bit for bit, across decimating,
// interpolating and mixed ratios and across odd chunk boundaries.
func TestResamplerFastPathMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, c := range []struct{ L, M, taps int }{
		{6, 125, 250}, // 3 MS/s → 144 kHz TETRA tap
		{1, 4, 48},    // shared decimate-by-4
		{24, 125, 63}, // 750 kHz → 144 kHz
		{1, 50, 600},  // 2.4 MS/s → 48 kHz
		{3, 2, 16},    // interpolating
		{5, 7, 12},    // mixed
		{1, 1, 8},     // unity
	} {
		fast := NewResampler(c.L, c.M, c.taps, 7.0)
		ref := newRefResampler(fast)
		var fastOut, refOut []complex64
		for chunk := 0; chunk < 40; chunk++ {
			n := 1 + rng.Intn(1500)
			src := make([]complex64, n)
			for i := range src {
				src[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
			}
			fastOut = append(fastOut, fast.Process(nil, src)...)
			refOut = append(refOut, ref.process(nil, src)...)
		}
		if len(fastOut) != len(refOut) {
			t.Fatalf("L/M %d/%d: fast %d outputs, reference %d", c.L, c.M, len(fastOut), len(refOut))
		}
		for i := range fastOut {
			if fastOut[i] != refOut[i] {
				t.Fatalf("L/M %d/%d: output %d differs: fast %v reference %v", c.L, c.M, i, fastOut[i], refOut[i])
			}
		}
		if len(fastOut) == 0 {
			t.Fatalf("L/M %d/%d: no output", c.L, c.M)
		}
	}
}

// BenchmarkResamplerTETRATapAt3MSps is the per-tap cost the Airspy Mini / Pi 4
// report hinges on: a 144 kHz tap resampled straight off 3 MS/s (#1321).
func BenchmarkResamplerTETRATapAt3MSps(b *testing.B) {
	r := NewResampler(6, 125, 250, 7.0)
	src := make([]complex64, 30_000)
	for i := range src {
		src[i] = complex(float32(i%7)*0.1, float32(i%5)*0.1)
	}
	var dst []complex64
	b.SetBytes(int64(len(src) * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = r.Process(dst, src)
	}
}
