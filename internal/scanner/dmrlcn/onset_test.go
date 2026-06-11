package dmrlcn

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
	"time"
)

// makeIQ builds n complex samples: low-amplitude white noise, plus an
// optional tone at toneOffsetHz when tone is true.
func makeIQ(n int, fs float64, toneOffsetHz float64, toneAmp float64, noiseAmp float64, tone bool, phase *float64, rng *rand.Rand) []complex64 {
	out := make([]complex64, n)
	w := 2 * math.Pi * toneOffsetHz / fs
	for i := 0; i < n; i++ {
		var s complex128
		s += complex(noiseAmp*(rng.Float64()*2-1), noiseAmp*(rng.Float64()*2-1))
		if tone {
			s += complex(toneAmp, 0) * cmplx.Exp(complex(0, *phase))
			*phase += w
		}
		out[i] = complex64(s)
	}
	return out
}

func TestOnsetDetectorEmitsOnKeyup(t *testing.T) {
	const (
		fs           = 2_400_000.0
		center       = uint32(851_500_000)
		targetFreq   = uint32(851_650_000) // +150 kHz, on the 12.5 kHz grid
		frameSize    = 4096
		toneOffsetHz = 150_000.0
	)
	rng := rand.New(rand.NewSource(1))
	clock := time.Unix(1_700_000_000, 0)

	var onsets []CarrierOnset
	det := newOnsetDetector(onsetConfig{
		CenterHz:     center,
		SampleRateHz: uint32(fs),
		FFTSize:      frameSize,
		FrameRate:    1e9, // force one FFT per window for a deterministic test
		Debounce:     2,
		Now:          func() time.Time { return clock },
	}, func(o CarrierOnset) { onsets = append(onsets, o) })

	if det.channelCount() < 5 {
		t.Fatalf("grid has too few channels: %d", det.channelCount())
	}

	// Idle phase: noise only, lets the noise floor settle.
	phase := 0.0
	for i := 0; i < 40; i++ {
		clock = clock.Add(time.Millisecond)
		det.Process(makeIQ(frameSize, fs, toneOffsetHz, 0, 0.002, false, &phase, rng))
	}
	if len(onsets) != 0 {
		t.Fatalf("onset emitted during idle: %+v", onsets)
	}

	// Key-up: strong carrier on the target channel.
	keyAt := clock.Add(time.Millisecond)
	for i := 0; i < 20; i++ {
		clock = clock.Add(time.Millisecond)
		det.Process(makeIQ(frameSize, fs, toneOffsetHz, 0.3, 0.002, true, &phase, rng))
	}

	if len(onsets) != 1 {
		t.Fatalf("got %d onsets, want exactly 1: %+v", len(onsets), onsets)
	}
	if onsets[0].FreqHz != targetFreq {
		t.Errorf("onset freq = %d, want %d", onsets[0].FreqHz, targetFreq)
	}
	if onsets[0].At.Before(keyAt) {
		t.Errorf("onset time %v predates key-up %v", onsets[0].At, keyAt)
	}
}

func TestOnsetDetectorExcludesControlChannel(t *testing.T) {
	const (
		fs           = 2_400_000.0
		center       = uint32(851_500_000)
		targetFreq   = uint32(851_650_000)
		frameSize    = 4096
		toneOffsetHz = 150_000.0
	)
	rng := rand.New(rand.NewSource(2))
	clock := time.Unix(1_700_000_000, 0)

	var onsets []CarrierOnset
	det := newOnsetDetector(onsetConfig{
		CenterHz:     center,
		SampleRateHz: uint32(fs),
		FFTSize:      frameSize,
		FrameRate:    1e9,
		Debounce:     2,
		Exclude:      []uint32{targetFreq}, // pretend this is the CC carrier
		Now:          func() time.Time { return clock },
	}, func(o CarrierOnset) { onsets = append(onsets, o) })

	phase := 0.0
	for i := 0; i < 40; i++ {
		det.Process(makeIQ(frameSize, fs, toneOffsetHz, 0, 0.002, false, &phase, rng))
	}
	for i := 0; i < 20; i++ {
		det.Process(makeIQ(frameSize, fs, toneOffsetHz, 0.3, 0.002, true, &phase, rng))
	}
	if len(onsets) != 0 {
		t.Fatalf("excluded channel emitted onsets: %+v", onsets)
	}
}
