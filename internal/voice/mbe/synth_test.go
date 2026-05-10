package mbe

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// TestPredictLog2MlFirstFrameIsTl confirms that with no prev-frame
// state, the prediction term is zero and log2(Ml)[l] = Tl[l]. The
// mean subtraction in eq. 77 is mean(pred[l]) = 0, not mean(Tl), so
// Tl values pass through unchanged.
func TestPredictLog2MlFirstFrameIsTl(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 20}}
	for l := 1; l <= p.L; l++ {
		p.Tl[l] = float64(l) * 0.1
	}
	var dst [57]float64
	PredictLog2Ml(&s, p, &dst)
	for l := 1; l <= p.L; l++ {
		want := float64(l) * 0.1
		if !almostEqual(dst[l], want) {
			t.Errorf("first frame dst[%d] = %v, want %v", l, dst[l], want)
		}
	}
}

// TestPredictLog2MlConstantPrevCancels: when prev-frame log2(Ml) is
// constant c, the prediction is 0.65*c at every l and ave = 0.65*c
// — they cancel in eq. 77 and dst = Tl. This pins the eq. 77 mean
// subtraction (a flat-spectrum prev frame contributes no tilt to
// the current frame).
func TestPredictLog2MlConstantPrevCancels(t *testing.T) {
	const c = 4.2
	s := SynthState{PrevW0: math.Pi / 30, PrevL: 20}
	for l := 1; l <= s.PrevL; l++ {
		s.PrevLog2Ml[l] = c
	}
	p := Params{Header: Header{W0: math.Pi / 30, L: 20}}
	for l := 1; l <= p.L; l++ {
		p.Tl[l] = float64(l) - 10.5 // arbitrary signed values
	}
	var dst [57]float64
	PredictLog2Ml(&s, p, &dst)
	for l := 1; l <= p.L; l++ {
		if !almostEqual(dst[l], p.Tl[l]) {
			t.Errorf("constant-prev dst[%d] = %v, want Tl[%d] = %v",
				l, dst[l], l, p.Tl[l])
		}
	}
}

// TestPredictLog2MlSamePitchExactInterp: when ω₀_curr == ω₀_prev and
// L_curr == L_prev, the interpolation index pos = l × 1.0 lands on
// integer l for every harmonic — no fractional blending. Each
// dst[l] = 0.65 × prev[l] + Tl[l] − mean(0.65 × prev). Checks the
// integer-index path.
func TestPredictLog2MlSamePitchExactInterp(t *testing.T) {
	s := SynthState{PrevW0: math.Pi / 30, PrevL: 4}
	prev := [5]float64{0, 1.0, 2.0, 3.0, 4.0} // 1-indexed
	for l := 1; l <= s.PrevL; l++ {
		s.PrevLog2Ml[l] = prev[l]
	}
	p := Params{Header: Header{W0: math.Pi / 30, L: 4}}
	// Tl all zero so dst directly reflects prediction - mean(prediction).
	var dst [57]float64
	PredictLog2Ml(&s, p, &dst)

	predMean := PredictionGain * (1.0 + 2.0 + 3.0 + 4.0) / 4.0
	for l := 1; l <= p.L; l++ {
		want := PredictionGain*prev[l] - predMean
		if !almostEqual(dst[l], want) {
			t.Errorf("same-pitch dst[%d] = %v, want %v", l, dst[l], want)
		}
	}
}

