package dsp

import (
	"math"
	"math/cmplx"
	"testing"
)

// A tone at +offsetHz, mixed by an NCO tuned to the same offset, must land
// at DC: a constant phasor with the input's magnitude and ~zero residual
// frequency.
func TestNCOMixesToneToDC(t *testing.T) {
	const (
		fs     = 48000.0
		offset = 750.0
		amp    = 0.7
		n      = 200000 // long enough to exercise several renorm cycles
	)
	src := make([]complex64, n)
	for i := range src {
		ph := 2 * math.Pi * offset * float64(i) / fs
		src[i] = complex64(complex(amp*math.Cos(ph), amp*math.Sin(ph)))
	}
	out := NewNCO(offset, fs).Mix(nil, src)

	// Reference phase from the first output sample; every later sample
	// should share it (DC), and magnitude should be preserved.
	ref := cmplx.Phase(complex128(out[0]))
	var maxPhaseErr, maxMagErr float64
	for _, v := range out {
		c := complex128(v)
		d := math.Abs(cmplx.Phase(c) - ref)
		if d > math.Pi {
			d = 2*math.Pi - d
		}
		if d > maxPhaseErr {
			maxPhaseErr = d
		}
		if e := math.Abs(cmplx.Abs(c) - amp); e > maxMagErr {
			maxMagErr = e
		}
	}
	if maxPhaseErr > 1e-3 {
		t.Errorf("residual phase drift after mix = %.2e rad, want ~0 (tone not centred)", maxPhaseErr)
	}
	if maxMagErr > 1e-3 {
		t.Errorf("magnitude error after mix = %.2e, want ~0 (renorm/gain bug)", maxMagErr)
	}
}

// A zero offset is an identity mix (existing centred-capture behaviour).
func TestNCOZeroOffsetIdentity(t *testing.T) {
	src := []complex64{complex(1, 2), complex(-3, 0.5), complex(0, -1)}
	out := NewNCO(0, 48000).Mix(nil, append([]complex64(nil), src...))
	for i := range src {
		if d := complex128(out[i] - src[i]); cmplx.Abs(d) > 1e-6 {
			t.Errorf("sample %d: zero-offset mix changed %v -> %v", i, src[i], out[i])
		}
	}
}
