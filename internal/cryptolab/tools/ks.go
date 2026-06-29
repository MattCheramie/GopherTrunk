package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/keystream"
)

// ksTool exploits keystream reuse: additive stream ciphers (P25 DES-OFB and
// ADP/RC4, TETRA AIE) produce an identical keystream whenever the same key and
// IV recur, so frames sharing an IV (a P25 Message Indicator) leak
// plaintext^plaintext. This tool finds those collisions and recovers content
// from them — operator-misuse exploitation, not key recovery.
type ksTool struct{}

func (ksTool) Name() string     { return "ks" }
func (ksTool) Synopsis() string { return "keystream-reuse / many-time-pad recovery (IV/MI collisions)" }
func (ksTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{ksReuse{}, ksMTP{}, ksExtract{}}
}

// frameLine parses one corpus line into a Frame. Each non-empty, non-comment
// line is either a JSON object {"label","iv","ct"} (the format the decoder
// bridge emits) or a CSV triple label,iv_hex,ct_hex.
func frameLine(line string) (keystream.Frame, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return keystream.Frame{}, false, nil
	}
	if strings.HasPrefix(line, "{") {
		var rec struct {
			Label string `json:"label"`
			IV    string `json:"iv"`
			CT    string `json:"ct"`
			AlgID uint8  `json:"algid"`
			KeyID uint16 `json:"keyid"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return keystream.Frame{}, false, fmt.Errorf("bad JSON frame: %w", err)
		}
		iv, err := hex.DecodeString(rec.IV)
		if err != nil {
			return keystream.Frame{}, false, fmt.Errorf("frame %q: bad iv hex: %w", rec.Label, err)
		}
		ct, err := hex.DecodeString(rec.CT)
		if err != nil {
			return keystream.Frame{}, false, fmt.Errorf("frame %q: bad ct hex: %w", rec.Label, err)
		}
		return keystream.Frame{Label: rec.Label, IV: iv, CT: ct, AlgID: rec.AlgID, KeyID: rec.KeyID}, true, nil
	}
	parts := strings.Split(line, ",")
	if len(parts) != 3 {
		return keystream.Frame{}, false, fmt.Errorf("expected `label,iv_hex,ct_hex`, got %q", line)
	}
	iv, err := hex.DecodeString(strings.TrimSpace(parts[1]))
	if err != nil {
		return keystream.Frame{}, false, fmt.Errorf("frame %q: bad iv hex: %w", parts[0], err)
	}
	ct, err := hex.DecodeString(strings.TrimSpace(parts[2]))
	if err != nil {
		return keystream.Frame{}, false, fmt.Errorf("frame %q: bad ct hex: %w", parts[0], err)
	}
	return keystream.Frame{Label: strings.TrimSpace(parts[0]), IV: iv, CT: ct}, true, nil
}

func loadFrames(path string) ([]keystream.Frame, error) {
	data, err := readInput(path)
	if err != nil {
		return nil, err
	}
	var frames []keystream.Frame
	for i, line := range strings.Split(string(data), "\n") {
		f, ok, err := frameLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if ok {
			frames = append(frames, f)
		}
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames parsed from %s", path)
	}
	return frames, nil
}

type ksReuse struct{}

func (ksReuse) Name() string     { return "reuse" }
func (ksReuse) Synopsis() string { return "find frames that share an IV/MI (and thus a keystream)" }

func (ksReuse) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("ks reuse")
	in := fs.String("in", "", "frames file (JSONL {label,iv,ct} or CSV label,iv_hex,ct_hex) — required")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	frames, err := loadFrames(*in)
	if err != nil {
		return nil, err
	}
	groups := keystream.FindReuse(frames)

	res := &cryptolab.Result{Tool: "ks", Mode: "reuse",
		Summary: fmt.Sprintf("%d frames: %d IV/MI reuse group(s) found", len(frames), len(groups))}
	res.AddField("frames", len(frames))
	res.AddField("reuse_groups", len(groups))
	for _, g := range groups {
		labels := make([]string, len(g.Frames))
		for i, f := range g.Frames {
			labels[i] = f.Label
		}
		res.AddFinding(fmt.Sprintf("iv=%x", g.IV), float64(len(g.Frames)),
			map[string]any{"frames": len(g.Frames), "labels": labels})
	}
	if len(groups) == 0 {
		res.Note("no IV reuse: every frame uses a distinct IV/MI, so no keystream collision to exploit. This is the secure case.")
	} else {
		res.Note("reused IVs leak plaintext^plaintext — run `cryptolab ks mtp` on this same file to recover content from each group.")
	}
	return res, nil
}

type ksMTP struct{}

func (ksMTP) Name() string { return "mtp" }
func (ksMTP) Synopsis() string {
	return "recover plaintext from a reuse group via known-plaintext or crib-drag"
}

func (ksMTP) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("ks mtp")
	in := fs.String("in", "", "frames file (JSONL/CSV) — required")
	crib := fs.String("crib", " the ", "crib to drag across reuse groups")
	knownLabel := fs.String("known-label", "", "label of a frame whose plaintext is known")
	knownPTFile := fs.String("known-pt", "", "file with the known plaintext for -known-label")
	top := fs.Int("top", 10, "crib-drag hits to report per group")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	frames, err := loadFrames(*in)
	if err != nil {
		return nil, err
	}
	groups := keystream.FindReuse(frames)
	res := &cryptolab.Result{Tool: "ks", Mode: "mtp",
		Summary: fmt.Sprintf("%d frames: %d reuse group(s)", len(frames), len(groups))}
	res.AddField("frames", len(frames))
	res.AddField("reuse_groups", len(groups))
	if len(groups) == 0 {
		res.Note("no IV reuse to exploit.")
		return res, nil
	}

	// Known-plaintext path: the strongest recovery. Derive the keystream from
	// the named frame and decrypt its whole group.
	if *knownLabel != "" {
		if *knownPTFile == "" {
			return nil, fmt.Errorf("-known-label requires -known-pt")
		}
		pt, err := readInput(*knownPTFile)
		if err != nil {
			return nil, err
		}
		recovered := false
		for _, g := range groups {
			ks, decoded, ok := keystream.RecoverWithKnown(g, *knownLabel, pt)
			if !ok {
				continue
			}
			recovered = true
			res.AddField("keystream_hex", hex.EncodeToString(ks))
			for _, d := range decoded {
				res.AddFinding(d.Label, 1.0, map[string]any{
					"plaintext_hex":   hex.EncodeToString(d.Plaintext),
					"plaintext_ascii": printableASCII(d.Plaintext),
					"method":          "known-plaintext",
				})
			}
		}
		if !recovered {
			return nil, fmt.Errorf("frame %q not found in any reuse group", *knownLabel)
		}
		res.Note("recovered the keystream from the known frame and decrypted its reuse group; the same keystream decrypts any future frame that reuses this IV.")
		return res, nil
	}

	// Crib-drag path: no known plaintext, peel content with a crib.
	var anyHit bool
	for _, g := range groups {
		hits := keystream.CribDragGroup(g, []byte(*crib), *top)
		for _, h := range hits {
			anyHit = true
			res.AddFinding(fmt.Sprintf("iv=%x off=%d", g.IV, h.Offset), h.Score, map[string]any{
				"crib_in":    h.CribLabel,
				"reveals_in": h.PartnerLabel,
				"offset":     h.Offset,
				"revealed":   string(h.Revealed),
				"method":     "crib-drag",
			})
		}
	}
	if !anyHit {
		res.Note(fmt.Sprintf("crib %q produced no fully-printable placements; try a different crib (a known call sign, unit id, or common word) or supply -known-label/-known-pt.", *crib))
	} else {
		res.Note("each hit reveals plaintext in the partner frame at that offset; extend the highest-scoring cribs to grow the recovered text.")
	}
	return res, nil
}

type ksExtract struct{}

func (ksExtract) Name() string     { return "extract" }
func (ksExtract) Synopsis() string { return "keystream = plaintext XOR ciphertext (known-plaintext)" }

func (ksExtract) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("ks extract")
	ptPath := fs.String("pt", "", "known-plaintext file — required")
	ctPath := fs.String("ct", "", "ciphertext file — required")
	outPath := fs.String("out", "", "optional file to write the raw keystream")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *ptPath == "" || *ctPath == "" {
		return nil, fmt.Errorf("both -pt and -ct are required")
	}
	pt, err := readInput(*ptPath)
	if err != nil {
		return nil, err
	}
	ct, err := readInput(*ctPath)
	if err != nil {
		return nil, err
	}
	ks := keystream.XOR(pt, ct)
	res := &cryptolab.Result{Tool: "ks", Mode: "extract",
		Summary: fmt.Sprintf("recovered %d keystream bytes", len(ks))}
	res.AddField("keystream_bytes", len(ks))
	res.AddField("keystream_hex", hex.EncodeToString(ks))
	if *outPath != "" {
		if err := os.WriteFile(*outPath, ks, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", *outPath, err)
		}
		res.AddField("written", *outPath)
	}
	res.Note("feed this keystream to `cryptolab randomness battery` or `cryptolab lfsr bm` to test whether it is strong (random) or an exploitable linear generator.")
	return res, nil
}

// printableASCII renders bytes with non-printables shown as '.', for a
// human-skimmable preview alongside the hex.
func printableASCII(b []byte) string {
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

func init() { cryptolab.Register(ksTool{}) }
