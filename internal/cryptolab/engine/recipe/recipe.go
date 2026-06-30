// Package recipe is a CyberChef-style operation pipeline for RF payloads. A
// recipe is an ordered list of steps; each step is either a TRANSFORM (it
// rewrites the working buffer — XOR, a cipher decrypt, a bit reversal, a
// spectral inversion) or an ANALYSIS (it measures the current buffer — entropy,
// randomness — without changing it). The bytes flow from one step to the next,
// so an analyst can encode the de-obfuscation sequence they repeat by hand
// (e.g. hex-decode → xor → stats → randomness) as a single reusable artifact.
//
// The transform ops reuse the toolkit's existing engines (p25crypto for the
// real ADP/DES/3DES/AES ciphers, voice for spectral inversion, stats and
// randomness for the analyses), so the pipeline composes the same primitives
// the standalone tools expose.
package recipe

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/randomness"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/voice"
)

// Step is one recipe operation: an op name plus its parameters.
type Step struct {
	Op     string
	Params map[string]any
}

// StepResult records what one step did.
type StepResult struct {
	Op        string         `json:"op"`
	Transform bool           `json:"transform"`
	BytesIn   int            `json:"bytes_in"`
	BytesOut  int            `json:"bytes_out"`
	Info      map[string]any `json:"info,omitempty"`
	Note      string         `json:"note,omitempty"`
}

// Report is the pipeline outcome.
type Report struct {
	InputBytes int          `json:"input_bytes"`
	Steps      []StepResult `json:"steps"`
	FinalBytes []byte       `json:"-"`
	FinalHex   string       `json:"final_hex"`
	FinalASCII string       `json:"final_ascii"`
}

type opFunc func(buf []byte, p params) ([]byte, map[string]any, string, error)

type opDef struct {
	transform bool
	synopsis  string
	fn        opFunc
}

// ops is the operation registry.
var ops = map[string]opDef{
	"xor":               {true, "repeating-key XOR (key=hex)", opXOR},
	"not":               {true, "bitwise NOT of every byte", opNot},
	"reverse-bits":      {true, "reverse the bit order within each byte (MSB↔LSB framing)", opReverseBits},
	"hex-decode":        {true, "decode an ASCII hex string to bytes", opHexDecode},
	"hex-encode":        {true, "encode bytes as an ASCII hex string", opHexEncode},
	"base64-decode":     {true, "decode standard base64 to bytes", opBase64Decode},
	"slice":             {true, "keep bytes [offset, offset+length) (length=0 → to end)", opSlice},
	"adp-decrypt":       {true, "P25 ADP/RC4 decrypt (key=hex 5B, mi=hex)", cipherOp(p25crypto.AlgADP)},
	"des-ofb-decrypt":   {true, "P25 DES-OFB decrypt (key=hex 8B, mi=hex)", cipherOp(p25crypto.AlgDESOFB)},
	"tdes-ofb-decrypt":  {true, "P25 Triple-DES-OFB decrypt (key=hex 24B, mi=hex)", cipherOp(p25crypto.AlgTDES)},
	"aes-ofb-decrypt":   {true, "P25 AES-OFB decrypt (key=hex 16/32B, mi=hex)", cipherOp(p25crypto.AlgAES256)},
	"descramble-invert": {true, "full-band spectral inversion of s16le mono PCM (self-inverse)", opDescramble},
	"stats":             {false, "report entropy / index-of-coincidence / chi-square", opStats},
	"randomness":        {false, "quick NIST randomness subset (strong vs structured)", opRandomness},
}

// Run executes the steps over input, threading the working buffer through the
// transforms. A step error aborts the pipeline and is returned with the
// partial report.
func Run(input []byte, steps []Step) (*Report, error) {
	rep := &Report{InputBytes: len(input)}
	buf := append([]byte(nil), input...)
	for i, s := range steps {
		def, ok := ops[s.Op]
		if !ok {
			return rep, fmt.Errorf("step %d: unknown op %q (see `cryptolab recipe ops`)", i+1, s.Op)
		}
		out, info, note, err := def.fn(buf, params(s.Params))
		if err != nil {
			return rep, fmt.Errorf("step %d (%s): %w", i+1, s.Op, err)
		}
		sr := StepResult{Op: s.Op, Transform: def.transform, BytesIn: len(buf), Info: info, Note: note}
		if def.transform {
			buf = out
		}
		sr.BytesOut = len(buf)
		rep.Steps = append(rep.Steps, sr)
	}
	rep.FinalBytes = buf
	rep.FinalHex = hexPreview(buf, 64)
	rep.FinalASCII = asciiPreview(buf, 64)
	return rep, nil
}

