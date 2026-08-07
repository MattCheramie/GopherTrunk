package diversity

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// --- test signal helpers -------------------------------------------------

// genSignal returns n unit-magnitude samples with random phase — a proxy
// for a modulated carrier feeding the reference branch.
func genSignal(rng *rand.Rand, n int) []complex64 {
	s := make([]complex64, n)
	for i := range s {
		ph := rng.Float64() * 2 * math.Pi
		s[i] = complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
	}
	return s
}

// rotate applies a constant phase rotation theta to every sample — the
// static hardware phase delta a shared-LO front-end imposes on branch k.
func rotate(s []complex64, theta float64) []complex64 {
	r := complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	out := make([]complex64, len(s))
	for i := range s {
		out[i] = s[i] * r
	}
	return out
}

// addNoise returns s plus i.i.d. complex Gaussian noise; sigma is the
// per-axis standard deviation.
func addNoise(rng *rand.Rand, s []complex64, sigma float64) []complex64 {
	out := make([]complex64, len(s))
	for i := range s {
		nr := rng.NormFloat64() * sigma
		ni := rng.NormFloat64() * sigma
		out[i] = s[i] + complex(float32(nr), float32(ni))
	}
	return out
}

// errPower returns sum|a[i]-ref[i]|^2.
func errPower(a, ref []complex64) float64 {
	var e float64
	for i := range a {
		e += mag2(a[i] - ref[i])
	}
	return e
}

// --- tests ---------------------------------------------------------------

// Compile-time proof StaticCalibrator is a drop-in Combiner.
var _ Combiner = (*StaticCalibrator)(nil)

// TestStaticCalibratorRecoversStaticPhase pins the core estimator: from a
// clean two-branch window where branch 1 is branch 0 rotated by a fixed
// angle, Calibrate must recover that angle and Combine must coherently
// reconstruct the reference waveform (not a faded/cancelled version).
func TestStaticCalibratorRecoversStaticPhase(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const theta = 150.0 * math.Pi / 180.0
	s := genSignal(rng, 512)
	b0 := s
	b1 := rotate(s, theta)

	c := NewStaticCalibrator(2)
	if err := c.Calibrate([][]complex64{b0, b1}); err != nil {
		t.Fatal(err)
	}

	// h_0 must be exactly 1; h_1 must be e^{jtheta}.
	g := c.Gains()
	if cmplx.Abs(complex128(g[0])-1) > 1e-5 {
		t.Errorf("h_0 = %v, want 1", g[0])
	}
	wantH1 := cmplx.Rect(1, theta)
	if cmplx.Abs(complex128(g[1])-wantH1) > 1e-3 {
		t.Errorf("h_1 = %v, want %v (e^{j150°})", g[1], wantH1)
	}

	out, err := c.Combine([][]complex64{b0, b1})
	if err != nil {
		t.Fatal(err)
	}
	// Coherent MRC of two noise-free copies reconstructs the reference.
	if e := errPower(out, s) / float64(len(s)); e > 1e-6 {
		t.Errorf("mean |out-s|^2 = %g, want ~0 (reference reconstructed)", e)
	}
}

// TestStaticCalibratorSNRGainOverSingleBranch is the payoff test: after a
// one-time calibration on a clean sync window, combining two equal-power
// noisy branches must beat the single reference branch by ~3 dB (the
// ideal for two equal branches). Fails first without the coherent
// alignment — a phase-blind sum of rotated branches does not gain 3 dB.
func TestStaticCalibratorSNRGainOverSingleBranch(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	const theta = 73.0 * math.Pi / 180.0
	const sigma = 0.30
	const n = 8192

	// Calibrate on a high-SNR sync burst (noise-free copies) — you only
	// lock on a good sync, so the calibration window is clean.
	cal := genSignal(rng, 256)
	c := NewStaticCalibrator(2)
	if err := c.Calibrate([][]complex64{cal, rotate(cal, theta)}); err != nil {
		t.Fatal(err)
	}

	// Payload: same signal seen on both branches with independent noise,
	// branch 1 statically rotated.
	s := genSignal(rng, n)
	b0 := addNoise(rng, s, sigma)
	b1 := addNoise(rng, rotate(s, theta), sigma)

	out, err := c.Combine([][]complex64{b0, b1})
	if err != nil {
		t.Fatal(err)
	}

	errBranch0 := errPower(b0, s)
	errOut := errPower(out, s)
	ratio := errOut / errBranch0
	gainDB := 10 * math.Log10(errBranch0/errOut)
	t.Logf("noise-power ratio out/branch0 = %.3f (%.2f dB gain)", ratio, gainDB)

	if ratio > 0.60 { // ideal is 0.5; margin for finite-N + estimation error
		t.Errorf("combined noise ratio = %.3f, want < 0.60 (>~2.2 dB gain)", ratio)
	}
	if ratio < 0.40 { // sanity: shouldn't be *better* than the 3 dB ideal
		t.Errorf("combined noise ratio = %.3f, implausibly low", ratio)
	}
}

