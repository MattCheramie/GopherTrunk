package trunking

import "strings"

// EncryptedMode controls how Engine.HandleGrant and the in-call
// encryption handlers treat calls discovered to be encrypted (a P25
// ALGID other than clear, or a DMR/grant Encrypted flag).
//
//   - EncryptedFollow (default): hold a voice SDR for the full duration
//     of an encrypted call, exactly like a clear call. This is the
//     backwards-compatible behaviour — pre-existing configs see no
//     change because it is the zero value.
//   - EncryptedMetadata: follow the call briefly so the traffic channel's
//     metadata (P25 Phase 2 MAC PDUs — talker alias, source RID,
//     encryption sync) can be captured, then release the voice SDR a
//     configurable window (metadata_follow_ms) after the call is known to
//     be encrypted. Frees scarce voice tuners while preserving alias / RID
//     discovery.
//   - EncryptedIgnore: never tie up a voice SDR on an encrypted call —
//     drop the grant before allocation when encryption is known up front,
//     and release the tuner the moment encryption is discovered mid-call.
//
// In every mode a call the operator has a configured decryption key for
// (a trunking.systems[].encryption_keys entry whose key_id matches the
// call) is exempt and always followed — see Engine.keyConfigured.
type EncryptedMode uint8

const (
	EncryptedFollow EncryptedMode = iota
	EncryptedMetadata
	EncryptedIgnore
)

// String renders the wire form used by config + REST + TUI clients.
func (m EncryptedMode) String() string {
	switch m {
	case EncryptedMetadata:
		return "metadata"
	case EncryptedIgnore:
		return "ignore"
	default:
		return "follow"
	}
}

// ParseEncryptedMode is the inverse of String. Empty string maps to the
// safe default (follow = current behaviour); unknown values also map to
// follow so a typo in config never silently stops following calls.
func ParseEncryptedMode(s string) EncryptedMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "metadata":
		return EncryptedMetadata
	case "ignore":
		return EncryptedIgnore
	default:
		return EncryptedFollow
	}
}
