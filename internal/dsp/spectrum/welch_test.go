package spectrum

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

func hannSum(win []float64) float64 {
	var s float64
	for _, v := range win {
		s += v * v
	}
	return s
}

func TestWelchPeaksAtTone(t *testing.T) {
	const (
		size     = 256
		sampleHz = 48000.0
		toneHz   = 6000.0
		segments = 8
	)
	plan := fft.New(size)
	win := window.Hann(size)
	winSum := hannSum(win)

	// A full tone spanning several overlapping segments.
	total := size * segments / 2 // 50% overlap → ~segments periodograms
	in := make([]complex64, total)
	w := 2 * math.Pi * toneHz / sampleHz
	for n := 0; n < total; n++ {
		in[n] = complex64(complex(math.Cos(w*float64(n)), math.Sin(w*float64(n))))
	}

	dst := Welch(plan, win, winSum, in, size/2)
	if len(dst) != size {
		t.Fatalf("Welch len = %d, want %d", len(dst), size)
	}

	center := size / 2
	wantBin := center + int(math.Round(toneHz*size/sampleHz))
	peak, peakVal := 0, float32(math.Inf(-1))
	for i, v := range dst {
		if v > peakVal {
			peak, peakVal = i, v
		}
	}
	if peak != wantBin {
		t.Errorf("Welch peak bin = %d, want %d", peak, wantBin)
	}
	if peakVal < -3 || peakVal > 1 {
		t.Errorf("Welch peak level = %.2f dBFS, want ~0", peakVal)
	}
}

func TestWelchAveragingLowersNoiseVariance(t *testing.T) {
	const (
		size = 256
	)
	plan := fft.New(size)
	win := window.Hann(size)
	winSum := hannSum(win)

	// White complex noise long enough for many segments.
	rng := rand.New(rand.NewSource(1))
	const total = size * 32
	noise := make([]complex64, total)
	for i := range noise {
		noise[i] = complex64(complex(rng.NormFloat64(), rng.NormFloat64()))
	}

	// Single periodogram (one segment) vs averaged Welch over all segments.
	single := Welch(plan, win, winSum, noise[:size], 0)
	avg := Welch(plan, win, winSum, noise, size/2)

	varOf := func(x []float32) float64 {
		var mean float64
		for _, v := range x {
			mean += float64(v)
		}
		mean /= float64(len(x))
		var ss float64
		for _, v := range x {
			d := float64(v) - mean
			ss += d * d
		}
		return ss / float64(len(x))
	}

	if varOf(avg) >= varOf(single) {
		t.Errorf("averaged PSD variance %.3f not below single-segment %.3f",
			varOf(avg), varOf(single))
	}
}

func TestWelchTooShortReturnsFloor(t *testing.T) {
	const size = 64
	plan := fft.New(size)
	win := window.Hann(size)
	winSum := hannSum(win)

	dst := Welch(plan, win, winSum, make([]complex64, size-1), 0)
	if len(dst) != size {
		t.Fatalf("len = %d, want %d", len(dst), size)
	}
	for i, v := range dst {
		if v > -200 {
			t.Errorf("bin %d = %v, want the clamp floor for an empty average", i, v)
		}
	}
}
