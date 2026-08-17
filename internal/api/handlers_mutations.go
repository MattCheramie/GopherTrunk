package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// handleMutationStatus reports the daemon's mutation policy and
// whether the current request would be accepted. Always returns 200;
// clients use can_mutate to light up write-side keybindings without
// having to probe a real endpoint for 401 / 403.
//
//	auth_mode      — "auto" | "required" | "disabled"
//	can_mutate     — true when the current request would pass auth
//	allow_mutations— legacy alias for can_mutate (deprecated)
//	engine_writable / retention_writable / tones_writable — wiring
func (s *Server) handleMutationStatus(w http.ResponseWriter, r *http.Request) {
	canMutate := s.auth.canMutate(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"auth_mode":          s.auth.mode.String(),
		"can_mutate":         canMutate,
		"allow_mutations":    canMutate, // legacy alias
		"engine_writable":    s.mutator != nil,
		"retention_writable": s.retention != nil,
		"tones_writable":     s.tones != nil,
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
//	POST /api/v1/calls/00000001/end
//	Content-Type: application/json
//	{"reason":"manual"}
//
// Responses:
//
//	200 {"ok":true,"device_serial":"...","reason":"manual"}
//	404 if no active call holds the device
//	503 if the daemon doesn't have an EngineMutator wired
func (s *Server) handleEndCall(w http.ResponseWriter, r *http.Request) {
	if s.mutator == nil {
		s.writeError(w, http.StatusServiceUnavailable, "engine not wired for mutations")
		return
	}
	serial := r.PathValue("deviceSerial")
	if serial == "" {
		s.writeError(w, http.StatusBadRequest, "deviceSerial required")
		return
	}
	var req endCallRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	reason := parseEndReason(req.Reason)
	ok := s.mutator.EndCall(serial, reason)
	if !ok {
		s.writeError(w, http.StatusNotFound, "no active call on that device")
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
	// AlphaTag / Description / Tag / Group are the operator-applied name and
	// its descriptive fields. Unlike the policy fields below they are
	// persisted when a label store is wired, so a name applied while watching
	// live traffic survives a restart.
	AlphaTag    *string `json:"alpha_tag"`
	Description *string `json:"description"`
	Tag         *string `json:"tag"`
	Group       *string `json:"group"`
	Priority    *int    `json:"priority"`
	Lockout     *bool   `json:"lockout"`
	Scan        *bool   `json:"scan"`
	Stream      *bool   `json:"stream"`
	Record      *bool   `json:"record"`
	Mute        *bool   `json:"mute"`
	Icon        *string `json:"icon"`
}

// empty reports whether the request carries no fields at all.
func (r updateTalkgroupRequest) empty() bool {
	return r.AlphaTag == nil && r.Description == nil && r.Tag == nil &&
		r.Group == nil && r.Priority == nil && r.Lockout == nil &&
		r.Scan == nil && r.Stream == nil && r.Record == nil &&
		r.Mute == nil && r.Icon == nil
}

// handleUpdateTalkgroup updates a talkgroup's operator-applied name and its
// mutable policy fields. The full updated record is returned.
//
//	PATCH /api/v1/talkgroups/42?system=250_013
//	Content-Type: application/json
//	{"alpha_tag":"TAC-1","priority":3}
//
// A talkgroup that is not in the catalogue is CREATED rather than 404'd: an
// operator naming a talkgroup they can see on the air should not first have to
// add it to a file and restart the daemon. The optional ?system= scopes the
// persisted label to one system; omitted, the name applies wherever the id is
// seen, matching the daemon's single merged catalogue.
//
// Responses:
//
//	200 with the updated TalkgroupDTO
//	400 if the id can't be parsed or the body is malformed
//	503 if no talkgroup catalogue is wired
func (s *Server) handleUpdateTalkgroup(w http.ResponseWriter, r *http.Request) {
	if s.talkgroups == nil {
		s.writeError(w, http.StatusServiceUnavailable, "talkgroup catalogue not wired")
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid talkgroup id")
		return
	}
	var req updateTalkgroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.empty() {
		s.writeError(w, http.StatusBadRequest,
			"supply at least one of alpha_tag, description, tag, group, priority, lockout, scan, stream, record, mute, icon")
		return
	}
	apply := func(tg *trunking.TalkGroup) {
		if req.AlphaTag != nil {
			tg.AlphaTag = *req.AlphaTag
		}
		if req.Description != nil {
			tg.Description = *req.Description
		}
		if req.Tag != nil {
			tg.Tag = *req.Tag
		}
		if req.Group != nil {
			tg.Group = *req.Group
		}
		if req.Priority != nil {
			tg.Priority = *req.Priority
		}
		if req.Lockout != nil {
			tg.Lockout = *req.Lockout
		}
		if req.Scan != nil {
			tg.Scan = *req.Scan
		}
		if req.Stream != nil {
			tg.Stream = *req.Stream
		}
		if req.Record != nil {
			tg.Record = *req.Record
		}
		if req.Mute != nil {
			tg.Mute = *req.Mute
		}
		if req.Icon != nil {
			tg.Icon = *req.Icon
		}
	}
	if !s.talkgroups.UpdateFields(uint32(id), apply) {
		// Not catalogued: synthesise a record with the CSV loader's defaults
		// so the row behaves like a loaded one. Deliberately NOT tagged
		// Discovered — DeleteDiscovered would then silently retract the
		// operator's name.
		tg := &trunking.TalkGroup{ID: uint32(id), Scan: true, Stream: true, Record: true}
		apply(tg)
		s.talkgroups.Add(tg)
	}
	tg := s.talkgroups.Lookup(uint32(id))
	s.persistTalkgroupLabel(r.URL.Query().Get("system"), tg)
	writeJSON(w, http.StatusOK, talkgroupToDTO(tg))
}

// handleRetentionSweep kicks off one immediate retention sweep
// (call-log rows + recordings, depending on what's configured).
// The sweep runs synchronously inside the request — typical sweep
// runs are sub-second; if a deployment outgrows that we'll move
// it to a goroutine + 202.
//
//	POST /api/v1/retention/sweep
//
// Responses:
//
//	200 {"ok":true}
//	503 if the daemon doesn't have a Retention wired (e.g. no
//	    call-log persistence configured)
func (s *Server) handleRetentionSweep(w http.ResponseWriter, r *http.Request) {
	if s.retention == nil {
		s.writeError(w, http.StatusServiceUnavailable, "retention not wired")
		return
	}
	s.retention.SweepOnce(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleToneReset clears per-device tone-out match progress for a
// given device, leaving the cooldown clock intact (so an in-flight
// false alarm doesn't immediately re-fire).
//
//	POST /api/v1/devices/00000001/tone-reset
//
// Responses:
//
//	200 {"ok":true,"device_serial":"..."}
//	503 if the daemon doesn't have a tone detector wired
func (s *Server) handleToneReset(w http.ResponseWriter, r *http.Request) {
	if s.tones == nil {
		s.writeError(w, http.StatusServiceUnavailable, "tone detector not wired")
		return
	}
	serial := r.PathValue("serial")
	if serial == "" {
		s.writeError(w, http.StatusBadRequest, "serial required")
		return
	}
	s.tones.ResetDevice(serial)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"device_serial": serial,
	})
}
