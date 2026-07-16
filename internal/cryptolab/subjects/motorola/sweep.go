package motorola

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"
)

// This file implements the chosen-plaintext differential analysis developed on
// #773 — the techniques that a controlled sweep corpus (moto_alias_t.zip)
// unlocks and that the passive-corpus modes (gauge/structure/cells/fromseed)
// cannot do. In order:
//
//   - sweep detection: group records that share plaintext at every character
//     but one, at a single message length;
//   - differential state pinning: at the swept character the pre-char state is
//     fixed, so value = char·(L|1) + H fits exactly and pins (Hodd, Modd);
//   - fold-input determination: cross-sweep, match the next-high-byte by char
//     vs by value and count distinct deltas — the input the update folds is the
//     one that collapses the deltas;
//   - the g-law / two-state test: match by value and confirm the delta is two
//     values, one equal to the state high-byte difference, i.e.
//     Hnext = H_state + g(value) + branch;
//   - g(value) recovery and coverage;
//   - the low-byte observability accounting: how many (L_in → L_out) update
//     transitions the sweeps actually expose (the wall documented on #773).

// Sweep is a set of chosen-plaintext records that share plaintext at every
// character except CharPos, at one message Length. Varying a single character
// while holding the rest fixed holds the cipher state constant up to that
// character, which is what makes the state pinnable.
type Sweep struct {
	Length  int
	CharPos int // 0-based character index that varies across Rows
	Rows    []Row
}

// PinnedState is a fully recovered odd-position accumulator byte pair, fitted
// from one sweep: value = char·(L|1) + H at the swept byte.
type PinnedState struct {
	Length  int
	CharPos int
	BytePos int   // odd cipher byte index = 2·CharPos+1
	Hodd    uint8 // accumulator high byte at the swept position
	LoddOr1 uint8 // low byte forced odd (L|1); the decode multiplier is its inverse
	Modd    uint8 // decode multiplier = inv(L|1)
	Total   int   // records in the sweep
}

// detectSweeps groups a chosen-plaintext corpus into single-position sweeps.
// Blanking the varied position collapses a sweep's rows to one bucket; blanking
// any other position leaves them differing at the varied one, so only the true
// varied position forms a bucket with several distinct characters. minDistinct
// guards against coincidental 2-row groups.
func detectSweeps(rows []Row, minDistinct int) []Sweep {
	type key struct {
		n, pos int
		blank  string
	}
	buckets := map[key][]Row{}
	for _, r := range rows {
		n := len(r.Enc) / 2
		if r.Alias == "" || n == 0 || n != len(r.Alias) {
			continue
		}
		for p := 0; p < n; p++ {
			b := []byte(r.Alias)
			b[p] = 0
			k := key{n: n, pos: p, blank: string(b)}
			buckets[k] = append(buckets[k], r)
		}
	}
	var out []Sweep
	for k, rs := range buckets {
		distinct := map[byte]bool{}
		for _, r := range rs {
			distinct[r.Alias[k.pos]] = true
		}
		if len(distinct) < minDistinct {
			continue
		}
		out = append(out, Sweep{Length: k.n, CharPos: k.pos, Rows: rs})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Length != out[j].Length {
			return out[i].Length < out[j].Length
		}
		return out[i].CharPos < out[j].CharPos
	})
	return out
}

// pinSweep fits value = char·(L|1) + H at the swept odd byte and returns the
// pinned odd-position state. ok is false when the fit is not a clean affine map
// with an odd slope over every record (i.e. the "sweep" is not a single fixed
// state — e.g. rows of differing prefixes slipped into the group).
func pinSweep(t *Table, sw Sweep) (PinnedState, bool) {
	bytePos := 2*sw.CharPos + 1
	type pt struct{ char, val uint8 }
	pts := make([]pt, 0, len(sw.Rows))
	for _, r := range sw.Rows {
		if bytePos >= len(r.Enc) {
			return PinnedState{}, false
		}
		pts = append(pts, pt{char: r.Alias[sw.CharPos], val: t.Fwd[r.Enc[bytePos]]})
	}
	// Solve the slope from a pair with an odd character difference (invertible
	// mod 256), then verify it against every record.
	var s, b uint8
	found := false
	for i := 1; i < len(pts) && !found; i++ {
		dc := pts[i].char - pts[0].char
		if dc&1 == 1 {
			s = (pts[i].val - pts[0].val) * gauge.ModInv(dc)
			b = pts[0].val - s*pts[0].char
			found = true
		}
	}
	if !found || s&1 == 0 {
		return PinnedState{}, false
	}
	for _, p := range pts {
		if s*p.char+b != p.val {
			return PinnedState{}, false
		}
	}
	return PinnedState{
		Length: sw.Length, CharPos: sw.CharPos, BytePos: bytePos,
		Hodd: b, LoddOr1: s, Modd: gauge.ModInv(s), Total: len(pts),
	}, true
}