// TestPredictLog2MlInterpolationFraction: when ω₀_curr / ω₀_prev =
// 0.5, l=2 maps to prev-pos 1.0 and l=3 maps to prev-pos 1.5. The
// fractional blend at l=3 is (0.5 × prev[1] + 0.5 × prev[2]). Pins
// the fractional-index path.
func TestPredictLog2MlInterpolationFraction(t *testing.T) {
	s := SynthState{PrevW0: math.Pi / 30, PrevL: 4}
	s.PrevLog2Ml[1] = 2.0
	s.PrevLog2Ml[2] = 4.0
	s.PrevLog2Ml[3] = 6.0
	s.PrevLog2Ml[4] = 8.0
	// Curr ω₀ is half of prev → curr harmonic l sits at prev pos l/2.
	p := Params{Header: Header{W0: math.Pi / 60, L: 6}}
	var dst [57]float64
	PredictLog2Ml(&s, p, &dst)

	// Predicted-then-mean values at curr harmonics:
	//  l=1 → pos=0.5 (clamped to 1) → prev[1] = 2
	//  l=2 → pos=1.0 → prev[1] = 2
	//  l=3 → pos=1.5 → 0.5×prev[1] + 0.5×prev[2] = 3
	//  l=4 → pos=2.0 → prev[2] = 4
	//  l=5 → pos=2.5 → 0.5×prev[2] + 0.5×prev[3] = 5
	//  l=6 → pos=3.0 → prev[3] = 6
	rawPred := [7]float64{0, 2, 2, 3, 4, 5, 6}
	var sum float64
	for l := 1; l <= 6; l++ {
		sum += PredictionGain * rawPred[l]
	}
	mean := sum / 6.0
	for l := 1; l <= p.L; l++ {
		want := PredictionGain*rawPred[l] - mean // Tl is zero
		if !almostEqual(dst[l], want) {
			t.Errorf("interp dst[%d] = %v, want %v", l, dst[l], want)
		}
	}
}

// TestPredictLog2MlClampsBeyondPrevL: when curr L exceeds the
// available prev-frame harmonics (e.g. pitch dropped sharply so we
// have many more harmonics now), positions past PrevL clamp to the
// last prev value. Pins the upper-bound clamp.
func TestPredictLog2MlClampsBeyondPrevL(t *testing.T) {
	s := SynthState{PrevW0: math.Pi / 10, PrevL: 3}
	s.PrevLog2Ml[1] = 1.0
	s.PrevLog2Ml[2] = 2.0
	s.PrevLog2Ml[3] = 3.0
	// Half pitch → curr L bigger; positions for high l exceed PrevL.
	p := Params{Header: Header{W0: math.Pi / 20, L: 10}}
	var dst [57]float64
	PredictLog2Ml(&s, p, &dst)

	// Verify high harmonics see the last-prev value (3.0) before mean
	// subtraction. Only check the relative tilt: dst[l] − dst[l′]
	// matches the raw pred difference for l, l′ both past clamp.
	// l=8 → pos=4.0 → clamped to 3 → 3.0
	// l=9 → pos=4.5 → clamped to 3 → 3.0
	// l=10 → pos=5.0 → clamped to 3 → 3.0
	for _, l := range []int{8, 9, 10} {
		if !almostEqual(dst[l], dst[8]) {
			t.Errorf("clamp: dst[%d] = %v should equal dst[8] = %v",
				l, dst[l], dst[8])
		}
	}
}

// TestPredictLog2MlSilentFrameLeavesDstUntouched: silence frames
// short-circuit the prediction; dst keeps its prior contents so
// callers can detect "no update happened". The synthesizer is
// expected to either play silence or hold the last frame.
func TestPredictLog2MlSilentFrameLeavesDstUntouched(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{Silent: true}}
	dst := [57]float64{}
	for i := range dst {
		dst[i] = 99 // sentinel
	}
	PredictLog2Ml(&s, p, &dst)
	for i := range dst {
		if dst[i] != 99 {
			t.Errorf("silent frame mutated dst[%d] = %v, want sentinel 99",
				i, dst[i])
		}
	}
}

// TestUpdateLog2MlRollsState: the helper copies log2(Ml)[1..L] into
// PrevLog2Ml, stores ω₀ + L, and zeroes any prior PrevLog2Ml entries
// past the new L (so a subsequent shorter-L frame doesn't leak old
// high harmonics into the prediction).
func TestUpdateLog2MlRollsState(t *testing.T) {
	s := SynthState{PrevL: 30}
	for l := 1; l <= 30; l++ {
		s.PrevLog2Ml[l] = 99
	}
	p := Params{Header: Header{W0: 0.42, L: 5}}
	src := [57]float64{}
	src[1] = 1
	src[2] = 2
	src[3] = 3
	src[4] = 4
	src[5] = 5
	s.UpdateLog2Ml(p, &src)
	if s.PrevW0 != 0.42 {
		t.Errorf("PrevW0 = %v, want 0.42", s.PrevW0)
	}
	if s.PrevL != 5 {
		t.Errorf("PrevL = %d, want 5", s.PrevL)
	}
	for l := 1; l <= 5; l++ {
		if s.PrevLog2Ml[l] != float64(l) {
			t.Errorf("PrevLog2Ml[%d] = %v, want %d", l, s.PrevLog2Ml[l], l)
		}
	}
	for l := 6; l <= 56; l++ {
		if s.PrevLog2Ml[l] != 0 {
			t.Errorf("PrevLog2Ml[%d] = %v, want 0 (cleared past new L)",
				l, s.PrevLog2Ml[l])
		}
	}
}