// TestStaticCalibratorAntiPhaseWouldCancelWithoutCalibration is the
// failing-first contrast in one test: with branch 1 in anti-phase, a
// naive elementwise sum cancels the signal to ~zero, while the calibrated
// combiner reconstructs it. This is exactly the failure diversity.go
// warns about ("with two equal-power branches in anti-phase, MRC cancels
// the signal") and what the phase calibration exists to prevent.
func TestStaticCalibratorAntiPhaseWouldCancelWithoutCalibration(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	s := genSignal(rng, 512)
	b0 := s
	b1 := rotate(s, math.Pi) // 180° — anti-phase

	// Naive uncalibrated sum: the branches destructively cancel.
	var naiveMag float64
	for i := range s {
		naiveMag += float64(cmplx.Abs(complex128(b0[i] + b1[i])))
	}
	naiveMag /= float64(len(s))
	if naiveMag > 1e-4 {
		t.Fatalf("precondition: anti-phase branches should cancel in a "+
			"naive sum, got mean |x0+x1| = %g", naiveMag)
	}

	// Calibrated combiner: recovers the reference instead of cancelling.
	c := NewStaticCalibrator(2)
	if err := c.Calibrate([][]complex64{b0, b1}); err != nil {
		t.Fatal(err)
	}
	out, err := c.Combine([][]complex64{b0, b1})
	if err != nil {
		t.Fatal(err)
	}
	if e := errPower(out, s) / float64(len(s)); e > 1e-6 {
		t.Errorf("calibrated combine mean |out-s|^2 = %g, want ~0 "+
			"(signal recovered, not cancelled)", e)
	}
}

// TestStaticCalibratorUncalibratedPassthrough: before any calibration,
// Combine returns the reference branch verbatim in a fresh buffer — never
// a blind sum.
func TestStaticCalibratorUncalibratedPassthrough(t *testing.T) {
	c := NewStaticCalibrator(2)
	if c.Calibrated() {
		t.Fatal("fresh calibrator should not report Calibrated()")
	}
	b0 := []complex64{complex(1, 2), complex(3, 4)}
	b1 := []complex64{complex(9, 9), complex(9, 9)}
	out, err := c.Combine([][]complex64{b0, b1})
	if err != nil {
		t.Fatal(err)
	}
	for i := range b0 {
		if out[i] != b0[i] {
			t.Errorf("out[%d] = %v, want reference %v", i, out[i], b0[i])
		}
	}
	out[0] = 0 // mutating output must not touch the input branch
	if b0[0] == 0 {
		t.Error("passthrough returned a shared buffer, want a copy")
	}
}

// TestStaticCalibratorSilentReferenceRejected: a dead reference branch
// can't anchor the phase; Calibrate must error and stay uncalibrated so
// the caller retries on the next sync burst.
func TestStaticCalibratorSilentReferenceRejected(t *testing.T) {
	c := NewStaticCalibrator(2)
	b0 := make([]complex64, 16) // all zero
	b1 := make([]complex64, 16)
	for i := range b1 {
		b1[i] = complex(0.5, 0.5)
	}
	if err := c.Calibrate([][]complex64{b0, b1}); err == nil {
		t.Fatal("expected error calibrating against a silent reference")
	}
	if c.Calibrated() {
		t.Error("calibrator should remain uncalibrated after a failed Calibrate")
	}
}

// TestStaticCalibratorResetReArms: Reset drops the constant and returns
// to passthrough, matching the per-retune re-calibration lifecycle.
func TestStaticCalibratorResetReArms(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	s := genSignal(rng, 64)
	c := NewStaticCalibrator(2)
	if err := c.Calibrate([][]complex64{s, rotate(s, 1.0)}); err != nil {
		t.Fatal(err)
	}
	if !c.Calibrated() {
		t.Fatal("should be calibrated")
	}
	c.Reset()
	if c.Calibrated() {
		t.Error("Reset should clear the calibration state")
	}
	// Back to reference passthrough.
	out, err := c.Combine([][]complex64{s, rotate(s, 1.0)})
	if err != nil {
		t.Fatal(err)
	}
	if e := errPower(out, s); e != 0 {
		t.Errorf("after Reset, expected reference passthrough, err=%g", e)
	}
}

// TestStaticCalibratorSingleBranchPassthrough: one branch is a valid
// degenerate case — h_0=1, output equals the input.
func TestStaticCalibratorSingleBranchPassthrough(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	s := genSignal(rng, 128)
	c := NewStaticCalibrator(1)
	if err := c.Calibrate([][]complex64{s}); err != nil {
		t.Fatal(err)
	}
	out, err := c.Combine([][]complex64{s})
	if err != nil {
		t.Fatal(err)
	}
	if e := errPower(out, s) / float64(len(s)); e > 1e-6 {
		t.Errorf("single-branch mean |out-s|^2 = %g, want ~0", e)
	}
}

// TestStaticCalibratorRejectsShapeErrors: branch-count and length
// mismatches surface as errors from both Calibrate and Combine rather
// than silent truncation.
func TestStaticCalibratorRejectsShapeErrors(t *testing.T) {
	c := NewStaticCalibrator(2)

	// Wrong branch count.
	if err := c.Calibrate([][]complex64{{complex(1, 0)}}); err == nil {
		t.Error("Calibrate: expected branch-count error")
	}
	// Length mismatch across branches.
	if err := c.Calibrate([][]complex64{
		{complex(1, 0), complex(2, 0)},
		{complex(1, 0)},
	}); err == nil {
		t.Error("Calibrate: expected length-mismatch error")
	}

	// Same guards on Combine.
	if _, err := c.Combine([][]complex64{{complex(1, 0)}}); err == nil {
		t.Error("Combine: expected branch-count error")
	}
	if _, err := c.Combine([][]complex64{
		{complex(1, 0), complex(2, 0)},
		{complex(1, 0)},
	}); err == nil {
		t.Error("Combine: expected length-mismatch error")
	}
}

func TestNewStaticCalibratorRejectsZeroBranches(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on zero branches")
		}
	}()
	NewStaticCalibrator(0)
}
