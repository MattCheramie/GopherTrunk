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

	"github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
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

// OpParam describes one parameter of an operation, enough for the web recipe
// builder to render an input control.
type OpParam struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Kind  string `json:"kind"` // hex | int
	Help  string `json:"help,omitempty"`
}

type opDef struct {
	transform bool
	synopsis  string
	params    []OpParam
	fn        opFunc
}

// param spec presets shared by several ops.
var (
	keyParam   = OpParam{Name: "key", Label: "Key (hex)", Kind: "hex"}
	miParam    = OpParam{Name: "mi", Label: "MI / IV (hex)", Kind: "hex"}
	cipherKeys = []OpParam{keyParam, miParam}
)

// ops is the operation registry.
var ops = map[string]opDef{
	"xor":               {transform: true, synopsis: "repeating-key XOR (key=hex)", params: []OpParam{keyParam}, fn: opXOR},
	"not":               {transform: true, synopsis: "bitwise NOT of every byte", fn: opNot},
	"reverse-bits":      {transform: true, synopsis: "reverse the bit order within each byte (MSB↔LSB framing)", fn: opReverseBits},
	"hex-decode":        {transform: true, synopsis: "decode an ASCII hex string to bytes", fn: opHexDecode},
	"hex-encode":        {transform: true, synopsis: "encode bytes as an ASCII hex string", fn: opHexEncode},
	"base64-decode":     {transform: true, synopsis: "decode standard base64 to bytes", fn: opBase64Decode},
	"slice":             {transform: true, synopsis: "keep bytes [offset, offset+length) (length=0 → to end)", params: []OpParam{{Name: "offset", Label: "Offset", Kind: "int"}, {Name: "length", Label: "Length", Kind: "int"}}, fn: opSlice},
	"rc4-decrypt":       {transform: true, synopsis: "raw RC4 decrypt with a caller-supplied key (DMR Enhanced Privacy / generic)", params: []OpParam{{Name: "key", Label: "Full RC4 key (hex)", Kind: "hex", Help: "the complete RC4 key bytes, e.g. privacy key ‖ IV"}}, fn: opRC4},
	"adp-decrypt":       {transform: true, synopsis: "P25 ADP/RC4 decrypt (key=hex 5B, mi=hex)", params: cipherKeys, fn: cipherOp(p25crypto.AlgADP)},
	"des-ofb-decrypt":   {transform: true, synopsis: "P25 DES-OFB decrypt (key=hex 8B, mi=hex)", params: cipherKeys, fn: cipherOp(p25crypto.AlgDESOFB)},
	"tdes-ofb-decrypt":  {transform: true, synopsis: "P25 Triple-DES-OFB decrypt (key=hex 24B, mi=hex)", params: cipherKeys, fn: cipherOp(p25crypto.AlgTDES)},
	"aes-ofb-decrypt":   {transform: true, synopsis: "P25 AES-OFB decrypt (key=hex 16/32B, mi=hex)", params: cipherKeys, fn: cipherOp(p25crypto.AlgAES256)},
	"descramble-invert": {transform: true, synopsis: "full-band spectral inversion of s16le mono PCM (self-inverse)", fn: opDescramble},
	"stats":             {transform: false, synopsis: "report entropy / index-of-coincidence / chi-square", fn: opStats},
	"randomness":        {transform: false, synopsis: "quick NIST randomness subset (strong vs structured)", fn: opRandomness},
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

// OpSpec is the full, web-facing description of one operation.
type OpSpec struct {
	Name      string    `json:"name"`
	Synopsis  string    `json:"synopsis"`
	Transform bool      `json:"transform"`
	Params    []OpParam `json:"params,omitempty"`
}

// Specs returns every operation with its parameters, sorted by name — the
// source the web recipe builder renders its palette and step forms from.
func Specs() []OpSpec {
	out := make([]OpSpec, 0, len(ops))
	for name, def := range ops {
		out = append(out, OpSpec{Name: name, Synopsis: def.synopsis, Transform: def.transform, Params: def.params})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Ops returns the registered operation names and synopses, sorted (CLI -list).
func Ops() []struct {
	Name, Synopsis string
	Transform      bool
} {
	specs := Specs()
	out := make([]struct {
		Name, Synopsis string
		Transform      bool
	}, len(specs))
	for i, s := range specs {
		out[i] = struct {
			Name, Synopsis string
			Transform      bool
		}{s.Name, s.Synopsis, s.Transform}
	}
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

func opRC4(buf []byte, p params) ([]byte, map[string]any, string, error) {
	key, err := p.hex("key")
	if err != nil {
		return nil, nil, "", err
	}
	if len(key) == 0 {
		return nil, nil, "", fmt.Errorf("rc4-decrypt: key is required")
	}
	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, nil, "", fmt.Errorf("rc4-decrypt: %w", err)
	}
	out := make([]byte, len(buf))
	c.XORKeyStream(out, buf)
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
	// Tolerate spaces and a 0x prefix so pasted keys ("11 22 aa" / "0x1122")
	// just work — the same leniency the config builder's key field offers.
	s := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ':' {
			return -1
		}
		return r
	}, p.str(key))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
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
