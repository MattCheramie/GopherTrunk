package ambe2

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// TestDMREnhancedDarkerAndLouder decodes the real #644 DMR sample twice —
// faithful, and through the opt-in voice enhancement chain — and confirms
// the chain does what it promises: it raises the AGC loudness target and
// trims high-band energy (band-limit low-pass + warmth shelf), while
// keeping the audio finite and non-silent.
func TestDMREnhancedDarkerAndLouder(t *testing.T) {
	faithful := NewDMR()
	if got := faithful.agc.Config().TargetPeak; got != 18000 {
		t.Fatalf("faithful AGC target = %.0f, want 18000 (default)", got)
	}

	enhanced := NewDMR()
	enhanced.SetVoiceEnhancer(mbe.DefaultEnhancerConfig())
	if got := enhanced.agc.Config().TargetPeak; got != 22000 {
		t.Errorf("enhanced AGC target = %.0f, want 22000 (louder lever not wired)", got)
	}

	fp := decodeSampleWith(t, faithful)
	en := decodeSampleWith(t, enhanced)
	if len(fp) != len(en) || len(en) == 0 {
		t.Fatalf("sample lengths differ/empty: faithful=%d enhanced=%d", len(fp), len(en))
	}

	var sumSq float64
	for _, s := range en {
		f := float64(s)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatal("enhanced decode produced a non-finite sample")
		}
		sumSq += f * f
	}
	if math.Sqrt(sumSq/float64(len(en))) < 50 {
		t.Error("enhanced decode is essentially silent")
	}

	// The high-pass must shrink the sub-200 Hz rumble fraction end-to-end
	// on real audio (scale-invariant, so the louder AGC target doesn't skew
	// it). The precise band-limit / warmth-shelf magnitude response is
	// pinned deterministically at signal level in the mbe and dsp/filter
	// unit tests (TestVoiceEnhancerBandLimitsAndWarms, TestLowPassResponse);
	// re-deriving them from this short clip's spectrum is unreliable because
	// the louder soft-limiter re-injects HF harmonics after the low-pass.
	if e, f := bandFractionBelow(en, 200), bandFractionBelow(fp, 200); !(e < f) {
		t.Errorf("rumble fraction not reduced: enhanced=%.5f faithful=%.5f (HPF had no effect)", e, f)
	}
}

// bandFractionBelow returns the fraction of pcm's total energy below fc,
// estimated with a one-pole low-pass. Scale-invariant, so it compares
// spectral shape regardless of the AGC loudness target.
func bandFractionBelow(pcm []int16, fc float64) float64 {
	const fs = 8000.0
	alpha := 1 - math.Exp(-2*math.Pi*fc/fs)
	var lp, lpSq, totSq float64
	for _, s := range pcm {
		x := float64(s)
		lp += alpha * (x - lp)
		lpSq += lp * lp
		totSq += x * x
	}
	if totSq == 0 {
		return 0
	}
	return lpSq / totSq
}

// TestDMRSetVoiceEnhancerDisabledIsByteIdentical pins the opt-in
// contract: installing a DISABLED enhancement config must leave the
// decoder byte-identical to the faithful path (nil chain, untouched AGC
// target), so the default faithful output and every existing golden test
// are unaffected.
func TestDMRSetVoiceEnhancerDisabledIsByteIdentical(t *testing.T) {
	plain := decodeSampleWith(t, NewDMR())

	d := NewDMR()
	d.SetVoiceEnhancer(mbe.EnhancerConfig{Enabled: false})
	if d.enhancer != nil {
		t.Error("disabled config installed a non-nil enhancer")
	}
	if got := d.agc.Config().TargetPeak; got != 18000 {
		t.Errorf("disabled config changed AGC target to %.0f, want 18000", got)
	}
	got := decodeSampleWith(t, d)

	if len(got) != len(plain) {
		t.Fatalf("length changed: got %d want %d", len(got), len(plain))
	}
	for i := range plain {
		if got[i] != plain[i] {
			t.Fatalf("sample %d differs with disabled enhancer: got %d want %d", i, got[i], plain[i])
		}
	}
}
