package diversity

import "math"

// Tracking-loop guards. Named after snapshot_cma.go's divergence guard, and
// there for the same reason: an adaptive estimate that meets a pathological
// window must not be able to poison every window after it.
const (
	// trackingDivergeCeilSq / trackingDivergeFloorSq bound |h|². A real branch
	// imbalance is a few dB; 1024 is 30 dB, past which the "weaker" branch
	// contributes nothing to a maximal-ratio combine anyway and the estimate is
	// far more likely to be numerical garbage than a measurement.
	trackingDivergeCeilSq  = 1024.0
	trackingDivergeFloorSq = 1.0 / 1024.0

	// trackingMaxStepRad / trackingMaxStepRatio clamp how far one accepted
	// update may move the APPLIED gain. In normal operation these never bind
	// (a 2 ms window at |rho|=0.5 estimates phase to well under a degree, and
	// the one-pole scales that by alpha again); they exist so the per-update
	// output phase step has a hard bound that holds regardless of sample rate,
	// window length, or a freak window that slipped past the coherence gate.
	trackingMaxStepRad   = 0.05 // ~2.9°
	trackingMaxStepRatio = 1.25

	// TrackingDefaultAlpha is the one-pole coefficient for a 2 ms window,
	// giving a ~200 ms time constant: far slower than one TETRA burst (14 ms),
	// so intra-burst phase is effectively constant, and far faster than the
	// relative drift of two independently-locked front-end PLLs.
	TrackingDefaultAlpha = 0.01

	// DefaultLockPhaseSigmaRad (~5.7 deg) and DefaultTrackPhaseSigmaRad
	// (~9.2 deg) are the default estimate-quality bounds. Via LockGate/TrackGate
	// they put the minimum acceptable |rho| at ~8x and ~5x the noise-only
	// coherence floor sqrt(pi/4N): the chance of two INDEPENDENT branches
	// clearing them is exp(-N*rho^2) — about e^-50 and 3e-9 per window — so
	// noise cannot lock and can produce at most one freak (damped, clamped)
	// tracking update a year, while a genuinely coherent pair passes however
	// much noise-only bandwidth surrounds its carrier.
	DefaultLockPhaseSigmaRad  = 0.10
	DefaultTrackPhaseSigmaRad = 0.16
)

// coherenceGateFor converts a phase-error bound into the minimum |rho| an
// n-sample window must reach: sigma^2 ~ (1-rho^2)/(2N rho^2) inverted for rho.
// An empty window can justify nothing, so n <= 0 gates at 1.
func coherenceGateFor(sigmaRad float64, n int) float64 {
	if n <= 0 || sigmaRad <= 0 {
		return 1
	}
	return 1 / math.Sqrt(1+2*float64(n)*sigmaRad*sigmaRad)
}

// LockGate is the minimum |rho| an n-sample window needs for a FIRST estimate
// under these options (defaults applied). Exposed so an operator-facing health
// line can print the actual bar a stream is being held to.
func (o TrackingOptions) LockGate(n int) float64 {
	return coherenceGateFor(o.withDefaults().LockPhaseSigmaRad, n)
}

// TrackGate is the minimum |rho| an n-sample window needs for a subsequent
// tracking update under these options (defaults applied).
func (o TrackingOptions) TrackGate(n int) float64 {
	return coherenceGateFor(o.withDefaults().TrackPhaseSigmaRad, n)
}

