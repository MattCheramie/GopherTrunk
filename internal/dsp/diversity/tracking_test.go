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

// TestTrackingCalibratorIsDifferentialSafe is the guard CLAUDE.md mandates for
// anything upstream of a differential decoder. The combiner's own contribution
// to s·conj(prev) must be negligible: the output phase is anchored to the
// reference branch by h_0 == 1, so a weight update may only perturb it by the
// (damped, clamped) estimate error.
func TestTrackingCalibratorIsDifferentialSafe(t *testing.T) {
	const n = 1 << 18
	const window = 4096
	rng := rand.New(rand.NewSource(202))
	ref := genSignal(rng, n)
	// Constant channel: any residual output-phase motion is the loop's own.
	x := addNoise(rng, rotate(ref, 1.05), 0.3)
	refN := addNoise(rng, ref, 0.3)

	c := NewTrackingCalibrator(2, TrackingOptions{WindowSamples: window, Alpha: TrackingDefaultAlpha})
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
		t.Errorf("max per-window output phase step %.3f°, want <=1° (45° decision spacing)", maxStep)
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
