package spectrum

import (
	"math"
	"sort"
	"testing"
)

func TestSmoothCopiesWhenWidthNonPositive(t *testing.T) {
	in := []float32{1, 2, 3, 4}
	out := Smooth(in, 0)
	if &out[0] == &in[0] {
		t.Fatal("Smooth(w<=0) must return a copy, not the same backing array")
	}
	for i := range in {
		if out[i] != in[i] {
			t.Fatalf("bin %d: got %v want %v", i, out[i], in[i])
		}
	}
}

func TestSmoothBoxAverage(t *testing.T) {
	// Half-width 1 ⇒ centered 3-wide mean, clipped at the edges.
	in := []float32{0, 3, 6, 9}
	out := Smooth(in, 1)
	want := []float32{
		(0 + 3) / 2.0,     // edge: bins 0,1
		(0 + 3 + 6) / 3.0, // bins 0,1,2
		(3 + 6 + 9) / 3.0, // bins 1,2,3
		(6 + 9) / 2.0,     // edge: bins 2,3
	}
	for i := range want {
		if math.Abs(float64(out[i]-want[i])) > 1e-5 {
			t.Errorf("bin %d: got %v want %v", i, out[i], want[i])
		}
	}
}

func TestPercentileMatchesSortedReference(t *testing.T) {
	bins := []float32{9, 1, 7, 3, 5, 2, 8, 4, 6, 0}
	for _, p := range []float64{0, 0.25, 0.5, 0.9, 1} {
		cp := append([]float32(nil), bins...)
		sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
		idx := int(p * float64(len(cp)-1))
		want := cp[idx]
		if got := Percentile(bins, p); got != want {
			t.Errorf("Percentile(p=%.2f) = %v, want %v", p, got, want)
		}
	}
}

func TestPercentileEmpty(t *testing.T) {
	if got := Percentile(nil, 0.5); got != 0 {
		t.Errorf("Percentile(nil) = %v, want 0", got)
	}
}

// A symmetric hump's power-weighted centroid is its geometric center.
func TestWeightedCentroidHzSymmetricHump(t *testing.T) {
	// Bins in dB: a triangular hump peaking at bin 5, floor elsewhere.
	bins := make([]float32, 11)
	for i := range bins {
		bins[i] = -60 // below floor
	}
	bins[3], bins[4], bins[5], bins[6], bins[7] = -20, -10, 0, -10, -20
	const baseHz, binHz = 100_000.0, 1_000.0
	got := WeightedCentroidHz(bins, 5, 4, -30, baseHz, binHz)
	want := baseHz + 5*binHz // symmetric ⇒ center bin
	if math.Abs(got-want) > 1.0 {
		t.Errorf("centroid = %.1f Hz, want %.1f", got, want)
	}
}

// When no bin clears the floor the centroid falls back to the peak bin's
// frequency.
func TestWeightedCentroidHzAllBelowFloor(t *testing.T) {
	bins := []float32{-90, -90, -90, -90, -90}
	const baseHz, binHz = 0.0, 500.0
	got := WeightedCentroidHz(bins, 2, 2, -30, baseHz, binHz)
	if want := baseHz + 2*binHz; got != want {
		t.Errorf("centroid = %.1f, want fallback %.1f", got, want)
	}
}
