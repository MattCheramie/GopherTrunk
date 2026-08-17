package diversity

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// twoBranchScene builds a window of two-branch samples carrying a wanted
// signal, an optional directional interferer with its own spatial signature,
// and per-branch thermal noise. Everything is expressed relative to branch 0,
// so h[0] = 1 by construction — the same convention the least-squares channel
// estimate uses.
type twoBranchScene struct {
	n int
	// wantedH1 is the wanted signal's gain at branch 1 relative to branch 0.
	wantedH1 complex128
	// interfAmp is the interferer's amplitude at branch 0 (0 disables it) and
	// interfH1 its gain at branch 1. An interferer arriving from a DIFFERENT
	// direction has a different h — that difference is the only thing a spatial
	// null can exploit.
	interfAmp float64
	interfH1  complex128
	noiseAmp  float64
	seed      int64
}

func (s twoBranchScene) build() (branches [][]complex64, wanted []complex64) {
	rng := rand.New(rand.NewSource(s.seed))
	b0 := make([]complex64, s.n)
	b1 := make([]complex64, s.n)
	want := make([]complex64, s.n)

	// Independent QPSK streams so the wanted signal and the interferer are
	// uncorrelated, as two unrelated transmitters are.
	qpsk := func() complex128 {
		const r = math.Sqrt2 / 2
		switch rng.Intn(4) {
		case 0:
			return complex(r, r)
		case 1:
			return complex(-r, r)
		case 2:
			return complex(-r, -r)
		default:
			return complex(r, -r)
		}
	}
	noise := func() complex128 {
		return complex(rng.NormFloat64()*s.noiseAmp, rng.NormFloat64()*s.noiseAmp)
	}

	for i := 0; i < s.n; i++ {
		sig := qpsk()
		var intf complex128
		if s.interfAmp > 0 {
			intf = qpsk() * complex(s.interfAmp, 0)
		}
		x0 := sig + intf + noise()
		x1 := sig*s.wantedH1 + intf*s.interfH1 + noise()
		b0[i] = complex64(x0)
		b1[i] = complex64(x1)
		want[i] = complex64(sig)
	}
	return [][]complex64{b0, b1}, want
}

// sinrDb measures how well a combined stream recovers the wanted signal, after
// removing the best complex scale (an overall gain/rotation is not an error a
// differential decoder cares about). Higher is better.
func sinrDb(got, want []complex64) float64 {
	var num complex128
	var den float64
	for i := range want {
		w := complex128(want[i])
		g := complex128(got[i])
		num += g * cmplx.Conj(w)
		den += real(w * cmplx.Conj(w))
	}
	if den == 0 {
		return math.Inf(-1)
	}
	a := num / complex(den, 0)
	var sig, err float64
	for i := range want {
		w := complex128(want[i])
		g := complex128(got[i])
		sig += real(a * w * cmplx.Conj(a*w))
		e := g - a*w
		err += real(e * cmplx.Conj(e))
	}
	if err == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(sig/err)
}

func runCalibrator(t *testing.T, cal interface {
	Observe([][]complex64) ObserveResult
	Combine([][]complex64) ([]complex64, error)
}, branches [][]complex64, chunk int) []complex64 {
	t.Helper()
	n := len(branches[0])
	out := make([]complex64, 0, n)
	for pos := 0; pos < n; pos += chunk {
		end := min(pos+chunk, n)
		part := [][]complex64{branches[0][pos:end], branches[1][pos:end]}
		cal.Observe(part)
		y, err := cal.Combine(part)
		if err != nil {
			t.Fatalf("Combine at %d: %v", pos, err)
		}
		out = append(out, y...)
	}
	return out
}