// TestResetClearsState: Reset zeroes everything so the next frame
// behaves like the first frame on a fresh stream.
func TestResetClearsState(t *testing.T) {
	s := SynthState{PrevW0: 0.5, PrevL: 20}
	for l := 1; l <= 20; l++ {
		s.PrevLog2Ml[l] = float64(l)
	}
	s.Reset()
	if s.PrevW0 != 0 || s.PrevL != 0 {
		t.Errorf("Reset: PrevW0=%v PrevL=%d, want 0/0", s.PrevW0, s.PrevL)
	}
	for l := 0; l <= 56; l++ {
		if s.PrevLog2Ml[l] != 0 {
			t.Errorf("Reset: PrevLog2Ml[%d] = %v, want 0", l, s.PrevLog2Ml[l])
		}
	}
}

// TestPredictLog2MlTwoFrameSequence runs two consecutive frames
// through Predict→Update to confirm the state machine threads
// correctly: frame 2's prediction must see frame 1's log2(Ml).
func TestPredictLog2MlTwoFrameSequence(t *testing.T) {
	var s SynthState

	// Frame 1: fresh state. dst1 = Tl1 (no prediction).
	p1 := Params{Header: Header{W0: math.Pi / 30, L: 4}}
	for l := 1; l <= p1.L; l++ {
		p1.Tl[l] = float64(l)
	}
	var dst1 [57]float64
	PredictLog2Ml(&s, p1, &dst1)
	for l := 1; l <= p1.L; l++ {
		if !almostEqual(dst1[l], float64(l)) {
			t.Fatalf("frame 1 dst[%d] = %v, want %d", l, dst1[l], l)
		}
	}
	s.UpdateLog2Ml(p1, &dst1)

	// Frame 2: same pitch + L; prev = {1,2,3,4}. With Tl2 = 0,
	// dst2[l] = 0.65*l - mean(0.65*{1,2,3,4}) = 0.65*l - 0.65*2.5.
	p2 := Params{Header: Header{W0: math.Pi / 30, L: 4}}
	var dst2 [57]float64
	PredictLog2Ml(&s, p2, &dst2)
	predMean := PredictionGain * (1.0 + 2.0 + 3.0 + 4.0) / 4.0
	for l := 1; l <= p2.L; l++ {
		want := PredictionGain*float64(l) - predMean
		if !almostEqual(dst2[l], want) {
			t.Errorf("frame 2 dst[%d] = %v, want %v", l, dst2[l], want)
		}
	}
}

// TestPredictLog2MlMeanCenteredOutput: regardless of prev-frame
// content, when Tl is mean-centered the eq. 77 output is also
// mean-centered (the prediction's DC bias is removed). Pins the
// "Tl carries the only intentional offset" property.
func TestPredictLog2MlMeanCenteredOutput(t *testing.T) {
	s := SynthState{PrevW0: math.Pi / 25, PrevL: 8}
	for l := 1; l <= s.PrevL; l++ {
		s.PrevLog2Ml[l] = float64(l) * 1.7
	}
	p := Params{Header: Header{W0: math.Pi / 30, L: 6}}
	// Mean-centered Tl: sums to zero.
	tlVals := []float64{-3, -1, 0, 0, 1, 3}
	for l := 1; l <= p.L; l++ {
		p.Tl[l] = tlVals[l-1]
	}
	var dst [57]float64
	PredictLog2Ml(&s, p, &dst)

	var sum float64
	for l := 1; l <= p.L; l++ {
		sum += dst[l]
	}
	if !almostEqual(sum, 0) {
		t.Errorf("mean-centered Tl: sum(dst) = %v, want 0", sum)
	}
}
