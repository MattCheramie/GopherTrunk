package fit

import "testing"

func TestFitAffine1Exact(t *testing.T) {
	t.Parallel()
	// Y = 7*X + 13 (7 is odd/invertible).
	var obs []Obs1
	for x := 0; x < 256; x++ {
		obs = append(obs, Obs1{X: uint8(x), Y: uint8(7*x + 13)})
	}
	got := FitAffine1(obs, true)
	if got.Conflicts != 0 || got.A != 7 || got.B != 13 {
		t.Fatalf("FitAffine1 = %+v, want A=7 B=13 conflicts=0", got)
	}
}

func TestFitAffine1NonAffineHasConflicts(t *testing.T) {
	t.Parallel()
	// Y = X*X is not affine; the best affine model must leave conflicts.
	var obs []Obs1
	for x := 0; x < 256; x++ {
		obs = append(obs, Obs1{X: uint8(x), Y: uint8(x * x)})
	}
	if got := FitAffine1(obs, false); got.Conflicts == 0 {
		t.Fatalf("quadratic data fit with 0 conflicts: %+v", got)
	}
}

func TestFitAffine2Exact(t *testing.T) {
	t.Parallel()
	// Z = 5*X + 9*Y + 200, with X and Y varied independently so the model
	// is determined (collinear X,Y would admit other affine solutions).
	var obs []Obs2
	for xi := 0; xi < 16; xi++ {
		for yi := 0; yi < 16; yi++ {
			x := uint8(xi * 17)
			y := uint8(yi * 13)
			obs = append(obs, Obs2{X: x, Y: y, Z: 5*x + 9*y + 200})
		}
	}
	got := FitAffine2(obs)
	if got.Conflicts != 0 {
		t.Fatalf("FitAffine2 conflicts = %d, want 0 (%+v)", got.Conflicts, got)
	}
	// Verify the recovered model reproduces a fresh independent point.
	if got.A*3+got.B*4+got.C != 5*3+9*4+200 {
		t.Fatalf("recovered model wrong: %+v", got)
	}
}