// The case IRC exists for: a directional interferer whose spatial signature
// differs from the wanted signal's. MRC has no way to tell them apart and
// combines the interferer right along with the signal; IRC nulls it — GIVEN a
// training sequence that says which signal is the wanted one.
func TestIRCRejectsADirectionalInterfererThatMRCAmplifies(t *testing.T) {
	scene := twoBranchScene{
		n:         200_000,
		wantedH1:  cmplx.Rect(0.95, 0.7),
		interfAmp: 0.8,
		// A different direction of arrival means a different relative phase.
		interfH1: cmplx.Rect(1.0, -1.9),
		noiseAmp: 0.05,
		seed:     7,
	}
	branches, want := scene.build()

	opts := TrackingOptions{WindowSamples: 8192, Alpha: 0}
	mrcOut := runCalibrator(t, NewTrackingCalibrator(2, opts), branches, 4096)
	ircOut := runTrainedIRC(t, NewIRCCalibrator(2, opts), branches, want, 4096)

	// Score only the tail, after both have locked on their first window.
	tail := len(want) / 2
	mrcDb := sinrDb(mrcOut[tail:], want[tail:])
	ircDb := sinrDb(ircOut[tail:], want[tail:])
	t.Logf("SINR: mrc=%.1f dB trained-irc=%.1f dB (gain %.1f dB)", mrcDb, ircDb, ircDb-mrcDb)

	if ircDb < mrcDb+6 {
		t.Errorf("trained IRC gained only %.1f dB over MRC against a directional "+
			"interferer; want at least 6 dB", ircDb-mrcDb)
	}
}

// The blind arm on the SAME scene, which is the measurement that says where
// this algorithm can and cannot live. Without a reference the channel estimate
// is cov(x_k,x_0)/var(x_0) = a power-weighted blend of the wanted and
// interfering channels, so the null is steered at a mixture and buys nothing.
// The driver combines the wideband stream, where no training sequence exists —
// so this is why IRC is not offered there.
func TestIRCBlindCannotSeparateCoChannelSignals(t *testing.T) {
	scene := twoBranchScene{
		n:         200_000,
		wantedH1:  cmplx.Rect(0.95, 0.7),
		interfAmp: 0.8,
		interfH1:  cmplx.Rect(1.0, -1.9),
		noiseAmp:  0.05,
		seed:      7,
	}
	branches, want := scene.build()
	opts := TrackingOptions{WindowSamples: 8192, Alpha: 0}
	mrcOut := runCalibrator(t, NewTrackingCalibrator(2, opts), branches, 4096)
	blindOut := runCalibrator(t, NewIRCCalibrator(2, opts), branches, 4096)

	tail := len(want) / 2
	mrcDb := sinrDb(mrcOut[tail:], want[tail:])
	blindDb := sinrDb(blindOut[tail:], want[tail:])
	t.Logf("SINR: mrc=%.1f dB blind-irc=%.1f dB (gain %.1f dB)", mrcDb, blindDb, blindDb-mrcDb)

	// Documenting the ceiling, not asserting a defect: if a future blind
	// estimator ever beats MRC by a real margin here, this test should fail so
	// the claim gets re-examined rather than quietly assumed still true.
	if blindDb > mrcDb+3 {
		t.Errorf("blind IRC gained %.1f dB over MRC — the co-channel contamination "+
			"argument in irc.go needs revisiting", blindDb-mrcDb)
	}
}

// runTrainedIRC drives the reference-fed path, handing the calibrator the known
// wanted-signal samples alongside each chunk.
func runTrainedIRC(t *testing.T, cal *IRCCalibrator, branches [][]complex64,
	reference []complex64, chunk int) []complex64 {
	t.Helper()
	n := len(branches[0])
	out := make([]complex64, 0, n)
	for pos := 0; pos < n; pos += chunk {
		end := min(pos+chunk, n)
		part := [][]complex64{branches[0][pos:end], branches[1][pos:end]}
		cal.ObserveWithReference(part, reference[pos:end])
		y, err := cal.Combine(part)
		if err != nil {
			t.Fatalf("Combine at %d: %v", pos, err)
		}
		out = append(out, y...)
	}
	return out
}