// Ops returns the registered operation names and synopses, sorted.
func Ops() []struct {
	Name, Synopsis string
	Transform      bool
} {
	var out []struct {
		Name, Synopsis string
		Transform      bool
	}
	for name, def := range ops {
		out = append(out, struct {
			Name, Synopsis string
			Transform      bool
		}{name, def.synopsis, def.transform})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// --- transform ops ---

func opXOR(buf []byte, p params) ([]byte, map[string]any, string, error) {
	key, err := p.hex("key")
	if err != nil {
		return nil, nil, "", err
	}
	if len(key) == 0 {
		return nil, nil, "", fmt.Errorf("xor: key is required")
	}
	out := make([]byte, len(buf))
	for i := range buf {
		out[i] = buf[i] ^ key[i%len(key)]
	}
	return out, map[string]any{"key_len": len(key)}, "", nil
}

func opNot(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = ^b
	}
	return out, nil, "", nil
}

func opReverseBits(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = bits.Reverse8(b)
	}
	return out, nil, "", nil
}

func opHexDecode(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	s := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, string(buf))
	out, err := hex.DecodeString(s)
	if err != nil {
		return nil, nil, "", fmt.Errorf("hex-decode: %w", err)
	}
	return out, nil, "", nil
}

func opHexEncode(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	return []byte(hex.EncodeToString(buf)), nil, "", nil
}

func opBase64Decode(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	out, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(buf)))
	if err != nil {
		return nil, nil, "", fmt.Errorf("base64-decode: %w", err)
	}
	return out, nil, "", nil
}

func opSlice(buf []byte, p params) ([]byte, map[string]any, string, error) {
	off := p.intDefault("offset", 0)
	length := p.intDefault("length", 0)
	if off < 0 || off > len(buf) {
		return nil, nil, "", fmt.Errorf("slice: offset %d out of range (len %d)", off, len(buf))
	}
	end := len(buf)
	if length > 0 && off+length < end {
		end = off + length
	}
	return append([]byte(nil), buf[off:end]...), map[string]any{"offset": off, "length": end - off}, "", nil
}

// cipherOp builds a decrypt transform for a P25 algorithm: keystream =
// Keystream(alg, key, mi, len(buf)); out = buf ⊕ keystream.
func cipherOp(alg uint8) opFunc {
	return func(buf []byte, p params) ([]byte, map[string]any, string, error) {
		key, err := p.hex("key")
		if err != nil {
			return nil, nil, "", err
		}
		mi, err := p.hex("mi")
		if err != nil {
			return nil, nil, "", err
		}
		ks, err := p25crypto.Keystream(alg, key, mi, len(buf))
		if err != nil {
			return nil, nil, "", err
		}
		out := make([]byte, len(buf))
		for i := range buf {
			out[i] = buf[i] ^ ks[i]
		}
		return out, map[string]any{"algorithm": p25crypto.AlgName(alg)}, "", nil
	}
}

func opDescramble(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	if len(buf)%2 != 0 {
		return nil, nil, "", fmt.Errorf("descramble-invert: need s16le PCM (even byte count), got %d", len(buf))
	}
	samples := make([]int16, len(buf)/2)
	for i := range samples {
		samples[i] = int16(uint16(buf[2*i]) | uint16(buf[2*i+1])<<8)
	}
	inv := voice.SpectralInvertInt16(samples)
	out := make([]byte, len(inv)*2)
	for i, s := range inv {
		out[2*i] = byte(uint16(s))
		out[2*i+1] = byte(uint16(s) >> 8)
	}
	return out, map[string]any{"samples": len(samples)}, "", nil
}

// --- analysis ops (buffer unchanged) ---

func opStats(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	ent := stats.ShannonEntropy(buf)
	info := map[string]any{
		"entropy_bits":         ent,
		"index_of_coincidence": stats.IndexOfCoincidence(buf),
		"chi_square_uniform":   stats.ChiSquareUniform(buf),
	}
	note := "intermediate structure"
	switch {
	case ent > 7.5:
		note = "high entropy — looks encrypted/compressed"
	case ent < 4.8:
		note = "low entropy — looks like plaintext/structured data"
	}
	return buf, info, note, nil
}

func opRandomness(buf []byte, _ params) ([]byte, map[string]any, string, error) {
	rep := randomness.Quick(lfsr.BitsFromBytes(buf), randomness.DefaultAlpha)
	note := "structured (fails randomness) — exploitable"
	if rep.LooksRandom() {
		note = "indistinguishable from random — strong"
	}
	return buf, map[string]any{"passed": rep.Passed, "failed": rep.Failed, "looks_random": rep.LooksRandom()}, note, nil
}

// --- params helpers ---

type params map[string]any

func (p params) str(key string) string {
	if p == nil {
		return ""
	}
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (p params) hex(key string) ([]byte, error) {
	s := strings.TrimSpace(p.str(key))
	if s == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("param %q: bad hex %q: %w", key, s, err)
	}
	return b, nil
}

func (p params) intDefault(key string, def int) int {
	if p == nil {
		return def
	}
	switch v := p[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return def
}

func hexPreview(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return hex.EncodeToString(b)
}

func asciiPreview(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
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
