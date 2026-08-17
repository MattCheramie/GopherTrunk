package diversity

import (
	"math"
	"math/cmplx"
)

// MMSE-IRC: Minimum Mean Square Error, Interference Rejection Combining.
//
// MRC maximises SNR against spatially white noise. It has no way to tell a
// strong wanted signal from a strong DIRECTIONAL interferer, so in a dense
// urban band — a co-channel neighbour, or a second simulcast tower arriving
// with a path delay — it happily amplifies the thing that is ruining the
// constellation. IRC estimates the spatial covariance of everything that is
// NOT the wanted signal and steers a null at it:
//
//	w = R_nn^-1 h        (then scaled so w^H h = 1)
//
// IRC NEEDS TO BE TOLD WHICH SIGNAL IS THE WANTED ONE. Use ObserveWithReference
// and give it a known training sequence. Two things go wrong without one, and
// only the first is fixable inside the algorithm:
//
//  1. The obvious residual e_k = x_k - h_k*x_0 is degenerate. h is estimated by
//     least squares against x_0, so h_0 = 1 by construction and e_0 = 0
//     identically: R_nn's reference row and column are zero, the matrix is
//     rank-deficient, the diagonal-loading constant alone sets the reference
//     weight, and no null can form. Taking the residual against the MRC output
//     instead — what blind Observe does — leaves every residual non-trivial.
//
//  2. The channel estimate itself is contaminated, and nothing inside a blind
//     estimator can undo that. With a co-channel interferer,
//     cov(x_k,x_0) = h_wanted*P_signal + h_interf*P_interf, so least squares
//     returns a POWER-WEIGHTED BLEND of the two channels. Measured on the
//     synthetic scene in irc_test.go: a true wanted channel of 0.95∠40° reads
//     back as 0.32∠1°. Steering a null with that h nulls a mixture, and the
//     blind arm measures no gain over MRC at all (pinned by
//     TestIRCBlindCannotSeparateCoChannelSignals).
//
// Which is the whole reason this cannot live where MRC lives. The driver
// combines the WIDEBAND stream, where there is no training sequence and no
// notion of which carrier is wanted. A training sequence exists only after the
// per-channel DDC — TETRA's midamble, P25's sync — which is the same place the
// per-channel combining that mrc.go's KNOWN LIMITATION calls for would have to
// go. Both roads lead past the DDC.
//
// Differential-decoder safety, the property that makes this usable ahead of
// TETRA's or P25's differential decode: h is anchored to branch 0 (h_0 = 1 by
// the least-squares construction), and the w^H h = 1 normalisation makes the
// output track the wanted signal AS SEEN AT BRANCH 0. The absolute output phase
// is therefore pinned to the reference receiver's own, exactly as it is for
// TrackingCalibrator — not free to wander the way a blind CMA equalizer's is.
// Pinned by TestIRCCalibratorIsDifferentialSafe.
//
// EXPERIMENTAL and 2-branch only. It is wired into the offline replay harness
// (cmd/gophertrunk/diversity_replay_test.go) so it can be scored against MRC on
// real captures by CRC-clean frame yield; it is deliberately NOT selectable in
// the driver, both because that measurement does not exist yet and because the
// driver has no training sequence to give it. A combiner that tidies a
// constellation while decoding nothing is the failure mode this repo has
// already been burnt by.
type IRCCalibrator struct {
	branches int
	opts     TrackingOptions

	// stats relates branch 1 to the reference for the channel estimate and the
	// coherence gate — the same estimator and the same gates TrackingCalibrator
	// uses, so an A/B between them differs only in the weight computation.
	stats CrossStats
	// window buffers the branch samples of the current estimation window. The
	// residual needs the samples themselves, not just their second-order
	// statistics, so unlike MRC this cannot be a streaming accumulator.
	window [][]complex64

	// refStats / refWindow back the trained path: h estimated against a known
	// wanted signal rather than against branch 0.
	refStats  [2]CrossStats
	refWindow []complex64

	weights    []complex64
	calibrated bool
	coherence  float64
	updates    uint64
	holds      uint64
}