// The blind formulation the RFC proposed — residuals taken against the
// reference branch — cannot do this: h_0 = 1 by construction makes e_0
// identically zero, R_nn rank-deficient, and no null can form. This pins that
// the shipped residual is taken against the wanted-signal estimate instead, by
// checking the covariance actually has a non-trivial reference row.
func TestIRCResidualIsNotDegenerateAtTheReferenceBranch(t *testing.T) {
	scene := twoBranchScene{
		n: 20_000, wantedH1: cmplx.Rect(1.0, 0.4),
		interfAmp: 0.7, interfH1: cmplx.Rect(0.9, -1.5),
		noiseAmp: 0.05, seed: 11,
	}
	branches, _ := scene.build()
	h := [2]complex128{1, cmplx.Rect(1.0, 0.4)}

	w, ok := ircWeights(branches, h)
	if !ok {
		t.Fatal("ircWeights rejected a healthy window")
	}
	// A degenerate R_nn leaves the reference weight set purely by the loading
	// constant, which makes it enormous relative to the other branch's.
	ratio := cmplx.Abs(w[0]) / cmplx.Abs(w[1])
	if ratio > 100 || ratio < 0.01 {
		t.Errorf("weight ratio |w0|/|w1| = %.3g — looks like a rank-deficient R_nn", ratio)
	}
	// And the distortionless-response normalisation must hold.
	resp := cmplx.Conj(w[0])*h[0] + cmplx.Conj(w[1])*h[1]
	if math.Abs(cmplx.Abs(resp)-1) > 1e-9 {
		t.Errorf("w^H h = %v, want unit magnitude", resp)
	}
}

// π/4-DQPSK sits behind a differential decoder, where a time-VARYING output
// phase does not cancel in s·conj(prev) and corrupts every dibit. The weights
// are anchored to branch 0 (h_0 = 1, w^H h = 1), so the output phase is pinned
// to the reference receiver's own and only the estimate error can move it.
// Same property TestTrackingCalibratorIsDifferentialSafe pins for MRC.
func TestIRCCalibratorIsDifferentialSafe(t *testing.T) {
	scene := twoBranchScene{
		n: 300_000, wantedH1: cmplx.Rect(0.9, 1.1),
		interfAmp: 0.5, interfH1: cmplx.Rect(1.1, -0.8),
		noiseAmp: 0.08, seed: 3,
	}
	branches, _ := scene.build()

	cal := NewIRCCalibrator(2, TrackingOptions{WindowSamples: 8192})
	const chunk = 8192
	var prev complex128
	var maxStep float64
	for pos := 0; pos+chunk <= len(branches[0]); pos += chunk {
		part := [][]complex64{branches[0][pos : pos+chunk], branches[1][pos : pos+chunk]}
		cal.Observe(part)
		if !cal.Calibrated() {
			continue
		}
		w := cal.Weights()
		// The output phase for a wanted signal at branch 0 is arg(w^H h); with
		// the normalisation that is arg(1) at convergence, so track arg(w0).
		cur := complex128(w[0])
		if prev != 0 {
			step := math.Abs(cmplx.Phase(cur / prev))
			if step > maxStep {
				maxStep = step
			}
		}
		prev = cur
	}
	// π/4-DQPSK decisions are 45° apart; a per-window step must stay far below
	// that or the differential decode is corrupted by the combiner itself.
	const limitRad = 5 * math.Pi / 180
	if maxStep > limitRad {
		t.Errorf("largest per-window output phase step = %.2f°, want < %.0f°",
			maxStep*180/math.Pi, limitRad*180/math.Pi)
	}
}

