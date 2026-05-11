package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// handleMutationStatus reports whether the daemon was started with
// mutations enabled. Always returns 200 — clients use this to
// light up write-side keybindings without having to probe for 403s.
func (s *Server) handleMutationStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"allow_mutations":  s.allowMutations,
		"engine_writable":  s.mutator != nil,
		"retention_writable": s.retention != nil,
		"tones_writable":   s.tones != nil,
	})
}

// endCallRequest is the body of POST /api/v1/calls/{deviceSerial}/end.
// reason is optional; defaults to "manual".
type endCallRequest struct {
	Reason string `json:"reason"`
}

// handleEndCall forces the engine to release the call held on the
// given device serial, publishing a CallEnd event with the supplied
// reason (default: "manual").
//
//   POST /api/v1/calls/00000001/end
//   Content-Type: application/json
//   {"reason":"manual"}
//
// Responses:
//   200 {"ok":true,"device_serial":"...","reason":"manual"}
//   404 if no active call holds the device
//   503 if the daemon doesn't have an EngineMutator wired
func (s *Server) handleEndCall(w http.ResponseWriter, r *http.Request) {
	if s.mutator == nil {
		writeError(w, http.StatusServiceUnavailable, "engine not wired for mutations")
		return
	}
	serial := r.PathValue("deviceSerial")
	if serial == "" {
		writeError(w, http.StatusBadRequest, "deviceSerial required")
		return
	}
	var req endCallRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	reason := parseEndReason(req.Reason)
	ok := s.mutator.EndCall(serial, reason)
	if !ok {
		writeError(w, http.StatusNotFound, "no active call on that device")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"device_serial": serial,
		"reason":        reason.String(),
	})
}

func parseEndReason(s string) trunking.EndReason {
	switch s {
	case "", "manual":
		return trunking.EndReasonManual
	case "normal":
		return trunking.EndReasonNormal
	case "lockout":
		return trunking.EndReasonLockout
	case "preempted":
		return trunking.EndReasonPreempted
	case "timeout":
		return trunking.EndReasonTimeout
	case "error":
		return trunking.EndReasonError
	}
	return trunking.EndReasonManual
}

// updateTalkgroupRequest is the PATCH body shape. All fields are
// pointers so JSON-omitted fields aren't accidentally zeroed: only
// supplied fields are applied.
type updateTalkgroupRequest struct {
	Priority *int  `json:"priority"`
	Lockout  *bool `json:"lockout"`
	Scan     *bool `json:"scan"`
}

// handleUpdateTalkgroup updates a talkgroup's mutable policy fields
// (priority and/or lockout). The full updated record is returned.
//
//   PATCH /api/v1/talkgroups/42
//   Content-Type: application/json
//   {"priority":3,"lockout":false}
//
// Responses:
//   200 with the updated TalkgroupDTO
//   400 if the id can't be parsed or the body is malformed
//   404 if no such talkgroup exists
func (s *Server) handleUpdateTalkgroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid talkgroup id")
		return
	}
	var req updateTalkgroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Priority == nil && req.Lockout == nil && req.Scan == nil {
		writeError(w, http.StatusBadRequest, "supply priority and/or lockout and/or scan")
		return
	}
	ok := s.talkgroups.UpdateFields(uint32(id), func(tg *trunking.TalkGroup) {
		if req.Priority != nil {
			tg.Priority = *req.Priority
		}
		if req.Lockout != nil {
			tg.Lockout = *req.Lockout
		}
		if req.Scan != nil {
			tg.Scan = *req.Scan
		}
	})
	if !ok {
		writeError(w, http.StatusNotFound, "talkgroup not found")
		return
	}
	tg := s.talkgroups.Lookup(uint32(id))
	writeJSON(w, http.StatusOK, talkgroupToDTO(tg))
}

// handleRetentionSweep kicks off one immediate retention sweep
// (call-log rows + recordings, depending on what's configured).
// The sweep runs synchronously inside the request — typical sweep
// runs are sub-second; if a deployment outgrows that we'll move
// it to a goroutine + 202.
//
//   POST /api/v1/retention/sweep
//
// Responses:
//   200 {"ok":true}
//   503 if the daemon doesn't have a Retention wired (e.g. no
//       call-log persistence configured)
func (s *Server) handleRetentionSweep(w http.ResponseWriter, r *http.Request) {
	if s.retention == nil {
		writeError(w, http.StatusServiceUnavailable, "retention not wired")
		return
	}
	s.retention.SweepOnce(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleToneReset clears per-device tone-out match progress for a
// given device, leaving the cooldown clock intact (so an in-flight
// false alarm doesn't immediately re-fire).
//
//   POST /api/v1/devices/00000001/tone-reset
//
// Responses:
//   200 {"ok":true,"device_serial":"..."}
//   503 if the daemon doesn't have a tone detector wired
func (s *Server) handleToneReset(w http.ResponseWriter, r *http.Request) {
	if s.tones == nil {
		writeError(w, http.StatusServiceUnavailable, "tone detector not wired")
		return
	}
	serial := r.PathValue("serial")
	if serial == "" {
		writeError(w, http.StatusBadRequest, "serial required")
		return
	}
	s.tones.ResetDevice(serial)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"device_serial": serial,
	})
}
