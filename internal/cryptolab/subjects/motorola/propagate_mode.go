package motorola

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"
)

// ---- Mode 5: state-machine propagation ----
//
// propagateMode recovers the per-character state machine explicitly. The
// cell solver (mode 3) pins the per-character state (Hodd, Modd) for a
// context (HPrev, HCur) only when enough observations of that context
// intersect to a single candidate; rare contexts stay ambiguous, which is
// the documented coverage wall. This mode goes one step further: from the
// pinned states it harvests
//
//	the per-character transition  T:(state, eo) -> next-state, and
//	the length seed               seed:(n) -> state at position 0,
//
// then propagates them forward and backward through every message to pin
// contexts the intersection alone could not. It reports how far the
// recovered machine generalizes — the decisive measure of whether a passive
// corpus can close the cipher, and the explicit model of the "nonlinear
// state update" the obfuscator uses (the transition is functional but, per
// the cryptanalysis notes, has no closed form — consistent with an internal
// substitution table distinct from the output LUT).
type propagateMode struct{}

func (propagateMode) Name() string { return "propagate" }
func (propagateMode) Synopsis() string {
	return "harvest the (state,eo)->state transition + length seed and report decode coverage"
}

// transKey is one transition-table key: the packed current state and the
// odd-position ciphertext byte that drives the step.
type transKey struct {
	State uint16 // packed (Hi<<8 | Mul)
	Eo    uint8
}

func unpackState(u uint16) gauge.State {
	return gauge.State{Hi: uint8(u >> 8), Mul: uint8(u)}
}

func (propagateMode) Run(ctx context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	rows, err := parseCorpus("propagate", args, env, nil)
	if err != nil {
		return nil, err
	}
	t := Recovered()

	// Pre-extract each message's ordered per-character observations once.
	// CharObservations returns positions in k order, so obs[0] is the
	// length-seeded position and consecutive entries are transition steps.
	msgs := make([][]CharObs, 0, len(rows))
	for _, row := range rows {
		if o := t.CharObservations(row); len(o) > 0 {
			msgs = append(msgs, o)
		}
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("alias propagate: no labeled rows (need ASCII aliases matching the encoded length)")
	}

	// Seed pinning: intersect candidate states per context (the cell solver).
	cand := map[Context][]uint16{}
	for _, obs := range msgs {
		for _, o := range obs {
			c := candidateSet(t, o)
			if prev, ok := cand[o.Ctx]; ok {
				cand[o.Ctx] = intersect(prev, c)
			} else {
				cand[o.Ctx] = c
			}
		}
	}
	pinned := map[Context]uint16{}
	for cx, s := range cand {
		if len(s) == 1 {
			pinned[cx] = s[0]
		}
	}
	initialPinned := len(pinned)

	// Cascade: harvest the transition + seed from the currently-pinned
	// states, propagate them through every message, and pin the contexts
	// they reach. Monotone — pinned only grows — so it converges.
	var trans map[transKey]uint16
	var transConflicts int
	var seed map[int]uint16
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		trans, transConflicts, seed = harvest(t, msgs, pinned)
		inv := invertTransition(trans)
		newlyPinned := 0
		for _, obs := range msgs {
			st, ok := assignStates(obs, pinned, seed, trans, inv)
			for k, o := range obs {
				if !ok[k] {
					continue
				}
				if _, exists := pinned[o.Ctx]; !exists {
					pinned[o.Ctx] = st[k]
					newlyPinned++
				}
			}
		}
		if newlyPinned == 0 {
			break
		}
	}

	// Coverage: a message decodes when every position's context is pinned;
	// verify the recovered state round-trips to the known plaintext char.
	fullMessages, exactMessages, charsDecodable, charsTotal := 0, 0, 0, 0
	for _, obs := range msgs {
		full := true
		for _, o := range obs {
			charsTotal++
			if _, ok := pinned[o.Ctx]; ok {
				charsDecodable++
			} else {
				full = false
			}
		}
		if !full {
			continue
		}
		fullMessages++
		exact := true
		for _, o := range obs {
			s := unpackState(pinned[o.Ctx])
			if t.DecodeChar(s.Hi, s.Mul, o.Eo) != byte(o.Char) {
				exact = false
				break
			}
		}
		if exact {
			exactMessages++
		}
	}

	res := &cryptolab.Result{Tool: "alias", Mode: "propagate",
		Summary: fmt.Sprintf("recovered %d transition keys (%d conflicts) + %d length seeds; %d/%d messages fully decoded",
			len(trans), transConflicts, len(seed), fullMessages, len(msgs))}
	res.AddField("messages", len(msgs))
	res.AddField("pinned_initial", initialPinned)
	res.AddField("pinned_final", len(pinned))
	res.AddField("transition_keys", len(trans))
	res.AddField("transition_conflicts", transConflicts)
	res.AddField("seed_lengths", len(seed))
	res.AddField("chars_decodable", charsDecodable)
	res.AddField("chars_total", charsTotal)
	res.AddField("messages_full", fullMessages)
	res.AddField("messages_exact", exactMessages)
	res.AddFinding("state-machine transition (state,eo)->state", 1-conflictRate(transConflicts, len(trans)),
		map[string]any{"keys": len(trans), "conflicts": transConflicts})
	res.Note("the transition is functional up to a handful of bit-error conflicts, confirming a deterministic per-character state machine — but it carries no closed form (see the cryptanalysis notes), consistent with an internal substitution table distinct from the output LUT.")
	res.Note("decode coverage is bounded by how many transition/seed keys the corpus exposes; pinned contexts and transition keys grow ~linearly with corpus size (they do not saturate), so a passive corpus — even a dense differential one — cannot reach full coverage. Closing it needs a closed form for the transition or chosen-plaintext captures.")
	return res, nil
}

