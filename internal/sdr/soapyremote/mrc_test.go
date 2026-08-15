package soapyremote

import (
	"bytes"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"
)

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
	ch0 := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35)}
	ch1 := scaleC(ch0, complex(0, 1)) // +90° rotation
	payload := encodeCS16(append(append([]complex64{}, ch0...), ch1...))

	m := newMRCCombiner(formatCS16)
	out := m.combine(payload, len(ch0))

	if !m.cal.Calibrated() {
		t.Fatal("combiner did not calibrate on a signal-bearing window")
	}
	if !approxEqualC(out, ch0, 2e-3) {
		t.Errorf("combined = %v, want ≈ reference %v (rotated branch must align, not cancel)", out, ch0)
	}
}

// TestMRCCombinerPassthroughBelowFloor: on a noise-floor-level window the
// combiner must not calibrate — it passes the reference branch through so a
// bad low-SNR window can't freeze a garbage phase constant.
func TestMRCCombinerPassthroughBelowFloor(t *testing.T) {
	// ~ -60 dBFS per branch, below mrcCalFloorDbFS (-40).
	ch0 := []complex64{complex(1e-3, -1e-3), complex(1e-3, 1e-3)}
	ch1 := scaleC(ch0, complex(0, 1))
	payload := encodeCS16(append(append([]complex64{}, ch0...), ch1...))

	m := newMRCCombiner(formatCS16)
	out := m.combine(payload, len(ch0))

	if m.cal.Calibrated() {
		t.Error("combiner calibrated on a below-floor (noise) window; want deferred")
	}
	// Uncalibrated Combine returns a copy of the reference branch.
	if !approxEqualC(out, ch0, 1e-3) {
		t.Errorf("below-floor output = %v, want reference passthrough %v", out, ch0)
	}
}

// TestMRCCombinerRecalibrateOnRequest: after a retune the frozen constant is
// stale. requestRecalibrate must force a fresh estimate on the next window;
// without it, a branch whose relationship flipped (ch1 = -ch0) would cancel.
func TestMRCCombinerRecalibrateOnRequest(t *testing.T) {
	ref := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35)}

	m := newMRCCombiner(formatCS16)

	// First lock: ch1 == ch0. Calibrates h1 = 1.
	pA := encodeCS16(append(append([]complex64{}, ref...), ref...))
	_ = m.combine(pA, len(ref))
	if !m.cal.Calibrated() {
		t.Fatal("did not calibrate on first window")
	}

	// Retune: the relationship flips to ch1 = -ch0. Re-arm calibration.
	m.requestRecalibrate()
	ch1 := scaleC(ref, complex(-1, 0))
	pB := encodeCS16(append(append([]complex64{}, ref...), ch1...))
	out := m.combine(pB, len(ref))

	// With a fresh estimate (h1 = -1) the branches add in phase → ≈ ref.
	// With a STALE h1 = 1 they would cancel to ≈ 0.
	if !approxEqualC(out, ref, 2e-3) {
		t.Errorf("recalibrated output = %v, want ≈ %v (stale gain would cancel to ~0)", out, ref)
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
	dead := []complex64{complex(1e-3, -1e-3), complex(-1e-3, 1e-3), complex(1e-3, 1e-3)}
	live := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35)}
	payload := encodeCS16(append(append([]complex64{}, dead...), live...))

	m := newMRCCombiner(formatCS16)
	out := m.combine(payload, len(live))

	if !m.cal.Calibrated() {
		t.Fatal("combiner did not calibrate on a window where RX1 carries signal")
	}
	if h := m.health(); h.refIdx != 1 {
		t.Errorf("reference branch = %d, want 1 (the live receiver)", h.refIdx)
	}
	if p := refPowerDbFS(out); p < mrcCalFloorDbFS {
		t.Fatalf("combined power = %.1f dBFS, want the live branch — a dead RX0 must not silence the stream", p)
	}
	if !approxEqualC(out, live, 2e-2) {
		t.Errorf("combined = %v, want ≈ the live branch %v", out, live)
	}
}

// TestMRCCombinerHealthFlagsDeadBranch: a branch far below the reference is
// reported so an operator can see it in GopherTrunk's own log rather than in
// SoapySDRServer's UHD debug output.
func TestMRCCombinerHealthFlagsDeadBranch(t *testing.T) {
	live := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35)}
	dead := scaleC(live, complex(1e-3, 0)) // 60 dB down
	m := newMRCCombiner(formatCS16)
	_ = m.combine(encodeCS16(append(append([]complex64{}, live...), dead...)), len(live))

	h := m.health()
	if h.deadBranch != 1 {
		t.Errorf("deadBranch = %d, want 1", h.deadBranch)
	}
	if h.degenerate {
		t.Error("degenerate = true, want false (both channels were delivered)")
	}

	// Two healthy branches must NOT trip it.
	m2 := newMRCCombiner(formatCS16)
	_ = m2.combine(encodeCS16(append(append([]complex64{}, live...), live...)), len(live))
	if h2 := m2.health(); h2.deadBranch != -1 {
		t.Errorf("deadBranch = %d on two equal branches, want -1", h2.deadBranch)
	}
}

// TestMRCCombinerSingleChannelPayloadPassesThrough: a server that honours only
// one channel of the 2-channel request must yield that receiver verbatim (and
// be flagged), not a spliced half-and-half stream.
func TestMRCCombinerSingleChannelPayloadPassesThrough(t *testing.T) {
	x := []complex64{complex(0.5, 0.2), complex(0.3, -0.4), complex(-0.2, 0.35), complex(0.1, 0.1)}
	m := newMRCCombiner(formatCS16)
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
	m := newMRCCombiner(formatCS16)
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
	on := []string{"mrc", "MRC", " mrc "}
	for _, s := range on {
		d, err := parseDiversity(s)
		if err != nil || !d {
			t.Errorf("parseDiversity(%q) = %v, %v; want true, nil", s, d, err)
		}
	}
	off := []string{"", "none", "off", "OFF"}
	for _, s := range off {
		d, err := parseDiversity(s)
		if err != nil || d {
			t.Errorf("parseDiversity(%q) = %v, %v; want false, nil", s, d, err)
		}
	}
	if _, err := parseDiversity("selection"); err == nil {
		t.Error("parseDiversity(selection) should error")
	}
}
