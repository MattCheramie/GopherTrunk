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
func (statsTool) Modes() []cryptolab.Mode { return []cryptolab.Mode{statsScan{}} }

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

func init() { cryptolab.Register(statsTool{}) }