// ircLoadingFactor sets the diagonal loading as a fraction of the mean residual
// power, so regularisation is scale-invariant — an absolute constant would mean
// something different at every gain setting, which is the trap absolute-dBFS
// gating already taught this package. Large enough to keep a clean,
// interference-free environment (where R_nn is near-singular) from producing
// wild weights, small enough not to erase a real null.
const ircLoadingFactor = 1e-3

// NewIRCCalibrator builds a 2-branch MMSE-IRC combiner. Branch 0 is the
// reference: the output tracks the wanted signal as seen there.
func NewIRCCalibrator(branches int, opts TrackingOptions) *IRCCalibrator {
	if branches != 2 {
		panic("diversity: IRCCalibrator supports exactly 2 branches")
	}
	c := &IRCCalibrator{
		branches: branches,
		opts:     opts.withDefaults(),
		window:   make([][]complex64, branches),
		weights:  make([]complex64, branches),
	}
	c.weights[0] = complex(1, 0)
	return c
}

// Observe feeds one chunk of branch samples (reference at index 0). Weights are
// recomputed once per completed window.
func (c *IRCCalibrator) Observe(branches [][]complex64) ObserveResult {
	res := ObserveResult{Gains: c.weights}
	if len(branches) != c.branches {
		return res
	}
	c.stats.Accumulate(branches[0], branches[1])
	for k := range branches {
		c.window[k] = append(c.window[k], branches[k]...)
	}
	if len(c.window[0]) < c.opts.WindowSamples {
		return res
	}
	res.WindowDone = true
	res.Updated = c.estimate()
	res.Coherence = c.coherence
	c.stats.Reset()
	for k := range c.window {
		c.window[k] = c.window[k][:0]
	}
	return res
}

// ObserveWithReference is the path that can actually form a spatial null. The
// caller supplies the known wanted-signal samples for this chunk — a decoded or
// regenerated training sequence, aligned with the branch samples — so the
// channel estimate is of the WANTED signal rather than a power-weighted blend
// of everything on the carrier, and the residual is genuinely "everything that
// is not the wanted signal".
//
// reference must be the same length as each branch chunk. A caller that only
// knows the reference over part of a burst should pass just that part, in its
// own Observe call.
func (c *IRCCalibrator) ObserveWithReference(branches [][]complex64, reference []complex64) ObserveResult {
	res := ObserveResult{Gains: c.weights}
	if len(branches) != c.branches || len(reference) != len(branches[0]) {
		return res
	}
	// h_k is estimated against the REFERENCE here, not against branch 0, so
	// h[0] is a measured value rather than 1 by construction.
	c.refStats[0].Accumulate(reference, branches[0])
	c.refStats[1].Accumulate(reference, branches[1])
	for k := range branches {
		c.window[k] = append(c.window[k], branches[k]...)
	}
	c.refWindow = append(c.refWindow, reference...)
	if len(c.window[0]) < c.opts.WindowSamples {
		return res
	}
	res.WindowDone = true
	res.Updated = c.estimateTrained()
	res.Coherence = c.coherence
	c.refStats[0].Reset()
	c.refStats[1].Reset()
	c.refWindow = c.refWindow[:0]
	for k := range c.window {
		c.window[k] = c.window[k][:0]
	}
	return res
}

// estimateTrained runs one window's decision against a known reference. The
// coherence gate here measures how well the reference explains the branches —
// i.e. whether the wanted signal is actually present — rather than how well the
// two receivers agree.
func (c *IRCCalibrator) estimateTrained() bool {
	if c.calibrated && c.opts.Alpha == 0 {
		return false
	}
	gate := c.opts.TrackCoherence
	if !c.calibrated {
		gate = c.opts.LockCoherence
	}

	var h [2]complex128
	worst := math.Inf(1)
	for k := 0; k < 2; k++ {
		st := &c.refStats[k]
		rho := st.Coherence()
		if rho < worst {
			worst = rho
		}
		if st.RefPower() < c.opts.MinBranchPower || st.BranchPower() < c.opts.MinBranchPower {
			c.coherence = rho
			c.holds++
			return false
		}
		g, ok := st.Gain()
		if !ok || !finiteWithin(g, trackingDivergeFloorSq, trackingDivergeCeilSq) {
			c.coherence = rho
			c.holds++
			return false
		}
		h[k] = complex128(g)
	}
	c.coherence = worst
	if worst < gate {
		c.holds++
		return false
	}

	w, ok := ircWeightsAgainstReference(c.window, c.refWindow, h)
	if !ok {
		c.holds++
		return false
	}
	c.weights[0] = complex64(w[0])
	c.weights[1] = complex64(w[1])
	c.calibrated = true
	c.updates++
	return true
}

