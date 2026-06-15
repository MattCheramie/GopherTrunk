package spectrum

import (
	"math"
	"math/cmplx"
	"testing"
)

// A unit-amplitude complex tone should read ~0 dBFS in the bin it lands on and
// well below that elsewhere, matching PowerDB's normalization.
func TestAverageDBTonePeaksAtZeroDBFS(t *testing.T) {
	const n = 256
	const rate = 48_000.0
	// Place the tone four bins above DC so it sits on an exact FFT bin.
	const k = 4
	iq := make([]complex64, n*8)
	for i := range iq {
		ph := 2 * math.Pi * float64(k) * float64(i) / float64(n)
		iq[i] = complex64(cmplx.Exp(complex(0, ph)))
	}

	frame := AverageDB(iq, 100_000_000, rate, n)
	if len(frame.Bins) != n {
		t.Fatalf("Bins len = %d, want %d", len(frame.Bins), n)
	}
	if frame.SampleRate != uint32(rate) {
		t.Errorf("SampleRate = %d, want %d", frame.SampleRate, uint32(rate))
	}

	// FFT-shifted: DC is at n/2, so the tone lands at n/2 + k. With a Hann
	// window the main lobe spreads the tone across ~3 bins, so the peak bin
	// reads a couple dB below 0 dBFS — PowerDB documents "~0 dBFS at its bin".
	peakBin := n/2 + k
	if frame.Bins[peakBin] < -3 || frame.Bins[peakBin] > 1 {
		t.Errorf("tone bin = %.2f dBFS, want ~0", frame.Bins[peakBin])
	}
	// A bin far from the tone should be deep in the noise floor.
	if frame.Bins[n/2-40] > -40 {
		t.Errorf("off-tone bin = %.2f dBFS, want < -40", frame.Bins[n/2-40])
	}
}

// Averaging identical windows must equal a single window: the per-bin dB mean
// of N copies of the same frame is that frame.
func TestAverageDBStableAcrossIdenticalWindows(t *testing.T) {
	const n = 128
	one := make([]complex64, n)
	for i := range one {
		ph := 2 * math.Pi * 3 * float64(i) / float64(n)
		one[i] = complex64(cmplx.Exp(complex(0, ph)))
	}
	// Tile the same window eight times.
	many := make([]complex64, 0, n*8)
	for j := 0; j < 8; j++ {
		many = append(many, one...)
	}

	single := AverageDB(one, 0, 48_000, n)
	avg := AverageDB(many, 0, 48_000, n)
	for i := range single.Bins {
		if d := math.Abs(float64(single.Bins[i] - avg.Bins[i])); d > 1e-3 {
			t.Fatalf("bin %d: single=%.4f avg=%.4f differ by %.4g", i, single.Bins[i], avg.Bins[i], d)
		}
	}
}

// Fewer than one FFT window still yields a full-length frame via the padded
// single pass.
func TestAverageDBShortBufferPads(t *testing.T) {
	const n = 64
	short := make([]complex64, 10) // < n
	for i := range short {
		short[i] = complex(1, 0)
	}
	frame := AverageDB(short, 0, 48_000, n)
	if len(frame.Bins) != n {
		t.Fatalf("Bins len = %d, want %d (padded pass)", len(frame.Bins), n)
	}
}
