package siglab

import (
	"math"
	"math/cmplx"
	"testing"
)

// TestDownconvertShiftsToneToDC feeds a pure tone offset +100 kHz from centre
// through Downconvert tuned to that offset and confirms two things: the output
// rate collapses to the requested ~50 kHz channel rate (≈48× fewer samples),
// and the tone lands at DC — i.e. the decimated stream is a near-constant
// phasor, so |mean| ≈ mean|·| (an off-DC tone would average toward zero).
func TestDownconvertShiftsToneToDC(t *testing.T) {
	const (
		rate   = 2_400_000.0
		offset = 100_000.0
		target = 50_000.0
		nsamp  = 240_000 // 0.1 s
	)
	in := make([]complex64, nsamp)
	for n := range in {
		ph := 2 * math.Pi * offset * float64(n) / rate
		in[n] = complex64(cmplx.Exp(complex(0, ph)))
	}

	out, outRate := Downconvert(in, rate, offset, target)

	if math.Abs(outRate-target) > 1 {
		t.Fatalf("output rate = %g, want ~%g", outRate, target)
	}
	if ratio := float64(len(in)) / float64(len(out)); ratio < 40 || ratio > 56 {
		t.Fatalf("decimation ratio = %.1f, want ~48", ratio)
	}

	// Skip the filter warm-up transient, then measure the DC-ness of the tail.
	tail := out[len(out)/2:]
	var sum complex128
	var mag float64
	for _, s := range tail {
		sum += complex128(s)
		mag += cmplx.Abs(complex128(s))
	}
	meanVec := cmplx.Abs(sum) / float64(len(tail))
	meanMag := mag / float64(len(tail))
	if meanMag == 0 || meanVec/meanMag < 0.95 {
		t.Fatalf("tone not at DC: |mean|/mean|·| = %.3f (want ≈1)", meanVec/meanMag)
	}
}
