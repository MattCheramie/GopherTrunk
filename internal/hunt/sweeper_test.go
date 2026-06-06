package hunt

import (
	"context"
	"math"
	"math/cmplx"
	"testing"
)

// toneIQ generates a unit-amplitude complex exponential at offsetHz relative to
// DC, sampled at rateHz, n samples long.
func toneIQ(n int, offsetHz, rateHz float64) []complex64 {
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		phase := 2 * math.Pi * offsetHz * float64(i) / rateHz
		out[i] = complex64(cmplx.Exp(complex(0, phase)))
	}
	return out
}

func TestSweeper_FindsCarrierAtExpectedFreq(t *testing.T) {
	const (
		rate     = 2_048_000
		fftSize  = 1024
		guard    = 0.1
		offsetHz = 400_000.0 // carrier 400 kHz above the step center (off-DC)
	)
	// One-step band: with guardFrac 0.1, step = rate*0.8, halfUsable = step/2.
	step := uint32(float64(rate) * (1 - 2*guard))
	halfUsable := step / 2
	bandLow := uint32(450_000_000)
	center := bandLow + halfUsable
	bandHigh := bandLow + step - 1 // exactly one step wide
	wantHz := center + uint32(offsetHz)

	src := NewFileIQSource(toneIQ(fftSize*4, offsetHz, rate), rate)
	sw, err := NewSweeper(SweepOptions{
		Source:    src,
		Bands:     []Band{{LowHz: bandLow, HighHz: bandHigh}},
		FFTSize:   fftSize,
		GuardFrac: guard,
		PeakOpts:  PeakOptions{ThresholdDb: 10},
	})
	if err != nil {
		t.Fatalf("NewSweeper: %v", err)
	}
	cands, err := sw.Sweep(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if len(cands) == 0 {
		t.Fatalf("no candidates found")
	}
	binHz := uint32(float64(rate) / fftSize)
	found := false
	for _, c := range cands {
		if absDiff(c.FreqHz, wantHz) <= 2*binHz {
			found = true
		}
	}
	if !found {
		t.Errorf("carrier near %d Hz not found; candidates=%+v (tol %d Hz)", wantHz, cands, 2*binHz)
	}
	// The source was tuned to the single step center.
	if len(src.TuneCalls()) == 0 || src.TuneCalls()[0] != center {
		t.Errorf("first tune = %v, want center %d", src.TuneCalls(), center)
	}
}

func TestSweeper_SilenceYieldsNoCandidates(t *testing.T) {
	const rate = 2_048_000
	src := NewFileIQSource(make([]complex64, 4096), rate) // all-zero IQ
	sw, err := NewSweeper(SweepOptions{
		Source:   src,
		Bands:    []Band{{LowHz: 450_000_000, HighHz: 451_000_000}},
		FFTSize:  1024,
		PeakOpts: PeakOptions{ThresholdDb: 10},
	})
	if err != nil {
		t.Fatalf("NewSweeper: %v", err)
	}
	cands, err := sw.Sweep(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if len(cands) != 0 {
		t.Errorf("silence produced candidates: %+v", cands)
	}
}

func TestNewSweeper_Validation(t *testing.T) {
	src := NewFileIQSource(toneIQ(2048, 1000, 48000), 48000)
	if _, err := NewSweeper(SweepOptions{Source: src}); err == nil {
		t.Error("expected error for no bands")
	}
	if _, err := NewSweeper(SweepOptions{Bands: []Band{{LowHz: 1, HighHz: 2}}}); err == nil {
		t.Error("expected error for nil source")
	}
	if _, err := NewSweeper(SweepOptions{Source: src, Bands: []Band{{1, 2}}, FFTSize: 1000}); err == nil {
		t.Error("expected error for non-power-of-two FFT size")
	}
}
