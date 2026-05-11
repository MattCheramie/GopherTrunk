package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// handleScannerStatus returns the unified scanner snapshot the TUI
// Scanner panel renders. Always 200 — when the cockpit is nil
// (scanner subsystem not wired), an empty status is returned so the
// TUI can still render "no systems / no conventional channels"
// instead of a 503.
func (s *Server) handleScannerStatus(w http.ResponseWriter, _ *http.Request) {
	if s.scanner == nil {
		writeJSON(w, http.StatusOK, ScannerStatus{
			ScanMode: "all",
		})
		return
	}
	writeJSON(w, http.StatusOK, s.scanner.Status())
}

// scannerSetModeRequest is the PATCH /api/v1/scanner body shape.
type scannerSetModeRequest struct {
	ScanMode string `json:"scan_mode"`
}

func (s *Server) handleScannerSetMode(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	var req scannerSetModeRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	if req.ScanMode == "" {
		writeError(w, http.StatusBadRequest, "scan_mode required")
		return
	}
	prev, err := s.scanner.SetScanMode(req.ScanMode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"scan_mode":     req.ScanMode,
		"previous_mode": prev,
	})
}

func (s *Server) handleHuntHold(w http.ResponseWriter, r *http.Request) {
	s.huntOp(w, r, s.scanner.HoldHunt)
}
func (s *Server) handleHuntResume(w http.ResponseWriter, r *http.Request) {
	s.huntOp(w, r, s.scanner.ResumeHunt)
}
func (s *Server) handleHuntRetune(w http.ResponseWriter, r *http.Request) {
	s.huntOp(w, r, s.scanner.ForceRetuneHunt)
}

// huntOp is the shared mechanics for the three per-system hunt
// operations: nil cockpit → 503, empty path → 400, unknown system
// → 404. The actual mutation is delegated to the supplied func.
func (s *Server) huntOp(w http.ResponseWriter, r *http.Request, op func(string) bool) {
	if s.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	system := r.PathValue("system")
	if system == "" {
		writeError(w, http.StatusBadRequest, "system required")
		return
	}
	if !op(system) {
		writeError(w, http.StatusNotFound, "no such system")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "system": system})
}

func (s *Server) handleConvHold(w http.ResponseWriter, _ *http.Request) {
	if s.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	if !s.scanner.HoldConventional() {
		writeError(w, http.StatusServiceUnavailable, "conventional scanner not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConvResume(w http.ResponseWriter, _ *http.Request) {
	if s.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	if !s.scanner.ResumeConventional() {
		writeError(w, http.StatusServiceUnavailable, "conventional scanner not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConvDwell(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	idxStr := r.PathValue("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		writeError(w, http.StatusBadRequest, "invalid index")
		return
	}
	if !s.scanner.DwellConventional(idx) {
		writeError(w, http.StatusNotFound, "channel index out of range")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "index": idx})
}