// transferMap returns, for a pinned sweep that has a following even byte, the
// map value → next high byte: value = FWD[C] at the swept byte (the update
// input), next high byte = FWD[C] at the following even byte (the update output
// high byte). Empty when the swept character is the last one.
func transferMap(t *Table, sw Sweep) map[uint8]uint8 {
	nextEven := 2 * (sw.CharPos + 1)
	out := map[uint8]uint8{}
	for _, r := range sw.Rows {
		if nextEven >= len(r.Enc) {
			return nil
		}
		out[t.Fwd[r.Enc[2*sw.CharPos+1]]] = t.Fwd[r.Enc[nextEven]]
	}
	return out
}

// charTransferMap is transferMap keyed by the plaintext char instead of the
// value — used only to contrast the two fold-input hypotheses.
func charTransferMap(t *Table, sw Sweep) map[uint8]uint8 {
	nextEven := 2 * (sw.CharPos + 1)
	out := map[uint8]uint8{}
	for _, r := range sw.Rows {
		if nextEven >= len(r.Enc) {
			return nil
		}
		out[r.Alias[sw.CharPos]] = t.Fwd[r.Enc[nextEven]]
	}
	return out
}

// distinctDeltas counts the distinct (mb[k] − ma[k]) mod 256 over the shared
// keys of two maps, and reports whether one of them equals want (the state
// high-byte difference, the g-law's signature).
func distinctDeltas(ma, mb map[uint8]uint8, want uint8) (distinct int, hasWant bool, common int) {
	set := map[uint8]bool{}
	for k, va := range ma {
		vb, ok := mb[k]
		if !ok {
			continue
		}
		common++
		d := vb - va
		set[d] = true
		if d == want {
			hasWant = true
		}
	}
	return len(set), hasWant, common
}

// ---- Mode: chosen-plaintext sweep analysis ----

type sweepMode struct{}

func (sweepMode) Name() string { return "sweep" }
func (sweepMode) Synopsis() string {
	return "chosen-plaintext differential: detect sweeps, pin states, find the fold input and Hnext law"
}

