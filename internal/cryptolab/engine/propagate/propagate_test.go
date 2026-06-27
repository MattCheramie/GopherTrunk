package propagate

import "testing"

func TestPropagateConsistentWiring(t *testing.T) {
	t.Parallel()
	// Build constraints where HNext is genuinely a table of (Hcur ^ eo).
	table := make([]uint8, 256)
	for i := range table {
		table[i] = uint8(i*31 + 7)
	}
	var cons []Constraint
	for h := 0; h < 64; h++ {
		for e := 0; e < 4; e++ {
			idx := uint8(h) ^ uint8(e*40)
			cons = append(cons, Constraint{HPrev: 1, HCur: uint8(h), Eo: uint8(e * 40), HNext: table[idx]})
		}
	}
	w := Wiring{Inputs: []Source{HCur, Eo}, Ops: []Op{Xor}}
	if r := Propagate(w, cons); r.Conflicts != 0 {
		t.Fatalf("consistent wiring had %d conflicts", r.Conflicts)
	}
	// A wrong wiring (Hprev alone, which is constant here) collapses all
	// constraints onto one index and conflicts.
	bad := Wiring{Inputs: []Source{HPrev}}
	if r := Propagate(bad, cons); r.Conflicts == 0 {
		t.Fatal("degenerate wiring should conflict")
	}
}

func TestEnumerateWiringsNonEmpty(t *testing.T) {
	t.Parallel()
	ws := EnumerateWirings()
	if len(ws) < 3+18+54 {
		t.Fatalf("enumerated only %d wirings", len(ws))
	}
	// Spot check stringification is stable/non-empty.
	for _, w := range ws {
		if w.String() == "" || w.String() == "(empty)" {
			t.Fatalf("bad wiring string for %+v", w)
		}
	}
}
