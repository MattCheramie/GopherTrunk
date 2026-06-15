package hunt

import "testing"

// TestSnapLoneCarrierCorrectsBinOffset: the field repro — a P25 trunk really on
// 440.125000 MHz reported a few hundred Hz off by FFT-bin quantization must snap
// back to the true channel centre (so it tunes dead-on and locks). A single
// candidate falls back to the finest standard step (6.25 kHz, phase 0).
func TestSnapLoneCarrierCorrectsBinOffset(t *testing.T) {
	const rate, fft = 2_400_000, 4096 // binHz ≈ 586
	in := []Candidate{{FreqHz: 440_124_700, SNRDb: 20}}
	out := snapCandidatesToGrid(in, rate, fft)
	if len(out) != 1 {
		t.Fatalf("got %d candidates, want 1", len(out))
	}
	if out[0].FreqHz != 440_125_000 {
		t.Errorf("snapped to %d Hz, want 440125000", out[0].FreqHz)
	}
}

// TestInferGridPicksTwelveAndHalf: a set of carriers on a 12.5 kHz raster is
// detected as a 12.5 kHz step (not the finer 6.25 kHz, which also fits — the
// coarsest tightly-clustered raster wins).
func TestInferGridPicksTwelveAndHalf(t *testing.T) {
	base := uint32(851_012_500)
	var cands []Candidate
	for i := uint32(0); i < 6; i++ {
		cands = append(cands, Candidate{FreqHz: base + i*12_500})
	}
	step, phase := inferGrid(cands)
	if step != 12_500 {
		t.Errorf("inferred step %d Hz, want 12500", step)
	}
	if phase != base%12_500 {
		t.Errorf("inferred phase %d Hz, want %d", phase, base%12_500)
	}
}

// TestSnapMergesJitteredDuplicates: two bin-jittered detections of the same
// channel collapse to one snapped candidate, strongest SNR winning.
func TestSnapMergesJitteredDuplicates(t *testing.T) {
	const rate, fft = 2_400_000, 4096
	in := []Candidate{
		{FreqHz: 460_000_300, SNRDb: 12}, // +300 above 460.000 (within one bin)
		{FreqHz: 459_999_500, SNRDb: 18}, // -500 below 460.000 (stronger, within one bin)
	}
	out := snapCandidatesToGrid(in, rate, fft)
	if len(out) != 1 {
		t.Fatalf("got %d candidates, want 1 (merged)", len(out))
	}
	if out[0].FreqHz != 460_000_000 {
		t.Errorf("merged candidate at %d Hz, want 460000000", out[0].FreqHz)
	}
	if out[0].SNRDb != 18 {
		t.Errorf("merged SNR %.0f, want 18 (strongest)", out[0].SNRDb)
	}
}

// TestSnapLeavesFarOffGridCarrier: a carrier more than one FFT bin off any grid
// point is genuinely off-raster and must be kept untouched (snap-only, never
// fabricate a frequency).
func TestSnapLeavesFarOffGridCarrier(t *testing.T) {
	const rate, fft = 2_400_000, 4096 // binHz ≈ 586
	// 3 kHz off the nearest 6.25 kHz point — far beyond one bin.
	in := []Candidate{{FreqHz: 462_003_000, SNRDb: 10}}
	out := snapCandidatesToGrid(in, rate, fft)
	if len(out) != 1 || out[0].FreqHz != 462_003_000 {
		t.Errorf("off-grid carrier changed to %v, want unchanged 462003000", out)
	}
}
