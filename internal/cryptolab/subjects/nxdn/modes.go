package nxdn

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	radionxdn "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
)

// loadFrames reads scrambled frames: one frame per non-blank, non-"#" line as
// hex (packed bits, MSB-first). A file with a single line is one frame. Each
// frame is returned as a bit slice (0/1, MSB-first) so the scrambler and the
// bit-oriented CRC checks can operate on it directly.
func loadFrames(path string) ([][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var frames [][]byte
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		raw, err := hex.DecodeString(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad hex: %w", i+1, err)
		}
		frames = append(frames, framing.UnpackBitsMSB(raw, 8*len(raw)))
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames parsed from %s", path)
	}
	return frames, nil
}

// oracle decides whether a descrambled frame looks like valid plaintext, which
// is how a brute-force key guess is confirmed. The interface is deliberately
// small so the planned AMBE/voice-FEC oracle (roadmap phase 2) can slot in
// without reshaping the sweep.
type oracle interface {
	name() string
	// check reports a score in [0,1] for ranking (1 = strongest) and whether
	// the frame passes outright.
	check(bits []byte) (score float64, pass bool)
}

// crcOracle confirms a key by re-checking an NXDN frame CRC after descrambling.
// It reuses the production CRC routines so the oracle matches the real decoder.
type crcOracle struct{ frameType string }

func (o crcOracle) name() string { return "crc:" + o.frameType }

func (o crcOracle) check(bits []byte) (float64, bool) {
	switch o.frameType {
	case "sacch": // 32-bit SACCH info block: 26 payload + CRC-6 trailer.
		if len(bits) != radionxdn.SACCHInfoBits {
			return 0, false
		}
		if radionxdn.VerifySACCHCRC6(bits) {
			return 1, true
		}
	case "cac": // 88-bit CAC message: type(8) + payload(64) + CRC-CCITT(16).
		if len(bits) != 88 {
			return 0, false
		}
		if _, err := radionxdn.ParseCAC(framing.PackBitsMSB(bits)); err == nil {
			return 1, true
		}
	case "ccitt": // generic: whole bytes, last two are CRC-CCITT/FALSE over the rest.
		b := framing.PackBitsMSB(bits)
		if len(b) < 3 {
			return 0, false
		}
		if framing.CRCCCITT(b[:len(b)-2]) == binary.BigEndian.Uint16(b[len(b)-2:]) {
			return 1, true
		}
	}
	return 0, false
}

// knownPtOracle confirms a key when a snippet of known descrambled content is
// available — it ranks by the fraction of leading bits that match and passes
// only on an exact match. Works for voice or data, unlike the CRC oracle.
type knownPtOracle struct{ want []byte }

func (knownPtOracle) name() string { return "known-pt" }

func (o knownPtOracle) check(bits []byte) (float64, bool) {
	n := len(o.want)
	if n == 0 || len(bits) < n {
		return 0, false
	}
	match := 0
	for i := 0; i < n; i++ {
		if bits[i]&1 == o.want[i]&1 {
			match++
		}
	}
	return float64(match) / float64(n), match == n
}

// ---- Mode: descramble ----

type descrambleMode struct{}

func (descrambleMode) Name() string { return "descramble" }
func (descrambleMode) Synopsis() string {
	return "descramble an NXDN information field with a known scrambler key"
}

func (descrambleMode) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := flag.NewFlagSet("cryptolab nxdn descramble", flag.ContinueOnError)
	in := fs.String("in", "", "scrambled payload file: hex per line (packed bits, MSB-first) — required")
	key := fs.Int("key", -1, "scrambler key 0..32767 — required")
	frameType := fs.String("frame-type", "", "optional CRC check after descramble: sacch|cac|ccitt")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *in == "" {
		return nil, fmt.Errorf("nxdn descramble: -in is required")
	}
	if *key < 0 || *key > radionxdn.ScramblerKeyMax {
		return nil, fmt.Errorf("nxdn descramble: -key must be 0..%d", radionxdn.ScramblerKeyMax)
	}
	frames, err := loadFrames(*in)
	if err != nil {
		return nil, err
	}

	var orc oracle
	if *frameType != "" {
		orc = crcOracle{frameType: *frameType}
	}
	res := &cryptolab.Result{Tool: "nxdn", Mode: "descramble"}
	res.AddField("key", *key)
	res.AddField("frames", len(frames))
	okCount := 0
	for i, f := range frames {
		plain := radionxdn.Descramble(f, uint16(*key))
		detail := map[string]any{
			"hex":  hex.EncodeToString(framing.PackBitsMSB(plain)),
			"bits": len(plain),
		}
		score := 0.0
		if orc != nil {
			if _, pass := orc.check(plain); pass {
				okCount++
				score = 1
				detail["crc_ok"] = true
			} else {
				detail["crc_ok"] = false
			}
		}
		res.AddFinding(fmt.Sprintf("frame[%d]", i), score, detail)
	}
	if orc != nil {
		res.AddField("crc_ok_frames", okCount)
		res.Summary = fmt.Sprintf("descrambled %d frame(s) with key %d; %d/%d pass %s CRC",
			len(frames), *key, okCount, len(frames), *frameType)
	} else {
		res.Summary = fmt.Sprintf("descrambled %d frame(s) with key %d", len(frames), *key)
	}
	return res, nil
}

