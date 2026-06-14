package carriers

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
)

func TestDetectPeaksFindsTonesAndSorts(t *testing.T) {
	const n, center, rate = 1024, 851_000_000, 2_048_000
	bins := make([]float32, n)
	for i := range bins {
		bins[i] = -80
	}
	bins[300] = -20 // strongest
	bins[800] = -30
	frame := spectrum.Frame{CenterHz: center, SampleRate: rate, Bins: bins}

	peaks := DetectPeaks(frame, PeakOptions{ThresholdDb: 10})
	if len(peaks) != 2 {
		t.Fatalf("len(peaks) = %d, want 2: %+v", len(peaks), peaks)
	}
	step := float64(rate) / n
	base := float64(center) - float64(rate)/2
	want300 := uint32(base + 300*step)
	if absDiff(peaks[0].FreqHz, want300) > uint32(step) {
		t.Errorf("peaks[0] = %d Hz, want strongest near %d Hz", peaks[0].FreqHz, want300)
	}
}

func TestInferGridPicksTwelveAndHalfAndSnaps(t *testing.T) {
	base := uint32(851_012_500)
	freqs := make([]uint32, 0, 6)
	for i := uint32(0); i < 6; i++ {
		freqs = append(freqs, base+i*12_500)
	}
	step, phase := InferGrid(freqs)
	if step != 12_500 {
		t.Fatalf("inferred step %d Hz, want 12500", step)
	}
	if phase != base%12_500 {
		t.Errorf("inferred phase %d Hz, want %d", phase, base%12_500)
	}
	// A carrier ~300 Hz off the raster snaps back within one bin's correction.
	if got := SnapHz(base+12_500+300, step, phase, 586); got != base+12_500 {
		t.Errorf("SnapHz = %d, want %d", got, base+12_500)
	}
	// A carrier 3 kHz off-raster is left untouched.
	if got := SnapHz(base+3_000, step, phase, 586); got != base+3_000 {
		t.Errorf("far-off-grid carrier changed to %d, want unchanged", got)
	}
}
