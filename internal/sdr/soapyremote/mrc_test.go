package soapyremote

import (
	"math"
	"testing"
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

	got := formatCS16.deinterleave(payload, 2)
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
	got := formatCS16.deinterleave(payload, 1)
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
	out := m.combine(payload)

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
	out := m.combine(payload)

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
	_ = m.combine(pA)
	if !m.cal.Calibrated() {
		t.Fatal("did not calibrate on first window")
	}

	// Retune: the relationship flips to ch1 = -ch0. Re-arm calibration.
	m.requestRecalibrate()
	ch1 := scaleC(ref, complex(-1, 0))
	pB := encodeCS16(append(append([]complex64{}, ref...), ch1...))
	out := m.combine(pB)

	// With a fresh estimate (h1 = -1) the branches add in phase → ≈ ref.
	// With a STALE h1 = 1 they would cancel to ≈ 0.
	if !approxEqualC(out, ref, 2e-3) {
		t.Errorf("recalibrated output = %v, want ≈ %v (stale gain would cancel to ~0)", out, ref)
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
