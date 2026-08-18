package soapyremote

import (
	"bytes"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// mrcTestWindow is a calibration window at the default (unknown-rate) sizing,
// so a fixture that feeds this many samples per branch completes exactly one
// estimation window. The combiner deliberately no longer estimates from a
// handful of samples: a least-squares phase estimate over 184 samples (one
// datagram at the default MTU) is worth about 9.5 degrees at the tracking gate.
const mrcTestWindow = mrcCalWindowDefaultSamples

// twoBranchPayload encodes ch0 then ch1 as the contiguous per-channel blocks a
// multi-channel SoapyRemote datagram carries.
func twoBranchPayload(ch0, ch1 []complex64) []byte {
	return encodeCS16(append(append([]complex64{}, ch0...), ch1...))
}

// mrcSignal returns n samples of unit-power random-phase carrier scaled to amp.
func mrcSignal(rng *rand.Rand, n int, amp float64) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		ph := rng.Float64() * 2 * math.Pi
		out[i] = complex(float32(amp*math.Cos(ph)), float32(amp*math.Sin(ph)))
	}
	return out
}

// mrcNoise adds independent complex noise with per-axis sd sigma.
func mrcNoise(rng *rand.Rand, x []complex64, sigma float64) []complex64 {
	out := make([]complex64, len(x))
	for i := range x {
		out[i] = x[i] + complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
	}
	return out
}

// approxEqual reports whether two complex64 slices match within tol per part.
func approxEqualC(a, b []complex64, tol float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if float32(math.Abs(float64(real(a[i])-real(b[i])))) > tol ||
			float32(math.Abs(float64(imag(a[i])-imag(b[i])))) > tol {
			return false
		}
	}
	return true
}

// scale returns x with every sample multiplied by g (a rotation/gain).
func scaleC(x []complex64, g complex64) []complex64 {
	out := make([]complex64, len(x))
	for i, z := range x {
		out[i] = z * g
	}
	return out
}

func TestDeinterleaveTwoChannelCS16(t *testing.T) {
	ch0 := []complex64{complex(0.5, -0.5), complex(0.25, 0.25)}
	ch1 := []complex64{complex(-0.5, 0.5), complex(0.125, -0.125)}
	// Contiguous per-channel blocks: [ch0...][ch1...].
	payload := encodeCS16(append(append([]complex64{}, ch0...), ch1...))

	got := formatCS16.deinterleave(payload, 2, len(ch0))
	if len(got) != 2 {
		t.Fatalf("branches = %d, want 2", len(got))
	}
	if !approxEqualC(got[0], ch0, 1e-3) {
		t.Errorf("branch 0 = %v, want %v", got[0], ch0)
	}
	if !approxEqualC(got[1], ch1, 1e-3) {
		t.Errorf("branch 1 = %v, want %v", got[1], ch1)
	}
}

func TestDeinterleaveSingleChannelIsConvert(t *testing.T) {
	x := []complex64{complex(0.5, -0.5), complex(0.25, 0.25)}
	payload := encodeCS16(x)
	got := formatCS16.deinterleave(payload, 1, len(x))
	if len(got) != 1 || !approxEqualC(got[0], x, 1e-3) {
		t.Fatalf("single-channel deinterleave = %v, want one branch %v", got, x)
	}
}

// TestMRCCombinerAlignsRotatedBranch is the core issue-#1062 property: a
// second branch that is a phase-rotated copy of the reference must be phase
// aligned and add coherently, NOT cancel. With equal-amplitude branches
// ch1 = e^{jθ}·ch0, coherent MRC recovers ch0 exactly.
func TestMRCCombinerAlignsRotatedBranch(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ch0 := mrcSignal(rng, mrcTestWindow, 0.4)
	ch1 := scaleC(ch0, complex(0, 1)) // +90° rotation

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	if !m.cal.Calibrated() {
		t.Fatal("combiner did not calibrate on a coherent window")
	}
	out := m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	if !approxEqualC(out, ch0, 2e-3) {
		t.Error("combined output is not the aligned reference (a rotated branch must align, not cancel)")
	}
}

