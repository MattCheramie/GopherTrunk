// Package p25 holds protocol-neutral helpers shared by the P25 Phase 1
// and Phase 2 packages. The TIA-102 Algorithm ID registry lookup lives
// here so log lines, the SSE/REST API and the TUI / web frontend all
// render encryption metadata the same way.
package p25

import "fmt"

// AlgorithmClear is the Algorithm ID a clear (unencrypted) P25 call
// advertises in its Encryption Sync. Mirrors the phase1 constant; kept
// here so callers outside the phase1 package can reference it without
// pulling in the LDU2 parser.
const AlgorithmClear uint8 = 0x80

// AlgorithmName returns the TIA-102 algorithm mnemonic for id, or
// "unknown" when the value is not in the registry. The table covers
// the algorithms currently in use on systems GopherTrunk observers
// monitor; new IDs can be appended as they show up in the wild.
//
// Source: TIA-102.AACE-A "Algorithm IDs" registry.
func AlgorithmName(id uint8) string {
	switch id {
	case 0x80:
		return "CLEAR"
	case 0x81:
		return "DES-OFB"
	case 0x83:
		return "TDES-2"
	case 0x84:
		return "AES-256"
	case 0x85:
		return "AES-128"
	case 0x86:
		return "TDES"
	case 0x89:
		return "AES-256-OFB"
	case 0xAA:
		return "ADP/RC4"
	case 0x9F:
		return "DES-XL"
	}
	return "unknown"
}

// AlgorithmKnown reports whether id is a registered TIA-102 Algorithm ID
// (i.e. AlgorithmName has a mnemonic for it, clear included). It exists so
// the call path can gate crypto metadata before surfacing it: a bit-error
// in a traffic-channel Encryption Sync smears the Algorithm ID roughly
// uniformly across 0x00-0xFF (with a near-distinct Key ID per call), so an
// out-of-set value is provably a mis-decode and, surfaced, is downstream
// indistinguishable from a real key. Callers omit algorithm_id/key_id when
// this returns false rather than emit plausible-looking garbage. The set
// tracks AlgorithmName, so a genuinely new algorithm is admitted the moment
// it is added there. Issues #813, #924.
func AlgorithmKnown(id uint8) bool {
	return AlgorithmName(id) != "unknown"
}

// FormatAlgorithm renders id as "0x84 (AES-256)" for human-readable
// log lines and UI tooltips. Unknown values render as "0xNN (unknown)"
// so an operator can still cross-reference the raw ID with SDRtrunk
// or the TIA registry.
func FormatAlgorithm(id uint8) string {
	return fmt.Sprintf("0x%02X (%s)", id, AlgorithmName(id))
}
