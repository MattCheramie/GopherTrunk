package motorola

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/propagate"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/progress"
)

// exportTransitions writes the dense high-byte recurrence as a
// Hprev,Hcur,eo,Hnext CSV for the optional Z3 structural search.
func exportTransitions(dir string, cons []propagate.Constraint) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "high-transitions.csv"))
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	fmt.Fprintln(bw, "Hprev,Hcur,eo,Hnext")
	for _, c := range cons {
		fmt.Fprintf(bw, "%d,%d,%d,%d\n", c.HPrev, c.HCur, c.Eo, c.HNext)
	}
	return bw.Flush()
}

// loadFlags parses the shared -csv flag (and returns the remaining FlagSet
// so a mode can add its own knobs first). The corpus path is required.
func parseCorpus(modeName string, args []string, env cryptolab.Env, extra func(*flag.FlagSet)) ([]Row, error) {
	fs := flag.NewFlagSet("cryptolab alias "+modeName, flag.ContinueOnError)
	csvPath := fs.String("csv", "", "ground-truth corpus CSV (rid,talkgroup,encoded_hex,alias) — required")
	if extra != nil {
		extra(fs)
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *csvPath == "" {
		return nil, fmt.Errorf("alias %s: -csv is required", modeName)
	}
	rows, err := LoadCSV(*csvPath, env.Logger)
	if err != nil {
		return nil, err
	}
	env.Logf("corpus loaded", "rows", len(rows))
	return rows, nil
}

// ---- Mode 1: gauge sweep ----

type gaugeMode struct{}

func (gaugeMode) Name() string { return "gauge" }
func (gaugeMode) Synopsis() string {
	return "sweep all 32,768 affine gauges for a clean coordinate frame"
}

func (gaugeMode) Run(ctx context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	rows, err := parseCorpus("gauge", args, env, nil)
	if err != nil {
		return nil, err
	}
	t := Recovered()

	// Recover per-context (Hodd, Modd) observations in gauge coordinates by
	// fitting the affine char = Modd·LUT[eo] − Modd·Hodd within each context.
	obs := recoverComponentObs(t, rows)
	if len(obs) < 2 {
		return nil, fmt.Errorf("alias gauge: too few resolvable contexts (%d)", len(obs))
	}
	env.Logf("component observations", "contexts", len(obs))

	// A clean *affine* relation is gauge-invariant, so it tells us nothing
	// about the gauge. The gauge only changes whether a *nonlinear* merged
	// index becomes functional. So per gauge we test whether Hodd_true is a
	// function of (HPrev_true XOR HCur_true) and report the conflict floor.
	jl, _ := progress.OpenJSONL(env.OutDir, "gauge-sweep.jsonl")
	defer jl.Close()
	thr := &progress.Throttle{Logger: env.Logger, Msg: "gauge swept", Every: 4096}

	type scored struct {
		g         gauge.Gauge
		conflicts int
	}
	best := scored{conflicts: len(obs) + 1}
	var cleanCount int
	for _, g := range gauge.All() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		conf := gaugeMergedIndexConflicts(g, obs)
		thr.Tick("mg", g.Mg, "cg", g.Cg, "conflicts", conf)
		if conf < best.conflicts {
			best = scored{g: g, conflicts: conf}
			thr.Best("new best gauge", "mg", g.Mg, "cg", g.Cg, "conflicts", conf)
			_ = jl.Append(map[string]any{"mg": g.Mg, "cg": g.Cg, "conflicts": conf})
		}
		if conf == 0 {
			cleanCount++
		}
	}

	res := &cryptolab.Result{Tool: "alias", Mode: "gauge", Summary: fmt.Sprintf("swept 32768 gauges; best conflict floor %d/%d contexts", best.conflicts, len(obs))}
	res.AddField("contexts", len(obs))
	res.AddField("best_mg", best.g.Mg)
	res.AddField("best_cg", best.g.Cg)
	res.AddField("best_conflicts", best.conflicts)
	res.AddField("clean_gauges", cleanCount)
	res.AddFinding(fmt.Sprintf("gauge Mg=%d Cg=%d", best.g.Mg, best.g.Cg), 1-float64(best.conflicts)/float64(len(obs)),
		map[string]any{"conflicts": best.conflicts})
	if cleanCount == 0 {
		res.Note("no gauge makes the odd high byte a clean function of the XOR merged index — consistent with the hidden internal table; the high byte's true dependency is on the unobserved low byte.")
		res.Note("to close this: feed more ciphertext-only captures (grows the high-byte recurrence) or labeled rows (grows the multiplier map), then re-run structure and cells.")
	}
	return res, nil
}

