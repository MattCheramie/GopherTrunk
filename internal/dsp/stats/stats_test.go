package stats

import (
	"math"
	"reflect"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestMeanVarianceStdDev(t *testing.T) {
	x := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	if got := Mean(x); !almost(got, 5) {
		t.Errorf("Mean = %v, want 5", got)
	}
	// Population variance of this classic set is 4, sample variance 32/7.
	if got := Variance(x, false); !almost(got, 4) {
		t.Errorf("population Variance = %v, want 4", got)
	}
	if got := Variance(x, true); !almost(got, 32.0/7.0) {
		t.Errorf("sample Variance = %v, want %v", got, 32.0/7.0)
	}
	if got := StdDev(x, false); !almost(got, 2) {
		t.Errorf("population StdDev = %v, want 2", got)
	}
}

func TestStatsEmptyAndSingleton(t *testing.T) {
	if got := Mean(nil); got != 0 {
		t.Errorf("Mean(nil) = %v, want 0", got)
	}
	if got := Variance([]float64{3}, true); got != 0 {
		t.Errorf("sample Variance of one point = %v, want 0", got)
	}
	if got := Variance([]float64{3}, false); got != 0 {
		t.Errorf("population Variance of one point = %v, want 0", got)
	}
}

func TestMinMaxArg(t *testing.T) {
	x := []float64{3, -1, 7, 7, 2}
	if v, i := Min(x); v != -1 || i != 1 {
		t.Errorf("Min = (%v,%v), want (-1,1)", v, i)
	}
	if v, i := Max(x); v != 7 || i != 2 {
		t.Errorf("Max = (%v,%v), want (7,2) (first of a tie)", v, i)
	}
	if got := ArgMax(x); got != 2 {
		t.Errorf("ArgMax = %v, want 2", got)
	}
	if got := ArgMin(x); got != 1 {
		t.Errorf("ArgMin = %v, want 1", got)
	}
	if got := ArgMax(nil); got != -1 {
		t.Errorf("ArgMax(nil) = %v, want -1", got)
	}
}

func TestArgMaxInt(t *testing.T) {
	if got := ArgMaxInt([]int{1, 5, 5, 2}); got != 1 {
		t.Errorf("ArgMaxInt = %v, want 1 (first of a tie)", got)
	}
	if got := ArgMaxInt(nil); got != -1 {
		t.Errorf("ArgMaxInt(nil) = %v, want -1", got)
	}
}

func TestHistogram(t *testing.T) {
	x := []float64{0, 1, 2, 3, 4}
	counts, edges := Histogram(x, 5)
	wantEdges := []float64{0, 0.8, 1.6, 2.4, 3.2, 4}
	if len(edges) != 6 {
		t.Fatalf("edges len = %d, want 6", len(edges))
	}
	for i := range wantEdges {
		if !almost(edges[i], wantEdges[i]) {
			t.Errorf("edge %d = %v, want %v", i, edges[i], wantEdges[i])
		}
	}
	// Each integer lands in its own bin; total count is preserved and the
	// right-most value folds into the last bin.
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != len(x) {
		t.Errorf("count total = %d, want %d", total, len(x))
	}
	if counts[len(counts)-1] < 1 {
		t.Errorf("last bin should hold the right-edge value, counts=%v", counts)
	}
	if c, _ := Histogram(nil, 4); c != nil {
		t.Errorf("Histogram(nil) counts = %v, want nil", c)
	}
}

func TestModeInt(t *testing.T) {
	if got := ModeInt([]int{4, 2, 4, 2, 4}); got != 4 {
		t.Errorf("ModeInt = %v, want 4", got)
	}
	// Tie between 2 and 5 (one each besides) resolves toward the smaller value.
	if got := ModeInt([]int{5, 2, 5, 2}); got != 2 {
		t.Errorf("ModeInt tie = %v, want 2 (smaller)", got)
	}
	if got := ModeInt(nil); got != 0 {
		t.Errorf("ModeInt(nil) = %v, want 0", got)
	}
}

// TestModeIntMatchesLegacy pins ModeInt to the behaviour of the original
// inline modal-spacing scan it replaced, including the tie rule.
func TestModeIntMatchesLegacy(t *testing.T) {
	cases := [][]int{
		{18, 18, 18, 9, 18},
		{1, 2, 3, 4},
		{7, 7, 3, 3, 1},
		{5},
	}
	for _, c := range cases {
		if got, want := ModeInt(c), legacyMode(c); got != want {
			t.Errorf("ModeInt(%v) = %d, legacy = %d", c, got, want)
		}
	}
}

// legacyMode is the pre-refactor inline implementation, kept here as the oracle.
func legacyMode(deltas []int) int {
	if len(deltas) == 0 {
		return 0
	}
	s := append([]int(nil), deltas...)
	for i := 1; i < len(s); i++ { // insertion sort, mirrors sort.Ints result
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
	bestVal, bestCount := s[0], 1
	curVal, curCount := s[0], 1
	for i := 1; i < len(s); i++ {
		if s[i] == curVal {
			curCount++
		} else {
			if curCount > bestCount {
				bestVal, bestCount = curVal, curCount
			}
			curVal, curCount = s[i], 1
		}
	}
	if curCount > bestCount {
		bestVal = curVal
	}
	return bestVal
}

func TestHistogramDegenerateRange(t *testing.T) {
	// All-equal input must not divide by a zero-width range.
	counts, edges := Histogram([]float64{3, 3, 3}, 2)
	if !reflect.DeepEqual(counts, []int{0, 3}) && !reflect.DeepEqual(counts, []int{3, 0}) {
		t.Logf("counts=%v edges=%v", counts, edges)
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 3 {
		t.Errorf("degenerate histogram lost counts: %v", counts)
	}
}
