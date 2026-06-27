// Package gauge models the global affine gauge ambiguity that shows up when
// a byte-oriented obfuscator's state is recovered from output only: the
// recovered high byte and multiplier are pinned just up to an affine change
// of coordinates (Mg·x + Cg with Mg odd, hence invertible mod 256). The
// gauge group is small — 128 odd multipliers × 256 offsets = 32,768
// elements — so it can be brute-forced in full, which is what the alias
// "gauge" recovery mode does to look for a coordinate frame in which a
// clean closed form appears.
package gauge

// State is the per-character FSM state of the kind the alias subject
// exposes: an observable high byte and an odd multiplier byte.
type State struct {
	Hi  uint8 // high byte (observable at even positions)
	Mul uint8 // odd multiplier
}

// Gauge is the affine change of coordinates (Mg odd, Cg). It acts on the
// high byte as Hi → Mg·Hi + Cg and on the multiplier as Mul → Mg·Mul, both
// mod 256.
type Gauge struct {
	Mg uint8 // must be odd (invertible mod 256)
	Cg uint8
}

// ModInv returns the multiplicative inverse of an odd byte mod 256. It
// panics on an even input, which has no inverse.
func ModInv(a uint8) uint8 {
	if a&1 == 0 {
		panic("gauge: even value has no inverse mod 256")
	}
	// Newton iteration converges in 3 steps for mod 2^8.
	x := a
	for i := 0; i < 3; i++ {
		x = x * (2 - a*x)
	}
	return x
}

// ToTrue maps a gauge-coordinate state into true coordinates.
func (g Gauge) ToTrue(s State) State {
	return State{
		Hi:  g.Mg*s.Hi + g.Cg,
		Mul: g.Mg * s.Mul,
	}
}

// FromTrue maps a true-coordinate state back into gauge coordinates. It is
// the exact inverse of ToTrue.
func (g Gauge) FromTrue(s State) State {
	inv := ModInv(g.Mg)
	return State{
		Hi:  inv * (s.Hi - g.Cg),
		Mul: inv * s.Mul,
	}
}

// All returns every gauge: 128 odd multipliers × 256 offsets = 32,768.
func All() []Gauge {
	out := make([]Gauge, 0, 128*256)
	for mg := 1; mg < 256; mg += 2 {
		for cg := 0; cg < 256; cg++ {
			out = append(out, Gauge{Mg: uint8(mg), Cg: uint8(cg)})
		}
	}
	return out
}
