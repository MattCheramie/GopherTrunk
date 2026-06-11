package spectrum

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

func TestNewRejectsBadFFTSize(t *testing.T) {
	if _, err := New(Options{FFTSize: 1000}); err == nil {
		t.Error("FFTSize=1000 should error (not power of two)")
	}
	if _, err := New(Options{FFTSize: -1}); err != nil {
		// Negative defaults to 4096; should NOT error.
		t.Errorf("FFTSize=-1 errored, want default fallback: %v", err)
	}
}

func TestProducerEmitsFrameWithExpectedShape(t *testing.T) {
	p, err := New(Options{
		FFTSize:      64,
		FrameRate:    100,
		CenterFreqHz: 851_000_000,
		SampleRateHz: 2_048_000,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	in := make(chan []complex64, 4)
	out := make(chan Frame, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = p.Run(ctx, in, out) }()

	// Feed a single FFT-sized chunk of DC (constant 1+0i).
	chunk := make([]complex64, 64)
	for i := range chunk {
		chunk[i] = complex(1, 0)
	}
	in <- chunk

	select {
	case f := <-out:
		if len(f.Bins) != 64 {
			t.Errorf("Bins len = %d, want 64", len(f.Bins))
		}
		if f.CenterHz != 851_000_000 {
			t.Errorf("CenterHz = %d, want 851000000", f.CenterHz)
		}
		if f.SampleRate != 2_048_000 {
			t.Errorf("SampleRate = %d", f.SampleRate)
		}
		// DC tone with FFT-shift should peak at index 32 (centre).
		var peakBin int
		var peakVal float32 = -math.MaxFloat32
		for i, v := range f.Bins {
			if v > peakVal {
				peakVal = v
				peakBin = i
			}
		}
		if peakBin != 32 {
			t.Errorf("DC peak at bin %d, want 32 (FFT-shifted centre)", peakBin)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive a frame within 1s")
	}
}

func TestProducerRateLimits(t *testing.T) {
	// 50 fps producer should emit ~5 frames in 100 ms regardless of
	// input rate.
	p, err := New(Options{FFTSize: 64, FrameRate: 50})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	in := make(chan []complex64, 16)
	out := make(chan Frame, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = p.Run(ctx, in, out) }()

	// Pump chunks much faster than 50 fps.
	go func() {
		defer close(in)
		deadline := time.Now().Add(150 * time.Millisecond)
		chunk := make([]complex64, 64)
		for time.Now().Before(deadline) {
			select {
			case in <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	count := len(out)
	// Expected ~5 frames at 50 fps over ~100 ms of active streaming.
	// Allow generous slop for goroutine scheduling jitter.
	if count < 3 || count > 15 {
		t.Errorf("frame count = %d, expected ~5 (rate-limited)", count)
	}
}

func TestProducerStopsOnInputClose(t *testing.T) {
	p, err := New(Options{FFTSize: 64, FrameRate: 100})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	in := make(chan []complex64, 1)
	out := make(chan Frame, 1)

	done := make(chan error, 1)
	go func() { done <- p.Run(context.Background(), in, out) }()

	close(in)
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v on input close, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after input channel closed")
	}
}

func TestProducerStopsOnContextCancel(t *testing.T) {
	p, err := New(Options{FFTSize: 64, FrameRate: 100})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	in := make(chan []complex64)
	out := make(chan Frame, 1)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, in, out) }()

	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Run err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after ctx cancel")
	}
}

func TestProducerSetCenterAndSampleRate(t *testing.T) {
	p, err := New(Options{FFTSize: 64, FrameRate: 100, CenterFreqHz: 100, SampleRateHz: 200})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetCenter(400_000_000)
	p.SetSampleRate(2_500_000)

	in := make(chan []complex64, 1)
	out := make(chan Frame, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = p.Run(ctx, in, out) }()

	in <- make([]complex64, 64)
	select {
	case f := <-out:
		if f.CenterHz != 400_000_000 {
			t.Errorf("CenterHz = %d, want 400_000_000", f.CenterHz)
		}
		if f.SampleRate != 2_500_000 {
			t.Errorf("SampleRate = %d, want 2_500_000", f.SampleRate)
		}
	case <-time.After(time.Second):
		t.Fatal("no frame")
	}
}

func TestProducerCounteractsBinTones(t *testing.T) {
	// Inject a complex exponential at +sampleRate/4. After FFT-shift,
	// the peak should land at bin 48 of 64 (= centre + N/4).
	const N = 64
	p, err := New(Options{FFTSize: N, FrameRate: 100, SampleRateHz: 2_000_000})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	in := make(chan []complex64, 1)
	out := make(chan Frame, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = p.Run(ctx, in, out) }()

	chunk := make([]complex64, N)
	for i := range chunk {
		theta := 2 * math.Pi * 0.25 * float64(i) // +Fs/4
		chunk[i] = complex(float32(math.Cos(theta)), float32(math.Sin(theta)))
	}
	in <- chunk

	f := <-out
	var peakBin int
	var peakVal float32 = -math.MaxFloat32
	for i, v := range f.Bins {
		if v > peakVal {
			peakVal = v
			peakBin = i
		}
	}
	if peakBin != 48 {
		t.Errorf("+Fs/4 peak at bin %d, want 48 (centre + N/4)", peakBin)
	}
}

// TestPowerDBPeaksAtTone checks the shared helper: a unit-amplitude
// complex tone reads ~0 dBFS at its FFT-shifted bin and well below
// elsewhere, with DC landing at the centre bin.
func TestPowerDBPeaksAtTone(t *testing.T) {
	const (
		size     = 256
		sampleHz = 48000.0
		toneHz   = 6000.0
	)
	plan := fft.New(size)
	win := window.Hann(size)
	var winSum float64
	for _, v := range win {
		winSum += v * v
	}

	in := make([]complex64, size)
	w := 2 * math.Pi * toneHz / sampleHz
	for n := 0; n < size; n++ {
		in[n] = complex64(complex(math.Cos(w*float64(n)), math.Sin(w*float64(n))))
	}

	bufIn := make([]complex128, size)
	bufOut := make([]complex128, size)
	dst := make([]float32, size)
	PowerDB(plan, win, winSum, in, bufIn, bufOut, dst)

	center := size / 2
	wantBin := center + int(math.Round(toneHz*size/sampleHz))
	peak, peakVal := 0, float32(math.Inf(-1))
	for i, v := range dst {
		if v > peakVal {
			peak, peakVal = i, v
		}
	}
	if peak != wantBin {
		t.Errorf("peak bin = %d, want %d", peak, wantBin)
	}
	// A windowed unit tone should peak near 0 dBFS.
	if peakVal < -3 || peakVal > 1 {
		t.Errorf("peak level = %.2f dBFS, want ~0", peakVal)
	}
	// An off-peak bin should be far down.
	if dst[center] > -20 {
		t.Errorf("centre (DC) bin = %.2f dBFS, want well below the +6 kHz peak", dst[center])
	}
}
