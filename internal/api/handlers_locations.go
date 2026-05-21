package api

import (
	"net/http"
	"strconv"
)

// handleLocations returns recent geographic fixes for the web map.
// When the location subsystem is not wired (no storage) an empty list
// is returned so the UI renders a stable shape.
func (s *Server) handleLocations(w http.ResponseWriter, r *http.Request) {
	if s.locations == nil {
		writeJSON(w, http.StatusOK, map[string]any{"locations": []LocationFix{}})
		return
	}
	limit := 500
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	fixes, err := s.locations.RecentLocations(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query locations: "+err.Error())
		return
	}
	if fixes == nil {
		fixes = []LocationFix{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": fixes})
}