// harvest collects the per-character transition T:(state,eo)->next-state and
// the length seed from the currently-pinned states, resolving the occasional
// bit-error disagreement by majority vote. It also returns how many
// transition keys had a conflicting (majority-losing) observation.
func harvest(t *Table, msgs [][]CharObs, pinned map[Context]uint16) (trans map[transKey]uint16, conflicts int, seed map[int]uint16) {
	transVotes := map[transKey]map[uint16]int{}
	seedVotes := map[int]map[uint16]int{}
	for _, obs := range msgs {
		n := len(obs)
		st := make([]uint16, n)
		ok := make([]bool, n)
		for k, o := range obs {
			if s, p := pinned[o.Ctx]; p {
				st[k], ok[k] = s, true
			}
		}
		if ok[0] {
			voteFor(seedVotes, n, st[0])
		}
		for k := 0; k+1 < n; k++ {
			if ok[k] && ok[k+1] {
				key := transKey{State: st[k], Eo: obs[k].Eo}
				if transVotes[key] == nil {
					transVotes[key] = map[uint16]int{}
				}
				transVotes[key][st[k+1]]++
			}
		}
	}
	trans = make(map[transKey]uint16, len(transVotes))
	for key, votes := range transVotes {
		best, total := majority(votes)
		trans[key] = best
		if len(votes) > 1 {
			conflicts += total
		}
	}
	seed = make(map[int]uint16, len(seedVotes))
	for n, votes := range seedVotes {
		best, _ := majority(votes)
		seed[n] = best
	}
	return trans, conflicts, seed
}

// assignStates fills the per-position state of one message: from pinned
// contexts, then the length seed at position 0, then forward via the
// transition and backward via its inverse. Returns the states and which
// positions were resolved.
func assignStates(obs []CharObs, pinned map[Context]uint16, seed map[int]uint16, trans, inv map[transKey]uint16) (st []uint16, ok []bool) {
	n := len(obs)
	st = make([]uint16, n)
	ok = make([]bool, n)
	for k, o := range obs {
		if s, p := pinned[o.Ctx]; p {
			st[k], ok[k] = s, true
		}
	}
	if !ok[0] {
		if s, p := seed[n]; p {
			st[0], ok[0] = s, true
		}
	}
	for k := 0; k+1 < n; k++ {
		if ok[k] && !ok[k+1] {
			if ns, p := trans[transKey{State: st[k], Eo: obs[k].Eo}]; p {
				st[k+1], ok[k+1] = ns, true
			}
		}
	}
	for k := n - 1; k > 0; k-- {
		if ok[k] && !ok[k-1] {
			if ps, p := inv[transKey{State: st[k], Eo: obs[k-1].Eo}]; p {
				st[k-1], ok[k-1] = ps, true
			}
		}
	}
	return st, ok
}

// invertTransition builds the inverse (next-state, eo) -> state map used to
// propagate state backward through a message. Keys that are not invertible
// (the same next-state/eo reached from two states) are dropped.
func invertTransition(trans map[transKey]uint16) map[transKey]uint16 {
	rev := map[transKey]map[uint16]bool{}
	for k, next := range trans {
		key := transKey{State: next, Eo: k.Eo}
		if rev[key] == nil {
			rev[key] = map[uint16]bool{}
		}
		rev[key][k.State] = true
	}
	inv := make(map[transKey]uint16, len(rev))
	for key, srcs := range rev {
		if len(srcs) == 1 {
			for s := range srcs {
				inv[key] = s
			}
		}
	}
	return inv
}

func voteFor(votes map[int]map[uint16]int, key int, val uint16) {
	if votes[key] == nil {
		votes[key] = map[uint16]int{}
	}
	votes[key][val]++
}

// majority returns the most-voted value and the count of votes that did not
// match it (the conflict count for that key).
func majority(votes map[uint16]int) (best uint16, conflicts int) {
	bestCount := -1
	total := 0
	for v, c := range votes {
		total += c
		// Deterministic tie-break on the value keeps results stable.
		if c > bestCount || (c == bestCount && v < best) {
			best, bestCount = v, c
		}
	}
	return best, total - bestCount
}

func conflictRate(conflicts, keys int) float64 {
	if keys == 0 {
		return 0
	}
	return float64(conflicts) / float64(keys)
}