// TestMRCCombinerCalibratesWeakButCoherentBranches is the failing-first
// regression for the gain-coupling bug. Both branches sit near -55 dBFS, well
// under the -40 dBFS absolute floor the predecessor gated on, but they are
// strongly correlated — which is the only thing that actually determines whether
// a gain estimate is trustworthy. An operator should never have to raise RF gain
// to make a software gate open, which is exactly what the old floor forced
// (65.0 dB -> 82.0 dB, landing 0.2 dB the right side of the constant).
func TestMRCCombinerCalibratesWeakButCoherentBranches(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	ch0 := mrcSignal(rng, mrcTestWindow, 1.8e-3) // ~ -55 dBFS
	ch1 := mrcNoise(rng, scaleC(ch0, complex(0, 1)), 3e-4)

	if p := refPowerDbFS(ch0); p > -50 || p < -60 {
		t.Fatalf("fixture power %.1f dBFS, want ~-55 (must be under the old -40 floor)", p)
	}

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	if !m.cal.Calibrated() {
		t.Fatal("did not calibrate on a weak but strongly coherent window — " +
			"calibration is still gated on absolute power")
	}
	if got := m.health().coherence; got < 0.9 {
		t.Errorf("coherence %.3f, want >=0.9 on this fixture", got)
	}
}

// TestMRCCombinerCalibratesOnBandwidthDilutedCoherence is the failing-first
// regression for the 18 Aug 2026 X310 report — the fixed-|rho| successor to the
// -40 dBFS trap above. The common signal is strong in the reference branch but
// sits ~16 dB down in the other branch under that branch's own broadband floor,
// so the whole-stream coherence measures ~0.25: under the old fixed 0.5 lock
// gate (and the 0.35 track gate) this NEVER calibrated, and the operator raised
// RF gain 5 dB purely to push the number past the constant. The estimate
// itself is accurate to a few degrees over a full window, which is the only
// thing the gate is entitled to demand.
func TestMRCCombinerCalibratesOnBandwidthDilutedCoherence(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	theta := 0.9
	rot := complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	var rho float64
	for i := 0; i < 3; i++ {
		common := mrcSignal(rng, mrcTestWindow, 0.2)
		ch0 := mrcNoise(rng, common, 0.02)
		ch1 := mrcNoise(rng, scaleC(common, complex(0.03/0.2, 0)*rot), 0.07)
		m.combine(twoBranchPayload(ch0, ch1), len(ch0))
		rho = m.health().coherence
	}
	// Fixture sanity: genuinely in the diluted regime, below both old gates.
	if rho >= 0.35 || rho < 0.1 {
		t.Fatalf("fixture coherence %.3f, want the diluted regime [0.1, 0.35)", rho)
	}
	if !m.cal.Calibrated() {
		t.Fatalf("did not calibrate at coherence %.3f — the lock gate is still a fixed "+
			"constant rather than scaling with the estimate quality (gate %.3f)",
			rho, m.lockGate)
	}
	g := m.cal.Gains()[1]
	got := math.Atan2(float64(imag(g)), float64(real(g)))
	if d := math.Abs(got - theta); d > 6*math.Pi/180 {
		t.Errorf("recovered branch phase %.1f deg, want %.1f deg within 6 deg",
			got*180/math.Pi, theta*180/math.Pi)
	}
}

// TestMRCCombinerRefusesIncoherentLoudBranches is the other half: two loud but
// UNRELATED receivers must not calibrate. A power gate cannot tell this case
// apart from a good one — it would freeze a meaningless gain and combine two
// unrelated signals — whereas coherence rejects it outright.
func TestMRCCombinerRefusesIncoherentLoudBranches(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	// Two unrelated carriers, both loud (~-10 dBFS) and both bounded so CS16
	// encoding cannot clip and muddy the comparison.
	ch0 := mrcSignal(rng, mrcTestWindow, 0.3)
	ch1 := mrcSignal(rng, mrcTestWindow, 0.3)

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	out := m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	if m.cal.Calibrated() {
		t.Error("calibrated on two independent (loud) noise branches")
	}
	// Uncalibrated Combine yields the anchored branch verbatim — never a blind
	// sum of two unrelated receivers. Which branch is anchored is whichever came
	// out louder, so compare against that one.
	want := []([]complex64){ch0, ch1}[m.health().refIdx]
	if !approxEqualC(out, want, 1e-3) {
		t.Error("incoherent output is not the anchored branch passed through")
	}
}

