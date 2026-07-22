package siglab

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestDiffPhaseSeriesRotation(t *testing.T) {
	// A constellation rotating by a fixed π/4 step per symbol — the canonical
	// π/4-DQPSK rotation — must yield a constant differential phase of π/4.
	const step = math.Pi / 4
	n := 200
	pts := make([]complex64, n)
	for i := range pts {
		pts[i] = complex64(cmplx.Exp(complex(0, step*float64(i))))
	}
	got := diffPhaseSeries(pts, 1000)
	if len(got) != n-1 {
		t.Fatalf("len = %d, want %d", len(got), n-1)
	}
	for i, v := range got {
		if math.Abs(float64(v)-step) > 1e-5 {
			t.Errorf("diff[%d] = %v, want %v", i, v, step)
		}
	}
}

func TestDiffPhaseSeriesDecimationCap(t *testing.T) {
	pts := make([]complex64, 5000)
	for i := range pts {
		pts[i] = complex64(cmplx.Exp(complex(0, 0.3*float64(i))))
	}
	const cap = 500
	got := diffPhaseSeries(pts, cap)
	if len(got) == 0 {
		t.Fatal("expected a decimated diff-phase series")
	}
	if len(got) > cap {
		t.Errorf("diff-phase series exceeds cap: %d > %d", len(got), cap)
	}
}

func TestDiffPhaseSeriesTooShort(t *testing.T) {
	if got := diffPhaseSeries([]complex64{complex(1, 0)}, 100); got != nil {
		t.Errorf("diffPhaseSeries of one point = %v, want nil", got)
	}
	if got := diffPhaseSeries(nil, 100); got != nil {
		t.Errorf("diffPhaseSeries(nil) = %v, want nil", got)
	}
}

func TestSymbolIQSeriesPreservesPoints(t *testing.T) {
	// Below the cap, every complex point is passed through unchanged as an
	// IQPoint — this is the 2-D symbol-decision constellation the viz plots.
	pts := []complex64{complex(1, 0), complex(0, 1), complex(-1, 0), complex(0, -1)}
	got := symbolIQSeries(pts, 1000)
	if len(got) != len(pts) {
		t.Fatalf("len = %d, want %d", len(got), len(pts))
	}
	for i, p := range pts {
		if got[i].I != real(p) || got[i].Q != imag(p) {
			t.Errorf("point[%d] = (%v,%v), want (%v,%v)", i, got[i].I, got[i].Q, real(p), imag(p))
		}
	}
}

func TestSymbolIQSeriesDecimationCap(t *testing.T) {
	pts := make([]complex64, 5000)
	for i := range pts {
		pts[i] = complex64(cmplx.Exp(complex(0, 0.3*float64(i))))
	}
	const cap = 500
	got := symbolIQSeries(pts, cap)
	if len(got) == 0 {
		t.Fatal("expected a decimated symbol-IQ series")
	}
	if len(got) > cap {
		t.Errorf("symbol-IQ series exceeds cap: %d > %d", len(got), cap)
	}
}

func TestSymbolIQSeriesEmpty(t *testing.T) {
	if got := symbolIQSeries(nil, 100); got != nil {
		t.Errorf("symbolIQSeries(nil) = %v, want nil", got)
	}
	if got := symbolIQSeries([]complex64{complex(1, 0)}, 0); got != nil {
		t.Errorf("symbolIQSeries(maxPoints=0) = %v, want nil", got)
	}
}
