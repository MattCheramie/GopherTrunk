package diversity

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// scale multiplies every sample by a real factor — an RF gain change.
func scale(s []complex64, k float64) []complex64 {
	out := make([]complex64, len(s))
	for i := range s {
		out[i] = complex(float32(float64(real(s[i]))*k), float32(float64(imag(s[i]))*k))
	}
	return out
}

// offset adds a constant to every sample — LO leakage / converter DC.
func offset(s []complex64, dc complex64) []complex64 {
	out := make([]complex64, len(s))
	for i := range s {
		out[i] = s[i] + dc
	}
	return out
}

// TestCrossStatsCoherenceIsScaleInvariant is the property that decouples
// calibration from RF gain: the same pair of branches, attenuated or amplified
// by four orders of magnitude, must report the same coherence. An absolute
// power gate cannot have this property, which is why one made an operator raise
// front-end gain from 65 dB to 82 dB purely to clear a software constant.
func TestCrossStatsCoherenceIsScaleInvariant(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	ref := genSignal(rng, 4096)
	x := addNoise(rng, rotate(ref, 0.9), 0.4)

	var base CrossStats
	base.Accumulate(ref, x)
	want := base.Coherence()
	if want < 0.5 || want > 0.999 {
		t.Fatalf("test fixture is degenerate: coherence %.4f", want)
	}

	for _, k := range []float64{1e-4, 1e-2, 1e2, 1e4} {
		var s CrossStats
		s.Accumulate(scale(ref, k), scale(x, k))
		if got := s.Coherence(); math.Abs(got-want) > 1e-6 {
			t.Errorf("scale %g: coherence %.9f, want %.9f", k, got, want)
		}
		// The gain estimate is likewise unchanged: both branches scaled
		// together leaves the branch-to-branch ratio alone.
		h, ok := s.Gain()
		if !ok {
			t.Fatalf("scale %g: Gain not ok", k)
		}
		hw, _ := base.Gain()
		if cmplx.Abs(complex128(h)-complex128(hw)) > 1e-4 {
			t.Errorf("scale %g: gain %v, want %v", k, h, hw)
		}
	}
}

// TestCrossStatsRejectsCommonDCOffset pins the DC trap. Two receivers of one
// front-end share an LO-leakage / converter DC offset, and that offset is
// perfectly common-mode. An uncentred correlator on two branches of INDEPENDENT
// noise plus a shared DC reports near-perfect correlation and would freeze a
// gain of dc1/dc0 — calibrating confidently on nothing, in exactly the weak
// signal regime the gate exists to protect.
func TestCrossStatsRejectsCommonDCOffset(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	// Independent noise on both branches: there is no common signal at all.
	a := addNoise(rng, make([]complex64, 8192), 1)
	b := addNoise(rng, make([]complex64, 8192), 1)
	dc := complex64(complex(20, -12)) // 20x the noise amplitude

	var s CrossStats
	s.Accumulate(offset(a, dc), offset(b, dc))
	if got := s.Coherence(); got > 0.05 {
		t.Errorf("coherence %.4f with a shared DC offset over independent noise, want <=0.05 "+
			"(the correlator is not removing DC)", got)
	}

	// And the DC must not leak into the power readings either: these branches
	// carry only noise, so AC power should be ~1 per axis pair, not ~544.
	if p := s.RefPower(); p > 4 {
		t.Errorf("RefPower %.2f, want ~2 (DC is being counted as signal)", p)
	}
}

// TestCrossStatsCoherenceTracksSNR pins the mapping the gate thresholds are
// justified by: for a common signal in independent noise at equal per-branch
// SNR gamma, |rho| = gamma/(1+gamma).
func TestCrossStatsCoherenceTracksSNR(t *testing.T) {
	for _, tc := range []struct{ snrDb, want float64 }{
		{20, 100.0 / 101.0},
		{10, 10.0 / 11.0},
		{0, 0.5},
		{-6, 0.2005},
	} {
		rng := rand.New(rand.NewSource(3))
		sig := genSignal(rng, 1<<16) // unit mean power
		gamma := math.Pow(10, tc.snrDb/10)
		sigma := math.Sqrt(1/gamma) / math.Sqrt2 // per-axis sd for noise power 1/gamma
		var s CrossStats
		s.Accumulate(addNoise(rng, sig, sigma), addNoise(rng, rotate(sig, 1.1), sigma))
		got := s.Coherence()
		if math.Abs(got-tc.want) > 0.02 {
			t.Errorf("snr %+.0f dB: coherence %.4f, want %.4f±0.02", tc.snrDb, got, tc.want)
		}
	}
}

// TestCrossStatsIndependentNoiseFloor pins that pure noise stays far below any
// usable gate, so a threshold is not clearing on luck.
func TestCrossStatsIndependentNoiseFloor(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	var s CrossStats
	s.Accumulate(addNoise(rng, make([]complex64, 4096), 1), addNoise(rng, make([]complex64, 4096), 1))
	// sqrt(pi/4N) ~= 0.014 at N=4096; allow generous headroom.
	if got := s.Coherence(); got > 0.1 {
		t.Errorf("independent-noise coherence %.4f, want <=0.1", got)
	}
}

// TestCrossStatsAccumulatesAcrossChunks pins that a window may span many
// datagrams: feeding one long pair and the same pair in pieces must agree.
func TestCrossStatsAccumulatesAcrossChunks(t *testing.T) {
	rng := rand.New(rand.NewSource(23))
	ref := genSignal(rng, 3000)
	x := addNoise(rng, rotate(ref, -0.4), 0.3)

	var whole CrossStats
	whole.Accumulate(ref, x)

	var pieces CrossStats
	for i := 0; i < len(ref); i += 137 {
		end := min(i+137, len(ref))
		pieces.Accumulate(ref[i:end], x[i:end])
	}

	if pieces.Samples() != whole.Samples() {
		t.Fatalf("samples %d, want %d", pieces.Samples(), whole.Samples())
	}
	if math.Abs(pieces.Coherence()-whole.Coherence()) > 1e-9 {
		t.Errorf("chunked coherence %.12f, whole %.12f", pieces.Coherence(), whole.Coherence())
	}
	hp, _ := pieces.Gain()
	hw, _ := whole.Gain()
	if cmplx.Abs(complex128(hp)-complex128(hw)) > 1e-5 {
		t.Errorf("chunked gain %v, whole %v", hp, hw)
	}
}

// TestCrossStatsSilentReference reports no opinion rather than a spurious
// perfect correlation.
func TestCrossStatsSilentReference(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	var s CrossStats
	s.Accumulate(make([]complex64, 1024), addNoise(rng, make([]complex64, 1024), 1))
	if got := s.Coherence(); got != 0 {
		t.Errorf("coherence %v with a silent reference, want 0", got)
	}
	if _, ok := s.Gain(); ok {
		t.Error("Gain reported ok with a silent reference")
	}
}
