package diversity

// StaticCalibrator is the phase-coherence front-end that turns the
// phase-blind MRC combiner into a coherent maximal-ratio combiner using
// a single, one-time channel calibration.
//
// Motivation. mrc.go's pilot mode combines optimally *if* someone hands
// it a per-branch complex gain estimate h_k; diversity.go states plainly
// that supplying that estimate — keeping the branches phase-aligned — is
// the caller's job. This type is that caller. It targets the specific
// hardware class where the estimate can be a frozen constant rather than
// a continuously-tracked loop: shared-LO, single-chip front-ends such as
// the AD9361 (USRP B210 and clones), whose RX1/RX2 phase relationship is
// deterministic and stable for as long as the RF synthesizer stays
// locked. The only unknowns are the fixed trace/cable skew and the
// power-up divider phase — both constant across a tune session — so a
// one-time estimate on lock is sufficient and no per-sample phase-error
// re-estimation is needed.
//
// How it works.
//
//	Calibrate(window)  Least-squares-estimate each branch's complex gain
//	                   relative to a reference branch (branch 0):
//
//	                       h_k = ( sum_i x_k[i]·conj(x_0[i]) )
//	                             / ( sum_i |x_0[i]|^2 )
//
//	                   over the supplied calibration window (in practice
//	                   the samples of the first valid protocol sync burst,
//	                   which is high-SNR by definition — you only lock on a
//	                   good sync). h_0 is 1 by construction. The estimates
//	                   are frozen and pushed into the wrapped MRC's pilot
//	                   mode.
//	Combine(chunk)     Coherent MRC with the frozen gains:
//	                       y = ( sum_k conj(h_k)·x_k ) / ( sum_k |h_k|^2 ).
//	                   Before the first successful Calibrate it degrades
//	                   safely to a copy of the reference branch — never a
//	                   blind sum, which would let anti-phase branches
//	                   cancel the signal.
//
// A retune re-randomizes the divider phase, so the intended lifecycle is
// Reset() then Calibrate() on every new carrier lock. StaticCalibrator
// satisfies the Combiner interface, so a pipeline can swap it for
// Selection/MRC without touching the call site.
type StaticCalibrator struct {
	branches   int
	mrc        *MRC
	gains      []complex64 // frozen h_k relative to branch 0; gains[0]==1
	calibrated bool
}

// NewStaticCalibrator builds a coherent combiner for `branches` inputs.
// Branch 0 is the reference; every other branch is aligned to it.
func NewStaticCalibrator(branches int) *StaticCalibrator {
	if branches <= 0 {
		panic("diversity: StaticCalibrator needs at least one branch")
	}
	return &StaticCalibrator{
		branches: branches,
		// alpha is irrelevant here — the wrapped MRC is driven purely in
		// pilot mode via SetGain and never smooths power.
		mrc:   NewMRC(branches, 0.05),
		gains: make([]complex64, branches),
	}
}

// Calibrate estimates and freezes the per-branch complex gains from a
// calibration window (one chunk of samples per branch, in branch order —
// typically the first valid sync burst). It returns an error if the
// branch shapes are invalid or the reference branch carries no energy in
// the window (a silent reference can't anchor the phase — retry on the
// next sync burst). On success it enters coherent mode; Combine then
// uses the frozen gains.
func (c *StaticCalibrator) Calibrate(branches [][]complex64) error {
	if _, err := validateBranches(branches); err != nil {
		return err
	}
	if len(branches) != c.branches {
		return errBranchCount{got: len(branches), want: c.branches}
	}

	ref := branches[0]
	var refPow float64
	for _, z := range ref {
		refPow += mag2(z)
	}
	if refPow == 0 {
		return errSilentReference{}
	}

	// h_0 = 1 by construction (branch 0 is the reference). For k>0,
	// h_k is the LS scalar best mapping x_0 -> x_k: correlate branch k
	// against the reference and normalise by the reference energy.
	// Accumulate in float64 for numerical stability, then narrow.
	c.gains[0] = complex(1, 0)
	for k := 1; k < c.branches; k++ {
		xk := branches[k]
		var numR, numI float64
		for i := range xk {
			ar, ai := float64(real(xk[i])), float64(imag(xk[i]))
			cr, ci := float64(real(ref[i])), float64(imag(ref[i]))
			// x_k · conj(x_0) = (ar+j·ai)(cr-j·ci)
			numR += ar*cr + ai*ci
			numI += ai*cr - ar*ci
		}
		c.gains[k] = complex(float32(numR/refPow), float32(numI/refPow))
	}

	for k := 0; k < c.branches; k++ {
		c.mrc.SetGain(k, c.gains[k])
	}
	c.calibrated = true
	return nil
}

// Combine emits one coherently-combined sample per input sample index.
// Until the first successful Calibrate it returns a copy of the
// reference branch (branch 0) so an uncalibrated pipeline sees the
// primary receiver verbatim rather than a phase-blind sum that could
// cancel. After calibration it delegates to the wrapped MRC in pilot
// mode with the frozen gains.
func (c *StaticCalibrator) Combine(branches [][]complex64) ([]complex64, error) {
	n, err := validateBranches(branches)
	if err != nil {
		return nil, err
	}
	if len(branches) != c.branches {
		return nil, errBranchCount{got: len(branches), want: c.branches}
	}
	if !c.calibrated {
		out := make([]complex64, n)
		copy(out, branches[0])
		return out, nil
	}
	return c.mrc.Combine(branches)
}

// Calibrated reports whether a calibration constant has been locked.
func (c *StaticCalibrator) Calibrated() bool { return c.calibrated }

// Gains returns a copy of the frozen per-branch complex gain estimates
// (h_0 == 1). Empty-valued until the first Calibrate. Diagnostics/tests.
func (c *StaticCalibrator) Gains() []complex64 {
	out := make([]complex64, len(c.gains))
	copy(out, c.gains)
	return out
}

// Reset drops the calibration constant and re-arms for a fresh estimate.
// Call on every retune: a new LO lock re-randomizes the front-end's
// divider phase, invalidating the previous constant.
func (c *StaticCalibrator) Reset() {
	for i := range c.gains {
		c.gains[i] = 0
	}
	c.calibrated = false
	c.mrc.Reset()
}

type errSilentReference struct{}

func (errSilentReference) Error() string {
	return "diversity: reference branch (0) has no energy in the " +
		"calibration window; cannot anchor phase"
}
