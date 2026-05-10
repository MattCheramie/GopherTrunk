package mbe

import (
	"math"
	"testing"
)

const enhanceEpsilon = 1e-9

// TestEnhanceAmplitudesSilentNoOp: silent frames leave M untouched
// so callers can invoke unconditionally on the synthesis path.
func TestEnhanceAmplitudesSilentNoOp(t *testing.T) {
	p := Params{Header: Header{Silent: true}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = float64(l) * 0.1
	}
	want := M
	EnhanceAmplitudes(p, &M)
	if M != want {
		t.Errorf("Silent: M mutated; got %v want %v", M, want)
	}
}

// TestEnhanceAmplitudesZeroLNoOp: L = 0 frames leave M untouched.
func TestEnhanceAmplitudesZeroLNoOp(t *testing.T) {
	p := Params{Header: Header{W0: 0.1, L: 0}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0
	}
	want := M
	EnhanceAmplitudes(p, &M)
	if M != want {
		t.Errorf("L=0: M mutated; got %v want %v", M, want)
	}
}

// TestEnhanceAmplitudesAllZeroNoOp: a degenerate frame with R_M0
// = 0 (every M[l] = 0) is left alone — there's nothing to enhance
// or rescale.
func TestEnhanceAmplitudesAllZeroNoOp(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	want := M
	EnhanceAmplitudes(p, &M)
	if M != want {
		t.Errorf("All-zero M: mutated; got %v want %v", M, want)
	}
}

// TestEnhanceAmplitudesEnergyPreserved: regardless of input, the
// total energy R_M0 = Σ M_l² before and after enhancement must
// agree (the per-harmonic multiply followed by the global rescale
// normalizes to the original total energy). Pins the energy
// preservation contract that makes the §6.2 enhancement safe to
// drop into the synthesis path.
func TestEnhanceAmplitudesEnergyPreserved(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 12}}
	var M [57]float64
	// Skewed input: amplitudes vary by 10x across harmonics.
	M[1] = 0.5
	M[2] = 1.0
	M[3] = 0.3
	M[4] = 2.0
	M[5] = 0.8
	M[6] = 0.4
	M[7] = 1.5
	M[8] = 0.9
	M[9] = 0.2
	M[10] = 1.1
	M[11] = 0.6
	M[12] = 0.7
	rm0Before := FrameEnergy(&M, p.L)

	EnhanceAmplitudes(p, &M)
	rm0After := FrameEnergy(&M, p.L)

	if math.Abs(rm0Before-rm0After) > 1e-9 {
		t.Errorf("energy not preserved: before=%v after=%v",
			rm0Before, rm0After)
	}
}

// TestEnhanceAmplitudesUniformBoundedAndNonzero: a uniform-amplitude
// spectrum doesn't pass through *unchanged* (per-harmonic W depends
// on cos(ω₀·l), which varies across l), but every harmonic must stay
// strictly positive and within a sane envelope. Pins that uniform
// input doesn't degenerate into zeros + that enhancement reshapes
// rather than destroys.
func TestEnhanceAmplitudesUniformBoundedAndNonzero(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 16}}
	var M [57]float64
	for l := 1; l <= p.L; l++ {
		M[l] = 1.0
	}

	EnhanceAmplitudes(p, &M)

	for l := 1; l <= p.L; l++ {
		if M[l] <= 0 {
			t.Errorf("uniform input l=%d: M=%v, want > 0", l, M[l])
		}
		// With a global rescale that preserves R_M0 = L = 16 and W
		// clamped in [0.5, 1.2], no harmonic can scale by more than
		// ~2.4× either direction even after the rescale.
		if M[l] > 5 || M[l] < 0.05 {
			t.Errorf("uniform input l=%d: M=%v out of sane envelope [0.05, 5]",
				l, M[l])
		}
	}
}

