package rtlsdr

import (
	"math"
	"testing"
)

func TestDCBlockerRemovesBias(t *testing.T) {
	const n = 4096
	in := make([]complex64, n)
	for i := range in {
		in[i] = complex(0.5, -0.3) // pure DC
	}
	d := NewDCBlocker(0.05)
	for k := 0; k < 20; k++ {
		buf := append([]complex64(nil), in...)
		d.Process(buf)
		if k == 19 {
			// After convergence, output should be near zero.
			var maxAbs float32
			for _, s := range buf {
				a := absComplex(s)
				if a > maxAbs {
					maxAbs = a
				}
			}
			if maxAbs > 0.05 {
				t.Errorf("residual after convergence = %f, want < 0.05", maxAbs)
			}
		}
	}
}

func TestPPMToHz(t *testing.T) {
	got := PPMToHz(50, 851_000_000)
	if got != 42_550 {
		t.Errorf("PPMToHz = %d, want 42550", got)
	}
}

func absComplex(c complex64) float32 {
	return float32(math.Hypot(float64(real(c)), float64(imag(c))))
}
