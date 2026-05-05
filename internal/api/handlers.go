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
