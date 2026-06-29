package cryptolab

import "sort"

// Param describes one input of a mode, enough for a web UI to render a form
// control and for the web server to translate the submitted value into a CLI
// flag. Name is the flag name (without the leading dash), so the same value
// reaches the mode's own flag parser unchanged.
type Param struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"` // file | outfile | string | int | bool | select
	Required bool     `json:"required"`
	Default  string   `json:"default,omitempty"`
	Help     string   `json:"help,omitempty"`
	Options  []string `json:"options,omitempty"`
}

// ModeSchema is the web-facing description of one tool/mode and its inputs.
type ModeSchema struct {
	Tool     string  `json:"tool"`
	Mode     string  `json:"mode"`
	Synopsis string  `json:"synopsis"`
	Params   []Param `json:"params"`
}

// modeParams maps "tool/mode" to its input parameters. It is the single
// source of truth the web UI renders; each Param.Name matches the flag the
// mode defines, so the web server can forward values straight to the mode's
// own flag parser. A schema test asserts every registered mode has an entry.
var modeParams = map[string][]Param{
	"stats/scan": {
		{Name: "in", Label: "Payload file", Kind: "file", Required: true, Help: "binary payload to analyze"},
		{Name: "max-keylen", Label: "Max XOR key length", Kind: "int", Default: "40"},
	},
	"stats/period": {
		{Name: "in", Label: "Payload file", Kind: "file", Required: true, Help: "binary payload to analyze"},
		{Name: "max-lag", Label: "Max autocorrelation lag (bytes)", Kind: "int", Default: "256"},
		{Name: "ngram", Label: "N-gram length", Kind: "int", Default: "3"},
		{Name: "top", Label: "Candidates to report", Kind: "int", Default: "8"},
	},
	"randomness/battery": {
		{Name: "in", Label: "Bitstream file", Kind: "file", Required: true, Help: "packed bytes, MSB-first (e.g. a recovered keystream)"},
		{Name: "alpha", Label: "Significance level", Kind: "string", Default: "0.01"},
	},
	"randomness/quick": {
		{Name: "in", Label: "Bitstream file", Kind: "file", Required: true, Help: "packed bytes, MSB-first"},
		{Name: "alpha", Label: "Significance level", Kind: "string", Default: "0.01"},
	},
	"classify/auto": {
		{Name: "in", Label: "Payload file", Kind: "file", Required: true, Help: "unknown payload to triage"},
		{Name: "max-keylen", Label: "Max XOR key length", Kind: "int", Default: "40"},
	},
	"ks/reuse": {
		{Name: "in", Label: "Frames file", Kind: "file", Required: true, Help: "JSONL {label,iv,ct} or CSV label,iv_hex,ct_hex"},
	},
	"ks/mtp": {
		{Name: "in", Label: "Frames file", Kind: "file", Required: true, Help: "JSONL/CSV frames sharing IVs"},
		{Name: "crib", Label: "Crib", Kind: "string", Default: " the ", Help: "known substring to drag across reuse groups"},
		{Name: "known-label", Label: "Known frame label", Kind: "string", Help: "label of a frame whose plaintext is known"},
		{Name: "known-pt", Label: "Known plaintext file", Kind: "file", Help: "plaintext for the known frame"},
		{Name: "top", Label: "Hits per group", Kind: "int", Default: "10"},
	},
	"ks/extract": {
		{Name: "pt", Label: "Known-plaintext file", Kind: "file", Required: true},
		{Name: "ct", Label: "Ciphertext file", Kind: "file", Required: true},
		{Name: "out", Label: "Output keystream file", Kind: "outfile"},
	},
	"assess/crypto": {
		{Name: "in", Label: "Frames file", Kind: "file", Required: true, Help: "JSONL {label,iv,ct,algid,…} or CSV captured encrypted frames"},
		{Name: "protocol", Label: "Protocol", Kind: "select", Default: "p25", Options: []string{"p25", "tetra", "dmr"}, Help: "algorithm-id namespace for the known-weakness advisory"},
		{Name: "known-label", Label: "Known frame label", Kind: "string", Help: "label of a frame whose plaintext is known (enables verified decryption)"},
		{Name: "known-pt", Label: "Known plaintext file", Kind: "file", Help: "plaintext for the known frame"},
		{Name: "keys", Label: "Candidate keys file", Kind: "file", Help: "one hex key per line for the weak-key method"},
		{Name: "brute-bits", Label: "Brute-force bits", Kind: "int", Default: "0", Help: "reduced-keyspace brute: search the low N key bits against the oracle (0 = off)"},
		{Name: "base-key", Label: "Base key (hex)", Kind: "string", Help: "fixed high key bits for the brute (default all-zero)"},
	},
	"brute/xor": {
		{Name: "in", Label: "Ciphertext file", Kind: "file", Required: true},
		{Name: "keylen", Label: "Key length (0 = auto)", Kind: "int", Default: "0"},
		{Name: "crib", Label: "Known-plaintext crib", Kind: "string", Help: "optional substring to boost scoring"},
	},
	"brute/caesar": {
		{Name: "in", Label: "Ciphertext file", Kind: "file", Required: true},
		{Name: "crib", Label: "Known-plaintext crib", Kind: "string", Help: "optional substring to boost scoring"},
	},
	"brute/vigenere": {
		{Name: "in", Label: "Ciphertext file", Kind: "file", Required: true},
		{Name: "keylen", Label: "Key length (0 = auto)", Kind: "int", Default: "0"},
		{Name: "crib", Label: "Known-plaintext crib", Kind: "string", Help: "optional substring to boost scoring"},
	},
	"brute/substitution": {
		{Name: "in", Label: "Ciphertext file", Kind: "file", Required: true, Help: "English plaintext assumed"},
		{Name: "restarts", Label: "Hill-climb restarts (0 = default)", Kind: "int", Default: "0"},
		{Name: "seed", Label: "RNG seed", Kind: "int", Default: "1"},
	},
	"lfsr/bm": {
		{Name: "in", Label: "Keystream file", Kind: "file", Required: true, Help: "packed bytes, MSB-first"},
	},
	"lfsr/keystream": {
		{Name: "pt", Label: "Known-plaintext file", Kind: "file", Required: true},
		{Name: "ct", Label: "Ciphertext file", Kind: "file", Required: true},
	},
	"crc/recover": {
		{Name: "in", Label: "Samples file", Kind: "file", Required: true, Help: "one `datahex,crchex` per line"},
		{Name: "widths", Label: "Widths to try", Kind: "string", Default: "16", Help: "comma-separated, e.g. 16,8"},
	},
	"crc/compute": {
		{Name: "in", Label: "Input file", Kind: "file", Required: true},
		{Name: "width", Label: "Width (bits)", Kind: "int", Default: "16"},
		{Name: "poly", Label: "Polynomial", Kind: "string", Default: "0x1021"},
		{Name: "init", Label: "Init", Kind: "string", Default: "0"},
		{Name: "xorout", Label: "XorOut", Kind: "string", Default: "0xFFFF"},
		{Name: "refin", Label: "Reflect input", Kind: "bool"},
		{Name: "refout", Label: "Reflect output", Kind: "bool"},
	},
	"descramble/invert": {
		{Name: "in", Label: "Scrambled PCM (s16le mono)", Kind: "file", Required: true},
		{Name: "out", Label: "Output file", Kind: "outfile", Required: true, Default: "descrambled.s16"},
	},
	"descramble/splitband": {
		{Name: "in", Label: "Scrambled PCM (s16le mono)", Kind: "file", Required: true},
		{Name: "out", Label: "Output file", Kind: "outfile", Required: true, Default: "descrambled.s16"},
		{Name: "split", Label: "Split point (fraction of Nyquist)", Kind: "string", Default: "0.5"},
	},
	"descramble/rolling": {
		{Name: "in", Label: "Scrambled PCM (s16le mono)", Kind: "file", Required: true},
		{Name: "out", Label: "Output file", Kind: "outfile", Required: true, Default: "descrambled.s16"},
		{Name: "frame", Label: "Frame length (samples)", Kind: "int", Default: "1024"},
		{Name: "schedule", Label: "Split schedule (CSV) or 'auto'", Kind: "string", Default: "auto"},
	},
	"alias/gauge":     {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
	"alias/structure": {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
	"alias/cells": {
		{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true},
		{Name: "resume", Label: "Resume checkpoint (optional)", Kind: "file"},
	},
	"alias/fromseed": {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
}

// Schema returns the web-facing description of every registered tool/mode,
// sorted by tool then mode. It joins the live registry (so every registered
// mode appears with its synopsis) with the parameter table above.
func Schema() []ModeSchema {
	var out []ModeSchema
	for _, t := range Tools() {
		for _, m := range t.Modes() {
			key := t.Name() + "/" + m.Name()
			out = append(out, ModeSchema{
				Tool:     t.Name(),
				Mode:     m.Name(),
				Synopsis: m.Synopsis(),
				Params:   modeParams[key],
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tool != out[j].Tool {
			return out[i].Tool < out[j].Tool
		}
		return out[i].Mode < out[j].Mode
	})
	return out
}
