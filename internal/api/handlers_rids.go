package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// handleListRIDs returns the merged static-alias + live-tracker view
// of every radio unit the daemon knows about. Empty `{rids: []}` when
// neither source is wired.
//
//	GET /api/v1/rids
//
// Configured rows always appear; live-only rows (over-the-air radios
// without an operator-configured alias) appear when the affiliation
// tracker has seen them since the last sweep.
func (s *Server) handleListRIDs(w http.ResponseWriter, r *http.Request) {
	// Optional ?system= scopes the list to radios whose most-recent (or live)
	// activity is on that system — the RID roster grows large across many
	// systems, so the UI offers a per-system selector. Empty = every system.
	system := r.URL.Query().Get("system")
	dtos := s.mergedRIDList(r.Context(), system)
	writeJSON(w, http.StatusOK, map[string]any{"rids": dtos})
}

// handleGetRID returns one merged row by id.
//
//	GET /api/v1/rids/{id}
func (s *Server) handleGetRID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid rid")
		return
	}
	id := uint32(id64)
	var dto *RIDDTO
	if s.rids != nil {
		dto = ridToDTO(s.rids.Lookup(id))
	}
	// Durable base from the persisted call log, so a radio that has aged out of
	// the live tracker (or predates a restart) is still found.
	if p, ok := s.history.(RIDSummaryProvider); ok {
		if sums, err := p.RIDSummaries(r.Context(), RIDSummaryFilter{SourceID: id, Limit: 1}); err != nil {
			s.log.Warn("api: rid summary query failed", "err", err, "rid", id)
		} else if len(sums) > 0 {
			allTime := sums[0]
			dto = mergeRIDSummary(dto, allTime)
		}
	}
	if s.affiliations != nil {
		for _, u := range s.affiliations.Affiliations() {
			if u.RadioID == id {
				dto = mergeRIDLiveKeepAllTime(dto, u)
				break
			}
		}
	}
	if dto == nil {
		s.writeError(w, http.StatusNotFound, "rid not found")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// handleRIDHistory returns the call_log rows where source_id == {id}.
// Mirrors handleCallHistory's query parameters except group_id and
// source_id (the path id is the source filter).
//
//	GET /api/v1/rids/{id}/history?limit=50
func (s *Server) handleRIDHistory(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		s.writeError(w, http.StatusServiceUnavailable, "call log persistence is not enabled")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid rid")
		return
	}
	f := HistoryFilter{
		System:   r.URL.Query().Get("system"),
		SourceID: uint32(id64),
		Limit:    100,
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			s.writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		if n > 1000 {
			n = 1000
		}
		f.Limit = n
	}
	if r.URL.Query().Get("only_ended") == "true" {
		f.OnlyEnded = true
	}
	rows, err := s.history.History(r.Context(), f)
	if err != nil {
		s.log.Warn("api: rid history query failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "history query failed")
		return
	}
	if rows == nil {
		rows = []CallRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": rows})
}

// updateRIDRequest is the JSON body for PATCH /api/v1/rids/{id}. Every
// field is a pointer so the request can update a single field without
// resetting the others.
type updateRIDRequest struct {
	Alias       *string `json:"alias"`
	Description *string `json:"description"`
	Tag         *string `json:"tag"`
	Group       *string `json:"group"`
	Owner       *string `json:"owner"`
	Priority    *int    `json:"priority"`
	Lockout     *bool   `json:"lockout"`
	Watch       *bool   `json:"watch"`
	Icon        *string `json:"icon"`
}

// handleUpdateRID mutates an RID's operator-policy fields in the
// in-memory RIDDB. Returns the updated DTO (merged with any live
// observation data so the response shape matches the list endpoint).
//
//	PATCH /api/v1/rids/{id}
//	Content-Type: application/json
//	{"alias":"CPL-SMITH","watch":true}
//
// Behaviour mirrors the talkgroup PATCH: mutations live in memory
// only; the on-disk rid_alias_file is not rewritten. RIDs only seen
// over the air (no static catalogue entry) cannot be patched and
// return 404 — operators must add the radio to the alias file first.
func (s *Server) handleUpdateRID(w http.ResponseWriter, r *http.Request) {
	if s.rids == nil {
		s.writeError(w, http.StatusServiceUnavailable, "rid catalogue not wired")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid rid")
		return
	}
	var req updateRIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Alias == nil && req.Description == nil && req.Tag == nil &&
		req.Group == nil && req.Owner == nil && req.Priority == nil &&
		req.Lockout == nil && req.Watch == nil && req.Icon == nil {
		s.writeError(w, http.StatusBadRequest,
			"supply at least one of alias, description, tag, group, owner, priority, lockout, watch, icon")
		return
	}
	id := uint32(id64)
	ok := s.rids.UpdateFields(id, func(rec *trunking.RID) {
		if req.Alias != nil {
			rec.Alias = *req.Alias
		}
		if req.Description != nil {
			rec.Description = *req.Description
		}
		if req.Tag != nil {
			rec.Tag = *req.Tag
		}
		if req.Group != nil {
			rec.Group = *req.Group
		}
		if req.Owner != nil {
			rec.Owner = *req.Owner
		}
		if req.Priority != nil {
			rec.Priority = *req.Priority
		}
		if req.Lockout != nil {
			rec.Lockout = *req.Lockout
		}
		if req.Watch != nil {
			rec.Watch = *req.Watch
		}
		if req.Icon != nil {
			rec.Icon = *req.Icon
		}
	})
	if !ok {
		s.writeError(w, http.StatusNotFound, "rid not found in static catalogue (add to rid_alias_file first)")
		return
	}
	dto := ridToDTO(s.rids.Lookup(id))
	if s.affiliations != nil {
		for _, u := range s.affiliations.Affiliations() {
			if u.RadioID == id {
				dto = mergeRIDLive(dto, u)
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, dto)
}

// mergedRIDList merges three sources by RadioID, in increasing order of
// freshness: the static RIDDB (operator catalogue), the persisted call-log
// aggregate (durable, all-time observations), then the live affiliation tracker
// (the most recent in-window observation). The call-log layer is what makes the
// list durable — a radio no longer drops out once the tracker's 30-minute idle
// TTL sweeps it, which is what made the list look like it "reset after each CC
// lock" (the re-hunt gaps let the TTL expire every radio, with nothing behind
// it). The output is sorted by ID for a stable list ordering — the UI applies
// its own sort on top.
func (s *Server) mergedRIDList(ctx context.Context, system string) []*RIDDTO {
	byID := map[uint32]*RIDDTO{}
	// The static operator catalogue carries no system association, so it is
	// only included in the unfiltered ("all systems") view. When a system is
	// selected, a static alias still surfaces for any RID that has call-log or
	// live activity on that system — the merge below seeds the alias for those
	// rows — so scoping to a system never hides an operator-named radio that
	// actually appears on it.
	if s.rids != nil && system == "" {
		for _, r := range s.rids.All() {
			byID[r.ID] = ridToDTO(r)
		}
	}
	// aliasFor returns the static-catalogue DTO for a RID (nil if none),
	// used to seed a filtered row so it keeps the operator's alias/tag.
	aliasFor := func(id uint32) *RIDDTO {
		if system == "" || s.rids == nil {
			return nil
		}
		return ridToDTO(s.rids.Lookup(id))
	}
	if p, ok := s.history.(RIDSummaryProvider); ok {
		if sums, err := p.RIDSummaries(ctx, RIDSummaryFilter{System: system}); err != nil {
			s.log.Warn("api: rid summaries query failed", "err", err)
		} else {
			for _, sum := range sums {
				dto := byID[sum.SourceID]
				if dto == nil {
					dto = aliasFor(sum.SourceID)
				}
				byID[sum.SourceID] = mergeRIDSummary(dto, sum)
			}
		}
	}
	if s.affiliations != nil {
		for _, u := range s.affiliations.Affiliations() {
			if system != "" && !strings.EqualFold(u.System, system) {
				continue
			}
			dto := byID[u.RadioID]
			if dto == nil {
				dto = aliasFor(u.RadioID)
			}
			byID[u.RadioID] = mergeRIDLiveKeepAllTime(dto, u)
		}
	}
	out := make([]*RIDDTO, 0, len(byID))
	for _, dto := range byID {
		out = append(out, dto)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// mergeRIDLiveKeepAllTime overlays the live tracker's fresh observation onto a
// DTO that may already carry the durable call-log aggregate, without letting the
// live counters clobber the all-time ones: CallCount keeps the larger value and
// FirstSeen the earlier, since the live tracker only counts calls seen since it
// (re)added the unit after its last idle sweep, whereas the call log is all-time.
func mergeRIDLiveKeepAllTime(dto *RIDDTO, u trunking.UnitActivity) *RIDDTO {
	allTimeCalls := uint64(0)
	var firstSeen time.Time
	if dto != nil {
		allTimeCalls, firstSeen = dto.CallCount, dto.FirstSeen
	}
	dto = mergeRIDLive(dto, u)
	if allTimeCalls > dto.CallCount {
		dto.CallCount = allTimeCalls
	}
	if !firstSeen.IsZero() && (dto.FirstSeen.IsZero() || firstSeen.Before(dto.FirstSeen)) {
		dto.FirstSeen = firstSeen
	}
	return dto
}
