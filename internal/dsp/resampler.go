package dsp

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// Resampler is a polyphase rational resampler with rate L/M. It interpolates
// by L (using polyphase branches of an LPF) and decimates by M.
type Resampler struct {
	L, M     int
	branches [][]float32 // length L; each branch is a phase of the prototype
	hist     []complex64 // 2·branchN: each sample stored twice (mirrored window)
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
		hist:     make([]complex64, 2*taps),
	}
}

// Process consumes len(src) input samples and returns approximately
// len(src)*L/M output samples. dst is reused if it has capacity.
//
// Each input sample advances the commutator by L phases; an output is due at
// every phase whose running decimation count is a multiple of M. The loop
// visits only those phases — for a decimating ratio (M ≫ L, the wideband DDC
// case: 6/125 for a 144 kHz tap off 3 MS/s) that is usually none — instead of
// stepping through all L phases per input sample, which used to cost as much
// as the filter itself. The outputs, their order and the branch each one is
// computed from are identical to the phase-by-phase walk (pinned by
// TestResamplerFastPathMatchesReference).
func (r *Resampler) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src)*r.L/r.M+8 {
		dst = make([]complex64, 0, len(src)*r.L/r.M+8)
	} else {
		dst = dst[:0]
	}
	L, M, N := r.L, r.M, r.branchN
	for _, x := range src {
		// Push into the mirrored history: the sample lives at histPos and
		// histPos+N so the newest N samples are always one contiguous
		// descending run ending at histPos-1+N (no wrap in computeBranch).
		r.hist[r.histPos] = x
		r.hist[r.histPos+N] = x
		r.histPos++
		if r.histPos == N {
			r.histPos = 0
		}
		// Phases s = 0..L-1 of this input sample; an output falls on every s
		// with (mCount + s) % M == 0, computed from branch (idx + s) % L.
		s := 0
		if r.mCount != 0 {
			s = M - r.mCount
		}
		for ; s < L; s += M {
			b := r.idx + s
			if b >= L {
				b -= L
			}
			dst = append(dst, r.computeBranch(b))
		}
		// Advance the commutator by the L phases just consumed. idx moves by
		// L ≡ 0 (mod L), so only the decimation count changes.
		r.mCount = (r.mCount + L) % M
	}
	return dst
}

func (r *Resampler) computeBranch(b int) complex64 {
	row := r.branches[b]
	var accI, accQ float32
	// Newest sample first (mirrored history: see Process), oldest last — the
	// same products in the same order as the ring-buffer walk.
	win := r.hist[r.histPos : r.histPos+r.branchN]
	for k, h := range row {
		s := win[r.branchN-1-k]
		accI += h * real(s)
		accQ += h * imag(s)
	}
	return complex(accI, accQ)
}

// Reset clears the sample history and the commutator / decimator
// state so the next Process call starts a fresh stream — leaving the
// prototype filter branches intact.
func (r *Resampler) Reset() {
	for i := range r.hist {
		r.hist[i] = 0
	}
	r.histPos = 0
	r.idx = 0
	r.mCount = 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