// TestMRCCombinerRecalibrateOnRequest: after a retune the frozen constant is
// stale. requestRecalibrate must force a fresh estimate on the next window;
// without it, a branch whose relationship flipped (ch1 = -ch0) would cancel.
func TestMRCCombinerRecalibrateOnRequest(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	ref := mrcSignal(rng, mrcTestWindow, 0.4)

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)

	// First lock: ch1 == ch0. Estimates h1 = 1.
	m.combine(twoBranchPayload(ref, ref), len(ref))
	if !m.cal.Calibrated() {
		t.Fatal("did not calibrate on first window")
	}

	// Retune: the relationship flips to ch1 = -ch0. Re-arm calibration.
	m.requestRecalibrate()
	flipped := scaleC(ref, complex(-1, 0))
	m.combine(twoBranchPayload(ref, flipped), len(ref))
	out := m.combine(twoBranchPayload(ref, flipped), len(ref))

	// With a fresh estimate (h1 = -1) the branches add in phase → ≈ ref.
	// With a STALE h1 = 1 they would cancel to ≈ 0.
	if !approxEqualC(out, ref, 2e-3) {
		t.Error("recalibrated output is not the reference (a stale gain would cancel it to ~0)")
	}
}

// TestDeinterleaveShortLastChannel pins the upstream block layout: the stride is
// derived from (total-elems), NOT assumed equal. The first N-1 channels always
// occupy a full stride-sized block; only the last is shortened to elems. An
// equal split would return ch0's stale tail as branch 1's first sample.
func TestDeinterleaveShortLastChannel(t *testing.T) {
	ch0 := []complex64{complex(0.5, -0.5), complex(0.25, 0.25), complex(-0.3, 0.4)}
	stale := complex64(complex(-0.9, 0.9))
	ch1 := []complex64{complex(0.1, 0.2), complex(-0.2, 0.3), complex(0.4, -0.1)}
	payload := encodeCS16(append(append(append([]complex64{}, ch0...), stale), ch1...))

	got := formatCS16.deinterleave(payload, 2, len(ch1))
	if len(got) != 2 {
		t.Fatalf("branches = %d, want 2", len(got))
	}
	if !approxEqualC(got[0], ch0, 1e-3) {
		t.Errorf("branch 0 = %v, want %v (stale tail past elems must be dropped)", got[0], ch0)
	}
	if !approxEqualC(got[1], ch1, 1e-3) {
		t.Errorf("branch 1 = %v, want %v (block stride, not an equal split)", got[1], ch1)
	}
}

// TestDeinterleaveSingleChannelPayloadUnderMultiChannelRequest: when the server
// delivers only one channel's worth of samples (elems == the whole payload)
// under a 2-channel request, that payload is a valid single-receiver time
// series. Report one branch rather than force-splitting it into two half-length
// fakes, so the caller can surface the shortfall instead of combining nonsense.
func TestDeinterleaveSingleChannelPayloadUnderMultiChannelRequest(t *testing.T) {
	x := []complex64{complex(0.5, -0.5), complex(0.25, 0.25), complex(-0.3, 0.4), complex(0.1, 0.1)}
	got := formatCS16.deinterleave(encodeCS16(x), 2, len(x))
	if len(got) != 1 {
		t.Fatalf("branches = %d, want 1 (single-channel payload)", len(got))
	}
	if !approxEqualC(got[0], x, 1e-3) {
		t.Errorf("branch 0 = %v, want the payload verbatim %v", got[0], x)
	}
}

