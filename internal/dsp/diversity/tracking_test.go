package diversity

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// feed pushes a signal pair through a calibrator in windowSamples-sized chunks,
// returning the concatenated combined output.
func feed(t *testing.T, c *TrackingCalibrator, ref, x []complex64, chunk int) []complex64 {
	t.Helper()
	var out []complex64
	for i := 0; i < len(ref); i += chunk {
		end := min(i+chunk, len(ref))
		br := [][]complex64{ref[i:end], x[i:end]}
		c.Observe(br)
		y, err := c.Combine(br)
		if err != nil {
			t.Fatalf("Combine: %v", err)
		}
		out = append(out, y...)
	}
	return out
}

// drift returns s rotated by a phase that ramps linearly from ph0 to ph1 — two
// independently-locked front-end PLLs walking apart.
func drift(s []complex64, ph0, ph1 float64) []complex64 {
	out := make([]complex64, len(s))
	for i := range s {
		f := float64(i) / float64(max1(len(s)-1))
		th := ph0 + (ph1-ph0)*f
		r := complex(float32(math.Cos(th)), float32(math.Sin(th)))
		out[i] = s[i] * r
	}
	return out
}

// residualPhaseDeg is the mean phase of y relative to ref over a span — how far
// the combined output has rotated away from the reference branch it is anchored
// to.
func residualPhaseDeg(y, ref []complex64) float64 {
	var acc complex128
	for i := range y {
		acc += complex128(y[i]) * cmplx.Conj(complex128(ref[i]))
	}
	if acc == 0 {
		return 0
	}
	return cmplx.Phase(acc) * 180 / math.Pi
}

// TestTrackingCalibratorFollowsDriftingPhase is the failing-first regression for
// independent-LO hardware: branch 1's phase walks 90° across the run, which a
// frozen constant cannot follow. It asserts both halves in one test — tracking
// stays aligned, and a static calibration on the same data does not.
func TestTrackingCalibratorFollowsDriftingPhase(t *testing.T) {
	const n = 1 << 19
	const window = 8192
	rng := rand.New(rand.NewSource(101))
	ref := genSignal(rng, n)
	// ~14 dB per-branch SNR: comfortably coherent, still noisy.
	x := addNoise(rng, drift(ref, 0.3, 0.3+math.Pi/2), 0.14)
	refN := addNoise(rng, ref, 0.14)

	track := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 0.2})
	yTrack := feed(t, track, refN, x, window)

	// Measure over the last eighth, after the drift has had its full effect.
	tail := n - n/8
	if got := math.Abs(residualPhaseDeg(yTrack[tail:], refN[tail:])); got > 3 {
		t.Errorf("tracking residual phase %.2f°, want <=3° (the loop is not following the drift)", got)
	}

	// A one-shot calibration (Alpha 0) locks on the first window and is stranded.
	static := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 0})
	yStatic := feed(t, static, refN, x, window)
	staticResid := math.Abs(residualPhaseDeg(yStatic[tail:], refN[tail:]))
	if staticResid < 10 {
		t.Fatalf("static residual phase %.2f° — fixture does not exercise drift", staticResid)
	}

	// And the drift costs static real combining gain: compare each arm's error
	// against the clean reference over the tail.
	errTrack := errPower(yTrack[tail:], ref[tail:])
	errStatic := errPower(yStatic[tail:], ref[tail:])
	if errTrack >= errStatic {
		t.Errorf("tracking error %.4g not better than static %.4g under drift", errTrack, errStatic)
	}
}

// genNarrowband returns a unit-magnitude carrier at normalised frequency f0
// (cycles per sample) whose phase also takes a small random walk — a proxy for
// one modulated channel occupying a small slice of a wideband stream.
func genNarrowband(rng *rand.Rand, n int, f0, walk float64) []complex64 {
	s := make([]complex64, n)
	th := 0.0
	for i := range s {
		th += 2*math.Pi*f0 + rng.NormFloat64()*walk
		s[i] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
	}
	return s
}

