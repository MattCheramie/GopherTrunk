package api

import (
	"net/http"
	"strconv"
	"time"
)

// grantsDefaultLimit is the number of recent grants returned when the caller
// does not pass ?limit.
const grantsDefaultLimit = 500

// grantsMaxLimit caps ?limit so a single request can't ask for an unbounded
// page; the tracker's ring is bounded anyway, but this keeps the response
// size predictable.
const grantsMaxLimit = 5000

// GrantLogEntry is one row of GET /api/v1/grants: the stable GrantDTO the SSE
// stream already emits for KindGrant, plus the time the grant was decoded so a
// polling consumer can order and de-duplicate across polls. Reusing GrantDTO
// keeps the REST snapshot and the SSE live stream on one schema.
type GrantLogEntry struct {
	GrantDTO
	At time.Time `json:"at"`
}

// handleGrants answers GET /api/v1/grants with the most-recent voice-channel
// grants decoded off the control channel, newest first — the pollable form of
// the KindGrant stream the SSE feed carries live (issue #915). It lets a
// telemetry consumer read the source RID (and talkgroup, frequency,
// encryption) straight off the grant the way SDRtrunk's control-channel grant
// log does, without holding an SSE connection.
//
// Query params:
//   - limit:  max rows to return (default grantsDefaultLimit, capped at
//     grantsMaxLimit).
//   - system: restrict to one trunking-system name (exact match).
//
// Always 200; an empty list when the grant tracker is not wired.
func (s *Server) handleGrants(w http.ResponseWriter, r *http.Request) {
	entries := []GrantLogEntry{}
	if s.grants != nil {
		limit := grantsDefaultLimit
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > grantsMaxLimit {
			limit = grantsMaxLimit
		}
		system := r.URL.Query().Get("system")

		// Without a system filter, ask the tracker for exactly `limit` newest.
		// With one, pull the full retained window and cap after filtering so a
		// filtered query still returns up to `limit` matching rows.
		fetch := limit
		if system != "" {
			fetch = 0
		}
		for _, g := range s.grants.RecentGrants(fetch) {
			if system != "" && g.System != system {
				continue
			}
			entries = append(entries, GrantLogEntry{GrantDTO: grantToDTO(g), At: g.At})
			if len(entries) >= limit {
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": entries})
}
