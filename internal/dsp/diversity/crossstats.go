package diversity

import "math"

// CrossStats accumulates the second-order statistics relating one diversity
// branch to a reference branch, across an arbitrary number of chunks, so a
// calibration window can span many stream datagrams rather than being stuck
// with whatever one datagram happens to hold.
//
// From the same accumulation it yields both quantities a coherent combiner
// needs:
//
//	Gain()      the least-squares complex gain  h = cov(x,ref)/var(ref)
//	Coherence() the normalised cross-correlation |rho| in [0,1]
//
// Coherence is the quality gate. It answers "are these two receivers seeing the
// same signal through a constant complex gain?", which is the question a
// calibrator actually needs answered, and — unlike an absolute power threshold
// — it is SCALE-INVARIANT. Gating on power instead couples calibration to RF
// gain: an operator ends up raising front-end gain not because the radio needs
// it but to push a number past a software constant, which is a gain-staging
// trap that re-fires on every new front-end.
//
// For two branches carrying a common signal in independent noise at equal
// per-branch SNR gamma, |rho| = gamma/(1+gamma). That mapping is what makes a
// threshold interpretable: 0.5 is 0 dB, 0.35 is about -2.7 dB. Two independent
// noise branches over N samples sit near sqrt(pi/(4N)), i.e. ~0.03 at N=1000.
//
// # DC removal is load-bearing
//
// Every statistic here is a DC-REMOVED covariance, not a raw sum. Both
// receivers of a shared front-end carry LO leakage and converter DC offset, and
// that offset is perfectly common-mode. A raw (uncentred) correlator on two
// branches of pure independent noise sharing a DC offset reports |rho| -> 1 and
// hands back h = dc1/dc0 — it would confidently calibrate on nothing, exactly in
// the weak-signal regime the coherence gate exists to protect. Accumulating the
// per-branch means costs one extra sum per branch and makes the centring correct
// across a multi-chunk window in a single pass.
//
// Not safe for concurrent use; the owner is the single stream goroutine.
type CrossStats struct {
	n int

	sumRefR, sumRefI float64 // Σ ref
	sumXR, sumXI     float64 // Σ x
	sumRefRef        float64 // Σ |ref|²
	sumXX            float64 // Σ |x|²
	sumXRefR         float64 // Re Σ x·conj(ref)
	sumXRefI         float64 // Im Σ x·conj(ref)
}

// Accumulate folds one chunk of a reference branch and one other branch into
// the running statistics. The two must be the same length; a mismatched pair is
// truncated to the shorter, which keeps a ragged final datagram from skewing
// the estimate rather than rejecting the whole window.
func (s *CrossStats) Accumulate(ref, x []complex64) {
	n := len(ref)
	if len(x) < n {
		n = len(x)
	}
	for i := 0; i < n; i++ {
		cr, ci := float64(real(ref[i])), float64(imag(ref[i]))
		ar, ai := float64(real(x[i])), float64(imag(x[i]))
		s.sumRefR += cr
		s.sumRefI += ci
		s.sumXR += ar
		s.sumXI += ai
		s.sumRefRef += cr*cr + ci*ci
		s.sumXX += ar*ar + ai*ai
		// x·conj(ref) = (ar+j·ai)(cr-j·ci)
		s.sumXRefR += ar*cr + ai*ci
		s.sumXRefI += ai*cr - ar*ci
	}
	s.n += n
}

// Samples is how many sample pairs have been accumulated.
func (s *CrossStats) Samples() int { return s.n }

// Reset clears the accumulation, re-arming for a fresh window.
func (s *CrossStats) Reset() { *s = CrossStats{} }

// varRef, varX and cov are the DC-removed second moments.
func (s *CrossStats) varRef() float64 {
	if s.n == 0 {
		return 0
	}
	n := float64(s.n)
	return s.sumRefRef - (s.sumRefR*s.sumRefR+s.sumRefI*s.sumRefI)/n
}

func (s *CrossStats) varX() float64 {
	if s.n == 0 {
		return 0
	}
	n := float64(s.n)
	return s.sumXX - (s.sumXR*s.sumXR+s.sumXI*s.sumXI)/n
}

// cov is Σ (x-x̄)·conj(ref-r̄).
func (s *CrossStats) cov() (re, im float64) {
	if s.n == 0 {
		return 0, 0
	}
	n := float64(s.n)
	// E[x·conj(ref)] - x̄·conj(r̄), scaled by n.
	re = s.sumXRefR - (s.sumXR*s.sumRefR+s.sumXI*s.sumRefI)/n
	im = s.sumXRefI - (s.sumXI*s.sumRefR-s.sumXR*s.sumRefI)/n
	return re, im
}

// RefPower is the mean DC-removed power of the reference branch (0 when empty).
// This is an AC power: a branch carrying nothing but a DC offset reads 0, which
// is the correct reading for a receiver that is not delivering signal.
func (s *CrossStats) RefPower() float64 {
	if s.n == 0 {
		return 0
	}
	return math.Max(0, s.varRef()/float64(s.n))
}

// BranchPower is the mean DC-removed power of the other branch.
func (s *CrossStats) BranchPower() float64 {
	if s.n == 0 {
		return 0
	}
	return math.Max(0, s.varX()/float64(s.n))
}

// Coherence is |rho| in [0,1], the normalised magnitude of the cross-
// correlation. It is 0 when either branch carries no AC energy (the estimate is
// 0/0 and "no opinion" must not read as "perfectly correlated").
func (s *CrossStats) Coherence() float64 {
	vr, vx := s.varRef(), s.varX()
	if vr <= 0 || vx <= 0 {
		return 0
	}
	re, im := s.cov()
	rho := math.Hypot(re, im) / math.Sqrt(vr*vx)
	if math.IsNaN(rho) {
		return 0
	}
	// Rounding can push a perfectly correlated pair a hair over 1.
	return math.Min(1, rho)
}

// Gain is the least-squares complex gain h mapping the reference branch onto
// this one: h = cov(x,ref)/var(ref). ok is false when the reference carries no
// AC energy, in which case the phase cannot be anchored and the caller must
// keep whatever estimate it already had.
func (s *CrossStats) Gain() (complex64, bool) {
	vr := s.varRef()
	if vr <= 0 {
		return 0, false
	}
	re, im := s.cov()
	h := complex(float32(re/vr), float32(im/vr))
	if !isFinite(h) {
		return 0, false
	}
	return h, true
}

func isFinite(z complex64) bool {
	r, i := float64(real(z)), float64(imag(z))
	return !math.IsNaN(r) && !math.IsInf(r, 0) && !math.IsNaN(i) && !math.IsInf(i, 0)
}
