package stats

import (
	"reflect"
	"testing"
)

func TestFindPeaksBasic(t *testing.T) {
	// Interior maxima at indices 2 (value 5) and 5 (value 4).
	y := []float64{0, 1, 5, 1, 2, 4, 0}
	got := FindPeaks(y, PeakOpts{})
	want := []int{2, 5} // strongest first
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindPeaks = %v, want %v", got, want)
	}
}

func TestFindPeaksExcludesEndsByDefault(t *testing.T) {
	y := []float64{9, 1, 5, 1, 9}
	got := FindPeaks(y, PeakOpts{})
	if !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("FindPeaks (no ends) = %v, want [2]", got)
	}
	withEnds := FindPeaks(y, PeakOpts{IncludeEnds: true})
	// Both endpoints (value 9) outrank the interior peak (value 5).
	if len(withEnds) != 3 || withEnds[2] != 2 {
		t.Errorf("FindPeaks (ends) = %v, want the two endpoints then index 2", withEnds)
	}
}

func TestFindPeaksRelHeightFloor(t *testing.T) {
	y := []float64{0, 10, 0, 0.5, 0, 9, 0}
	// RelHeight 0.5 keeps only peaks ≥ 5; the 0.5-tall bump at index 3 drops.
	got := FindPeaks(y, PeakOpts{RelHeight: 0.5})
	if !reflect.DeepEqual(got, []int{1, 5}) {
		t.Errorf("FindPeaks RelHeight = %v, want [1 5]", got)
	}
}

func TestFindPeaksMinDistance(t *testing.T) {
	y := []float64{0, 8, 0, 9, 0, 1, 0}
	// Peaks at 1,3,5; min distance 3 keeps the strongest (3) then 5 is too
	// close to nothing kept yet... index 5 is distance 2 from 3, suppressed;
	// index 1 is distance 2 from 3, suppressed. Only the strongest survives.
	got := FindPeaks(y, PeakOpts{MinDistance: 3})
	if !reflect.DeepEqual(got, []int{3}) {
		t.Errorf("FindPeaks MinDistance = %v, want [3]", got)
	}
}

func TestFindPeaksFlatTopTakesLeft(t *testing.T) {
	// A flat top spanning indices 2..3: the ">=" right rule keeps the left one.
	y := []float64{0, 1, 5, 5, 1, 0}
	got := FindPeaks(y, PeakOpts{})
	if !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("FindPeaks flat top = %v, want [2]", got)
	}
}

func TestFindPeaksEmpty(t *testing.T) {
	if got := FindPeaks(nil, PeakOpts{}); got != nil {
		t.Errorf("FindPeaks(nil) = %v, want nil", got)
	}
}
