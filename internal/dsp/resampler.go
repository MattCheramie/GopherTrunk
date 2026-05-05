package dsp

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// Resampler is a polyphase rational resampler with rate L/M. It interpolates
// by L (using polyphase branches of an LPF) and decimates by M.
type Resampler struct {
	L, M     int
	branches [][]float32 // length L; each branch is a phase of the prototype
	hist     []complex64
	histPos  int
	branchN  int // taps per branch
	idx      int // commutator state (0..L-1)
	mCount   int // decimator state (0..M-1)
}

// NewResampler builds a resampler with rate L/M using a Kaiser-window LPF
// of total length tapsPerBranch*L. Cutoff is min(0.5/L, 0.5/M) so that the
// filter rejects images and aliases for both interpolation and decimation.
func NewResampler(L, M, tapsPerBranch int, beta float64) *Resampler {
	if L <= 0 || M <= 0 || tapsPerBranch <= 0 {
		panic("resampler: L, M, tapsPerBranch must be positive")
	}
	N := tapsPerBranch * L
	if N%2 == 0 {
		N++
	}
	cutoff := 0.5 / float64(maxInt(L, M))
	proto := filter.LowpassKaiser(N, cutoff, beta)
	// Compensate for interpolator gain loss of 1/L.
	for i := range proto {
		proto[i] *= float32(L)
	}
	branches := make([][]float32, L)
	taps := (len(proto) + L - 1) / L
	for b := 0; b < L; b++ {
		row := make([]float32, taps)
		for k := 0; k < taps; k++ {
			j := k*L + b
			if j < len(proto) {
				row[k] = proto[j]
			}
		}
		branches[b] = row
	}
	return &Resampler{
		L: L, M: M,
		branches: branches,
		branchN:  taps,
		hist:     make([]complex64, taps),
	}
}

// Process consumes len(src) input samples and returns approximately
// len(src)*L/M output samples. dst is reused if it has capacity.
func (r *Resampler) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src)*r.L/r.M+8 {
		dst = make([]complex64, 0, len(src)*r.L/r.M+8)
	} else {
		dst = dst[:0]
	}
	for _, x := range src {
		// Push into history.
		r.hist[r.histPos] = x
		r.histPos++
		if r.histPos == r.branchN {
			r.histPos = 0
		}
		// Walk the L commutator phases for this single input sample.
		for p := 0; p < r.L; p++ {
			if r.mCount == 0 {
				out := r.computeBranch(r.idx)
				dst = append(dst, out)
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

func (r *Resampler) computeBranch(b int) complex64 {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