// ---- Mode 2: structure / wiring enumeration ----

type structureMode struct{}

func (structureMode) Name() string { return "structure" }
func (structureMode) Synopsis() string {
	return "enumerate merged-index table wirings over the dense high-byte recurrence"
}

func (structureMode) Run(ctx context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	rows, err := parseCorpus("structure", args, env, nil)
	if err != nil {
		return nil, err
	}
	t := Recovered()
	cons := t.HighConstraints(rows)
	if len(cons) == 0 {
		return nil, fmt.Errorf("alias structure: no high-byte constraints (need rows of >= 3 chars)")
	}
	env.Logf("high-byte constraints", "count", len(cons))

	// Export the dense transitions for the optional Z3 structural search
	// (internal/cryptolab/smt/solve_structure.py consumes this CSV).
	if env.OutDir != "" {
		if err := exportTransitions(env.OutDir, cons); err != nil {
			return nil, err
		}
	}

	jl, _ := progress.OpenJSONL(env.OutDir, "structure-survivors.jsonl")
	defer jl.Close()

	wirings := propagate.EnumerateWirings()
	type wr struct {
		w    propagate.Wiring
		conf int
		keys int
	}
	results := make([]wr, 0, len(wirings))
	thr := &progress.Throttle{Logger: env.Logger, Msg: "wiring tested", Every: 8}
	bestConf := len(cons) + 1
	for _, w := range wirings {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		r := propagate.Propagate(w, cons)
		results = append(results, wr{w: w, conf: r.Conflicts, keys: r.Keys})
		thr.Tick("wiring", w.String(), "conflicts", r.Conflicts, "keys", r.Keys)
		if r.Conflicts < bestConf {
			bestConf = r.Conflicts
			thr.Best("new best wiring", "wiring", w.String(), "conflicts", r.Conflicts)
		}
		if r.Conflicts == 0 {
			_ = jl.Append(map[string]any{"wiring": w.String(), "keys": r.Keys})
		}
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].conf < results[j].conf })

	res := &cryptolab.Result{Tool: "alias", Mode: "structure",
		Summary: fmt.Sprintf("tested %d merged-index wirings over %d high-byte transitions; best conflict floor %d", len(wirings), len(cons), bestConf)}
	res.AddField("constraints", len(cons))
	res.AddField("wirings", len(wirings))
	res.AddField("best_conflicts", bestConf)
	top := results
	if len(top) > 8 {
		top = top[:8]
	}
	for _, r := range top {
		res.AddFinding(r.w.String(), 1-float64(r.conf)/float64(len(cons)),
			map[string]any{"conflicts": r.conf, "keys": r.keys})
	}
	if bestConf > 0 {
		res.Note("no single merged-index lookup reproduces H[k+1]: the high byte is not a function of any 2-byte combination of the readable inputs — its true dependency is on the hidden low byte. This reproduces the documented wall.")
		res.Note("richer multi-round / two-table forms are the next step (see internal/cryptolab/smt for the optional Z3 search).")
	}
	return res, nil
}

// ---- Mode 3: incremental cell solver ----

type cellsMode struct{}

func (cellsMode) Name() string { return "cells" }
func (cellsMode) Synopsis() string {
	return "intersect per-context state candidates across the corpus (monotone, resumable)"
}

// cellCheckpoint is the resumable state: surviving candidate states per
// context key. Monotone — Ingest only intersects.
type cellCheckpoint struct {
	Version    int                 `json:"version"`
	Candidates map[string][]uint16 `json:"candidates"` // ctxKey -> packed (Hi<<8|Mul) states
}

func ctxKey(c Context) string { return fmt.Sprintf("%d:%d", c.HPrev, c.HCur) }

func packState(s gauge.State) uint16 { return uint16(s.Hi)<<8 | uint16(s.Mul) }