// TestEnhanceAmplitudesLowBandUntouchedBeforeRescale: harmonics with
// 8·l ≤ L are in the "low band" the §6.2 algorithm leaves at W = 1.
// They still get the global energy-preservation rescale, so the
// ratio between two low-band harmonics is preserved exactly.
func TestEnhanceAmplitudesLowBandUntouchedBeforeRescale(t *testing.T) {
	// L = 16 → low band is l ≤ 2 (since 8·2 = 16 ≤ 16, but 8·3 = 24 > 16).
	p := Params{Header: Header{W0: math.Pi / 30, L: 16}}
	var M [57]float64
	for l := 1; l <= p.L; l++ {
		M[l] = 1.0
	}
	M[1] = 0.7
	M[2] = 0.3

	EnhanceAmplitudes(p, &M)

	// Ratio M[1] / M[2] should still be 0.7 / 0.3 since both got
	// the same global rescale and no per-harmonic weight (W = 1 for
	// both — they're both in the low band).
	got := M[1] / M[2]
	want := 0.7 / 0.3
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("low-band ratio = %v, want %v", got, want)
	}
}

// TestEnhanceAmplitudesPreservesMidbandHarmonicCount: the
// enhancement may not zero any of the L harmonics — the W clamp
// at EnhanceWMin = 0.5 ensures every harmonic comes out positive
// for any non-degenerate frame. Pins that the §6.2 step doesn't
// accidentally silence harmonics.
func TestEnhanceAmplitudesPreservesMidbandHarmonicCount(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 22, L: 22}}
	var M [57]float64
	// Steeply skewed spectrum to exercise the clamp boundary.
	for l := 1; l <= p.L; l++ {
		M[l] = math.Pow(2, -float64(l)/3) // M[1]=0.79, M[22]=6e-3
	}
	EnhanceAmplitudes(p, &M)
	for l := 1; l <= p.L; l++ {
		if M[l] <= 0 {
			t.Errorf("M[%d] = %v after enhancement; should stay positive",
				l, M[l])
		}
	}
}

// TestEnhanceWMinMaxBound: a hand-picked frame produces a per-harmonic
// W that without clamping would fall outside [EnhanceWMin,
// EnhanceWMax]. Verify the clamp keeps the eventual M[l] / M_orig[l]
// (compensated for global rescale) within a sane band.
func TestEnhanceWMinMaxBound(t *testing.T) {
	// A near-pure-tone spectrum (most energy on one harmonic) makes
	// R_M1² ≈ R_M0², which sends den → 0 and would blow up xi if
	// unclamped.
	p := Params{Header: Header{W0: math.Pi / 30, L: 12}}
	var M [57]float64
	M[1] = 10.0
	for l := 2; l <= p.L; l++ {
		M[l] = 0.01
	}
	rm0Before := FrameEnergy(&M, p.L)

	EnhanceAmplitudes(p, &M)

	rm0After := FrameEnergy(&M, p.L)
	if math.Abs(rm0Before-rm0After) > 1e-6 {
		t.Errorf("near-pure-tone: energy not preserved %v → %v",
			rm0Before, rm0After)
	}
	for l := 1; l <= p.L; l++ {
		if math.IsNaN(M[l]) || math.IsInf(M[l], 0) {
			t.Errorf("M[%d] = %v (NaN / Inf) — clamp didn't catch the den→0 case",
				l, M[l])
		}
	}
}