// TrackingOptions configures a TrackingCalibrator. Zero fields take defaults.
type TrackingOptions struct {
	// WindowSamples is how many sample pairs to accumulate per branch before
	// (re-)estimating. A single stream datagram is far too short: at |rho|=0.35
	// a 184-sample window estimates phase to about 9.5°, an 8192-sample window
	// to 1.2°.
	WindowSamples int
	// Alpha is the one-pole smoothing coefficient applied to every update after
	// the first. ZERO MEANS ONE-SHOT: the first accepted window is frozen and
	// never revisited, which is the classic static-calibration behaviour for a
	// shared-LO front-end whose branch phase offset really is a constant.
	Alpha float64
	// LockPhaseSigmaRad / TrackPhaseSigmaRad bound the PROJECTED PHASE ERROR of
	// a window's least-squares gain estimate, sigma ~ sqrt((1-rho^2)/(2N rho^2)),
	// which is what actually decides whether an estimate is safe to apply. The
	// lock bound is stricter because a one-shot calibration lives with its first
	// estimate forever; a tracking update's error averages away over ~1/Alpha
	// windows and is clamped besides.
	//
	// These deliberately replace a fixed |rho| threshold. |rho| measured on a
	// WIDEBAND stream is diluted by every hertz of noise-only bandwidth around
	// the coherent carrier (rho_wb ~ rho_ch*sqrt(f0*f1) for in-channel power
	// fractions f_k), so a fixed rho constant makes calibration depend on the
	// configured capture bandwidth and on each branch's noise floor — a
	// bandwidth-staging trap, sibling of the -40 dBFS gain-staging trap the
	// coherence gate replaced. An operator with an X310 raised RF gain 70 -> 75 dB
	// purely to push wideband rho past 0.5 while their phase estimate had been
	// accurate to ~4 degrees all along. Bounding the phase error instead makes the
	// minimum rho fall as 1/sqrt(N) (see LockGate/TrackGate), so a long window may
	// trust a small coherence, and the gate answers the only question it owns:
	// "is this estimate trustworthy?"
	LockPhaseSigmaRad  float64
	TrackPhaseSigmaRad float64
	// MinBranchPower is a linear AC-power floor that rejects a DIGITALLY DEAD
	// branch — all zeros, or a receiver that never digitised — where |rho| is
	// 0/0. It is deliberately far below any signal level: it is not a
	// signal-presence test, because an absolute power threshold is exactly the
	// gain-staging trap this design exists to remove.
	MinBranchPower float64
}

func (o TrackingOptions) withDefaults() TrackingOptions {
	if o.WindowSamples <= 0 {
		o.WindowSamples = 8192
	}
	if o.Alpha < 0 || o.Alpha > 1 {
		o.Alpha = TrackingDefaultAlpha
	}
	if o.LockPhaseSigmaRad <= 0 {
		o.LockPhaseSigmaRad = DefaultLockPhaseSigmaRad
	}
	if o.TrackPhaseSigmaRad <= 0 {
		o.TrackPhaseSigmaRad = DefaultTrackPhaseSigmaRad
	}
	if o.MinBranchPower <= 0 {
		o.MinBranchPower = 1e-10 // -100 dBFS; one CS16 LSB is -90.3 dBFS
	}
	return o
}

// ObserveResult is what one Observe call learned. Gains is borrowed and only
// valid until the next Observe.
type ObserveResult struct {
	// WindowDone is true when this call completed an estimation window.
	WindowDone bool
	// Updated is true when that window was accepted and moved the applied gains.
	Updated bool
	// Coherence is |rho| of the completed window (0 when no window completed).
	Coherence float64
	// Gains are the currently applied per-branch complex gains (Gains[0] == 1).
	Gains []complex64
}

// TrackingCalibrator is a coherent maximal-ratio combiner whose per-branch
// complex gain is re-estimated from the stream itself and smoothed, rather than
// measured once and frozen.
//
// # Why tracking
//
// StaticCalibrator's frozen constant is correct for a shared-LO, single-chip
// front-end (an AD9361/B210), where the branch-to-branch phase is fixed trace
// skew plus a power-up divider phase and genuinely does not move for as long as
// the synthesizer stays locked. It is NOT correct when the two branches sit on
// separate daughterboards with independent PLLs — a USRP X310 with two TwinRX
// cards, say. Those are frequency-locked to a common reference, so there is no
// beat, but the relative phase is random at each lock and walks afterwards. A
// constant measured once decays.
//
// # Why this is safe ahead of a differential demodulator
//
// The expensive lesson recorded for SnapshotCMA is that a continuously-adapting
// filter destroys a differential decode: a time-varying output phase does not
// cancel in s·conj(prev), so every dibit is corrupted. That failure mode comes
// from CMA's cost being rotation-invariant — nothing constrains its absolute
// output phase, so it wanders freely as the taps adapt.
//
// This combiner is structurally different. The reference branch's weight is
// pinned to exactly 1+0j, so the output phase is ANCHORED to the reference
// branch's own phase. Writing x1 = h·x0 + n with estimate ĥ = h·e^{jε},
//
//	y = (x0 + conj(ĥ)·x1)/(1+|ĥ|²)  ⇒  arg(y/x0) ≈ -ε·|h|²/(1+|h|²)
//
// i.e. at most about ε/2, and exactly zero when the estimate is right. The only
// phase wander is the wander of the estimate ERROR, which the coherence gate
// bounds, the one-pole damps by alpha, and trackingMaxStepRad clamps outright.
// At a 2 ms window and alpha 0.01 the per-burst perturbation is a hundredth of
// a degree against a 45° pi/4-DQPSK decision spacing. Pinned by
// TestTrackingCalibratorIsDifferentialSafe.
//
// # Degradation
//
// On a shared-LO front-end the true h is constant, so every window returns the
// same estimate and the one-pole is a variance-reducing no-op: steady state is
// that same constant, measured over far more samples than a single datagram.
// With Alpha 0 the first accepted window is frozen and the behaviour is
// StaticCalibrator's, bit-for-bit on the same window.
//
// Not safe for concurrent use; the owner is the single stream goroutine.
type TrackingCalibrator struct {
	branches int
	opts     TrackingOptions
	mrc      *MRC

	// stats[k] relates branch k+1 to the reference (branch 0).
	stats []CrossStats
	// applied are the gains currently pushed into mrc; applied[0] == 1.
	applied []complex64

	calibrated bool
	coherence  float64
	updates    uint64
	holds      uint64
	holdStreak int
}

