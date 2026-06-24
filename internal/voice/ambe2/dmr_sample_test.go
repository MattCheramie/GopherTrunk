package ambe2

import (
	"math"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// decodeSampleWith decodes the committed #644 fixture through d and
// returns the PCM, skipping the test if the fixture is absent.
func decodeSampleWith(t *testing.T, d *Decoder) []int16 {
	t.Helper()
	raw, err := os.ReadFile("testdata/dmr-voice.raw")
	if err != nil {
		t.Skipf("no #644 sample at testdata/dmr-voice.raw: %v", err)
	}
	fs := d.FrameSize()
	var samples []int16
	for i := 0; i+fs <= len(raw); i += fs {
		out, err := d.Decode(raw[i : i+fs])
		if err != nil {
			t.Fatalf("decode frame %d: %v", i/fs, err)
		}
		samples = append(samples, out...)
	}
	return samples
}

// highBandRatio is the fraction of signal energy above ~1.5 kHz,
// estimated with a one-pole high-pass. Higher = brighter.
func highBandRatio(pcm []int16) float64 {
	const fs, fc = 8000.0, 1500.0
	alpha := 1 - math.Exp(-2*math.Pi*fc/fs)
	var lp, hpSq, totSq float64
	for _, s := range pcm {
		x := float64(s)
		lp += alpha * (x - lp)
		hp := x - lp
		hpSq += hp * hp
		totSq += x * x
	}
	if totSq == 0 {
		return 0
	}
	return hpSq / totSq
}

// TestDMRWarmVariantRegisteredAndDarker confirms the opt-in
// "ambe2-dmr-warm" vocoder is selectable via the registry and that its
// high-shelf measurably reduces high-band energy versus the plain
// "ambe2-dmr" decoder on the real #644 sample — while still producing
// finite, non-silent audio. This is the opt-in naturalness tone control;
// it does not change the default decoder (the warm tilt is nil there).
func TestDMRWarmVariantRegisteredAndDarker(t *testing.T) {
	if _, err := voice.DefaultRegistry.New(DMRWarmVocoderName); err != nil {
		t.Fatalf("warm vocoder %q not registered: %v", DMRWarmVocoderName, err)
	}

	plain := decodeSampleWith(t, NewDMR())
	warm := decodeSampleWith(t, NewDMRWarm())
	if len(plain) != len(warm) || len(warm) == 0 {
		t.Fatalf("sample lengths differ/empty: plain=%d warm=%d", len(plain), len(warm))
	}

	// Warm output stays finite and non-silent.
	var sumSq float64
	for _, s := range warm {
		f := float64(s)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatal("warm decode produced non-finite sample")
		}
		sumSq += f * f
	}
	if math.Sqrt(sumSq/float64(len(warm))) < 50 {
		t.Error("warm decode is essentially silent")
	}

	plainHi := highBandRatio(plain)
	warmHi := highBandRatio(warm)
	if !(warmHi < plainHi) {
		t.Errorf("warm high-band ratio %.4f not below plain %.4f — shelf had no effect", warmHi, plainHi)
	}
	// Expect a clear but modest reduction (the shelf trims ~2 dB of highs).
	if warmHi > 0.95*plainHi {
		t.Errorf("warm high-band reduction too small: warm=%.4f plain=%.4f (%.1f%% of plain)",
			warmHi, plainHi, 100*warmHi/plainHi)
	}
}

// TestDecodeDMRSampleIssue644 decodes the real DMR Tier III voice
// capture attached to issue #644 (the "machine voice" report) through
// the AMBE+2 3600x2450 DMR decoder and asserts the output is
// well-formed audio. It is a deterministic regression guard for the
// failure modes that issue cycled through:
//
//   - empty recordings (#654: the slot router dropped every superframe),
//   - decode errors / panics on real on-air frames, and
//   - all-silent or NaN/Inf output.
//
// It does NOT (and can not) assert "naturalness": two independent
// MBE-family decoders synthesise harmonics with different absolute
// phase, so a sample-level waveform comparison against a reference
// decoder (e.g. mbelib/DSD-FME) is uncorrelated by construction and
// the calibrate xcorr metric does not apply here. Perceptual
// regressions are caught upstream by the synthesis unit tests
// (e.g. TestSynthUnvoicedOverlapAddNoiseFloorIsFlat in the mbe pkg).
//
// The .raw fixture is 7-byte (49-bit) AMBE+2 frames, MSB-first packed,
// post-FEC — the on-disk sidecar format documented in testdata/README.md.
func TestDecodeDMRSampleIssue644(t *testing.T) {
	raw, err := os.ReadFile("testdata/dmr-voice.raw")
	if err != nil {
		t.Skipf("no #644 sample at testdata/dmr-voice.raw: %v", err)
	}
	d := NewDMR()
	fs := d.FrameSize()
	if len(raw)%fs != 0 {
		t.Fatalf("fixture size %d is not a multiple of frame size %d", len(raw), fs)
	}
	wantFrames := len(raw) / fs

	var samples []int16
	for i := 0; i+fs <= len(raw); i += fs {
		out, err := d.Decode(raw[i : i+fs])
		if err != nil {
			t.Fatalf("decode frame %d: %v", i/fs, err)
		}
		if len(out) != 160 {
			t.Fatalf("frame %d: got %d samples, want 160", i/fs, len(out))
		}
		samples = append(samples, out...)
	}
	if got := len(samples); got != wantFrames*160 {
		t.Fatalf("decoded %d samples, want %d (%d frames × 160)", got, wantFrames*160, wantFrames)
	}

	// Output must be finite, non-silent, and not pinned hard against the
	// rails for the whole call (a sign of a broken envelope / AGC).
	var sumSq float64
	var peak, nonZero int
	for _, s := range samples {
		f := float64(s)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatal("non-finite sample in decoded output")
		}
		sumSq += f * f
		if s != 0 {
			nonZero++
		}
		a := int(s)
		if a < 0 {
			a = -a
		}
		if a > peak {
			peak = a
		}
	}
	if nonZero == 0 {
		t.Fatal("decoded audio is all-silent — #644 empty-recording regression")
	}
	rms := math.Sqrt(sumSq / float64(len(samples)))
	// The AGC targets a peak of ~18k; a healthy voice call lands well
	// above the noise floor. These are deliberately loose sanity rails,
	// not a quality bar.
	if rms < 50 {
		t.Errorf("decoded RMS %.1f too low — output is essentially silent", rms)
	}
	if peak < 2000 {
		t.Errorf("decoded peak %d too low — AGC/envelope likely broken", peak)
	}
}