// TestMRCCombinerAnchorsOnTheLiveBranch is the #1062 field regression at the
// combiner level: with RX0 disconnected (noise) and RX1 carrying signal, the
// combiner must anchor its calibration on RX1 and emit RX1's signal. Anchoring
// on branch 0 unconditionally leaves it in passthrough forever, emitting noise.
func TestMRCCombinerAnchorsOnTheLiveBranch(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	live := mrcSignal(rng, mrcTestWindow, 0.4)
	dead := mrcNoise(rng, make([]complex64, mrcTestWindow), 1e-3)

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	m.combine(twoBranchPayload(dead, live), len(live))

	if h := m.health(); h.refIdx != 1 {
		t.Fatalf("reference branch = %d, want 1 (the live receiver)", h.refIdx)
	}
	out := m.combine(twoBranchPayload(dead, live), len(live))
	if p := refPowerDbFS(out); p < -20 {
		t.Fatalf("combined power = %.1f dBFS — a dead RX0 must not silence the stream", p)
	}
}

// TestMRCCombinerReferenceDoesNotFlipOnPowerCrossover pins the sticky anchor.
// Two healthy branches whose levels cross back and forth by ~1 dB must not move
// the phase anchor: the predecessor re-ran argmax on every datagram, so an
// ordinary crossover swapped receivers and handed the demodulator an arbitrary
// phase step. Only a persistently dead reference may move it.
func TestMRCCombinerReferenceDoesNotFlipOnPowerCrossover(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)

	// Short chunks so no estimation window ever completes: the anchor must be
	// stable while uncalibrated, which is precisely when the old code flapped.
	const chunk = 256
	var first, flips int
	first = -1
	for i := 0; i < 40; i++ {
		a, b := 0.40, 0.36 // ~1 dB apart
		if i%2 == 1 {
			a, b = b, a
		}
		ch0 := mrcSignal(rng, chunk, a)
		ch1 := mrcSignal(rng, chunk, b)
		m.combine(twoBranchPayload(ch0, ch1), chunk)
		got := m.health().refIdx
		if first < 0 {
			first = got
		} else if got != first {
			flips++
		}
	}
	if flips != 0 {
		t.Errorf("reference branch moved %d times across a ~1 dB power crossover, want 0", flips)
	}
}

// TestMRCCombinerSingleChannelPayloadPassesThrough: a server that honours only
// one channel of the 2-channel request must yield that receiver verbatim (and
// be flagged), not a spliced half-and-half stream.
func TestMRCCombinerSingleChannelPayloadPassesThrough(t *testing.T) {
	x := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35), complex(0.1, 0.1)}
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	out := m.combine(encodeCS16(x), len(x))

	if !approxEqualC(out, x, 1e-3) {
		t.Errorf("out = %v, want the payload verbatim %v", out, x)
	}
	h := m.health()
	if !h.degenerate || h.branches != 1 {
		t.Errorf("health = %+v, want degenerate with 1 branch delivered", h)
	}
}

// TestDiversityReporterWarnsOncePerInterval: the first datagram always reports
// (so enabling diversity shows both branch levels immediately), a dead branch
// reports at WARN, and the line is throttled thereafter.
func TestDiversityReporterWarnsOncePerInterval(t *testing.T) {
	live := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35)}
	dead := scaleC(live, complex(1e-3, 0))
	payload := encodeCS16(append(append([]complex64{}, live...), dead...))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	_ = m.combine(payload, len(live))

	var buf bytes.Buffer
	r := newDiversityReporter(slog.New(slog.NewTextHandler(&buf, nil)), "10.0.0.5:55132")
	now := time.Unix(1700000000, 0)
	r.now = func() time.Time { return now }

	r.observe(m)
	first := buf.String()
	if !strings.Contains(first, "level=WARN") || !strings.Contains(first, "dead_branch=1") {
		t.Errorf("first report = %q, want a WARN naming the dead branch", first)
	}
	if !strings.Contains(first, "ch0=") || !strings.Contains(first, "ch1=") {
		t.Errorf("first report = %q, want per-branch levels", first)
	}

	buf.Reset()
	now = now.Add(mrcHealthInterval / 2)
	r.observe(m)
	if buf.Len() != 0 {
		t.Errorf("reported again inside the throttle interval: %q", buf.String())
	}

	buf.Reset()
	now = now.Add(mrcHealthInterval)
	r.observe(m)
	if buf.Len() == 0 {
		t.Error("no report after the throttle interval elapsed")
	}
}

