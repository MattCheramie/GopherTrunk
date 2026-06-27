// Package propagate is the toolkit's fast constraint propagator for the
// "is the next state a table lookup of some combined index?" family of
// hypotheses. Given a corpus of (inputs → output) byte transitions and a
// wiring (which inputs feed the index and how they combine), it tries to
// fill a single 256-entry table directly from the data and reports
// conflicts. Zero conflicts means the wiring's single-lookup form is
// consistent; a nonzero floor across every wiring is a theorem that the
// output is not a function of any such merged index — which is exactly the
// wall the alias high-byte recurrence runs into.
package propagate

// Source selects one input byte of a transition.
type Source int

const (
	HPrev Source = iota // H[k-1]
	HCur                // H[k]
	Eo                  // odd ciphertext byte eo[k]
)

func (s Source) String() string {
	switch s {
	case HPrev:
		return "Hprev"
	case HCur:
		return "Hcur"
	case Eo:
		return "eo"
	default:
		return "?"
	}
}

// Op combines two index bytes.
type Op int

const (
	Xor Op = iota
	Add
	Sub
)

func (o Op) String() string {
	switch o {
	case Xor:
		return "^"
	case Add:
		return "+"
	case Sub:
		return "-"
	default:
		return "?"
	}
}

func (o Op) apply(a, b uint8) uint8 {
	switch o {
	case Xor:
		return a ^ b
	case Add:
		return a + b
	case Sub:
		return a - b
	default:
		return a
	}
}

// Constraint is one observed high-byte transition.
type Constraint struct {
	HPrev, HCur, Eo, HNext uint8
}

// Wiring builds an index byte by folding the selected inputs left-to-right
// with the given ops: index = (((in0 op0 in1) op1 in2) ...). len(Ops) must
// be len(Inputs)-1.
type Wiring struct {
	Inputs []Source
	Ops    []Op
}

func (w Wiring) index(c Constraint) uint8 {
	val := func(s Source) uint8 {
		switch s {
		case HPrev:
			return c.HPrev
		case HCur:
			return c.HCur
		case Eo:
			return c.Eo
		default:
			return 0
		}
	}
	if len(w.Inputs) == 0 {
		return 0
	}
	acc := val(w.Inputs[0])
	for i := 1; i < len(w.Inputs); i++ {
		acc = w.Ops[i-1].apply(acc, val(w.Inputs[i]))
	}
	return acc
}

func (w Wiring) String() string {
	if len(w.Inputs) == 0 {
		return "(empty)"
	}
	s := w.Inputs[0].String()
	for i := 1; i < len(w.Inputs); i++ {
		s += w.Ops[i-1].String() + w.Inputs[i].String()
	}
	return s
}

// Result reports how a wiring fared.
type Result struct {
	Wiring    Wiring
	Keys      int // distinct index values observed
	Conflicts int // index values mapping to more than one HNext
}

// Propagate fills a single 256-entry table T[index] = HNext directly from
// the constraints and counts conflicts (an index already bound to a
// different output). Zero conflicts means HNext is a function of this
// wiring's merged index.
func Propagate(w Wiring, cons []Constraint) Result {
	table := make(map[uint8]uint8, 256)
	conflicts := 0
	for _, c := range cons {
		idx := w.index(c)
		if prev, ok := table[idx]; ok {
			if prev != c.HNext {
				conflicts++
			}
		} else {
			table[idx] = c.HNext
		}
	}
	return Result{Wiring: w, Keys: len(table), Conflicts: conflicts}
}

// EnumerateWirings produces the single- and multi-input merged-index
// wirings over {Hprev, Hcur, eo} with combine ops {^,+,-}: the family the
// propagator can falsify in one pass.
func EnumerateWirings() []Wiring {
	sources := []Source{HPrev, HCur, Eo}
	ops := []Op{Xor, Add, Sub}
	var out []Wiring
	// 1-input.
	for _, s := range sources {
		out = append(out, Wiring{Inputs: []Source{s}})
	}
	// 2-input (ordered pairs of distinct sources).
	for _, a := range sources {
		for _, b := range sources {
			if a == b {
				continue
			}
			for _, op := range ops {
				out = append(out, Wiring{Inputs: []Source{a, b}, Ops: []Op{op}})
			}
		}
	}
	// 3-input (all three, one op between each).
	perms := [][]Source{
		{HPrev, HCur, Eo}, {HPrev, Eo, HCur}, {HCur, HPrev, Eo},
		{HCur, Eo, HPrev}, {Eo, HPrev, HCur}, {Eo, HCur, HPrev},
	}
	for _, p := range perms {
		for _, op0 := range ops {
			for _, op1 := range ops {
				out = append(out, Wiring{Inputs: append([]Source(nil), p...), Ops: []Op{op0, op1}})
			}
		}
	}
	return out
}
