package rfscope

import (
	"context"
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// synthGatedTone builds a wideband capture with a single complex tone at
// offsetHz from center, present only during [onSec, offSec], plus light white
// noise everywhere so a noise floor exists for carrier discovery.
func synthGatedTone(rate float64, durSec, onSec, offSec, offsetHz, amp float64) []complex64 {
	n := int(rate * durSec)
	iq := make([]complex64, n)
	rng := rand.New(rand.NewSource(1))
	onN, offN := int(onSec*rate), int(offSec*rate)
	w := 2 * math.Pi * offsetHz / rate
	for i := 0; i < n; i++ {
		var s complex128
		s = complex(0.01*(rng.Float64()-0.5), 0.01*(rng.Float64()-0.5))
		if i >= onN && i < offN {
			s += complex(amp, 0) * cmplx.Exp(complex(0, w*float64(i)))
		}
		iq[i] = complex64(s)
	}
	return iq
}

func TestSegmentGatedTone(t *testing.T) {
	t.Parallel()
	const (
		rate     = 240_000.0
		center   = 450_000_000
		offsetHz = 30_000.0
		onSec    = 0.10
		offSec   = 0.30
		durSec   = 0.50
	)
	iq := synthGatedTone(rate, durSec, onSec, offSec, offsetHz, 1.0)
	src := NewMemSource(iq, center, rate)

	scene, in, err := Segment(context.Background(), src, SegmentConfig{})
	if err != nil {
		t.Fatalf("Segment: %v", err)
	}
	if len(scene.Bursts) != 1 {
		t.Fatalf("want exactly 1 burst, got %d: %+v", len(scene.Bursts), scene.Bursts)
	}
	b := scene.Bursts[0]

	// Carrier frequency within one channel raster of center+offset.
	wantFreq := uint32(center + offsetHz)
	if d := absU32(b.FreqHz, wantFreq); d > 12_500 {
		t.Fatalf("burst freq %d off from %d by %d Hz", b.FreqHz, wantFreq, d)
	}
	// Onset/offset within ~20 ms of the gate edges.
	if math.Abs(b.StartSec-onSec) > 0.02 {
		t.Fatalf("start %.3fs not near %.3fs", b.StartSec, onSec)
	}
	if math.Abs(b.EndSec-offSec) > 0.02 {
		t.Fatalf("end %.3fs not near %.3fs", b.EndSec, offSec)
	}
	// A CW tone is tonal, not noise-like: flatness well below 1.
	if b.SpectralFlatness >= 0.6 {
		t.Fatalf("tone flatness %.3f should be ≪1", b.SpectralFlatness)
	}
	// BurstIQ serves the stored baseband.
	bb, r, err := in.BurstIQ(b)
	if err != nil || len(bb) == 0 || r <= 0 {
		t.Fatalf("BurstIQ: len=%d rate=%g err=%v", len(bb), r, err)
	}
}

func TestSegmentNoCarrier(t *testing.T) {
	t.Parallel()
	// Pure low-level noise: no carrier stands above the floor, so no bursts.
	rng := rand.New(rand.NewSource(2))
	n := int(240_000 * 0.3)
	iq := make([]complex64, n)
	for i := range iq {
		iq[i] = complex64(complex(0.01*(rng.Float64()-0.5), 0.01*(rng.Float64()-0.5)))
	}
	scene, _, err := Segment(context.Background(), NewMemSource(iq, 450_000_000, 240_000), SegmentConfig{})
	if err != nil {
		t.Fatalf("Segment: %v", err)
	}
	if len(scene.Bursts) != 0 {
		t.Fatalf("want no bursts from pure noise, got %d", len(scene.Bursts))
	}
	if scene.WindowSec <= 0 || scene.CenterHz != 450_000_000 {
		t.Fatalf("scene header wrong: %+v", scene)
	}
}

func TestSegmentShortCaptureErrors(t *testing.T) {
	t.Parallel()
	src := NewMemSource(makeIQ(100), 1000, 240_000)
	if _, _, err := Segment(context.Background(), src, SegmentConfig{}); err == nil {
		t.Fatalf("want error for capture shorter than FFT size")
	}
}

func absU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