// NewTrackingCalibrator builds a tracking coherent combiner for `branches`
// inputs. Branch 0 is the reference; every other branch is aligned to it.
func NewTrackingCalibrator(branches int, opts TrackingOptions) *TrackingCalibrator {
	if branches <= 0 {
		panic("diversity: TrackingCalibrator needs at least one branch")
	}
	c := &TrackingCalibrator{
		branches: branches,
		opts:     opts.withDefaults(),
		// alpha is irrelevant to the wrapped MRC: it is driven purely in pilot
		// mode via SetGain and never smooths power itself.
		mrc:     NewMRC(branches, 0.05),
		stats:   make([]CrossStats, branches-1),
		applied: make([]complex64, branches),
	}
	c.applied[0] = complex(1, 0)
	return c
}

// Observe feeds one chunk of branch samples (reference at index 0) to the
// estimator. The calibrator accumulates internally and decides, on its own
// window schedule and gates, whether to move the applied gains.
func (c *TrackingCalibrator) Observe(branches [][]complex64) ObserveResult {
	res := ObserveResult{Gains: c.applied}
	if len(branches) != c.branches || c.branches < 2 {
		return res
	}
	for k := 1; k < c.branches; k++ {
		c.stats[k-1].Accumulate(branches[0], branches[k])
	}
	if c.stats[0].Samples() < c.opts.WindowSamples {
		return res
	}
	res.WindowDone = true
	res.Updated = c.estimate()
	res.Coherence = c.coherence
	for k := range c.stats {
		c.stats[k].Reset()
	}
	return res
}

// estimate runs one window's decision. Returns whether the applied gains moved.
//
// A rejected window HOLDS the previous gains — it never falls back to
// reference-only. Dropping to passthrough mid-stream is itself a large phase
// discontinuity, which is the exact failure class this design is avoiding; a
// stale-but-bounded weight is strictly better than a step. (This is a
// deliberate divergence from SnapshotCMA's re-seed-to-passthrough guard: CMA's
// taps can diverge without bound, whereas a clamped |h| cannot.) The only reset
// is Reset(), on a retune or a fresh stream, where the stream is already
// discontinuous.
func (c *TrackingCalibrator) estimate() bool {
	// Freeze once locked in one-shot mode.
	if c.calibrated && c.opts.Alpha == 0 {
		return false
	}

	// The gate is derived from the window's actual sample count, so the same
	// options hold every stream to the same ESTIMATE quality regardless of
	// sample rate or how far a chunk overshot the nominal window.
	n := c.stats[0].Samples()
	gate := coherenceGateFor(c.opts.TrackPhaseSigmaRad, n)
	if !c.calibrated {
		gate = coherenceGateFor(c.opts.LockPhaseSigmaRad, n)
	}

	// The window's coherence is the weakest branch's: one incoherent branch is
	// enough to make the estimate untrustworthy.
	worst := math.Inf(1)
	proposed := make([]complex64, c.branches)
	proposed[0] = complex(1, 0)
	for k := 1; k < c.branches; k++ {
		s := &c.stats[k-1]
		rho := s.Coherence()
		if rho < worst {
			worst = rho
		}
		if s.RefPower() < c.opts.MinBranchPower || s.BranchPower() < c.opts.MinBranchPower {
			c.hold(rho)
			return false
		}
		h, ok := s.Gain()
		if !ok || !finiteWithin(h, trackingDivergeFloorSq, trackingDivergeCeilSq) {
			c.hold(rho)
			return false
		}
		proposed[k] = h
	}
	c.coherence = worst
	if worst < gate {
		c.hold(worst)
		return false
	}

	for k := 1; k < c.branches; k++ {
		if !c.calibrated {
			// First accepted window: snap. No smoothing and no clamp, so this is
			// bit-identical to a one-shot least-squares calibration on the same
			// window — a shared-LO device and a drifting one start the same way.
			c.applied[k] = proposed[k]
			continue
		}
		target := smoothGain(c.applied[k], proposed[k], c.opts.Alpha)
		c.applied[k] = clampStep(c.applied[k], target)
	}
	for k := 0; k < c.branches; k++ {
		c.mrc.SetGain(k, c.applied[k])
	}
	c.calibrated = true
	c.updates++
	c.holdStreak = 0
	return true
}