func TestParseDiversity(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want diversityMode
	}{
		{"mrc", diversityMRCTracking},
		{"MRC", diversityMRCTracking},
		{" mrc ", diversityMRCTracking},
		{"mrc-static", diversityMRCStatic},
		{"MRC-Static", diversityMRCStatic},
		{"mrc_static", diversityMRCStatic},
		{"", diversityOff},
		{"none", diversityOff},
		{"off", diversityOff},
		{"OFF", diversityOff},
	} {
		got, err := parseDiversity(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("parseDiversity(%q) = %v, %v; want %v, nil", tc.in, got, err, tc.want)
		}
	}
	if _, err := parseDiversity("selection"); err == nil {
		t.Error("parseDiversity(selection) should error")
	}
}

// TestMRCCombinerStaticModeFreezes pins the mrc-static escape hatch: its gain is
// estimated once and never revisited, so an operator can A/B it against tracking
// on the same hardware and attribute any difference to the update policy alone.
func TestMRCCombinerStaticModeFreezes(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	ref := mrcSignal(rng, mrcTestWindow, 0.4)

	m := newMRCCombiner(formatCS16, diversityMRCStatic, 0)
	m.combine(twoBranchPayload(ref, ref), len(ref))
	if !m.cal.Calibrated() {
		t.Fatal("static mode did not calibrate")
	}
	locked := m.cal.Gains()

	// A window whose true gain is different must not move it.
	rotated := scaleC(ref, complex(0, 1))
	m.combine(twoBranchPayload(ref, rotated), len(ref))
	if got := m.cal.Gains(); got[1] != locked[1] {
		t.Errorf("static gain moved from %v to %v", locked[1], got[1])
	}
	if u, _ := m.cal.Counters(); u != 1 {
		t.Errorf("static updates = %d, want exactly 1", u)
	}
}

