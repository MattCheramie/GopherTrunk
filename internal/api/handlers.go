package api

import (
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"now":    time.Now().UTC(),
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	v := s.version
	if v == "" {
		v = "dev"
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": v})
}

func (s *Server) handleListSystems(w http.ResponseWriter, _ *http.Request) {
	out := make([]SystemDTO, 0, len(s.systems))
	for _, sys := range s.systems {
		out = append(out, systemToDTO(sys))
	}
	writeJSON(w, http.StatusOK, map[string]any{"systems": out})
}

func (s *Server) handleGetSystem(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	for _, sys := range s.systems {
		if sys.Name == name {
			writeJSON(w, http.StatusOK, systemToDTO(sys))
			return
		}
	}
	writeError(w, http.StatusNotFound, "system not found")
}

func (s *Server) handleListTalkgroups(w http.ResponseWriter, _ *http.Request) {
	all := s.talkgroups.All()
	out := make([]*TalkgroupDTO, 0, len(all))
	for _, tg := range all {
		out = append(out, talkgroupToDTO(tg))
	}
	writeJSON(w, http.StatusOK, map[string]any{"talkgroups": out})
}

func (s *Server) handleGetTalkgroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid talkgroup id")
		return
	}
	tg := s.talkgroups.Lookup(uint32(id))
	if tg == nil {
		writeError(w, http.StatusNotFound, "talkgroup not found")
		return
	}
	writeJSON(w, http.StatusOK, talkgroupToDTO(tg))
}

func (s *Server) handleActiveCalls(w http.ResponseWriter, _ *http.Request) {
	if s.engine == nil {
		writeJSON(w, http.StatusOK, map[string]any{"calls": []ActiveCallDTO{}})
		return
	}
	active := s.engine.ActiveCalls()
	out := make([]ActiveCallDTO, 0, len(active))
	for _, ac := range active {
		out = append(out, activeCallToDTO(ac))
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": out})
}

// handleCallHistory queries the persisted call_log table.
//   ?system=<name>      filter by system
//   ?group_id=<n>       filter by talkgroup
//   ?since=<rfc3339>    only calls started at/after this time
//   ?until=<rfc3339>    only calls started before this time
//   ?limit=<n>          cap rows (default 100, max 1000)
//   ?only_ended=true    skip calls that haven't ended
func (s *Server) handleCallHistory(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		writeError(w, http.StatusServiceUnavailable, "call log persistence is not enabled")
		return
	}
	q := r.URL.Query()
	f := HistoryFilter{
		System: q.Get("system"),
		Limit:  100,
	}
	if v := q.Get("group_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid group_id")
			return
		}
		f.GroupID = uint32(n)
	}
	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since (want RFC3339)")
			return
		}
		f.Since = t
	}
	if v := q.Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid until (want RFC3339)")
			return
		}
		f.Until = t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		if n > 1000 {
			n = 1000
		}
		f.Limit = n
	}
	if q.Get("only_ended") == "true" {
		f.OnlyEnded = true
	}
	rows, err := s.history.History(r.Context(), f)
	if err != nil {
		s.log.Warn("api: history query failed", "err", err)
		writeError(w, http.StatusInternalServerError, "history query failed")
		return
	}
	if rows == nil {
		rows = []CallRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": rows})
}
