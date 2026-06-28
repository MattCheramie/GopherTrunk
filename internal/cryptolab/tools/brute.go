package tools

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/brute"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
)

type bruteTool struct{}

func (bruteTool) Name() string     { return "brute" }
func (bruteTool) Synopsis() string { return "keyspace brute force for classical / XOR ciphers" }
func (bruteTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{bruteXOR{}, bruteCaesar{}, bruteVigenere{}}
}

type bruteXOR struct{}

func (bruteXOR) Name() string { return "xor" }
func (bruteXOR) Synopsis() string {
	return "recover a single- or repeating-byte XOR key from ciphertext"
}

func (bruteXOR) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("brute xor")
	in := fs.String("in", "", "ciphertext file — required")
	keyLen := fs.Int("keylen", 0, "repeating-key length (0 = auto-detect via Hamming)")
	crib := fs.String("crib", "", "optional known-plaintext substring to boost scoring")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	ct, err := readInput(*in)
	if err != nil {
		return nil, err
	}

	var scorer brute.Scorer = brute.English{}
	if *crib != "" {
		scorer = brute.Crib{Crib: []byte(*crib), Base: brute.English{}}
	}

	res := &cryptolab.Result{Tool: "brute", Mode: "xor"}
	if *keyLen > 0 {
		// Explicit key length: solve it directly.
		cand := brute.SolveRepeatingXOR(ct, *keyLen, scorer)
		res.AddField("keylen", *keyLen)
		env.Logf("repeating XOR solved", "keylen", *keyLen)
		addXORFinding(res, cand)
		res.Summary = fmt.Sprintf("recovered %d-byte XOR key", len(cand.Key))
		return res, nil
	}

	// Auto: score the single-byte solution against the top guessed key
	// lengths and keep whichever the scorer likes best. This avoids
	// overfitting a single-byte cipher to a noisy long key-length guess.
	best := brute.SolveSingleByteXOR(ct, scorer)
	bestKL := 1
	guesses := stats.GuessXORKeyLength(ct, 40)
	res.AddField("auto_keylen_candidates", topLens(guesses, 3))
	for _, g := range firstN(guesses, 3) {
		if g.Length <= 1 {
			continue
		}
		c := brute.SolveRepeatingXOR(ct, g.Length, scorer)
		if c.Score > best.Score {
			best, bestKL = c, g.Length
		}
	}
	res.AddField("keylen", bestKL)
	env.Logf("auto XOR solved", "keylen", bestKL)
	addXORFinding(res, best)
	res.Summary = fmt.Sprintf("recovered %d-byte XOR key", len(best.Key))
	return res, nil
}

func firstN(g []stats.KeyLenScore, n int) []stats.KeyLenScore {
	if len(g) > n {
		return g[:n]
	}
	return g
}

// scorerFor returns an English scorer, optionally boosted by a known-plaintext crib.
func scorerFor(crib string) brute.Scorer {
	if crib != "" {
		return brute.Crib{Crib: []byte(crib), Base: brute.English{}}
	}
	return brute.English{}
}

type bruteCaesar struct{}

func (bruteCaesar) Name() string     { return "caesar" }
func (bruteCaesar) Synopsis() string { return "recover a single additive (Caesar/ROT) shift" }

func (bruteCaesar) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("brute caesar")
	in := fs.String("in", "", "ciphertext file — required")
	crib := fs.String("crib", "", "optional known-plaintext substring to boost scoring")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	ct, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	cand := brute.SolveCaesar(ct, scorerFor(*crib))
	res := &cryptolab.Result{Tool: "brute", Mode: "caesar"}
	shift := byte(0)
	if len(cand.Key) == 1 {
		shift = cand.Key[0]
	}
	res.AddField("shift", shift)
	addXORFinding(res, cand)
	res.Summary = fmt.Sprintf("recovered Caesar shift %d", shift)
	return res, nil
}

type bruteVigenere struct{}

func (bruteVigenere) Name() string { return "vigenere" }
func (bruteVigenere) Synopsis() string {
	return "recover a repeating additive (Vigenère) key from ciphertext"
}

func (bruteVigenere) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("brute vigenere")
	in := fs.String("in", "", "ciphertext file — required")
	keyLen := fs.Int("keylen", 0, "key length (0 = auto-detect via Hamming)")
	crib := fs.String("crib", "", "optional known-plaintext substring to boost scoring")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	ct, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	scorer := scorerFor(*crib)
	res := &cryptolab.Result{Tool: "brute", Mode: "vigenere"}
	kl := *keyLen
	if kl <= 0 {
		guesses := stats.GuessXORKeyLength(ct, 40) // periodicity test is cipher-agnostic
		res.AddField("auto_keylen_candidates", topLens(guesses, 3))
		if len(guesses) > 0 {
			kl = guesses[0].Length
		}
		if kl <= 0 {
			kl = 1
		}
	}
	cand := brute.SolveVigenere(ct, kl, scorer)
	res.AddField("keylen", kl)
	env.Logf("vigenere solved", "keylen", kl)
	addXORFinding(res, cand)
	res.Summary = fmt.Sprintf("recovered %d-byte Vigenère key", len(cand.Key))
	return res, nil
}

func addXORFinding(res *cryptolab.Result, c brute.Candidate) {
	preview := c.Plaintext
	if len(preview) > 64 {
		preview = preview[:64]
	}
	res.AddFinding(fmt.Sprintf("key=%x", c.Key), c.Score, map[string]any{
		"key_ascii":        printableOnly(c.Key),
		"plaintext_prefix": string(sanitize(preview)),
	})
	if res.Summary == "" {
		res.Summary = fmt.Sprintf("best key %x", c.Key)
	}
}

func topLens(g []stats.KeyLenScore, n int) []int {
	var out []int
	for i := 0; i < len(g) && i < n; i++ {
		out = append(out, g[i].Length)
	}
	return out
}

func printableOnly(b []byte) string {
	out := make([]byte, len(b))
	for i, c := range b {
		if c >= 0x20 && c <= 0x7e {
			out[i] = c
		} else {
			out[i] = '.'
		}
	}
	return string(out)
}

func sanitize(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		if c == '\n' || c == '\t' || (c >= 0x20 && c <= 0x7e) {
			out[i] = c
		} else {
			out[i] = '.'
		}
	}
	return out
}

func init() { cryptolab.Register(bruteTool{}) }