// TestTrackingCalibratorLocksOnBandwidthDilutedCoherence is the failing-first
// regression for the 18 Aug 2026 X310 report: MRC refused to calibrate at
// 200/250 kHz until the operator raised RF gain 5 dB purely to push wideband
// coherence past the fixed 0.5 lock constant. |rho| on the WIDEBAND stream is
// diluted by every hertz of noise-only bandwidth around the coherent carrier —
// rho_wb ~ rho_ch*sqrt(f0*f1) for in-channel power fractions f_k — so a fixed
// rho threshold is a bandwidth-staging trap, the direct sibling of the
// -40 dBFS gain-staging trap the coherence gate replaced. On those captures the
// per-window rho sat at ~0.16 while the branch phase was measurable to ~4
// degrees; the gate, not the estimate, was the blocker.
//
// The fixture reproduces that regime: a narrowband common carrier in a wide
// noise floor, the second branch ~9 dB down with its own broadband noise, so
// wideband coherence sits well below BOTH old fixed gates (0.5 lock, 0.35
// track) while the phase estimate is accurate. The calibrator must lock, keep
// updating, and recover the true branch phase.
func TestTrackingCalibratorLocksOnBandwidthDilutedCoherence(t *testing.T) {
	const n = 1 << 17
	const window = 8192
	const theta = 1.1 // true branch phase, rad
	rng := rand.New(rand.NewSource(909))

	carrier := genNarrowband(rng, n, 0.11, 0.02)
	// Reference branch: carrier-dominated (the operator's br1: 96% in-channel).
	ref := addNoise(rng, carrier, 0.1)
	// Other branch: the same carrier 16.5 dB down in amplitude under a
	// broadband floor (the operator's br0: 14% in-channel power fraction).
	x := addNoise(rng, rotate(scale(carrier, 0.15), theta), 0.5)

	// Fixture sanity: the wideband coherence really is below both old fixed
	// gates, or this test is not exercising the dilution regime at all.
	var s CrossStats
	s.Accumulate(ref[:window], x[:window])
	if rho := s.Coherence(); rho >= 0.35 || rho < 0.10 {
		t.Fatalf("fixture wideband coherence %.3f, want the diluted regime [0.10, 0.35)", rho)
	}

	// Alpha as the driver would derive it for a ~20 ms window (mrcTrackAlpha).
	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 0.1})
	feed(t, c, ref, x, window)

	if !c.Calibrated() {
		t.Fatalf("did not calibrate on a bandwidth-diluted but phase-measurable pair "+
			"(coherence %.3f, lock gate %.3f)", c.Coherence(), TrackingOptions{}.LockGate(window))
	}
	updates, _ := c.Counters()
	if updates < 2 {
		t.Errorf("updates = %d, want tracking to keep accepting windows in this regime", updates)
	}
	g := c.Gains()[1]
	got := math.Atan2(float64(imag(g)), float64(real(g)))
	if d := math.Abs(wrapPi(got - theta)); d > 5*math.Pi/180 {
		t.Errorf("recovered branch phase %.1f deg, want %.1f deg within 5 deg",
			got*180/math.Pi, theta*180/math.Pi)
	}
}

// TestTrackingCalibratorNeverLocksOnIndependentNoise pins the other side of the
// relaxed gate: two branches of pure independent noise must never calibrate,
// however many windows they are given. Their coherence sits at the
// sqrt(pi/4N) estimation floor, which the lock gate clears by ~8x.
func TestTrackingCalibratorNeverLocksOnIndependentNoise(t *testing.T) {
	const window = 4096
	rng := rand.New(rand.NewSource(1010))
	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window})
	for i := 0; i < 200; i++ {
		a := addNoise(rng, make([]complex64, window), 1)
		b := addNoise(rng, make([]complex64, window), 1)
		if res := c.Observe([][]complex64{a, b}); res.Updated {
			t.Fatalf("window %d: locked on independent noise (coherence %.4f)", i, res.Coherence)
		}
	}
	if c.Calibrated() {
		t.Fatal("calibrated on independent noise")
	}
}

// TestTrackingGatesScaleWithWindow pins the gate math: the minimum acceptable
// |rho| follows the estimate quality, so it falls as 1/sqrt(N) — a long window
// may trust a small coherence — and the lock bar stays strictly above the
// track bar and well above the noise-only floor at every window length.
func TestTrackingGatesScaleWithWindow(t *testing.T) {
	var o TrackingOptions
	if got := o.LockGate(4096); math.Abs(got-0.1098) > 0.002 {
		t.Errorf("LockGate(4096) = %.4f, want ~0.110", got)
	}
	for _, n := range []int{4096, 8192, 65536, 262144} {
		lock, track := o.LockGate(n), o.TrackGate(n)
		if track >= lock {
			t.Errorf("TrackGate(%d) = %.4f >= LockGate(%d) = %.4f", n, track, n, lock)
		}
		floor := math.Sqrt(math.Pi / (4 * float64(n)))
		if lock < 7*floor {
			t.Errorf("LockGate(%d) = %.4f is under 7x the noise floor %.4f — noise could lock", n, lock, floor)
		}
		if bigger := o.LockGate(4 * n); math.Abs(bigger-lock/2) > lock*0.05 {
			t.Errorf("LockGate(%d) = %.4f, want ~half of LockGate(%d) = %.4f", 4*n, bigger, n, lock)
		}
	}
	if got := o.LockGate(0); got != 1 {
		t.Errorf("LockGate(0) = %.4f, want 1 (an empty window can justify nothing)", got)
	}
}

