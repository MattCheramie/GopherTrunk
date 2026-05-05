// Package diversity combines IQ streams from N receivers tuned to the
// same frequency into a single per-sample IQ stream that's stronger and
// less faded than any one source. Two complementary strategies live
// here:
//
//   selection.go  Selection combining — pick the branch with the
//                 highest instantaneous |x|^2. Cheap, no calibration,
//                 graceful degradation when one branch goes silent.
//   mrc.go        Maximal-ratio combining — weight each branch by an
//                 estimate of its complex channel response and sum.
//                 Optimal in AWGN; needs SNR estimates per branch and
//                 cooperating front-ends (matched RF chains).
//
// Both combiners take []complex64 chunks per branch and emit a single
// []complex64 chunk. Branch alignment is the caller's job: a delay line
// before the combiner equalizes path-length skew across antennas, and
// a coarse phase recovery (e.g. a CMA equalizer per branch) keeps the
// branches phase-aligned. With one antenna this package is a no-op;
// with two or more it's where the diversity gain lives.
//
// Public surface: a Combiner interface and two implementations
// (NewSelection / NewMRC), so a higher-level pipeline can swap the
// strategy without changing the call site.
package diversity

import "errors"

// Combiner takes one chunk of complex samples per branch (in branch
// order) and returns a single combined chunk. All branch chunks must
// have the same length; mismatches return an error rather than
// silently truncating.
type Combiner interface {
	Combine(branches [][]complex64) ([]complex64, error)
}

// validateBranches enforces that callers supply ≥1 branch and that
// every branch has the same chunk length. Returns the common length.
func validateBranches(branches [][]complex64) (int, error) {
	if len(branches) == 0 {
		return 0, errors.New("diversity: at least one branch is required")
	}
	n := len(branches[0])
	for i, b := range branches {
		if len(b) != n {
			return 0, errBranchLenMismatch{i, len(b), n}
		}
	}
	return n, nil
}

type errBranchLenMismatch struct {
	branch, got, want int
}

func (e errBranchLenMismatch) Error() string {
	return "diversity: branch " + itoa(e.branch) + " length " +
		itoa(e.got) + " != expected " + itoa(e.want)
}

// itoa avoids importing strconv just for an error string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// mag2 returns |z|^2 without a sqrt. Used by both combiners.
func mag2(z complex64) float64 {
	r := float64(real(z))
	i := float64(imag(z))
	return r*r + i*i
}
