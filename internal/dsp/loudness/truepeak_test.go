package loudness

import (
	"math"
	"testing"
)

func TestTruePeak_DetectsInterSample(t *testing.T) {
	// A sine at fs/4 sampled with a π/4 phase offset never lands on its
	// crest: samples sit at ±amp/√2, but the true peak is amp. A 4x
	// oversampling true-peak meter should recover close to amp.
	const amp = 0.9
	s := sine(testRate*1, testRate/4, amp, math.Pi/4)

	samplePeak := 0.0
	for _, x := range s {
		if a := math.Abs(x); a > samplePeak {
			samplePeak = a
		}
	}
	tp := TruePeak(s, testRate)

	if tp <= samplePeak+1e-6 {
		t.Fatalf("true peak %.4f should exceed sample peak %.4f", tp, samplePeak)
	}
	if math.Abs(tp-amp) > 0.05 {
		t.Fatalf("true peak %.4f should be near the real crest %.4f", tp, amp)
	}
}

func TestTruePeak_ShortInput(t *testing.T) {
	if got := TruePeak([]float64{0.3}, testRate); math.Abs(got-0.3) > 1e-9 {
		t.Fatalf("single-sample true peak should equal |sample|, got %.4f", got)
	}
}
