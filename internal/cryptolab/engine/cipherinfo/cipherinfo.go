// Package cipherinfo is a knowledge base of the link-layer encryption
// algorithms used across the trunking protocols GopherTrunk decodes (P25,
// TETRA, DMR) and their published cryptographic weaknesses. The assessment
// harness consults it so it can report an informed verdict even for an
// algorithm whose keystream function it does not bundle: "this is TETRA TEA1,
// whose 80-bit key is reduced to 32 bits by design (CVE-2022-24402) and is
// brute-forceable in minutes" is an actionable finding on its own.
//
// Facts here are drawn from public sources (TIA-102 algorithm IDs, the ETSI
// TETRA standard, and the TETRA:BURST research by Midnight Blue, 2023). The
// package contains no cipher implementations — only the metadata an analyst
// needs to decide which attack to run.
package cipherinfo

import "strings"

// Strength is a coarse, decision-oriented grade.
type Strength string

const (
	// StrengthBroken — a practical full break is published / built in.
	StrengthBroken Strength = "broken"
	// StrengthWeak — a small or reduced keyspace that is feasibly brute-forced.
	StrengthWeak Strength = "weak"
	// StrengthLegacy — aging but not trivially broken (e.g. single DES).
	StrengthLegacy Strength = "legacy"
	// StrengthStrong — no known practical break with a properly managed key.
	StrengthStrong Strength = "strong"
	// StrengthUnknown — clear (unencrypted) or an algorithm we have no data on.
	StrengthUnknown Strength = "unknown"
)

// Info describes one (protocol, algorithm-id) pair.
type Info struct {
	Protocol         string
	AlgID            uint8
	Name             string
	Family           string // RC4, DES, 3DES, AES, TEA, proprietary, clear
	KeyBits          int
	EffectiveKeyBits int // after any documented reduction/backdoor (== KeyBits if none)
	Strength         Strength
	// BruteForceable reports whether EffectiveKeyBits is small enough to search
	// on commodity hardware (≤ ~40 bits).
	BruteForceable bool
	// Bundled reports whether p25crypto can realise this algorithm's keystream,
	// so the active weak-key / brute methods can actually attempt it.
	Bundled   bool
	Weakness  string // one-line summary of the published weakness, "" if none
	Reference string // CVE / paper, "" if none
}

const bruteThresholdBits = 40

