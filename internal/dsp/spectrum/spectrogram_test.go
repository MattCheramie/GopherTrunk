package spectrum

import (
	"math"
	"math/cmplx"
	"testing"
)

// A steady tone should land in the same FFT-shifted bin in every frame, near
// 0 dBFS, matching PowerDB/AverageDB's layout and normalization.
func TestSpectrogramSteadyTone(t *testing.T) {
	const n = 256
	const hop = 128
	const k = 6 // bins above DC
	iq := make([]complex64, n*8)
	for i := range iq {
		ph := 2 * math.Pi * float64(k) * float64(i) / float64(n)
		iq[i] = complex64(cmplx.Exp(complex(0, ph)))
	}

	rows := Spectrogram(iq, n, hop)
	wantFrames := (len(iq)-n)/hop + 1
	if len(rows) != wantFrames {
		t.Fatalf("frames = %d, want %d", len(rows), wantFrames)
	}
	peakBin := n/2 + k // FFT-shifted: DC at n/2
	for f, row := range rows {
		if len(row) != n {
			t.Fatalf("frame %d width = %d, want %d", f, len(row), n)
		}
		if row[peakBin] < -3 || row[peakBin] > 1 {
			t.Errorf("frame %d tone bin = %.2f dBFS, want ~0", f, row[peakBin])
		}
		if row[n/2-40] > -40 {
			t.Errorf("frame %d off-tone bin = %.2f dBFS, want < -40", f, row[n/2-40])
		}
	}
}

// Fewer than one window yields no frames; hop<=0 defaults to 50% overlap.
func TestSpectrogramEdgeCases(t *testing.T) {
	if rows := Spectrogram(make([]complex64, 10), 64, 32); rows != nil {
		t.Errorf("short input: got %d frames, want nil", len(rows))
	}
	iq := make([]complex64, 512)
	for i := range iq {
		iq[i] = complex(1, 0)
	}
	def := Spectrogram(iq, 128, 0)  // hop defaults to 64
	exp := Spectrogram(iq, 128, 64) // explicit
	if len(def) != len(exp) {
		t.Errorf("default hop frames = %d, want %d (50%% overlap)", len(def), len(exp))
	}
}
