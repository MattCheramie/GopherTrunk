package loudness

import (
	"math"
	"testing"
)

const testRate = 8000

// sine generates n samples of a sine at freqHz with the given amplitude
// and phase (radians).
func sine(n int, freqHz, amp, phase float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = amp * math.Sin(2*math.Pi*freqHz*float64(i)/testRate+phase)
	}
	return out
}

func TestIntegratedLUFS_Sanity(t *testing.T) {
	// A steady 1 kHz sine at amplitude 0.5. K-weighting is ~0 dB near
	// 1 kHz, so L ≈ -0.691 + 10*log10(0.5^2/2) ≈ -9.7 LUFS. Allow a band
	// for the shelf/HPF contribution.
	s := sine(testRate*3, 1000, 0.5, 0)
	lufs, ok := IntegratedLUFS(s, testRate)
	if !ok {
		t.Fatal("expected ok for a 3 s tone")
	}
	if lufs < -11.0 || lufs > -8.0 {
		t.Fatalf("integrated loudness %.2f LUFS outside expected band [-11,-8]", lufs)
	}
}

func TestIntegratedLUFS_Monotonic6dB(t *testing.T) {
	loud, ok := IntegratedLUFS(sine(testRate*3, 1000, 0.5, 0), testRate)
	if !ok {
		t.Fatal("loud not ok")
	}
	// Half the amplitude == -6.02 dB.
	quiet, ok := IntegratedLUFS(sine(testRate*3, 1000, 0.25, 0), testRate)
	if !ok {
		t.Fatal("quiet not ok")
	}
	diff := loud - quiet
	if math.Abs(diff-6.02) > 0.2 {
		t.Fatalf("expected ~6.02 LU difference for a 6 dB amplitude change, got %.3f", diff)
	}
}

func TestIntegratedLUFS_Gating(t *testing.T) {
	// 2 s of tone followed by 2 s of silence. The relative+absolute gates
	// should drop the silent blocks, so the integrated loudness tracks the
	// tone-only measurement rather than dropping ~3 dB for the silence.
	tone := sine(testRate*2, 1000, 0.5, 0)
	toneOnly, ok := IntegratedLUFS(tone, testRate)
	if !ok {
		t.Fatal("toneOnly not ok")
	}
	mixed := append(append([]float64{}, tone...), make([]float64, testRate*2)...)
	gated, ok := IntegratedLUFS(mixed, testRate)
	if !ok {
		t.Fatal("gated not ok")
	}
	if math.Abs(gated-toneOnly) > 0.5 {
		t.Fatalf("gated loudness %.2f should track tone-only %.2f (silence gated out)", gated, toneOnly)
	}
}

func TestIntegratedLUFS_Silence(t *testing.T) {
	if _, ok := IntegratedLUFS(make([]float64, testRate*3), testRate); ok {
		t.Fatal("silence should report ok=false")
	}
}

func TestIntegratedLUFS_TooShort(t *testing.T) {
	// Shorter than one 400 ms block.
	if _, ok := IntegratedLUFS(sine(testRate/10, 1000, 0.5, 0), testRate); ok {
		t.Fatal("sub-block-length input should report ok=false")
	}
}

func TestGainToTarget(t *testing.T) {
	// -23 LUFS measured, -16 target -> +7 dB -> linear ~2.2387.
	g := GainToTarget(-23, -16)
	if math.Abs(g-DBToLinear(7)) > 1e-9 {
		t.Fatalf("GainToTarget mismatch: %.6f", g)
	}
}

func TestNormalizeGain_QuietTone(t *testing.T) {
	// A quiet tone (~-22 LUFS) toward -16: gain should be applied and,
	// after applying it, the signal should re-measure near the target.
	p := NormalizeParams{}.WithDefaults()
	s := sine(testRate*3, 1000, 0.2, 0)
	gain, ok := NormalizeGain(s, testRate, p)
	if !ok {
		t.Fatal("expected ok")
	}
	if gain <= 1.0 {
		t.Fatalf("expected boost >1 for a quiet tone, got %.4f", gain)
	}
	gained := make([]float64, len(s))
	for i, v := range s {
		gained[i] = v * gain
	}
	lufs, _ := IntegratedLUFS(gained, testRate)
	if math.Abs(lufs-p.TargetLUFS) > 0.5 {
		t.Fatalf("after gain, loudness %.2f not near target %.2f", lufs, p.TargetLUFS)
	}
}

func TestNormalizeGain_TruePeakCeiling(t *testing.T) {
	// A near-full-scale tone can't be boosted to target without breaching
	// the ceiling, so the gain must hold the true peak under it.
	p := NormalizeParams{}.WithDefaults()
	s := sine(testRate*3, 1000, 0.95, 0)
	gain, ok := NormalizeGain(s, testRate, p)
	if !ok {
		t.Fatal("expected ok")
	}
	peak := TruePeak(s, testRate) * gain
	if peak > DBToLinear(p.TruePeakDBTP)+1e-3 {
		t.Fatalf("gained true peak %.4f exceeds ceiling", peak)
	}
}

func TestNormalizeGain_Silence(t *testing.T) {
	gain, ok := NormalizeGain(make([]float64, testRate*3), testRate, NormalizeParams{}.WithDefaults())
	if ok {
		t.Fatal("silence should report ok=false")
	}
	if gain != 1.0 {
		t.Fatalf("silence gain should be unity, got %.4f", gain)
	}
}
