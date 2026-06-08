package survey

import (
	"math"
	"testing"
)

// tone builds a real sinusoid of the given frequency and amplitude.
func tone(n int, rateHz, freqHz, amp float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = amp * math.Sin(2*math.Pi*freqHz*float64(i)/rateHz)
	}
	return out
}

// TestAnalyzeAnalogFM_CTCSS FM-modulates a 100 Hz CTCSS tone under voice and
// asserts AnalyzeAnalogFM reports the carrier active and detects the tone,
// exercising the reuse of the conventional scanner's CTCSS detector.
func TestAnalyzeAnalogFM_CTCSS(t *testing.T) {
	const n = 96_000 // 2 s at 48 kHz — several CTCSS Goertzel blocks
	msg := tone(n, testRateHz, 100.0, 1.0)
	// Mix in a little higher-frequency voice so it isn't a pure tone.
	for i, v := range tone(n, testRateHz, 900.0, 0.3) {
		msg[i] += v
	}
	iq := fmModulate(msg, testRateHz, 2500)

	rep := AnalyzeAnalogFM(iq, testRateHz)
	if !rep.Active {
		t.Fatalf("expected active carrier, got %+v", rep)
	}
	if math.Abs(rep.CTCSSHz-100.0) > 1.0 {
		t.Errorf("CTCSS = %.1f Hz, want ≈100", rep.CTCSSHz)
	}
}

// TestAnalyzeAnalogFM_Silence asserts a zero/near-zero buffer reads as inactive
// (below squelch), so the survey doesn't report phantom analog carriers.
func TestAnalyzeAnalogFM_Silence(t *testing.T) {
	rep := AnalyzeAnalogFM(make([]complex64, 48_000), testRateHz)
	if rep.Active {
		t.Errorf("silence reported active: %+v", rep)
	}
}

// TestCaptureAnalogAudioResamples checks the audio chain produces ~8 kHz PCM of
// the expected length from a 48 kHz channel capture.
func TestCaptureAnalogAudioResamples(t *testing.T) {
	const n = 48_000 // 1 s at 48 kHz
	iq := fmModulate(tone(n, testRateHz, 1000.0, 1.0), testRateHz, 2500)
	pcm := CaptureAnalogAudio(iq, testRateHz)
	// 48 kHz → 8 kHz is a 6:1 decimation; allow a small polyphase tail.
	want := n / 6
	if len(pcm) < want-64 || len(pcm) > want+64 {
		t.Errorf("audio length = %d, want ≈%d", len(pcm), want)
	}
}