// TestTrackingCalibratorIsDifferentialSafe is the guard CLAUDE.md mandates for
// anything upstream of a differential decoder. The combiner's own contribution
// to s·conj(prev) must be negligible: the output phase is anchored to the
// reference branch by h_0 == 1, so a weight update may only perturb it by the
// (damped, clamped) estimate error.
func TestTrackingCalibratorIsDifferentialSafe(t *testing.T) {
	// The soapyremote driver derives alpha from the ACTUAL window duration
	// against a 200 ms tau (mrcTrackAlpha; this package cannot import the
	// driver). At a 200 kHz stream the window clamps to 4096 samples = 20.48 ms
	// ⇒ alpha = 0.1024 — 10× the package default, and the alpha the 19 Aug
	// field run actually ran. Differential safety must hold at both.
	cases := []struct {
		name  string
		alpha float64
	}{
		{"defaultAlpha", TrackingDefaultAlpha},
		{"driver200kHzAlpha", 0.1024},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const n = 1 << 18
			const window = 4096
			rng := rand.New(rand.NewSource(202))
			ref := genSignal(rng, n)
			// Constant channel: any residual output-phase motion is the loop's own.
			x := addNoise(rng, rotate(ref, 1.05), 0.3)
			refN := addNoise(rng, ref, 0.3)

			c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: tc.alpha})
			y := feed(t, c, refN, x, window)

			// Per-window residual phase relative to the reference branch. The step
			// between consecutive windows is what a differential decode actually sees.
			var prev float64
			var maxStep float64
			first := true
			for i := window * 2; i+window <= n; i += window { // skip the lock window
				cur := residualPhaseDeg(y[i:i+window], refN[i:i+window])
				if !first {
					if d := math.Abs(cur - prev); d > maxStep {
						maxStep = d
					}
				}
				prev, first = cur, false
			}
			// pi/4-DQPSK decisions are 45° apart. Anything approaching a degree here
			// would already be suspicious; the design budget is hundredths.
			if maxStep > 1.0 {
				t.Errorf("alpha=%g: max per-window output phase step %.3f°, want <=1° (45° decision spacing)", tc.alpha, maxStep)
			}
		})
	}
}

// TestTrackingCalibratorHoldsOnIncoherentWindow pins the hold guard: windows of
// pure noise must not move the gains at all, and must not drop the combiner
// back to passthrough (which would itself be a phase discontinuity).
func TestTrackingCalibratorHoldsOnIncoherentWindow(t *testing.T) {
	const window = 4096
	rng := rand.New(rand.NewSource(303))
	ref := genSignal(rng, window*4)
	x := addNoise(rng, rotate(ref, 0.7), 0.2)

	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 0.2})
	feed(t, c, ref, x, window)
	if !c.Calibrated() {
		t.Fatal("did not calibrate on clean data")
	}
	locked := c.Gains()
	_, holdsBefore := c.Counters()

	// 50 windows of two independent noise branches.
	const rounds = 50
	for i := 0; i < rounds; i++ {
		a := addNoise(rng, make([]complex64, window), 1)
		b := addNoise(rng, make([]complex64, window), 1)
		br := [][]complex64{a, b}
		res := c.Observe(br)
		if res.Updated {
			t.Fatalf("round %d: accepted an update from independent noise (coherence %.3f)", i, res.Coherence)
		}
		if _, err := c.Combine(br); err != nil {
			t.Fatalf("round %d: Combine: %v", i, err)
		}
	}

	got := c.Gains()
	for k := range locked {
		if got[k] != locked[k] {
			t.Errorf("gain[%d] moved from %v to %v across noise windows", k, locked[k], got[k])
		}
	}
	if _, holds := c.Counters(); holds != holdsBefore+rounds {
		t.Errorf("holds = %d, want %d", holds-holdsBefore, rounds)
	}
	if !c.Calibrated() {
		t.Error("dropped out of calibrated state on noise — that is a phase step, not a safe fallback")
	}
}

