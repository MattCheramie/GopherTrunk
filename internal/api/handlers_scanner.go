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

// handleScannerManualTune adds a VFO-style temporary channel to the
// conventional scanner and forces dwell on it. Mirrors the muscle
// memory of a traditional scanner's "FREQ" / "MAN" / "TUNE" key.
//
//   POST /api/v1/scanner/manual_tune
//   Content-Type: application/json
//   {"frequency_hz":155895000,"label":"sheriff","mode":"fm"}
//
// Responses:
//
//	200 {"ok":true,"index":N,"frequency_hz":...}
//	400 if frequency_hz is missing or out of range
//	503 if the conventional scanner isn't wired (no Voice SDR
//	    carved out for it)
func (s *Server) handleScannerManualTune(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	var req ManualTuneRequest
	if r.Body == nil || r.ContentLength == 0 {
		writeError(w, http.StatusBadRequest, "body required")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.FrequencyHz == 0 {
		writeError(w, http.StatusBadRequest, "frequency_hz required")
		return
	}
	// Sanity check: 25 MHz – 1.3 GHz is the practical RTL-SDR tuning
	// range. Looser is fine but a zero or absurd value almost
	// certainly means the operator mis-typed.
	if req.FrequencyHz < 25_000_000 || req.FrequencyHz > 1_300_000_000 {
		writeError(w, http.StatusBadRequest, "frequency_hz outside 25 MHz – 1.3 GHz tuning range")
		return
	}
	if req.Mode != "" && req.Mode != "fm" && req.Mode != "nfm" {
		writeError(w, http.StatusBadRequest, "mode must be fm or nfm")
		return
	}
	idx, ok := s.scanner.ManualTune(req)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "conventional scanner not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"index":        idx,
		"frequency_hz": req.FrequencyHz,
	})
}

// handleScannerClearManualTune removes a temporary channel
// previously added via manual tune. Static (config-seeded)
// channels can't be removed at runtime — those return 404.
//
//   DELETE /api/v1/scanner/manual_tune/{index}
//
// Responses:
//
//	200 {"ok":true,"index":N}
//	400 if the index can't be parsed
//	404 if the index isn't a temp channel
//	503 if the scanner isn't wired
func (s *Server) handleScannerClearManualTune(w http.ResponseWriter, r *http.Request) {
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
	if !s.scanner.ClearManualTune(idx) {
		writeError(w, http.StatusNotFound, "no such temporary channel")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "index": idx})
}
