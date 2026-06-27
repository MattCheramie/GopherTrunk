// Package fit holds closed-form byte-function fitters over Z/256: given
// observed (input → output) byte pairs, find the affine coefficients that
// best explain them and report how many observations conflict. A
// zero-conflict fit means the data follows that closed form; a nonzero
// floor is the toolkit's evidence that no such low-degree form exists (the
// signature of a hidden table).
package fit

// Obs1 is one observation of a one-input byte function: Y = f(X).
type Obs1 struct{ X, Y uint8 }

// Affine1 is the best affine model Y = A·X + B (mod 256) for the data,
// together with the count of observations it fails to reproduce.
type Affine1 struct {
	A, B      uint8
	Conflicts int
}

// FitAffine1 returns the affine model minimizing conflicts. It is exact and
// fast: for each of the 256 candidate slopes A, the intercept is forced by
// the first observation and the rest are checked. RequireOddA restricts the
// slope to invertible (odd) multipliers — the regime the alias keystream
// fit lives in.
func FitAffine1(obs []Obs1, requireOddA bool) Affine1 {
	best := Affine1{Conflicts: len(obs) + 1}
	if len(obs) == 0 {
		return Affine1{}
	}
	for a := 0; a < 256; a++ {
		if requireOddA && a&1 == 0 {
			continue
		}
		au := uint8(a)
		b := obs[0].Y - au*obs[0].X
		conflicts := 0
		for _, o := range obs {
			if au*o.X+b != o.Y {
				conflicts++
			}
		}
		if conflicts < best.Conflicts {
			best = Affine1{A: au, B: b, Conflicts: conflicts}
			if conflicts == 0 {
				break
			}
		}
	}
	return best
}

// Obs2 is one observation of a two-input byte function: Z = f(X, Y).
type Obs2 struct{ X, Y, Z uint8 }

// Affine2 is the best affine model Z = A·X + B·Y + C (mod 256).
type Affine2 struct {
	A, B, C   uint8
	Conflicts int
}

// FitAffine2 fits Z = A·X + B·Y + C by brute-forcing the slope A and
// reducing the remainder to a one-input affine fit in Y. Cost is
// O(256·256·N); intended for bounded observation sets (callers subsample
// for the full 32,768-gauge sweep and log the cap).
func FitAffine2(obs []Obs2) Affine2 {
	best := Affine2{Conflicts: len(obs) + 1}
	if len(obs) == 0 {
		return Affine2{}
	}
	reduced := make([]Obs1, len(obs))
	for a := 0; a < 256; a++ {
		au := uint8(a)
		for i, o := range obs {
			reduced[i] = Obs1{X: o.Y, Y: o.Z - au*o.X}
		}
		r := FitAffine1(reduced, false)
		if r.Conflicts < best.Conflicts {
			best = Affine2{A: au, B: r.A, C: r.B, Conflicts: r.Conflicts}
			if r.Conflicts == 0 {
				break
			}
		}
	}
	return best
}
