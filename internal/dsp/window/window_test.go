package window

import (
	"math"
	"testing"
)

func TestSymmetry(t *testing.T) {
	for _, tc := range []struct {
		name string
		w    []float64
	}{
		{"hann", Hann(33)},
		{"hamming", Hamming(33)},
		{"blackman", Blackman(33)},
		{"kaiser", Kaiser(33, 8.6)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := len(tc.w)
			for i := 0; i < n/2; i++ {
				if math.Abs(tc.w[i]-tc.w[n-1-i]) > 1e-12 {
					t.Errorf("asymmetric at %d: %f vs %f", i, tc.w[i], tc.w[n-1-i])
				}
			}
		})
	}
}

func TestKaiserPeakAtCenter(t *testing.T) {
	w := Kaiser(65, 8.6)
	mid := w[len(w)/2]
	if mid < 0.999 {
		t.Errorf("Kaiser midpoint = %f, want ~1", mid)
	}
	if w[0] > 0.01 || w[len(w)-1] > 0.01 {
		t.Errorf("Kaiser endpoints not near zero: %f %f", w[0], w[len(w)-1])
	}
}
