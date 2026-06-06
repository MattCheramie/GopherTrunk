package hunt

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
)

// frameWithTones builds a synthetic spectrum frame: a flat noise floor with
// elevated bins at the given indices. n must be even (DC sits at n/2).
func frameWithTones(n int, centerHz, rateHz uint32, floor float32, tones map[int]float32) spectrum.Frame {
	bins := make([]float32, n)
	for i := range bins {
		bins[i] = floor
	}
	for idx, db := range tones {
		bins[idx] = db
	}
	return spectrum.Frame{CenterHz: centerHz, SampleRate: rateHz, Bins: bins}
}

func TestDetectPeaks_FindsTonesAtCorrectHz(t *testing.T) {
	const n = 1024
	const center = 851_000_000
	const rate = 2_048_000
	// Place tones at bins 300 and 800 (both off-DC, away from edges).
	frame := frameWithTones(n, center, rate, -80, map[int]float32{300: -20, 800: -30})

	peaks := DetectPeaks(frame, PeakOptions{ThresholdDb: 10})
	if len(peaks) != 2 {
		t.Fatalf("len(peaks) = %d, want 2: %+v", len(peaks), peaks)
	}
	// binToHz: base = center - rate/2, step = rate/n.
	step := float64(rate) / n
	base := float64(center) - float64(rate)/2
	want := map[int]uint32{
		300: uint32(base + 300*step),
		800: uint32(base + 800*step),
	}
	got := map[uint32]bool{}
	for _, p := range peaks {
		got[p.FreqHz] = true
	}
	for _, w := range want {
		if !nearAny(got, w, uint32(step)) {
			t.Errorf("missing peak near %d Hz; got %v", w, got)
		}
	}
	// Strongest (bin 300, -20) should sort first.
	if peaks[0].FreqHz != want[300] {
		t.Errorf("peaks[0] = %d Hz, want strongest tone at %d Hz", peaks[0].FreqHz, want[300])
	}
}

func TestDetectPeaks_RejectsDCAndThreshold(t *testing.T) {
	const n = 512
	// A tone exactly at DC (bin n/2) must be ignored, and a tone only 5 dB over
	// the floor must fail the 10 dB threshold.
	frame := frameWithTones(n, 100_000_000, 1_000_000, -80, map[int]float32{
		n / 2: -10, // DC spike — rejected
		100:   -75, // only +5 dB — below threshold
	})
	peaks := DetectPeaks(frame, PeakOptions{ThresholdDb: 10})
	if len(peaks) != 0 {
		t.Errorf("expected no peaks (DC + sub-threshold), got %+v", peaks)
	}
}

func TestDetectPeaks_MinSpacingKeepsStronger(t *testing.T) {
	const n = 1024
	const rate = 1_024_000 // 1000 Hz/bin
	// Two tones 2 bins apart (2 kHz). With MinSpacingHz=12.5 kHz only the
	// stronger survives.
	frame := frameWithTones(n, 450_000_000, rate, -90, map[int]float32{
		400: -40,
		402: -20, // stronger
	})
	peaks := DetectPeaks(frame, PeakOptions{ThresholdDb: 10, MinSpacingHz: 12_500})
	if len(peaks) != 1 {
		t.Fatalf("len(peaks) = %d, want 1 (min-spacing): %+v", len(peaks), peaks)
	}
	step := float64(rate) / n
	base := float64(450_000_000) - float64(rate)/2
	wantStronger := uint32(base + 402*step)
	if !nearAny(map[uint32]bool{peaks[0].FreqHz: true}, wantStronger, uint32(step)) {
		t.Errorf("kept peak = %d Hz, want the stronger tone near %d Hz", peaks[0].FreqHz, wantStronger)
	}
}

func TestDetectPeaks_GuardBinsRejectEdges(t *testing.T) {
	const n = 1024
	frame := frameWithTones(n, 1_000_000, 1_000_000, -80, map[int]float32{
		2:     -20, // near low edge
		n - 3: -20, // near high edge
		500:   -20, // valid, mid-band
	})
	peaks := DetectPeaks(frame, PeakOptions{ThresholdDb: 10, GuardBins: 16})
	if len(peaks) != 1 {
		t.Fatalf("len(peaks) = %d, want 1 (edges guarded): %+v", len(peaks), peaks)
	}
}

func nearAny(set map[uint32]bool, target, tol uint32) bool {
	for f := range set {
		if absDiff(f, target) <= tol {
			return true
		}
	}
	return false
}