// TestTrackingFirstUpdateMatchesStaticCalibrate pins the degradation claim: the
// first accepted window is snapped, not smoothed, so it is exactly the one-shot
// least-squares estimate StaticCalibrator would freeze on the same window.
func TestTrackingFirstUpdateMatchesStaticCalibrate(t *testing.T) {
	const window = 8192
	rng := rand.New(rand.NewSource(404))
	ref := genSignal(rng, window)
	x := addNoise(rng, rotate(ref, -0.8), 0.2)

	track := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window})
	track.Observe([][]complex64{ref, x})
	if !track.Calibrated() {
		t.Fatal("tracking did not calibrate")
	}

	static := NewStaticCalibrator(2)
	if err := static.Calibrate([][]complex64{ref, x}); err != nil {
		t.Fatalf("StaticCalibrator.Calibrate: %v", err)
	}

	// StaticCalibrator's estimate is uncentred; on a DC-free fixture the two
	// agree to estimator precision.
	gt, gs := track.Gains(), static.Gains()
	if d := cmplx.Abs(complex128(gt[1]) - complex128(gs[1])); d > 1e-3 {
		t.Errorf("tracking first gain %v, static %v (|diff| %.2e)", gt[1], gs[1], d)
	}
}

// TestTrackingCalibratorOneShotFreezes pins that Alpha 0 never revisits its
// estimate — the mrc-static escape hatch must be provably one-shot.
func TestTrackingCalibratorOneShotFreezes(t *testing.T) {
	const window = 4096
	rng := rand.New(rand.NewSource(505))
	ref := genSignal(rng, window*8)
	x := addNoise(rng, drift(ref, 0, math.Pi), 0.15)

	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 0})
	feed(t, c, ref, x, window)
	locked := c.Gains()

	feed(t, c, ref, x, window)
	if got := c.Gains(); got[1] != locked[1] {
		t.Errorf("one-shot gain moved from %v to %v", locked[1], got[1])
	}
	if updates, _ := c.Counters(); updates != 1 {
		t.Errorf("updates = %d, want exactly 1", updates)
	}
}

// TestTrackingCalibratorSlewClamped pins the hard bound on a single update. An
// adversarial window proposes a gain 180° away; the applied gain may only move
// by trackingMaxStepRad however convincing that window looked.
func TestTrackingCalibratorSlewClamped(t *testing.T) {
	const window = 2048
	rng := rand.New(rand.NewSource(606))
	ref := genSignal(rng, window)
	x := rotate(ref, 0.2)

	// Alpha 1 removes the one-pole damping so only the clamp is under test.
	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: 1})
	c.Observe([][]complex64{ref, x})
	if !c.Calibrated() {
		t.Fatal("did not lock")
	}
	before := c.Gains()[1]

	flipped := rotate(ref, 0.2+math.Pi)
	c.Observe([][]complex64{ref, flipped})
	after := c.Gains()[1]

	step := math.Abs(wrapPi(cmplx.Phase(complex128(after)) - cmplx.Phase(complex128(before))))
	if step > trackingMaxStepRad+1e-6 {
		t.Errorf("applied phase step %.4f rad, want <=%.4f", step, trackingMaxStepRad)
	}
}

// TestTrackingCalibratorRejectsDeadBranch pins that a digitally silent branch
// cannot produce a gain estimate, whatever the reference is doing.
func TestTrackingCalibratorRejectsDeadBranch(t *testing.T) {
	const window = 2048
	rng := rand.New(rand.NewSource(707))
	ref := genSignal(rng, window*2)
	dead := make([]complex64, window*2)

	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window})
	feed(t, c, ref, dead, window)
	if c.Calibrated() {
		t.Error("calibrated against an all-zero branch")
	}
}

// TestTrackingCalibratorResetRearms pins the retune lifecycle.
func TestTrackingCalibratorResetRearms(t *testing.T) {
	const window = 4096
	rng := rand.New(rand.NewSource(808))
	ref := genSignal(rng, window)
	x := addNoise(rng, rotate(ref, 0.5), 0.2)

	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window})
	c.Observe([][]complex64{ref, x})
	if !c.Calibrated() {
		t.Fatal("did not calibrate")
	}
	c.Reset()
	if c.Calibrated() {
		t.Error("still calibrated after Reset")
	}
	if g := c.Gains(); g[0] != complex(1, 0) {
		t.Errorf("reference gain %v after Reset, want 1+0i", g[0])
	}
	// And it combines as passthrough again rather than with a stale weight.
	y, err := c.Combine([][]complex64{ref, x})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if errPower(y, ref) != 0 {
		t.Error("post-Reset Combine is not the reference branch verbatim")
	}
}