// ---- Mode: brute ----

type bruteMode struct{}

func (bruteMode) Name() string { return "brute" }
func (bruteMode) Synopsis() string {
	return "brute-force the 15-bit NXDN scrambler key against a CRC or known-plaintext oracle"
}

func (bruteMode) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := flag.NewFlagSet("cryptolab nxdn brute", flag.ContinueOnError)
	in := fs.String("in", "", "scrambled frames file: one frame per line, hex (packed bits, MSB-first) — required")
	oracleName := fs.String("oracle", "crc", "key-confirmation oracle: crc|known-pt")
	frameType := fs.String("frame-type", "sacch", "CRC frame type for the crc oracle: sacch|cac|ccitt")
	knownPt := fs.String("known-pt", "", "known-plaintext file (hex, packed bits) for the known-pt oracle")
	top := fs.Int("top", 16, "max candidate keys to report")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *in == "" {
		return nil, fmt.Errorf("nxdn brute: -in is required")
	}
	frames, err := loadFrames(*in)
	if err != nil {
		return nil, err
	}

	var orc oracle
	switch *oracleName {
	case "crc":
		switch *frameType {
		case "sacch", "cac", "ccitt":
		default:
			return nil, fmt.Errorf("nxdn brute: -frame-type must be sacch|cac|ccitt, got %q", *frameType)
		}
		orc = crcOracle{frameType: *frameType}
	case "known-pt":
		if *knownPt == "" {
			return nil, fmt.Errorf("nxdn brute: -known-pt file is required for the known-pt oracle")
		}
		kp, err := loadFrames(*knownPt)
		if err != nil {
			return nil, fmt.Errorf("nxdn brute: known-pt: %w", err)
		}
		orc = knownPtOracle{want: kp[0]}
	case "ambe-fec":
		return nil, fmt.Errorf("nxdn brute: the ambe-fec oracle needs the NXDN voice decoder (roadmap phase 2); use crc or known-pt")
	default:
		return nil, fmt.Errorf("nxdn brute: -oracle must be crc|known-pt, got %q", *oracleName)
	}

	type candidate struct {
		key   uint16
		score float64
	}
	var candidates []candidate
	// The whole point: the key space is only 2^15, so an exhaustive sweep is
	// instantaneous. A key is a candidate only if it passes the oracle on every
	// supplied frame — the true key passes them all while false positives from a
	// short CRC drop out as more frames are added.
	for k := 0; k <= radionxdn.ScramblerKeyMax; k++ {
		passAll := true
		total := 0.0
		for _, f := range frames {
			s, pass := orc.check(radionxdn.Descramble(f, uint16(k)))
			if !pass {
				passAll = false
				break
			}
			total += s
		}
		if passAll {
			candidates = append(candidates, candidate{key: uint16(k), score: total / float64(len(frames))})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].key < candidates[j].key
	})

	res := &cryptolab.Result{Tool: "nxdn", Mode: "brute"}
	res.AddField("keyspace", radionxdn.ScramblerKeyMax+1)
	res.AddField("frames", len(frames))
	res.AddField("oracle", orc.name())
	res.AddField("candidates", len(candidates))
	for i, c := range candidates {
		if i >= *top {
			break
		}
		res.AddFinding(fmt.Sprintf("key=0x%04x (%d)", c.key, c.key), c.score, map[string]any{
			"key":     int(c.key),
			"key_hex": fmt.Sprintf("0x%04x", c.key),
		})
	}

	switch len(candidates) {
	case 0:
		res.Summary = "no scrambler key satisfied the oracle"
		res.Note("No key passed. Check that -frame-type matches the payload and that the data is actually scrambled. Note the scrambler model is synthetic (not yet hardware-confirmed) — a known-key capture will settle whether the polynomial/seed mapping needs adjusting.")
	case 1:
		res.Summary = fmt.Sprintf("recovered NXDN scrambler key %d (0x%04x)", candidates[0].key, candidates[0].key)
		env.Logf("nxdn scrambler key recovered", "key", int(candidates[0].key), "frames", len(frames), "oracle", orc.name())
	default:
		res.Summary = fmt.Sprintf("%d CRC-consistent candidate keys (a linear coset); a non-linear oracle is needed to pin one", len(candidates))
		res.Note("These keys are CRC-indistinguishable. The scrambler LFSR and the CRC are both linear, so the keys that satisfy the CRC form a fixed coset of size ~2^(15-crcbits) (e.g. ~2^9 for SACCH's 6-bit CRC) that is the same for every same-type frame — adding more same-type frames does NOT shrink it. Disambiguate with a known-plaintext crib (-oracle known-pt), a mix of different frame types, or — for voice, the way AOR does it — the AMBE/FEC oracle (roadmap phase 2). The true key is always in this set.")
	}
	return res, nil
}