// table is the canonical knowledge base, keyed by protocol + algid.
var table = map[string]map[uint8]Info{
	"p25": {
		0x80: {Name: "CLEAR", Family: "clear", Strength: StrengthUnknown, Weakness: "unencrypted"},
		0x81: {Name: "DES-OFB", Family: "DES", KeyBits: 56, EffectiveKeyBits: 56, Strength: StrengthLegacy, Bundled: true,
			Weakness: "56-bit single DES is exhaustively searchable with dedicated hardware/time (EFF Deep Crack, 1998); deprecated.", Reference: "FIPS 46-3 withdrawal"},
		0x83: {Name: "TDES-2", Family: "3DES", KeyBits: 112, EffectiveKeyBits: 112, Strength: StrengthStrong, Bundled: true,
			Weakness: "two-key 3DES has a ~2^112 meet-in-the-middle bound; no practical break, but aging."},
		0x84: {Name: "AES-256", Family: "AES", KeyBits: 256, EffectiveKeyBits: 256, Strength: StrengthStrong, Bundled: true},
		0x85: {Name: "AES-128", Family: "AES", KeyBits: 128, EffectiveKeyBits: 128, Strength: StrengthStrong, Bundled: true},
		0x86: {Name: "TDES", Family: "3DES", KeyBits: 168, EffectiveKeyBits: 112, Strength: StrengthStrong, Bundled: true,
			Weakness: "three-key 3DES has ~112-bit effective security; no practical break."},
		0x89: {Name: "AES-256-OFB", Family: "AES", KeyBits: 256, EffectiveKeyBits: 256, Strength: StrengthStrong, Bundled: true},
		0x9F: {Name: "DES-XL", Family: "proprietary", KeyBits: 56, EffectiveKeyBits: 56, Strength: StrengthLegacy,
			Weakness: "Motorola DES variant with ~56-bit key strength; proprietary keystream not bundled."},
		0xAA: {Name: "ADP/RC4", Family: "RC4", KeyBits: 40, EffectiveKeyBits: 40, Strength: StrengthWeak, BruteForceable: true, Bundled: true,
			Weakness: "40-bit key over RC4: a small, brute-forceable keyspace, compounded by RC4's keystream biases. Often deployed with default keys.", Reference: "ADP 40-bit keyspace"},
	},
	"tetra": {
		0x01: {Name: "TEA1", Family: "TEA", KeyBits: 80, EffectiveKeyBits: 32, Strength: StrengthBroken, BruteForceable: true,
			Weakness: "the 80-bit key is compressed to 32 bits before keystream generation — a deliberate backdoor, brute-forceable on a laptop in minutes. Keystream function is proprietary (not bundled); supply an implementation to run the 2^32 brute.", Reference: "CVE-2022-24402 / TETRA:BURST (Midnight Blue, 2023)"},
		0x02: {Name: "TEA2", Family: "TEA", KeyBits: 80, EffectiveKeyBits: 80, Strength: StrengthStrong,
			Weakness: "restricted-distribution algorithm; no public full break, but proprietary and unauditable."},
		0x03: {Name: "TEA3", Family: "TEA", KeyBits: 80, EffectiveKeyBits: 80, Strength: StrengthLegacy,
			Weakness: "export-grade; TETRA:BURST found a non-uniformity in the S-box but no full key recovery.", Reference: "TETRA:BURST (Midnight Blue, 2023)"},
		0x04: {Name: "TEA4", Family: "TEA", KeyBits: 80, EffectiveKeyBits: 80, Strength: StrengthLegacy,
			Weakness: "export-grade variant; proprietary and unauditable."},
	},
	"dmr": {
		// DMRA / vendor privacy types as surfaced by common decoders.
		0x00: {Name: "Basic Privacy", Family: "proprietary", KeyBits: 8, EffectiveKeyBits: 8, Strength: StrengthBroken, BruteForceable: true,
			Weakness: "DMRA 'Basic Privacy' is a short keyed scrambler with a tiny keyspace — trivially recovered; not real encryption."},
		0x21: {Name: "RC4 (Enhanced Privacy)", Family: "RC4", KeyBits: 40, EffectiveKeyBits: 40, Strength: StrengthWeak, BruteForceable: true,
			Weakness: "vendor 'Enhanced Privacy' RC4 with a short key; small keyspace plus RC4 biases. Keystream construction is vendor-specific (not bundled)."},
		0x22: {Name: "DES-OFB", Family: "DES", KeyBits: 56, EffectiveKeyBits: 56, Strength: StrengthLegacy,
			Weakness: "single DES, exhaustively searchable; vendor key construction not bundled."},
		0x23: {Name: "Triple-DES", Family: "3DES", KeyBits: 168, EffectiveKeyBits: 112, Strength: StrengthStrong},
		0x24: {Name: "AES-128", Family: "AES", KeyBits: 128, EffectiveKeyBits: 128, Strength: StrengthStrong},
		0x25: {Name: "AES-256", Family: "AES", KeyBits: 256, EffectiveKeyBits: 256, Strength: StrengthStrong},
	},
}

// Lookup returns the Info for a (protocol, algid) pair. protocol is matched
// case-insensitively; a few common aliases map onto the canonical key.
func Lookup(protocol string, algid uint8) (Info, bool) {
	p := canonicalProtocol(protocol)
	m, ok := table[p]
	if !ok {
		return Info{}, false
	}
	info, ok := m[algid]
	if !ok {
		return Info{}, false
	}
	info.Protocol = p
	info.AlgID = algid
	if info.EffectiveKeyBits == 0 {
		info.EffectiveKeyBits = info.KeyBits
	}
	info.BruteForceable = info.BruteForceable || (info.EffectiveKeyBits > 0 && info.EffectiveKeyBits <= bruteThresholdBits)
	return info, true
}

// Protocols returns the protocol keys the knowledge base covers.
func Protocols() []string {
	out := make([]string, 0, len(table))
	for p := range table {
		out = append(out, p)
	}
	return out
}

func canonicalProtocol(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "", "p25", "p25-phase1", "p25-phase2", "apco25", "apco-25":
		return "p25"
	case "tetra":
		return "tetra"
	case "dmr", "dmr-tier2", "dmr-tier3", "dmr-tier1":
		return "dmr"
	default:
		return strings.ToLower(strings.TrimSpace(p))
	}
}