// Before the first accepted window the combiner must emit the reference branch
// verbatim — never a phase-blind sum, which can cancel outright.
func TestIRCUncalibratedPassesThroughTheReference(t *testing.T) {
	cal := NewIRCCalibrator(2, TrackingOptions{WindowSamples: 1 << 20})
	branches := [][]complex64{
		{complex(1, 0), complex(0, 1)},
		{complex(-1, 0), complex(0, -1)}, // exactly anti-phase: a blind sum is zero
	}
	cal.Observe(branches)
	out, err := cal.Combine(branches)
	if err != nil {
		t.Fatal(err)
	}
	for i := range out {
		if out[i] != branches[0][i] {
			t.Fatalf("out[%d] = %v, want the reference branch %v", i, out[i], branches[0][i])
		}
	}
}

// A digitally dead branch (all zeros) has no usable estimate; the window must
// be held rather than producing garbage weights.
func TestIRCHoldsOnADeadBranch(t *testing.T) {
	cal := NewIRCCalibrator(2, TrackingOptions{WindowSamples: 1024})
	n := 2048
	live := make([]complex64, n)
	dead := make([]complex64, n)
	rng := rand.New(rand.NewSource(1))
	for i := range live {
		live[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
	}
	cal.Observe([][]complex64{live, dead})
	if cal.Calibrated() {
		t.Error("calibrated against a dead branch")
	}
	if _, holds := cal.Counters(); holds == 0 {
		t.Error("dead branch was not counted as a held window")
	}
}

// An incoherent pair — two receivers not seeing the same signal — must not
// calibrate, the same gate TrackingCalibrator applies.
func TestIRCHoldsWhenBranchesAreIncoherent(t *testing.T) {
	cal := NewIRCCalibrator(2, TrackingOptions{WindowSamples: 4096})
	n := 8192
	a := make([]complex64, n)
	b := make([]complex64, n)
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < n; i++ {
		a[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
		b[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
	}
	cal.Observe([][]complex64{a, b})
	if cal.Calibrated() {
		t.Errorf("calibrated on independent noise (coherence %.3f)", cal.Coherence())
	}
}

func TestIRCResetRearms(t *testing.T) {
	scene := twoBranchScene{n: 20_000, wantedH1: cmplx.Rect(1, 0.3), noiseAmp: 0.05, seed: 5}
	branches, _ := scene.build()
	cal := NewIRCCalibrator(2, TrackingOptions{WindowSamples: 8192})
	runCalibrator(t, cal, branches, 4096)
	if !cal.Calibrated() {
		t.Fatal("did not calibrate on a clean scene")
	}
	cal.Reset()
	if cal.Calibrated() || cal.Coherence() != 0 {
		t.Error("Reset did not re-arm the calibrator")
	}
	w := cal.Weights()
	if w[0] != complex(1, 0) || w[1] != 0 {
		t.Errorf("weights after Reset = %v, want reference passthrough", w)
	}
}

// With no interferer IRC must not be WORSE than MRC — an operator turning it on
// in a clean environment should not pay for it.
func TestIRCMatchesMRCWithoutAnInterferer(t *testing.T) {
	scene := twoBranchScene{
		n: 200_000, wantedH1: cmplx.Rect(0.9, 0.6), noiseAmp: 0.12, seed: 13,
	}
	branches, want := scene.build()
	opts := TrackingOptions{WindowSamples: 8192, Alpha: 0}
	mrcOut := runCalibrator(t, NewTrackingCalibrator(2, opts), branches, 4096)
	ircOut := runCalibrator(t, NewIRCCalibrator(2, opts), branches, 4096)

	tail := len(want) / 2
	mrcDb := sinrDb(mrcOut[tail:], want[tail:])
	ircDb := sinrDb(ircOut[tail:], want[tail:])
	t.Logf("clean scene SINR: mrc=%.1f dB irc=%.1f dB", mrcDb, ircDb)
	if ircDb < mrcDb-1 {
		t.Errorf("IRC lost %.1f dB to MRC on a clean scene", mrcDb-ircDb)
	}
}
