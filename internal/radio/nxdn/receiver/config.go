package receiver

import "strings"

// ParseSoftDecision maps a config / user-facing string into the
// Options.SoftDecision boolean, mirroring the P25 Phase 2 receiver's
// parser. Recognised values (case-insensitive): "" / "off" / "false" / "0"
// → false (the default — the hard slicer, byte-for-byte the historical
// behaviour); "on" / "true" / "1" → true (derive per-bit LLRs from the
// 4-level soft symbols and run the CAC decode through the true per-bit soft
// Viterbi, recovering the coding gain the hard slicer discards). Unknown
// strings return false with `ok = false` so callers can warn and fall back.
func ParseSoftDecision(s string) (on bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return false, true
	case "on", "true", "1":
		return true, true
	default:
		return false, false
	}
}
