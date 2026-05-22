package api

import (
	"net/http"
	"strconv"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
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

// handleAffiliations returns the affiliation tracker's unit-activity
// table — which radio units are currently active on which talkgroups.
// Always 200; an empty list when the tracker is not wired.
func (s *Server) handleAffiliations(w http.ResponseWriter, _ *http.Request) {
	units := []trunking.UnitActivity{}
	if s.affiliations != nil {
		if snap := s.affiliations.Affiliations(); snap != nil {
			units = snap
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"affiliations": units})
}
