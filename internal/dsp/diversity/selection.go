package diversity

// Selection is the simplest diversity combiner: per output sample,
// pick the branch with the highest |x|^2. No SNR calibration, no phase
// alignment, no per-branch weights — just the loudest sample wins.
//
// Properties:
//
//   - With M independent Rayleigh-faded branches, selection diversity
//     gives ~ M-fold reduction in deep-fade probability while only
//     paying the cost of one branch's data path downstream. The
//     theoretical SNR gain is 10·log10(sum_{k=1..M}(1/k)) dB above one
//     branch — about 4.8 dB for M=4.
//   - Phase-blind: it doesn't matter if branches are anti-phase.
//   - Robust to a silent branch: a dead branch contributes |x|^2=0
//     and never wins.
//
// The struct is empty (the algorithm has no per-call state) but kept
// concrete so future variants — selection with hysteresis, or hybrid
// switch-and-stay diversity — can hang state off it without breaking
// callers.
type Selection struct{}

// NewSelection constructs a selection combiner.
func NewSelection() *Selection { return &Selection{} }

// Combine selects, per sample index, the branch with the largest
// |x|^2. With a single branch the input chunk is returned verbatim
// (allocated copy so the caller can safely mutate the result).
func (s *Selection) Combine(branches [][]complex64) ([]complex64, error) {
	n, err := validateBranches(branches)
	if err != nil {
		return nil, err
	}
	out := make([]complex64, n)
	if len(branches) == 1 {
		copy(out, branches[0])
		return out, nil
	}
	for i := 0; i < n; i++ {
		bestK := 0
		best := mag2(branches[0][i])
		for k := 1; k < len(branches); k++ {
			if m := mag2(branches[k][i]); m > best {
				best = m
				bestK = k
			}
		}
		out[i] = branches[bestK][i]
	}
	return out, nil
}