// ircWeightsAgainstReference is ircWeights with the residual taken against a
// KNOWN wanted signal: e_k = x_k - h_k*ref. This is the textbook formulation,
// and it is only well-posed because ref is an independent signal rather than
// one of the branches.
func ircWeightsAgainstReference(window [][]complex64, ref []complex64, h [2]complex128) ([2]complex128, bool) {
	var w [2]complex128
	n := len(window[0])
	if n == 0 || len(window[1]) < n || len(ref) < n {
		return w, false
	}
	var r00, r11 float64
	var r01 complex128
	for i := 0; i < n; i++ {
		s := complex128(ref[i])
		e0 := complex128(window[0][i]) - h[0]*s
		e1 := complex128(window[1][i]) - h[1]*s
		r00 += real(e0 * cmplx.Conj(e0))
		r11 += real(e1 * cmplx.Conj(e1))
		r01 += e0 * cmplx.Conj(e1)
	}
	return solveIRC(r00, r11, r01, float64(n), h)
}

// estimate runs one window's decision. Like TrackingCalibrator, a rejected
// window HOLDS the previous weights rather than dropping to passthrough:
// falling back mid-stream is itself a large phase step, which is the failure
// class being avoided.
func (c *IRCCalibrator) estimate() bool {
	if c.calibrated && c.opts.Alpha == 0 {
		return false
	}
	gate := c.opts.TrackCoherence
	if !c.calibrated {
		gate = c.opts.LockCoherence
	}

	rho := c.stats.Coherence()
	c.coherence = rho
	if c.stats.RefPower() < c.opts.MinBranchPower ||
		c.stats.BranchPower() < c.opts.MinBranchPower {
		c.holds++
		return false
	}
	h1, ok := c.stats.Gain()
	if !ok || !finiteWithin(h1, trackingDivergeFloorSq, trackingDivergeCeilSq) {
		c.holds++
		return false
	}
	if rho < gate {
		c.holds++
		return false
	}

	h := [2]complex128{1, complex128(h1)}
	w, ok := ircWeights(c.window, h)
	if !ok {
		c.holds++
		return false
	}
	c.weights[0] = complex64(w[0])
	c.weights[1] = complex64(w[1])
	c.calibrated = true
	c.updates++
	return true
}

// ircWeights computes w = R_nn^-1 h, scaled so w^H h = 1, from one window of
// branch samples and a channel estimate. Returns false when the window is
// unusable (empty, or a covariance that will not invert even after loading).
func ircWeights(window [][]complex64, h [2]complex128) ([2]complex128, bool) {
	var w [2]complex128
	n := len(window[0])
	if n == 0 || len(window[1]) < n {
		return w, false
	}

	// The wanted-signal estimate: the MRC output for this channel estimate.
	// Using it rather than the reference branch is what keeps R_nn full-rank
	// (see the type comment).
	hNormSq := real(h[0]*cmplx.Conj(h[0]) + h[1]*cmplx.Conj(h[1]))
	if hNormSq <= 0 || math.IsNaN(hNormSq) || math.IsInf(hNormSq, 0) {
		return w, false
	}

	var r00, r11 float64
	var r01 complex128
	for i := 0; i < n; i++ {
		x0 := complex128(window[0][i])
		x1 := complex128(window[1][i])
		s := (cmplx.Conj(h[0])*x0 + cmplx.Conj(h[1])*x1) / complex(hNormSq, 0)
		e0 := x0 - h[0]*s
		e1 := x1 - h[1]*s
		r00 += real(e0 * cmplx.Conj(e0))
		r11 += real(e1 * cmplx.Conj(e1))
		r01 += e0 * cmplx.Conj(e1)
	}
	return solveIRC(r00, r11, r01, float64(n), h)
}

