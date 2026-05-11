package trunking

import "strings"

// ScanMode controls how Engine.HandleGrant filters incoming grants
// against the talkgroup database's `Scan` flag.
//
//   - ScanModeAll  (default): every non-locked-out grant is dispatched,
//     regardless of TalkGroup.Scan. This is the backwards-compatible
//     behavior — pre-scanner configs see no change.
//   - ScanModeList: only grants whose talkgroup carries Scan==true (or
//     whose grant is flagged Emergency, parallel to the Lockout
//     exception) are dispatched. Unknown talkgroup IDs are dropped
//     because there's no way to know they're scannable.
type ScanMode uint8

const (
	ScanModeAll ScanMode = iota
	ScanModeList
)

// String renders the wire form used by config + REST + TUI clients.
func (m ScanMode) String() string {
	switch m {
	case ScanModeList:
		return "list"
	default:
		return "all"
	}
}

// ParseScanMode is the inverse of String. Empty string maps to the
// safe default (all); unknown values also map to all so a typo in
// config doesn't accidentally silence the daemon.
func ParseScanMode(s string) ScanMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "list":
		return ScanModeList
	default:
		return ScanModeAll
	}
}
