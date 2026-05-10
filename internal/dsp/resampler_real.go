package dsp

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// RealResampler is the real-valued counterpart of Resampler. Same
// polyphase decomposition, same L/M rate, but operates on float32
// audio instead of complex64 IQ. Sized for the post-demod chain in
// internal/voice/composer where the FM demod hands real audio to a
// resampler that retunes 48 kHz → 8 kHz (or arbitrary rational
// ratios) with proper anti-aliasing built into the polyphase
// prototype filter.
//
// The complex Resampler stays where it is — IQ paths still want it.
// Splitting them keeps each loop hot on the right data type without
// the complex64 ↔ float32 conversion overhead that an interface
// would impose.
//
// RealResampler is not safe for concurrent Process calls — pin it
// to a single demod goroutine and Reset between calls.
type RealResampler struct {
	L, M     int
	branches [][]float32 // length L; each branch is a phase of the prototype
	hist     []float32
	histPos  int
	branchN  int // taps per branch
	idx      int // commutator state (0..L-1)
	mCount   int // decimator state (0..M-1)
}

// NewRealResampler builds a real-valued resampler with rate L/M
// using a Kaiser-window LPF of total length tapsPerBranch*L. Cutoff
// is min(0.5/L, 0.5/M) so the prototype rejects images and aliases
// for both the interpolation and decimation steps. Bad parameters
// trip a panic at construction so misconfiguration shows up loudly.
func NewRealResampler(L, M, tapsPerBranch int, beta float64) *RealResampler {
	if L <= 0 || M <= 0 || tapsPerBranch <= 0 {
		panic("dsp: NewRealResampler requires positive L, M, tapsPerBranch")
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
	return &RealResampler{
		L: L, M: M,
		branches: branches,
		branchN:  taps,
		hist:     make([]float32, taps),
	}
}

// Reset clears the running history and commutator state so the next
// Process call starts from silence.
func (r *RealResampler) Reset() {
	for i := range r.hist {
		r.hist[i] = 0
	}
	r.histPos = 0
	r.idx = 0
	r.mCount = 0
}

// Process consumes len(src) input samples and returns approximately
// len(src)*L/M output samples. dst is reused if it has capacity.
func (r *RealResampler) Process(dst, src []float32) []float32 {
	want := len(src)*r.L/r.M + 8
	if cap(dst) < want {
		dst = make([]float32, 0, want)
	} else {
		dst = dst[:0]
	}
	for _, x := range src {
		r.hist[r.histPos] = x
		r.histPos++
		if r.histPos == r.branchN {
			r.histPos = 0
		}
		// Walk the L commutator phases for this single input sample.
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

func (r *RealResampler) computeBranch(b int) float32 {
	row := r.branches[b]
	var acc float32
	idx := r.histPos - 1
	if idx < 0 {
		idx = r.branchN - 1
	}
	for k := 0; k < r.branchN; k++ {
		acc += row[k] * r.hist[idx]
		idx--
		if idx < 0 {
			idx = r.branchN - 1
		}
	}
	return acc
}
