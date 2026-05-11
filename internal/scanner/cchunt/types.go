// Package cchunt is the multi-system control-channel scanner.
//
// The supervisor owns one control SDR and multiplexes
// `trunking.Hunter` runs across every configured trunked system. On
// success a system parks until cc.lost; on failure the supervisor
// exponentially backs off, publishes KindHuntFailed, and advances to
// the next system. Operators can hold a system on its current lock,
// resume the loop, or force a re-hunt — these mutation surfaces are
// exposed through the API cockpit and the TUI Scanner panel.
//
// This package is the orchestration layer only. It does not produce
// `cc.locked` events itself — those must come from the IQ-domain
// protocol decoders (P25 / DMR / NXDN / YSF / ...). Without those
// upstream feeders the supervisor will always exhaust the candidate
// list and report failed; that's intentional, not a bug. See the
// README "Roadmap" / "Known gaps" sections for the broader
// IQ-domain decoder dependency.
package cchunt

import "time"

// HuntState is the per-system status surfaced through the REST cockpit
// + TUI. It maps onto a small set of user-facing words rather than
// the raw supervisor internals.
type HuntState string

const (
	// StateIdle means the supervisor hasn't run for this system yet
	// (only seen briefly at startup before the first hunt round).
	StateIdle HuntState = "idle"
	// StateHunting means the supervisor is actively walking the
	// system's candidate CCs.
	StateHunting HuntState = "hunting"
	// StateLocked means a hunt succeeded — a cc.locked event was
	// observed within the dwell window and persisted to the cache.
	StateLocked HuntState = "locked"
	// StateFailed means the last hunt round exhausted without a
	// lock; the supervisor is in backoff before retrying.
	StateFailed HuntState = "failed"
	// StateHeld means the operator pinned the supervisor on its
	// current lock (or on its current failed state) — retunes are
	// suppressed until Resume.
	StateHeld HuntState = "held"
)

// SystemStatus is the per-system snapshot the supervisor exposes to
// the REST handler (and indirectly to the TUI cockpit panel).
// Time-valued fields are zero when not applicable (e.g. LockedAt is
// zero before the first successful hunt).
type SystemStatus struct {
	Name            string    `json:"name"`
	Protocol        string    `json:"protocol"`
	State           HuntState `json:"state"`
	AttemptedFreqHz uint32    `json:"attempted_freq_hz,omitempty"`
	AttemptIndex    int       `json:"attempt_index,omitempty"`
	TotalCandidates int       `json:"total_candidates,omitempty"`
	LockedFreqHz    uint32    `json:"locked_freq_hz,omitempty"`
	LockedAt        time.Time `json:"locked_at,omitempty"`
	NAC             uint16    `json:"nac,omitempty"`
	LastFailedAt    time.Time `json:"last_failed_at,omitempty"`
	BackoffMs       int       `json:"backoff_ms,omitempty"`
	LastGrantAt     time.Time `json:"last_grant_at,omitempty"`
}
