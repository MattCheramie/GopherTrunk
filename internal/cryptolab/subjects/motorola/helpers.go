package motorola

import (
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/fit"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"
)

// componentObs is one recovered per-context state: the readable context
// (HPrev,HCur) and the odd-position high byte / multiplier fitted for it.
type componentObs struct {
	HPrev, HCur uint8
	Hodd, Modd  uint8
}

// recoverComponentObs groups character observations by context and, within
// each context, fits the unique odd multiplier Modd and the odd high byte
// Hodd from char = Modd·(LUT[eo] − Hodd). Only contexts that fit cleanly
// with at least two distinct LUT[eo] inputs are returned (otherwise the
// fit is underdetermined).
func recoverComponentObs(t *Table, rows []Row) []componentObs {
	type grp struct {
		obs   []fit.Obs1
		seenX map[uint8]bool
	}
	groups := map[Context]*grp{}
	for _, row := range rows {
		for _, o := range t.CharObservations(row) {
			g := groups[o.Ctx]
			if g == nil {
				g = &grp{seenX: map[uint8]bool{}}
				groups[o.Ctx] = g
			}
			x := t.Fwd[o.Eo]
			g.obs = append(g.obs, fit.Obs1{X: x, Y: o.Char})
			g.seenX[x] = true
		}
	}
	var out []componentObs
	for ctx, g := range groups {
		if len(g.seenX) < 2 {
			continue // underdetermined
		}
		f := fit.FitAffine1(g.obs, true) // require odd slope (Modd)
		if f.Conflicts != 0 || f.A&1 == 0 {
			continue
		}
		// char = A·LUT[eo] + B with A=Modd, B = −Modd·Hodd ⇒ Hodd = −B·Modd⁻¹.
		hodd := (0 - f.B) * gauge.ModInv(f.A)
		out = append(out, componentObs{HPrev: ctx.HPrev, HCur: ctx.HCur, Hodd: hodd, Modd: f.A})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].HPrev != out[j].HPrev {
			return out[i].HPrev < out[j].HPrev
		}
		return out[i].HCur < out[j].HCur
	})
	return out
}

// gaugeMergedIndexConflicts transforms the recovered odd high byte and its
// context into true coordinates under g, then tests whether Hodd_true is a
// function of the XOR merged index (HPrev_true ^ HCur_true). Returns the
// number of conflicting observations. (An affine relation is gauge-
// invariant, so only a nonlinear merged index like XOR is gauge-sensitive —
// which is the whole point of sweeping gauges.)
func gaugeMergedIndexConflicts(g gauge.Gauge, obs []componentObs) int {
	tr := func(b uint8) uint8 { return g.Mg*b + g.Cg }
	table := make(map[uint8]uint8, 256)
	conflicts := 0
	for _, o := range obs {
		idx := tr(o.HPrev) ^ tr(o.HCur)
		out := tr(o.Hodd)
		if prev, ok := table[idx]; ok {
			if prev != out {
				conflicts++
			}
		} else {
			table[idx] = out
		}
	}
	return conflicts
}

// candidateSet returns the packed candidate states consistent with one
// character observation.
func candidateSet(t *Table, o CharObs) []uint16 {
	states := t.CandidatesForChar(o.Char, o.Eo)
	out := make([]uint16, len(states))
	for i, s := range states {
		out[i] = packState(s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// intersect returns the sorted intersection of two sorted candidate lists.
func intersect(a, b []uint16) []uint16 {
	var out []uint16
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			out = append(out, a[i])
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return out
}

// decodeCoverage reports how many characters are decodable (their context
// is pinned to a single state) and how many whole messages decode fully.
func decodeCoverage(t *Table, rows []Row, ck *cellCheckpoint) (charsDecodable, charsTotal, msgsFull int) {
	pinned := map[string]bool{}
	for k, c := range ck.Candidates {
		if len(c) == 1 {
			pinned[k] = true
		}
	}
	for _, row := range rows {
		obs := t.CharObservations(row)
		if len(obs) == 0 {
			continue
		}
		full := true
		for _, o := range obs {
			charsTotal++
			if pinned[ctxKey(o.Ctx)] {
				charsDecodable++
			} else {
				full = false
			}
		}
		if full {
			msgsFull++
		}
	}
	return charsDecodable, charsTotal, msgsFull
}

// family is one candidate accumulator update for the from-seed brute.
type family struct {
	Name   string
	Seed   func(n int) uint16
	Update func(s uint16, eo uint8) uint16
}

// defaultFamilies are example update families. They are expected to be
// rejected (the LCG class is ruled out); they exist to exercise the
// auto-gauge simulation framework, which users extend with new candidates.
func defaultFamilies() []family {
	return []family{
		{
			Name:   "lcg-293-0x72E9", // the placeholder constants from internal/radio/p25 alias.go
			Seed:   func(n int) uint16 { return uint16(n) },
			Update: func(s uint16, eo uint8) uint16 { return s*293 + 0x72E9 + uint16(eo) + 1 },
		},
		{
			Name:   "lcg-25173-13849",
			Seed:   func(n int) uint16 { return uint16(n) },
			Update: func(s uint16, eo uint8) uint16 { return s*25173 + 13849 + uint16(eo) },
		},
	}
}

// simulateFamily runs a candidate update from the length seed across the
// corpus, solves the affine output gauge once over the collected
// (predicted, observed) high-byte pairs, and returns the high-byte mismatch
// count and sample total.
func simulateFamily(t *Table, rows []Row, fam family) (mismatches, total int) {
	type pair struct{ pred, obs uint8 }
	var pairs []pair
	for _, row := range rows {
		enc := row.Enc
		n := len(enc) / 2
		if n == 0 {
			continue
		}
		s := fam.Seed(n)
		for i := 0; i < len(enc); i++ {
			if i%2 == 0 { // even position exposes the high byte
				pairs = append(pairs, pair{pred: uint8(s >> 8), obs: t.Fwd[enc[i]]})
			}
			s = fam.Update(s, enc[i])
		}
	}
	if len(pairs) < 2 {
		return 0, 0
	}
	// Auto-solve the affine gauge alpha·pred + beta = obs from a pair with
	// an odd predicted-difference (invertible mod 256).
	alpha, beta := uint8(1), uint8(0)
	for i := 1; i < len(pairs); i++ {
		d := pairs[0].pred - pairs[i].pred
		if d&1 == 1 {
			alpha = (pairs[0].obs - pairs[i].obs) * gauge.ModInv(d)
			beta = pairs[0].obs - alpha*pairs[0].pred
			break
		}
	}
	for _, p := range pairs {
		total++
		if alpha*p.pred+beta != p.obs {
			mismatches++
		}
	}
	return mismatches, total
}
