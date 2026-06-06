package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// handleHuntStatus returns the live system-discovery snapshot. Always 200 —
// when the cockpit is nil (no SDR / hunt not wired) an idle status is returned
// so the UI renders "no run" rather than a 503.
func (s *Server) handleHuntStatus(w http.ResponseWriter, _ *http.Request) {
	if s.hunt == nil {
		writeJSON(w, http.StatusOK, HuntStatus{State: "idle"})
		return
	}
	writeJSON(w, http.StatusOK, s.hunt.Status())
}

// handleHuntStart launches a live hunt run.
func (s *Server) handleHuntStart(w http.ResponseWriter, r *http.Request) {
	if s.hunt == nil {
		s.writeError(w, http.StatusServiceUnavailable, "hunt not wired (no SDR available)")
		return
	}
	var req HuntStartRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	runID, err := s.hunt.Start(req)
	if err != nil {
		// "already in progress" is a conflict; everything else is a bad request.
		status := http.StatusBadRequest
		if isHuntConflict(err) {
			status = http.StatusConflict
		}
		s.writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "run_id": runID})
}

// handleHuntStop cancels the active run.
func (s *Server) handleHuntStop(w http.ResponseWriter, _ *http.Request) {
	if s.hunt == nil {
		s.writeError(w, http.StatusServiceUnavailable, "hunt not wired")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stopped": s.hunt.Stop()})
}

// handleHuntExport streams the discovered system in the requested format.
func (s *Server) handleHuntExport(w http.ResponseWriter, r *http.Request) {
	if s.hunt == nil {
		s.writeError(w, http.StatusServiceUnavailable, "hunt not wired")
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "bundle"
	}
	data, filename, err := s.hunt.Export(format)
	if err != nil {
		// No discovered system yet → 409; a bad format → 400.
		status := http.StatusConflict
		if isHuntBadFormat(err) {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentTypeForExport(format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// huntCommitRequest is the optional POST /api/v1/hunt/commit body.
type huntCommitRequest struct {
	Force  bool `json:"force,omitempty"`
	DryRun bool `json:"dry_run,omitempty"`
}

// handleHuntCommit merges the discovered system into config.yaml.
func (s *Server) handleHuntCommit(w http.ResponseWriter, r *http.Request) {
	if s.hunt == nil {
		s.writeError(w, http.StatusServiceUnavailable, "hunt not wired")
		return
	}
	var req huntCommitRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	changes, err := s.hunt.Commit(req.Force, req.DryRun)
	if err != nil {
		s.writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dry_run": req.DryRun, "changes": changes})
}

func contentTypeForExport(format string) string {
	switch format {
	case "trunk-recorder", "trunkrecorder", "tr":
		return "application/json"
	case "rr", "radioreference":
		return "text/markdown; charset=utf-8"
	default:
		return "text/csv; charset=utf-8"
	}
}

// isHuntConflict / isHuntBadFormat classify cockpit errors for HTTP status
// mapping without coupling the handler to the hunt package's error values.
func isHuntConflict(err error) bool {
	return err != nil && containsAny(err.Error(), "already in progress", "in progress")
}

func isHuntBadFormat(err error) bool {
	return err != nil && containsAny(err.Error(), "unknown export format", "unknown format")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && indexOfSub(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOfSub(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