// TestDiversityReporterDoesNotCryIncoherentBeforeMeasuring pins that the
// first-datagram report — which always fires so an operator sees both branch
// levels immediately — cannot claim the branches are incoherent before a single
// estimation window has completed. Saying so at t=0 would send them after the
// antennas when nothing has been measured yet.
func TestDiversityReporterDoesNotCryIncoherentBeforeMeasuring(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	// One short datagram: well under a window, so no estimate is possible.
	ch0 := mrcSignal(rng, 256, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	m.combine(twoBranchPayload(ch0, ch1), len(ch0))

	var buf bytes.Buffer
	r := newDiversityReporter(slog.New(slog.NewTextHandler(&buf, nil)), "10.0.0.5:55132")
	r.now = func() time.Time { return time.Unix(1700000000, 0) }
	r.observe(m)

	got := buf.String()
	if strings.Contains(got, "not coherent") {
		t.Errorf("reported incoherence before any window completed: %q", got)
	}
	if !strings.Contains(got, "coherence=") {
		t.Errorf("report = %q, want the coherence attribute present", got)
	}
}

// TestDiversityReporterWarnsWhenBranchesNeverCorrelate is the other half: once
// windows HAVE been measured and none passed the gate, the operator is told —
// with the actual (window-scaled) lock bar in the line, and guidance that names
// a branch buried under its own front-end floor as a cause. (The line used to
// claim "raising RF gain will NOT help"; an operator's 18 Aug 2026 A/B proved
// that wrong — their weak branch's floor did not scale with gain, so +5 dB
// lifted its in-channel fraction and coherence with it.)
func TestDiversityReporterWarnsWhenBranchesNeverCorrelate(t *testing.T) {
	rng := rand.New(rand.NewSource(12))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	// Two loud but unrelated receivers, long enough to complete windows.
	for i := 0; i < 3; i++ {
		a := mrcSignal(rng, mrcTestWindow, 0.3)
		b := mrcSignal(rng, mrcTestWindow, 0.3)
		m.combine(twoBranchPayload(a, b), len(a))
	}
	if _, holds := m.cal.Counters(); holds == 0 {
		t.Fatal("fixture completed no windows")
	}

	var buf bytes.Buffer
	r := newDiversityReporter(slog.New(slog.NewTextHandler(&buf, nil)), "10.0.0.5:55132")
	r.now = func() time.Time { return time.Unix(1700000000, 0) }
	r.observe(m)

	got := buf.String()
	if !strings.Contains(got, "level=WARN") || !strings.Contains(got, "not coherent") {
		t.Errorf("report = %q, want a WARN about incoherent branches", got)
	}
	if !strings.Contains(got, "gain staging") {
		t.Errorf("report = %q, want it to point at per-branch gain staging", got)
	}
	if !strings.Contains(got, "lock_gate=") {
		t.Errorf("report = %q, want the effective lock gate in the line", got)
	}
}

// TestDiversityReporterWarnsWhenCalibrationGoesStale covers the gap an
// operator's hour-long X310 capture exposed: the combiner locked once, then
// every window for the next eleven minutes failed the coherence gate — updates
// frozen while holds climbed by ~6000 per interval — and the health line kept
// reporting INFO because "calibrated" was still true. The combine was applying
// a gain measured eleven minutes earlier and nothing said so.
func TestDiversityReporterWarnsWhenCalibrationGoesStale(t *testing.T) {
	rng := rand.New(rand.NewSource(21))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	// Lock on a coherent pair.
	for i := 0; i < 2; i++ {
		a := mrcSignal(rng, mrcTestWindow, 0.3)
		b := scaleC(a, complex(0.9, 0.4))
		m.combine(twoBranchPayload(a, b), len(a))
	}
	if !m.cal.Calibrated() {
		t.Fatal("fixture did not calibrate")
	}

	var buf bytes.Buffer
	now := time.Unix(1700000000, 0)
	r := newDiversityReporter(slog.New(slog.NewTextHandler(&buf, nil)), "10.0.0.5:55132")
	r.now = func() time.Time { return now }
	r.observe(m)

	// Then feed uncorrelated branches, so every further window is held.
	for interval := 0; interval <= mrcStaleUpdateIntervals; interval++ {
		for i := 0; i < 2; i++ {
			a := mrcSignal(rng, mrcTestWindow, 0.3)
			b := mrcSignal(rng, mrcTestWindow, 0.3)
			m.combine(twoBranchPayload(a, b), len(a))
		}
		now = now.Add(mrcHealthInterval)
		buf.Reset()
		r.observe(m)
	}

	got := buf.String()
	if !strings.Contains(got, "level=WARN") || !strings.Contains(got, "stale gain") {
		t.Errorf("report = %q, want a WARN about a stale calibration", got)
	}
	if !strings.Contains(got, "wideband") {
		t.Errorf("report = %q, want it to name the wideband-combine limitation", got)
	}
}

// A healthy combiner that keeps accepting windows must NOT trip the stale
// warning, or the signal is worthless.
func TestDiversityReporterStaysQuietWhileCalibrationKeepsUpdating(t *testing.T) {
	rng := rand.New(rand.NewSource(22))
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	var buf bytes.Buffer
	now := time.Unix(1700000000, 0)
	r := newDiversityReporter(slog.New(slog.NewTextHandler(&buf, nil)), "10.0.0.5:55132")
	r.now = func() time.Time { return now }

	for interval := 0; interval <= mrcStaleUpdateIntervals+2; interval++ {
		for i := 0; i < 2; i++ {
			a := mrcSignal(rng, mrcTestWindow, 0.3)
			b := scaleC(a, complex(0.9, 0.4))
			m.combine(twoBranchPayload(a, b), len(a))
		}
		now = now.Add(mrcHealthInterval)
		buf.Reset()
		r.observe(m)
	}
	if got := buf.String(); strings.Contains(got, "stale gain") {
		t.Errorf("healthy combiner reported a stale calibration: %q", got)
	}
}

// TestMRCTrackAlphaMatchesDocumentedTimeConstant pins that the loop bandwidth is
// derived from the ACTUAL window duration and the stated time constant rather
// than hardcoded, so neither changing the window nor the min-samples clamp can
// silently move it.
func TestMRCTrackAlphaMatchesDocumentedTimeConstant(t *testing.T) {
	// Unclamped rate: 2 MHz gives a 4096-sample window ≈ the nominal 2 ms, so
	// alpha stays at the historical 0.01.
	rate := 2_048_000.0
	if got, want := mrcTrackAlpha(diversityMRCTracking, mrcCalWindowSamples(rate), rate), mrcCalWindowMs/mrcTrackTauMs; got != want {
		t.Errorf("tracking alpha = %v, want %v", got, want)
	}
	if got := mrcTrackAlpha(diversityMRCStatic, mrcCalWindowSamples(rate), rate); got != 0 {
		t.Errorf("static alpha = %v, want 0 (one-shot)", got)
	}
}

// TestMRCTrackAlphaHonorsClampedWindow is the low-rate regression (operator's
// 250 kHz narrow TETRA capture): the min-samples clamp stretches the 2 ms
// window to 4096 samples = 16.384 ms, and alpha must follow the ACTUAL
// duration or the loop's 1/e time silently becomes ~1.6 s instead of the
// documented 200 ms. Fails against the old nominal-constant derivation
// (which returned 0.01 here).
func TestMRCTrackAlphaHonorsClampedWindow(t *testing.T) {
	const rate = 250_000.0
	window := mrcCalWindowSamples(rate)
	if window != mrcCalWindowMinSamples {
		t.Fatalf("expected the min-samples clamp to engage at %v Hz, got window %d", rate, window)
	}
	got := mrcTrackAlpha(diversityMRCTracking, window, rate)
	want := float64(window) / rate * 1000 / mrcTrackTauMs // ≈ 0.0819
	if got != want {
		t.Errorf("tracking alpha at %v Hz = %v, want %v (actual window duration / tau)", rate, got, want)
	}
	// Unknown rate: fall back to the nominal derivation rather than guessing.
	if got, want := mrcTrackAlpha(diversityMRCTracking, mrcCalWindowDefaultSamples, 0), mrcCalWindowMs/mrcTrackTauMs; got != want {
		t.Errorf("unknown-rate alpha = %v, want nominal %v", got, want)
	}
}

// TestMRCCombinerHealthFlagsADeadReference pins a gap the sticky anchor opened.
// The reference used to be the argmax by construction, so it could never be the
// dead branch and comparing everything to it was safe. Now that the anchor is
// latched and held, the REFERENCE receiver can go dark mid-session — an antenna
// falling off the primary — and a reference-relative comparison would report
// nothing at all, which is the one case an operator most needs told about.
func TestMRCCombinerHealthFlagsADeadReference(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	live := mrcSignal(rng, 512, 0.4)
	weak := scaleC(live, complex(1e-3, 0)) // 60 dB down

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	// Latch the anchor on branch 0 while it is the loud one.
	m.combine(twoBranchPayload(live, weak), len(live))
	if h := m.health(); h.refIdx != 0 {
		t.Fatalf("reference branch = %d, want 0", h.refIdx)
	}

	// Branch 0's antenna falls off: it is now the weak one, but the anchor holds.
	m.combine(twoBranchPayload(weak, live), len(live))
	h := m.health()
	if h.refIdx != 0 {
		t.Fatalf("reference moved to %d on one datagram; the anchor must be sticky", h.refIdx)
	}
	if h.deadBranch != 0 {
		t.Errorf("deadBranch = %d, want 0 — a dead REFERENCE must still be reported", h.deadBranch)
	}
}