// TestEnhanceAmplitudesAtOrPastClampHigh: a frame where the
// closed-form weight exceeds EnhanceWMax should get clamped down.
// Hard to engineer the exact w > 1.2 case from a clean input —
// instead just verify the bound holds across a sweep of L values
// + skewed amplitude profiles (no per-harmonic ratio after
// enhancement should jump by more than EnhanceWMax / EnhanceWMin
// = 2.4× compared to the global rescale).
func TestEnhanceAmplitudesAtOrPastClampHigh(t *testing.T) {
	for _, L := range []int{10, 20, 30, 40, 56} {
		p := Params{Header: Header{W0: math.Pi / float64(L+5), L: L}}
		var M [57]float64
		for l := 1; l <= L; l++ {
			M[l] = 1.0 + 0.5*math.Sin(float64(l)) // varied
		}
		var orig [57]float64
		copy(orig[:], M[:])

		EnhanceAmplitudes(p, &M)

		// Each harmonic ratio M[l] / M_orig[l] is W_l × global_scale.
		// The largest and smallest ratios across l differ by at most
		// EnhanceWMax / EnhanceWMin (the W bounds; global_scale is
		// constant across l).
		var minR, maxR = math.MaxFloat64, 0.0
		for l := 1; l <= L; l++ {
			if orig[l] == 0 {
				continue
			}
			r := M[l] / orig[l]
			if r < minR {
				minR = r
			}
			if r > maxR {
				maxR = r
			}
		}
		if maxR/minR > EnhanceWMax/EnhanceWMin+1e-9 {
			t.Errorf("L=%d: ratio span %v exceeds W clamp ratio %v",
				L, maxR/minR, EnhanceWMax/EnhanceWMin)
		}
	}
}

// TestEnhanceClosedFormSmallCase verifies the per-harmonic weight
// formula on a hand-derived input where every step can be checked
// analytically. L = 4 is below 8·1 (= 8), so all 4 harmonics are
// in the *enhancement* band (8·l > L for all l in 1..4).
//
// Inputs: ω₀ = π/2, M = [1, 2, 1, 0.5].
//
//	M² = [1, 4, 1, 0.25]; R_M0 = 6.25.
//	cos(ω₀·l) for l=1..4 = [0, -1, 0, 1]; R_M1 = M²·cos = [0, -4, 0, 0.25] → -3.75.
//	R_M0² = 39.0625; R_M1² = 14.0625.
//	den = R_M0 · (R_M0² − R_M1²) = 6.25 · 25 = 156.25.
//
// l=1: c=0; num = 39.0625 + 14.0625 − 0 = 53.125; ξ = 0.96·53.125/156.25 = 0.3264; W₁ = 0.3264^0.25.
// l=2: c=-1; num = 53.125 − 2·6.25·(-3.75)·(-1) = 53.125 − 46.875 = 6.25; ξ = 0.96·6.25/156.25 = 0.0384;
//      W₂ = 0.0384^0.25 ≈ 0.443 → clamped to 0.5.
// l=3: same as l=1 → W₃ = W₁.
// l=4: c=1; num = 53.125 + 46.875 = 100; ξ = 0.96·100/156.25 = 0.6144; W₄ = 0.6144^0.25.
//
// Verify pre-rescale ratios (rescale is a global multiplier so
// M[a]·W[b] / (M[b]·W[a]) is preserved).
func TestEnhanceClosedFormSmallCase(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 2, L: 4}}
	var M [57]float64
	M[1] = 1.0
	M[2] = 2.0
	M[3] = 1.0
	M[4] = 0.5

	EnhanceAmplitudes(p, &M)

	// W₁ and W₃ should be identical (same closed-form inputs).
	r1 := M[1] / 1.0
	r3 := M[3] / 1.0
	if math.Abs(r1-r3) > 1e-9 {
		t.Errorf("W₁ ≠ W₃ — closed form differs for symmetric harmonics: r1=%v r3=%v",
			r1, r3)
	}

	// W₄ should be larger than W₁ (ξ₄ > ξ₁ by the closed form;
	// pow preserves order on positive values).
	r4 := M[4] / 0.5
	if r4 <= r1 {
		t.Errorf("W₄ should be > W₁: r1=%v r4=%v", r1, r4)
	}

	// W₂ should be the smallest (clamped at 0.5; W₁ and W₄ both
	// land above 0.5 by the closed form).
	r2 := M[2] / 2.0
	if r2 >= r1 || r2 >= r4 {
		t.Errorf("W₂ should be smallest (clamped): r1=%v r2=%v r4=%v",
			r1, r2, r4)
	}
}