func (sweepMode) Run(ctx context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	rows, err := parseChosen("sweep", args, env)
	if err != nil {
		return nil, err
	}
	t := Recovered()

	sweeps := detectSweeps(rows, 3)
	env.Logf("sweeps detected", "count", len(sweeps))

	// Pin the odd-position state for each sweep (differential state pinning).
	type pinned struct {
		st  PinnedState
		val map[uint8]uint8 // value -> next high byte (nil if last char)
		chr map[uint8]uint8 // char  -> next high byte
	}
	var pins []pinned
	for _, sw := range sweeps {
		ps, ok := pinSweep(t, sw)
		if !ok {
			continue
		}
		pins = append(pins, pinned{st: ps, val: transferMap(t, sw), chr: charTransferMap(t, sw)})
		env.Logf("state pinned", "len", ps.Length, "char_pos", ps.CharPos,
			"H", ps.Hodd, "L_or1", ps.LoddOr1, "Modd", ps.Modd, "records", ps.Total)
	}

	res := &cryptolab.Result{Tool: "alias", Mode: "sweep"}
	res.AddField("rows", len(rows))
	res.AddField("sweeps_detected", len(sweeps))
	res.AddField("states_pinned", len(pins))
	for _, p := range pins {
		res.AddFinding(fmt.Sprintf("state len=%d pos=%d: H=%d L|1=%d Modd=%d",
			p.st.Length, p.st.CharPos, p.st.Hodd, p.st.LoddOr1, p.st.Modd), 1.0,
			map[string]any{"length": p.st.Length, "char_pos": p.st.CharPos,
				"H": p.st.Hodd, "L_or1": p.st.LoddOr1, "Modd": p.st.Modd, "records": p.st.Total})
	}
	if len(pins) < 2 {
		res.Summary = fmt.Sprintf("detected %d sweeps, pinned %d states (need >=2 with a following char for the transfer analysis)", len(sweeps), len(pins))
		res.Note("differential state pinning works; supply sweeps at >=2 positions that each have a following character to run the fold-input and Hnext-law analysis.")
		return res, nil
	}

	// Fold-input determination + g-law, over every pair of pinned states that
	// each have a transfer map (a following even byte).
	var withTransfer []pinned
	for _, p := range pins {
		if len(p.val) > 0 {
			withTransfer = append(withTransfer, p)
		}
	}
	byCharTotal, byValTotal := 0, 0
	pairs := 0
	glawOK := 0
	for i := 0; i < len(withTransfer); i++ {
		for j := i + 1; j < len(withTransfer); j++ {
			a, b := withTransfer[i], withTransfer[j]
			want := b.st.Hodd - a.st.Hodd
			dChar, _, _ := distinctDeltas(a.chr, b.chr, want)
			dVal, hasWant, common := distinctDeltas(a.val, b.val, want)
			if common == 0 {
				continue
			}
			pairs++
			byCharTotal += dChar
			byValTotal += dVal
			if dVal <= 2 && hasWant {
				glawOK++
			}
			env.Logf("pair analysed",
				"a", fmt.Sprintf("H%d", a.st.Hodd), "b", fmt.Sprintf("H%d", b.st.Hodd),
				"deltas_by_char", dChar, "deltas_by_value", dVal, "hdiff_present", hasWant)
		}
	}

	// g(value) recovery + coverage: value -> H_next − H_state, state-independent.
	gseen := map[uint8]bool{}
	for _, p := range withTransfer {
		for v := range p.val {
			gseen[v] = true
		}
	}

	// Low-byte observability: L is exposed (as L|1) only at swept odd bytes,
	// which are >=2 bytes apart, so no (L_in → L_out) single-step transition is
	// observed. Count adjacent-byte pinned pairs to make the floor explicit.
	lowTransitions := 0
	for i := 0; i < len(pins); i++ {
		for j := 0; j < len(pins); j++ {
			if pins[i].st.Length == pins[j].st.Length && pins[j].st.BytePos == pins[i].st.BytePos+1 {
				lowTransitions++
			}
		}
	}

	res.AddField("input_fold_by_char_avg_deltas", ratio(byCharTotal, pairs))
	res.AddField("input_fold_by_value_avg_deltas", ratio(byValTotal, pairs))
	res.AddField("glaw_pairs_confirmed", fmt.Sprintf("%d/%d", glawOK, pairs))
	res.AddField("g_value_coverage", fmt.Sprintf("%d/256", len(gseen)))
	res.AddField("low_byte_transitions_observed", lowTransitions)
	res.Summary = fmt.Sprintf("pinned %d states; fold input = %s; g-law confirmed %d/%d pairs; g covers %d/256; low-byte transitions observed %d",
		len(pins), foldVerdict(byCharTotal, byValTotal), glawOK, pairs, len(gseen), lowTransitions)

	if byValTotal < byCharTotal {
		res.Note("the update folds the ciphertext byte / value FWD[C], not the plaintext char: matching next-high-byte by value collapses the cross-state deltas far below matching by char.")
	}
	if glawOK == pairs && pairs > 0 {
		res.Note("Hnext = H_state + g(value) + branch confirmed: every state pair's next-high-byte delta is two values, one equal to the state high-byte difference. g(value) is state-independent; the residual branch is the hidden-low-byte selector.")
	}
	if lowTransitions == 0 {
		res.Note("low-byte observability floor: L is exposed only as L|1 at the swept (non-consecutive) odd bytes, so ZERO (L_in -> L_out) update transitions are observed — the low-byte update and branch table cannot be fitted from these sweeps. The decisive capture sweeps CONSECUTIVE positions so L_in -> L_out becomes observable.")
	}
	return res, nil
}

// parseChosen loads a chosen-plaintext corpus (encoded_hex is pure cipher
// output, no CRC) via the shared -csv flag.
func parseChosen(modeName string, args []string, env cryptolab.Env) ([]Row, error) {
	fs := flag.NewFlagSet("cryptolab alias "+modeName, flag.ContinueOnError)
	csvPath := fs.String("csv", "", "chosen-plaintext corpus CSV (rid,talkgroup,encoded_hex,alias); encoded_hex is pure cipher output, no CRC — required")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *csvPath == "" {
		return nil, fmt.Errorf("alias %s: -csv is required", modeName)
	}
	rows, err := LoadChosenCSV(*csvPath, env.Logger)
	if err != nil {
		return nil, err
	}
	env.Logf("chosen-plaintext corpus loaded", "rows", len(rows))
	return rows, nil
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func foldVerdict(byChar, byVal int) string {
	if byVal < byChar {
		return "value/ciphertext-byte"
	}
	if byChar < byVal {
		return "plaintext-char"
	}
	return "indeterminate"
}
