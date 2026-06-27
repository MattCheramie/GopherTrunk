package tools

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
)

type lfsrTool struct{}

func (lfsrTool) Name() string            { return "lfsr" }
func (lfsrTool) Synopsis() string        { return "Berlekamp–Massey LFSR recovery and keystream extraction" }
func (lfsrTool) Modes() []cryptolab.Mode { return []cryptolab.Mode{lfsrBM{}, lfsrKeystream{}} }

type lfsrBM struct{}

func (lfsrBM) Name() string { return "bm" }
func (lfsrBM) Synopsis() string {
	return "recover the shortest LFSR for a bit sequence (file = packed bytes)"
}

func (lfsrBM) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("lfsr bm")
	in := fs.String("in", "", "keystream file (packed bytes, MSB-first) — required")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	data, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	bits := lfsr.BitsFromBytes(data)
	r := lfsr.BerlekampMassey(bits)

	res := &cryptolab.Result{Tool: "lfsr", Mode: "bm",
		Summary: fmt.Sprintf("%d bits: linear complexity %d", len(bits), r.LinearComplexity)}
	res.AddField("bits", len(bits))
	res.AddField("linear_complexity", r.LinearComplexity)
	res.AddField("feedback_taps", r.Taps())
	res.AddFinding(fmt.Sprintf("L=%d taps=%v", r.LinearComplexity, r.Taps()), 1, map[string]any{
		"polynomial": r.Polynomial,
	})
	if r.LinearComplexity >= len(bits)/2 {
		res.Note("linear complexity near n/2: no short LFSR — the sequence is not a simple linear scrambler (or you need more bits).")
	} else {
		res.Note("short LFSR recovered: feed more bits to confirm the taps generalize beyond the observed sequence.")
	}
	return res, nil
}

type lfsrKeystream struct{}

func (lfsrKeystream) Name() string { return "keystream" }
func (lfsrKeystream) Synopsis() string {
	return "extract keystream = plaintext XOR ciphertext, then BM it"
}

func (lfsrKeystream) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("lfsr keystream")
	ptFile := fs.String("pt", "", "known-plaintext file — required")
	ctFile := fs.String("ct", "", "ciphertext file — required")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	pt, err := readInput(*ptFile)
	if err != nil {
		return nil, err
	}
	ct, err := readInput(*ctFile)
	if err != nil {
		return nil, err
	}
	ks := lfsr.Keystream(pt, ct)
	r := lfsr.BerlekampMassey(lfsr.BitsFromBytes(ks))

	res := &cryptolab.Result{Tool: "lfsr", Mode: "keystream",
		Summary: fmt.Sprintf("%d keystream bytes; linear complexity %d", len(ks), r.LinearComplexity)}
	res.AddField("keystream_bytes", len(ks))
	res.AddField("linear_complexity", r.LinearComplexity)
	res.AddFinding(fmt.Sprintf("keystream %x...", ksPrefix(ks)), 1, map[string]any{
		"taps": r.Taps(),
	})
	return res, nil
}

func ksPrefix(b []byte) []byte {
	if len(b) > 8 {
		return b[:8]
	}
	return b
}

func init() { cryptolab.Register(lfsrTool{}) }
