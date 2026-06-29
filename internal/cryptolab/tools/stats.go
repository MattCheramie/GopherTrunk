package tools

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
)

type statsTool struct{}

func (statsTool) Name() string { return "stats" }
func (statsTool) Synopsis() string {
	return "entropy / IC / chi-square / XOR key-length triage of a payload"
}
func (statsTool) Modes() []cryptolab.Mode { return []cryptolab.Mode{statsScan{}, statsPeriod{}} }

type statsScan struct{}

func (statsScan) Name() string     { return "scan" }
func (statsScan) Synopsis() string { return "report breakability statistics for a binary payload" }

func (statsScan) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("stats scan")
	in := fs.String("in", "", "binary payload to analyze — required")
	maxKeyLen := fs.Int("max-keylen", 40, "max repeating-XOR key length to score")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	data, err := readInput(*in)
	if err != nil {
		return nil, err
	}

	ent := stats.ShannonEntropy(data)
	ic := stats.IndexOfCoincidence(data)
	chi := stats.ChiSquareUniform(data)
	keylens := stats.GuessXORKeyLength(data, *maxKeyLen)

	res := &cryptolab.Result{Tool: "stats", Mode: "scan",
		Summary: fmt.Sprintf("%d bytes: entropy %.3f bits/byte, IC %.4f, chi-square %.1f", len(data), ent, ic, chi)}
	res.AddField("bytes", len(data))
	res.AddField("entropy_bits", ent)
	res.AddField("index_of_coincidence", ic)
	res.AddField("chi_square_uniform", chi)
	for i, kl := range keylens {
		if i >= 5 {
			break
		}
		res.AddFinding(fmt.Sprintf("keylen=%d", kl.Length), 1/(1+kl.Score),
			map[string]any{"normalized_hamming": kl.Score})
	}
	switch {
	case ent > 7.5:
		res.Note("entropy near 8 bits/byte: looks like strong encryption or compression — no weak-cipher structure to exploit.")
	case ic > 0.045:
		res.Note("elevated index of coincidence: looks like a mono-alphabetic / shift cipher or plaintext — try the brute tool.")
	default:
		res.Note("intermediate statistics: possibly a polyalphabetic / repeating-key cipher — try the top key lengths above with `cryptolab brute`.")
	}
	return res, nil
}

// statsPeriod reports byte-autocorrelation peaks and repeated n-grams, the
// measurements that reveal the period of a repeating-key cipher, a periodic
// scrambler, or a reused keystream before choosing a key length / frame size.
type statsPeriod struct{}

func (statsPeriod) Name() string { return "period" }
func (statsPeriod) Synopsis() string {
	return "autocorrelation period detection and repeated-n-gram histogram"
}

func (statsPeriod) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("stats period")
	in := fs.String("in", "", "binary payload to analyze — required")
	maxLag := fs.Int("max-lag", 256, "maximum autocorrelation lag (bytes) to scan")
	ngram := fs.Int("ngram", 3, "n-gram length for the repeated-block histogram")
	top := fs.Int("top", 8, "number of candidates to report")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	data, err := readInput(*in)
	if err != nil {
		return nil, err
	}

	cands := stats.DetectPeriod(data, *maxLag, *top)
	grams := stats.TopNGrams(data, *ngram, *top)

	res := &cryptolab.Result{Tool: "stats", Mode: "period",
		Summary: fmt.Sprintf("%d bytes: %d period candidate(s) within lag %d", len(data), len(cands), *maxLag)}
	res.AddField("bytes", len(data))
	res.AddField("max_lag", *maxLag)
	res.AddField("period_candidates", len(cands))

	for _, c := range cands {
		res.AddFinding(fmt.Sprintf("period=%d", c.Lag), c.ZScore,
			map[string]any{"coincidence": c.Coincidence, "z_score": c.ZScore})
	}
	for _, g := range grams {
		if g.Count < 2 {
			continue
		}
		res.AddFinding(fmt.Sprintf("ngram=%x", g.Bytes), float64(g.Count),
			map[string]any{"count": g.Count, "kind": "repeated_ngram"})
	}

	switch {
	case len(cands) == 0:
		res.Note("no autocorrelation peak stood out: no short repeating period within the scanned lag — consistent with strong encryption or a long/aperiodic keystream.")
	default:
		res.Note(fmt.Sprintf("strongest period is %d byte(s): for a repeating-XOR cipher try `cryptolab brute xor -keylen %d`; for a periodic scrambler this is the likely frame size.", cands[0].Lag, cands[0].Lag))
	}
	return res, nil
}

func init() { cryptolab.Register(statsTool{}) }
