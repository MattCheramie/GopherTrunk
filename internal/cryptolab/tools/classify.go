package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/randomness"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
)

// classifyTool is the "front door": it runs the cheap measurements the other
// tools rely on (entropy / IC / XOR key-length / autocorrelation period /
// randomness battery) and routes the operator to the right next command. It
// is the RF-framed analogue of Ciphey's auto-detect and CrypTool's cipher
// identifier — it knows about LFSR scramblers and reused keystreams, not just
// book ciphers.
type classifyTool struct{}

func (classifyTool) Name() string { return "classify" }
func (classifyTool) Synopsis() string {
	return "identify a payload's obfuscation class and recommend the next tool"
}
func (classifyTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{classifyAuto{}}
}

type classifyAuto struct{}

func (classifyAuto) Name() string { return "auto" }
func (classifyAuto) Synopsis() string {
	return "triage an unknown payload and recommend a next command"
}

type verdict struct {
	class       string
	confidence  float64
	rationale   string
	recommended string
}

func (classifyAuto) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("classify auto")
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
	printable, letters := charProfile(data)
	keylens := stats.GuessXORKeyLength(data, *maxKeyLen)
	periods := stats.DetectPeriod(data, 256, 4)

	// Randomness signal: the full battery when we have enough bits, else the
	// fast subset. The linear_complexity and spectral verdicts separate
	// "strong encryption" from "structured scrambler".
	bits := lfsr.BitsFromBytes(data)
	var rep randomness.Report
	if len(data) >= 2048 {
		rep = randomness.Battery(bits, randomness.DefaultAlpha)
	} else {
		rep = randomness.Quick(bits, randomness.DefaultAlpha)
	}
	lcFailed := failed(rep, "linear_complexity")
	spectralFailed := failed(rep, "spectral_dft")

	var vs []verdict

	if printable > 0.95 && ent < 4.8 {
		vs = append(vs, verdict{"plaintext", 0.6 + 0.4*printable,
			fmt.Sprintf("%.0f%% printable, entropy %.2f", printable*100, ent),
			"already plaintext — no decryption needed"})
	}
	if ic > 0.045 && letters > 0.6 {
		vs = append(vs, verdict{"substitution-or-shift", clamp01(ic / 0.066),
			fmt.Sprintf("index of coincidence %.4f near English, %.0f%% letters", ic, letters*100),
			"cryptolab brute substitution -in <file>   (or `brute caesar` for a single shift)"})
	}
	if len(keylens) > 0 && keylens[0].Length > 1 && keylens[0].Score < 3.6 {
		conf := clamp01((3.6 - keylens[0].Score) / 3.6)
		vs = append(vs, verdict{"repeating-xor", conf,
			fmt.Sprintf("repeating-XOR key length ~%d (normalized Hamming %.2f)", keylens[0].Length, keylens[0].Score),
			fmt.Sprintf("cryptolab brute xor -in <file> -keylen %d", keylens[0].Length)})
	}
	if len(periods) > 0 {
		vs = append(vs, verdict{"periodic-scrambler", clamp01(periods[0].ZScore / 12),
			fmt.Sprintf("autocorrelation peak at lag %d (z=%.1f)", periods[0].Lag, periods[0].ZScore),
			fmt.Sprintf("cryptolab stats period -in <file>   then `descramble rolling` (frame≈%d) or `ks mtp` if frames reuse an IV", periods[0].Lag)})
	}
	if ent > 7.0 && (lcFailed || spectralFailed) {
		vs = append(vs, verdict{"lfsr-or-keyless-scrambler", 0.7,
			fmt.Sprintf("high entropy %.2f but failing %s test(s)", ent, failedNames(rep)),
			"cryptolab lfsr bm -in <file>   (recover the shortest LFSR / linear scrambler)"})
	}
	if ent > 7.9 && rep.LooksRandom() {
		vs = append(vs, verdict{"strong-encrypted", clamp01((ent - 7.9) / 0.1),
			fmt.Sprintf("entropy %.3f and passes the randomness battery", ent),
			"no weak structure to attack — if frames reuse an IV/MI try `cryptolab ks reuse`, else key material is required"})
	}
	if len(vs) == 0 {
		vs = append(vs, verdict{"inconclusive", 0.2,
			fmt.Sprintf("intermediate statistics (entropy %.2f, IC %.4f)", ent, ic),
			"cryptolab stats scan -in <file>   then try the brute tool on the top key lengths"})
	}

	sort.SliceStable(vs, func(i, j int) bool { return vs[i].confidence > vs[j].confidence })

	res := &cryptolab.Result{Tool: "classify", Mode: "auto",
		Summary: fmt.Sprintf("%d bytes: likely %s (confidence %.0f%%)", len(data), vs[0].class, vs[0].confidence*100)}
	res.AddField("bytes", len(data))
	res.AddField("entropy_bits", ent)
	res.AddField("index_of_coincidence", ic)
	res.AddField("chi_square_uniform", chi)
	res.AddField("printable_fraction", printable)
	res.AddField("randomness_failed", rep.Failed)
	res.AddField("recommended", vs[0].recommended)
	for _, v := range vs {
		res.AddFinding(v.class, v.confidence, map[string]any{
			"rationale":   v.rationale,
			"recommended": v.recommended,
		})
	}
	res.Note("classify only triages byte payloads; analog voice scrambling (spectral inversion) operates on PCM — use the descramble tool directly for s16le audio.")
	return res, nil
}

func charProfile(data []byte) (printable, letters float64) {
	if len(data) == 0 {
		return 0, 0
	}
	var p, l int
	for _, b := range data {
		if b >= 0x20 && b <= 0x7e {
			p++
		}
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
			l++
		}
	}
	n := float64(len(data))
	return float64(p) / n, float64(l) / n
}

func failed(r randomness.Report, name string) bool {
	for _, t := range r.Tests {
		if t.Name == name && t.Applicable && !t.Passed {
			return true
		}
	}
	return false
}

func failedNames(r randomness.Report) string {
	var names []string
	for _, t := range r.Tests {
		if t.Applicable && !t.Passed {
			names = append(names, t.Name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func init() { cryptolab.Register(classifyTool{}) }