// solveIRC normalises an accumulated 2x2 Hermitian residual covariance, loads
// its diagonal, inverts it analytically and returns w = R^-1 h scaled so
// w^H h = 1.
func solveIRC(r00, r11 float64, r01 complex128, n float64, h [2]complex128) ([2]complex128, bool) {
	var w [2]complex128
	inv := 1 / n
	r00 *= inv
	r11 *= inv
	r01 *= complex(inv, 0)

	// Scale-invariant diagonal loading.
	load := ircLoadingFactor * (r00 + r11) / 2
	if load <= 0 || math.IsNaN(load) || math.IsInf(load, 0) {
		return w, false
	}
	r00 += load
	r11 += load

	// R is Hermitian, so R = [[r00, r01], [conj(r01), r11]] and
	// det = r00*r11 - |r01|^2, real and positive for a loaded covariance.
	det := complex(r00*r11, 0) - r01*cmplx.Conj(r01)
	if cmplx.Abs(det) <= 0 || cmplx.IsNaN(det) || cmplx.IsInf(det) {
		return w, false
	}
	invDet := 1 / det
	i00 := complex(r11, 0) * invDet
	i01 := -r01 * invDet
	i10 := -cmplx.Conj(r01) * invDet
	i11 := complex(r00, 0) * invDet

	w[0] = i00*h[0] + i01*h[1]
	w[1] = i10*h[0] + i11*h[1]

	// Distortionless-response normalisation: w^H h = 1, so the output tracks
	// the wanted signal as seen at branch 0 rather than an arbitrary scaling
	// of it. This is also what anchors the output phase.
	resp := cmplx.Conj(w[0])*h[0] + cmplx.Conj(w[1])*h[1]
	if cmplx.Abs(resp) <= 0 || cmplx.IsNaN(resp) || cmplx.IsInf(resp) {
		return w, false
	}
	scale := cmplx.Conj(1 / resp)
	w[0] *= scale
	w[1] *= scale
	if cmplx.IsNaN(w[0]) || cmplx.IsNaN(w[1]) || cmplx.IsInf(w[0]) || cmplx.IsInf(w[1]) {
		return [2]complex128{}, false
	}
	return w, true
}

// Combine emits y[n] = w^H x[n]. Before the first accepted window it returns a
// copy of the reference branch, never a phase-blind sum.
func (c *IRCCalibrator) Combine(branches [][]complex64) ([]complex64, error) {
	n, err := validateBranches(branches)
	if err != nil {
		return nil, err
	}
	if len(branches) != c.branches {
		return nil, errBranchCount{got: len(branches), want: c.branches}
	}
	out := make([]complex64, n)
	if !c.calibrated {
		copy(out, branches[0])
		return out, nil
	}
	w0 := conj64(c.weights[0])
	w1 := conj64(c.weights[1])
	for i := 0; i < n; i++ {
		out[i] = w0*branches[0][i] + w1*branches[1][i]
	}
	return out, nil
}

// Calibrated reports whether a usable weight vector has been accepted.
func (c *IRCCalibrator) Calibrated() bool { return c.calibrated }

// Coherence is |rho| of the last completed window.
func (c *IRCCalibrator) Coherence() float64 { return c.coherence }

// Counters returns accepted and held window counts.
func (c *IRCCalibrator) Counters() (updates, holds uint64) { return c.updates, c.holds }

// Weights returns a copy of the applied combining weights.
func (c *IRCCalibrator) Weights() []complex64 {
	out := make([]complex64, len(c.weights))
	copy(out, c.weights)
	return out
}

// Reset drops the estimate and re-arms. Call on every retune.
func (c *IRCCalibrator) Reset() {
	c.weights[0] = complex(1, 0)
	for i := 1; i < len(c.weights); i++ {
		c.weights[i] = 0
	}
	c.stats.Reset()
	c.refStats[0].Reset()
	c.refStats[1].Reset()
	c.refWindow = c.refWindow[:0]
	for k := range c.window {
		c.window[k] = c.window[k][:0]
	}
	c.calibrated = false
	c.coherence = 0
}

func conj64(v complex64) complex64 {
	return complex(real(v), -imag(v))
}
