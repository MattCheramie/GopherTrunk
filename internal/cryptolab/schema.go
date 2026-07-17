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
	Tool string `json:"tool"`
	Mode string `json:"mode"`
	// Category is the workflow stage the tool serves, so the web console can
	// group modes by what the analyst is trying to DO (triage → attack →
	// verify) instead of an alphabetical tool list. Derived from the tool via
	// toolCategory; unmapped tools fall back to "Other".
	Category string  `json:"category"`
	Synopsis string  `json:"synopsis"`
	Params   []Param `json:"params"`
}

// categoryOrder is the workflow order the console lists categories in — the
// triage → analyze → attack → verify pipeline cryptolab is designed around
// (see doc.go). Schema() sorts by this order so the frontend can group the
// returned list in place.
var categoryOrder = []string{
	"Triage",             // what am I looking at?
	"Statistical",        // characterize the structure
	"Keystream & reuse",  // the operator-misuse attacks
	"Classical ciphers",  // XOR / shift / substitution keyspace brute
	"Voice descrambling", // analog spectral inversion
	"Protocol subjects",  // NXDN / Motorola proprietary obfuscators
	"Pipelines & CRC",    // recipe chaining + CRC parameter recovery
	"Security test",      // the assess harness verdict
}

// toolCategory maps each registered tool to its workflow category. Modes under
// a tool share its category. Keep in sync with the registered tools; an
// unlisted tool is grouped under "Other" (and the schema stays valid — the
// coverage test only requires params, not a category).
var toolCategory = map[string]string{
	"classify":   "Triage",
	"stats":      "Statistical",
	"randomness": "Statistical",
	"ks":         "Keystream & reuse",
	"lfsr":       "Keystream & reuse",
	"brute":      "Classical ciphers",
	"descramble": "Voice descrambling",
	"nxdn":       "Protocol subjects",
	"alias":      "Protocol subjects",
	"recipe":     "Pipelines & CRC",
	"crc":        "Pipelines & CRC",
	"assess":     "Security test",
}

// categoryFor returns a tool's workflow category, defaulting to "Other".
func categoryFor(tool string) string {
	if c, ok := toolCategory[tool]; ok {
		return c
	}
	return "Other"
}

// categoryRank returns a category's position in categoryOrder, sorting any
// unlisted category ("Other") to the end.
func categoryRank(cat string) int {
	for i, c := range categoryOrder {
		if c == cat {
			return i
		}
	}
	return len(categoryOrder)
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
	"recipe/run": {
		{Name: "in", Label: "Input file", Kind: "file", Required: true},
		{Name: "recipe", Label: "Recipe file (JSON/YAML)", Kind: "file", Required: true, Help: "ordered steps: a list of {op, params} or {steps: [...]}"},
		{Name: "out", Label: "Output file", Kind: "outfile", Help: "write the final transformed bytes"},
		{Name: "list", Label: "List operations", Kind: "bool", Help: "list available ops instead of running"},
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
	"nxdn/descramble": {
		{Name: "in", Label: "Scrambled payload", Kind: "file", Required: true, Help: "hex per line (packed bits, MSB-first)"},
		{Name: "key", Label: "Scrambler key (0-32767)", Kind: "int", Required: true},
		{Name: "frame-type", Label: "CRC check (optional)", Kind: "select", Default: "", Options: []string{"", "sacch", "cac", "ccitt"}, Help: "verify a frame CRC after descrambling"},
	},
	"nxdn/brute": {
		{Name: "in", Label: "Scrambled frames", Kind: "file", Required: true, Help: "one scrambled frame per line, hex (packed bits, MSB-first)"},
		{Name: "oracle", Label: "Oracle", Kind: "select", Default: "crc", Options: []string{"crc", "known-pt"}, Help: "how to confirm a key guess"},
		{Name: "frame-type", Label: "Frame type (crc oracle)", Kind: "select", Default: "sacch", Options: []string{"sacch", "cac", "ccitt"}},
		{Name: "known-pt", Label: "Known plaintext (known-pt oracle)", Kind: "file", Help: "hex of known descrambled bits for one frame"},
		{Name: "top", Label: "Max candidate keys to report", Kind: "int", Default: "16"},
	},
	"alias/gauge":     {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
	"alias/structure": {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
	"alias/cells": {
		{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true},
		{Name: "resume", Label: "Resume checkpoint (optional)", Kind: "file"},
	},
	"alias/fromseed":  {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
	"alias/propagate": {{Name: "csv", Label: "Ground-truth corpus CSV", Kind: "file", Required: true}},
	"alias/sweep":     {{Name: "csv", Label: "Chosen-plaintext corpus CSV (no CRC)", Kind: "file", Required: true, Help: "encoded_hex is pure cipher output, exactly 2·len(alias) bytes"}},
}

// Schema returns the web-facing description of every registered tool/mode,
// sorted by workflow category (the triage → attack → verify order), then tool,
// then mode. It joins the live registry (so every registered mode appears with
// its synopsis) with the parameter table above and the workflow category.
func Schema() []ModeSchema {
	var out []ModeSchema
	for _, t := range Tools() {
		for _, m := range t.Modes() {
			key := t.Name() + "/" + m.Name()
			out = append(out, ModeSchema{
				Tool:     t.Name(),
				Mode:     m.Name(),
				Category: categoryFor(t.Name()),
				Synopsis: m.Synopsis(),
				Params:   modeParams[key],
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if ri, rj := categoryRank(out[i].Category), categoryRank(out[j].Category); ri != rj {
			return ri < rj
		}
		if out[i].Tool != out[j].Tool {
			return out[i].Tool < out[j].Tool
		}
		return out[i].Mode < out[j].Mode
	})
	return out
}