func (c *TrackingCalibrator) hold(rho float64) {
	c.coherence = rho
	c.holds++
	c.holdStreak++
}

// Combine emits one coherently-combined sample per input sample index. Before
// the first accepted window it returns a copy of the reference branch, so an
// uncalibrated pipeline sees the primary receiver verbatim rather than a
// phase-blind sum that could cancel.
func (c *TrackingCalibrator) Combine(branches [][]complex64) ([]complex64, error) {
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

// Calibrated reports whether a usable gain estimate has been accepted.
func (c *TrackingCalibrator) Calibrated() bool { return c.calibrated }

// Coherence is |rho| of the last completed window (0 before the first).
func (c *TrackingCalibrator) Coherence() float64 { return c.coherence }

// Counters returns how many windows were accepted and how many were held back
// by a gate. A climbing hold count against a flat update count is the signature
// of two receivers that are not seeing the same signal.
func (c *TrackingCalibrator) Counters() (updates, holds uint64) { return c.updates, c.holds }

// Gains returns a copy of the applied per-branch complex gains (Gains[0] == 1).
func (c *TrackingCalibrator) Gains() []complex64 {
	out := make([]complex64, len(c.applied))
	copy(out, c.applied)
	return out
}

// Reset drops the estimate and re-arms. Call on every retune: a new LO lock
// re-randomizes the front-end's divider phase, invalidating the old constant.
func (c *TrackingCalibrator) Reset() {
	for i := range c.applied {
		c.applied[i] = 0
	}
	if len(c.applied) > 0 {
		c.applied[0] = complex(1, 0)
	}
	for k := range c.stats {
		c.stats[k].Reset()
	}
	c.calibrated = false
	c.coherence = 0
	c.holdStreak = 0
	c.mrc.Reset()
}

// smoothGain is the one-pole update h <- (1-a)h + a*target.
func smoothGain(cur, target complex64, a float64) complex64 {
	ar := float32(a)
	return complex(
		real(cur)*(1-ar)+real(target)*ar,
		imag(cur)*(1-ar)+imag(target)*ar,
	)
}

// clampStep bounds how far one update may move the applied gain: at most
// trackingMaxStepRad of phase and a factor trackingMaxStepRatio of magnitude.
// Returns `want` unchanged when it is already within bounds.
func clampStep(cur, want complex64) complex64 {
	curMag := math.Hypot(float64(real(cur)), float64(imag(cur)))
	wantMag := math.Hypot(float64(real(want)), float64(imag(want)))
	if curMag == 0 || wantMag == 0 {
		return want
	}
	curPh := math.Atan2(float64(imag(cur)), float64(real(cur)))
	wantPh := math.Atan2(float64(imag(want)), float64(real(want)))
	dPh := wrapPi(wantPh - curPh)
	if dPh > trackingMaxStepRad {
		dPh = trackingMaxStepRad
	} else if dPh < -trackingMaxStepRad {
		dPh = -trackingMaxStepRad
	}
	mag := wantMag
	if mag > curMag*trackingMaxStepRatio {
		mag = curMag * trackingMaxStepRatio
	} else if mag < curMag/trackingMaxStepRatio {
		mag = curMag / trackingMaxStepRatio
	}
	ph := curPh + dPh
	return complex(float32(mag*math.Cos(ph)), float32(mag*math.Sin(ph)))
}

func wrapPi(x float64) float64 {
	for x > math.Pi {
		x -= 2 * math.Pi
	}
	for x < -math.Pi {
		x += 2 * math.Pi
	}
	return x
}

func finiteWithin(h complex64, loSq, hiSq float64) bool {
	if !isFinite(h) {
		return false
	}
	m := mag2(h)
	return m >= loSq && m <= hiSq
}
