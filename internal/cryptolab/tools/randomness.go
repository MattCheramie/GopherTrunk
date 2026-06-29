package tools

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/randomness"
)

type randomnessTool struct{}

func (randomnessTool) Name() string { return "randomness" }
func (randomnessTool) Synopsis() string {
	return "NIST SP 800-22 statistical randomness battery for keystreams / payloads"
}
func (randomnessTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{randomnessBattery{}, randomnessQuick{}}
}

// runBattery is shared by both modes; quick selects the fast subset.
func runBattery(modeName, in string, alpha float64, quick bool) (*cryptolab.Result, error) {
	data, err := readInput(in)
	if err != nil {
		return nil, err
	}
	bits := lfsr.BitsFromBytes(data)
	var rep randomness.Report
	if quick {
		rep = randomness.Quick(bits, alpha)
	} else {
		rep = randomness.Battery(bits, alpha)
	}

	res := &cryptolab.Result{Tool: "randomness", Mode: modeName,
		Summary: fmt.Sprintf("%d bits: %d/%d tests passed, %d failed, %d skipped (alpha %.3f)",
			rep.Bits, rep.Passed, rep.Passed+rep.Failed, rep.Failed, rep.Skipped, rep.Alpha)}
	res.AddField("bits", rep.Bits)
	res.AddField("alpha", rep.Alpha)
	res.AddField("passed", rep.Passed)
	res.AddField("failed", rep.Failed)
	res.AddField("skipped", rep.Skipped)
	res.AddField("looks_random", rep.LooksRandom())

	// Findings: one per test, weakest (most non-random) first. A failing test
	// scores higher so it sorts to the top of the ranked findings.
	for _, t := range rep.WeakestTests() {
		detail := map[string]any{"p_value": t.PValue, "passed": t.Passed}
		if t.Note != "" {
			detail["note"] = t.Note
		}
		score := 0.0
		if !t.Passed {
			score = 1 - t.PValue
		}
		res.AddFinding(t.Name, score, detail)
	}
	for _, t := range rep.Tests {
		if !t.Applicable {
			res.Note(fmt.Sprintf("%s skipped: %s", t.Name, t.Note))
		}
	}

	switch {
	case rep.LooksRandom():
		res.Note("indistinguishable from random at this alpha: consistent with strong keyed encryption (AES/DES-class) or good compression — no weak structure to brute-force. If you have known plaintext for a reused IV/MI, try `cryptolab ks`.")
	case rep.Failed > 0:
		res.Note("one or more tests failed: the stream carries exploitable structure. A failing linear_complexity or spectral test points at an LFSR / keyless scrambler — try `cryptolab lfsr bm` and `cryptolab stats period`.")
	}
	return res, nil
}

type randomnessBattery struct{}

func (randomnessBattery) Name() string { return "battery" }
func (randomnessBattery) Synopsis() string {
	return "run the full NIST SP 800-22 subset (10 tests) over a bitstream"
}
func (randomnessBattery) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("randomness battery")
	in := fs.String("in", "", "bitstream file (packed bytes, MSB-first) — required")
	alpha := fs.Float64("alpha", randomness.DefaultAlpha, "significance level")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return runBattery("battery", *in, *alpha, false)
}

type randomnessQuick struct{}

func (randomnessQuick) Name() string { return "quick" }
func (randomnessQuick) Synopsis() string {
	return "fast randomness subset (monobit / block-freq / runs / cusum) for short captures"
}
func (randomnessQuick) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("randomness quick")
	in := fs.String("in", "", "bitstream file (packed bytes, MSB-first) — required")
	alpha := fs.Float64("alpha", randomness.DefaultAlpha, "significance level")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return runBattery("quick", *in, *alpha, true)
}

func init() { cryptolab.Register(randomnessTool{}) }