func (cellsMode) Run(ctx context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	// -resume is also accepted as a mode flag (the web console passes an
	// uploaded checkpoint this way); it falls back to the CLI-global
	// env.ResumePath when not given.
	var resumeFlag string
	rows, err := parseCorpus("cells", args, env, func(fs *flag.FlagSet) {
		fs.StringVar(&resumeFlag, "resume", "", "checkpoint file to resume from")
	})
	if err != nil {
		return nil, err
	}
	t := Recovered()
	resumePath := resumeFlag
	if resumePath == "" {
		resumePath = env.ResumePath
	}

	ck := &cellCheckpoint{Version: 1, Candidates: map[string][]uint16{}}
	if resumePath != "" {
		if found, err := progress.LoadCheckpoint(resumePath, ck); err != nil {
			return nil, err
		} else if found {
			env.Logf("resumed checkpoint", "contexts", len(ck.Candidates))
		}
	}

	// Intersect candidate state sets per context across every observation.
	for _, row := range rows {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		for _, o := range t.CharObservations(row) {
			key := ctxKey(o.Ctx)
			cand := candidateSet(t, o)
			if prev, ok := ck.Candidates[key]; ok {
				ck.Candidates[key] = intersect(prev, cand)
			} else {
				ck.Candidates[key] = cand
			}
		}
	}

	pinned := 0
	for _, c := range ck.Candidates {
		if len(c) == 1 {
			pinned++
		}
	}
	// Decode coverage: a char is decodable when its context is pinned.
	charsDecodable, charsTotal, msgsFull := decodeCoverage(t, rows, ck)

	// Persist the (refined) checkpoint. Prefer the artifact dir so the web
	// console can offer it as a top-level download; otherwise write back to
	// the resume path so a CLI `-resume` run accumulates in place.
	ckPath := ""
	switch {
	case env.OutDir != "":
		ckPath = filepath.Join(env.OutDir, "checkpoint.json")
	case resumePath != "":
		ckPath = resumePath
	}
	if ckPath != "" {
		if err := progress.SaveCheckpoint(ckPath, ck); err != nil {
			return nil, err
		}
		env.Logf("checkpoint saved", "path", ckPath)
	}
	env.Logf("coverage", "pinned", pinned, "contexts", len(ck.Candidates),
		"chars_decodable", charsDecodable, "chars_total", charsTotal, "messages_full", msgsFull)

	res := &cryptolab.Result{Tool: "alias", Mode: "cells",
		Summary: fmt.Sprintf("pinned %d/%d contexts; %d/%d chars decodable; %d messages fully decoded",
			pinned, len(ck.Candidates), charsDecodable, charsTotal, msgsFull)}
	res.AddField("contexts", len(ck.Candidates))
	res.AddField("pinned", pinned)
	res.AddField("chars_decodable", charsDecodable)
	res.AddField("chars_total", charsTotal)
	res.AddField("messages_full", msgsFull)
	res.Note("monotone & resumable: re-run with -resume and more captures (even ciphertext-only) and coverage only grows. Contexts never seen stay blank — that is the passive-corpus coverage limit, not a bug.")
	return res, nil
}

// ---- Mode 4: from-seed simulation brute ----

type fromseedMode struct{}

func (fromseedMode) Name() string { return "fromseed" }
func (fromseedMode) Synopsis() string {
	return "simulate candidate state updates from the length seed, auto-gauge, and cull"
}

func (fromseedMode) Run(ctx context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	rows, err := parseCorpus("fromseed", args, env, nil)
	if err != nil {
		return nil, err
	}
	t := Recovered()
	fams := defaultFamilies()

	jl, _ := progress.OpenJSONL(env.OutDir, "fromseed-survivors.jsonl")
	defer jl.Close()

	res := &cryptolab.Result{Tool: "alias", Mode: "fromseed"}
	bestRate := 1.0
	for _, fam := range fams {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		mism, total := simulateFamily(t, rows, fam)
		rate := 1.0
		if total > 0 {
			rate = float64(mism) / float64(total)
		}
		env.Logf("family simulated", "name", fam.Name, "mismatch_rate", rate, "samples", total)
		res.AddFinding(fam.Name, 1-rate, map[string]any{"mismatch_rate": rate, "samples": total})
		if rate < bestRate {
			bestRate = rate
		}
		if rate < 0.01 {
			_ = jl.Append(map[string]any{"family": fam.Name, "mismatch_rate": rate})
		}
	}
	res.Summary = fmt.Sprintf("simulated %d candidate families; best high-byte mismatch rate %.3f", len(fams), bestRate)
	res.AddField("families", len(fams))
	res.AddField("best_mismatch_rate", bestRate)
	if bestRate > 0.01 {
		res.Note("no built-in candidate family reproduces the high byte (matches the ruled-out LCG/etc. families). Add families to defaultFamilies() to extend the search; the auto-gauge handles the affine ambiguity.")
	}
	return res, nil
}
