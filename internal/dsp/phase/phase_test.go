package phase

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestUnwrapRamp(t *testing.T) {
	// A phase ramp of +0.6π per step wraps in the raw (atan2-style) sequence
	// but must come out monotonic and continuous after unwrapping.
	stepTrue := 0.6 * math.Pi
	n := 12
	wrapped := make([]float64, n)
	for i := range wrapped {
		// Wrap the true phase into (-π, π].
		wrapped[i] = math.Atan2(math.Sin(stepTrue*float64(i)), math.Cos(stepTrue*float64(i)))
	}
	got := Unwrap(wrapped)
	for i := 1; i < n; i++ {
		d := got[i] - got[i-1]
		if math.Abs(d-stepTrue) > 1e-9 {
			t.Errorf("step %d = %v, want %v (continuous ramp)", i, d, stepTrue)
		}
	}
}

func TestUnwrapShortInputs(t *testing.T) {
	if got := Unwrap(nil); len(got) != 0 {
		t.Errorf("Unwrap(nil) len = %d, want 0", len(got))
	}
	if got := Unwrap([]float64{1.5}); len(got) != 1 || got[0] != 1.5 {
		t.Errorf("Unwrap singleton = %v, want [1.5]", got)
	}
}

func TestUnwrapDoesNotMutateInput(t *testing.T) {
	in := []float64{3, -3, 3}
	_ = Unwrap(in)
	if in[1] != -3 {
		t.Errorf("Unwrap mutated its input: %v", in)
	}
}

func TestDiffConstantRotation(t *testing.T) {
	// A phasor rotating by a fixed step has a constant differential phase.
	step := math.Pi / 4
	n := 8
	z := make([]complex128, n)
	for i := range z {
		z[i] = cmplx.Exp(complex(0, step*float64(i)))
	}
	d := Diff(z)
	if len(d) != n-1 {
		t.Fatalf("Diff len = %d, want %d", len(d), n-1)
	}
	for i, v := range d {
		if math.Abs(v-step) > 1e-9 {
			t.Errorf("Diff[%d] = %v, want %v", i, v, step)
		}
	}
}

func TestFromComplex(t *testing.T) {
	z := []complex128{complex(1, 0), complex(0, 1), complex(-1, 0)}
	got := FromComplex(z)
	want := []float64{0, math.Pi / 2, math.Pi}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-12 {
			t.Errorf("FromComplex[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestDiffShort(t *testing.T) {
	if got := Diff([]complex128{complex(1, 0)}); got != nil {
		t.Errorf("Diff of one sample = %v, want nil", got)
	}
}
