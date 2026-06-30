package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/assess"
)

// assessTool is the cryptographic security-test harness. It takes captured
// encrypted frames and actively attempts decryption by every applicable
// method, reporting each method's effectiveness — from 0% (the encryption
// held) up to 100% (complete decryption, i.e. the cipher failed the test) —
// plus any recovered data. This is the security test itself: trying to break
// the encryption and grading how far each method got.
type assessTool struct{}

func (assessTool) Name() string { return "assess" }
func (assessTool) Synopsis() string {
	return "security test: attempt decryption by every applicable method and grade each"
}
func (assessTool) Modes() []cryptolab.Mode { return []cryptolab.Mode{assessCrypto{}} }

type assessCrypto struct{}

func (assessCrypto) Name() string { return "crypto" }
func (assessCrypto) Synopsis() string {
	return "run all applicable attacks against captured encrypted frames and report effectiveness"
}

func (assessCrypto) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("assess crypto")
	in := fs.String("in", "", "frames file (JSONL {label,iv,ct,algid,…} or CSV) — required")
	protocol := fs.String("protocol", "p25", "protocol whose algorithm-id namespace applies: p25|tetra|dmr")
	knownLabel := fs.String("known-label", "", "label of a frame whose plaintext is known (enables verified decryption)")
	knownPTFile := fs.String("known-pt", "", "file with the known plaintext for -known-label")
	keysFile := fs.String("keys", "", "optional file of candidate keys (one hex key per line) for the weak-key method")
	bruteBits := fs.Int("brute-bits", 0, "reduced-keyspace brute: search the low N bits of the key against the known-plaintext oracle (0 = off)")
	baseKeyHex := fs.String("base-key", "", "fixed high bits of the key for -brute-bits, as hex (default all-zero)")
	externCmd := fs.String("extern-cmd", "", "external cipher program for an unbundled cipher (e.g. a TETRA TEA1 tool); used with -brute-bits to brute it natively")
	externAlgHex := fs.String("extern-algid", "", "algid the -extern-cmd cipher handles, as hex (e.g. 0x01 for tetra TEA1)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	frames, err := loadFrames(*in)
	if err != nil {
		return nil, err
	}
	input := assess.Input{Frames: frames, Protocol: *protocol, KnownLabel: *knownLabel, BruteBits: *bruteBits, ExternCmd: *externCmd}
	if *externAlgHex != "" {
		alg, err := parseAlgID(*externAlgHex)
		if err != nil {
			return nil, fmt.Errorf("-extern-algid: %w", err)
		}
		input.ExternAlgID = alg
	}
	if *baseKeyHex != "" {
		bk, err := hex.DecodeString(*baseKeyHex)
		if err != nil {
			return nil, fmt.Errorf("-base-key: bad hex: %w", err)
		}
		input.BaseKey = bk
	}
	if *knownPTFile != "" {
		pt, err := readInput(*knownPTFile)
		if err != nil {
			return nil, err
		}
		input.KnownPT = pt
	}
	if *knownLabel != "" && len(input.KnownPT) == 0 {
		return nil, fmt.Errorf("-known-label requires -known-pt")
	}
	if *keysFile != "" {
		keys, err := loadHexKeys(*keysFile)
		if err != nil {
			return nil, err
		}
		input.ExtraKeys = keys
	}

	rep := assess.Run(input)

	// Name the strongest applicable method for the headline.
	topName, topEff := "none", 0.0
	for _, m := range rep.Methods {
		if m.Applicable && m.Effectiveness >= topEff {
			topName, topEff = m.Name, m.Effectiveness
		}
	}

	res := &cryptolab.Result{Tool: "assess", Mode: "crypto",
		Summary: fmt.Sprintf("%s — strongest method %q at %.0f%% over %d ciphertext bytes across %d frames",
			rep.Verdict, topName, topEff*100, rep.TotalCipherBytes, rep.Frames)}
	res.AddField("frames", rep.Frames)
	res.AddField("total_cipher_bytes", rep.TotalCipherBytes)
	res.AddField("verdict", rep.Verdict)
	res.AddField("overall_effectiveness", rep.OverallEffectiveness)

	// One finding per method, scored by effectiveness so the most successful
	// attack sorts to the top.
	for _, mth := range rep.Methods {
		detail := map[string]any{
			"applicable":      mth.Applicable,
			"verified":        mth.Verified,
			"effectiveness":   mth.Effectiveness,
			"recovered_bytes": mth.RecoveredBytes,
		}
		if mth.SampleASCII != "" {
			detail["sample_ascii"] = mth.SampleASCII
			detail["sample_hex"] = mth.SampleHex
		}
		if len(mth.Notes) > 0 {
			detail["notes"] = strings.Join(mth.Notes, " ")
		}
		for k, v := range mth.Detail {
			detail[k] = v
		}
		res.AddFinding(mth.Name, mth.Effectiveness, detail)
	}

	switch rep.Verdict {
	case assess.VerdictBroken:
		res.Note("BROKEN: a method achieved verified complete decryption — the encryption FAILED this test. See the top finding for the method and recovered plaintext.")
	case assess.VerdictPartial:
		res.Note("PARTIAL: no full break, but information leaked (IV reuse, structured ciphertext, or a recovered keystream segment). Review each method to see what is exposed and harden accordingly.")
	default:
		res.Note("RESISTANT: no applicable method recovered plaintext — the encryption held against every attack tried here. Note this reflects only the methods and data provided.")
	}
	res.Note("effectiveness is the fraction of captured ciphertext each method recovered or exposed; 100% from a verified method is a complete break.")
	return res, nil
}

// parseAlgID parses an algorithm id given as hex (0x01), decimal, or bare hex.
func parseAlgID(s string) (uint8, error) {
	s = strings.TrimSpace(s)
	base := 0 // let ParseUint infer 0x / decimal
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		// A bare token like "AA" is hex by convention for algids.
		base = 16
	}
	v, err := strconv.ParseUint(s, base, 8)
	if err != nil {
		return 0, fmt.Errorf("bad algid %q: %w", s, err)
	}
	return uint8(v), nil
}

// loadHexKeys reads a candidate-key file: one hex-encoded key per line, blank
// lines and #-comments ignored.
func loadHexKeys(path string) ([][]byte, error) {
	data, err := readInput(path)
	if err != nil {
		return nil, err
	}
	var keys [][]byte
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, err := hex.DecodeString(line)
		if err != nil {
			return nil, fmt.Errorf("keys line %d: bad hex: %w", i+1, err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func init() { cryptolab.Register(assessTool{}) }
