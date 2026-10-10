package api

import (
	"encoding/json"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"io"
	"net/http"
	"strconv"
	"time"
)

// handleScannerStatus returns the unified scanner snapshot the TUI
// Scanner panel renders. Always 200 — when the cockpit is nil
// (scanner subsystem not wired), an empty status is returned so the
// TUI can still render "no systems / no conventional channels"
// instead of a 503.
func (s *Server) handleScannerStatus(w http.ResponseWriter, _ *http.Request) {
	if s.scanner == nil {
		writeJSON(w, http.StatusOK, normalizeScannerStatus(ScannerStatus{
			ScanMode: "all",
		}))
		return
	}
	writeJSON(w, http.StatusOK, normalizeScannerStatus(s.scanner.Status()))
}

// normalizeScannerStatus guarantees the slice fields marshal as JSON arrays
// rather than null. A nil Go slice encodes to `null`, which the web Scanner
// tab then dereferences (`systems.length` / `channels.length`) and crashes the
// whole panel behind the error boundary ("Cannot read properties of null").
// Emitting `[]` keeps the wire contract array-typed regardless of whether any
// systems / conventional channels are configured.
func normalizeScannerStatus(st ScannerStatus) ScannerStatus {
	if st.Systems == nil {
		st.Systems = []SystemHuntStatusDTO{}
	}
	if st.Conventional.Channels == nil {
		st.Conventional.Channels = []ConvChannelStatusDTO{}
	}
	if st.Avoids == nil {
		st.Avoids = []trunking.Avoid{}
	}
	return st
}

// scannerHoldRequest is the POST /api/v1/scanner/hold body.
type scannerHoldRequest struct {
	System    string `json:"system"`
	Talkgroup uint32 `json:"talkgroup"`
}

// handleScannerHold pins the engine to one talkgroup (the scanner "Hold"
// key): every other grant is dropped until DELETE /api/v1/scanner/hold.
func (s *Server) handleScannerHold(w http.ResponseWriter, r *http.Request) {
	if s.scanControl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "engine not wired for scan control")
		return
	}
	var req scannerHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Talkgroup == 0 {
		s.writeError(w, http.StatusBadRequest, "talkgroup required")
		return
	}
	st := s.scanControl.Hold(req.System, req.Talkgroup)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hold": st})
}

// handleScannerReleaseHold clears the talkgroup hold.
func (s *Server) handleScannerReleaseHold(w http.ResponseWriter, _ *http.Request) {
	if s.scanControl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "engine not wired for scan control")
		return
	}
	had := s.scanControl.ReleaseHold()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "released": had})
}

// talkgroupAvoidRequest is the POST /api/v1/talkgroups/{id}/avoid body:
// a duration ("30m", "2h") or minutes; system scopes the avoid (empty =
// any system). Default 30 minutes.
type talkgroupAvoidRequest struct {
	System   string `json:"system"`
	Duration string `json:"duration"`
	Minutes  int    `json:"minutes"`
}

// handleTalkgroupAvoid temporarily locks a talkgroup out (the scanner
// "Avoid" key's one-press temporary form).
func (s *Server) handleTalkgroupAvoid(w http.ResponseWriter, r *http.Request) {
	if s.scanControl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "engine not wired for scan control")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil || id == 0 {
		s.writeError(w, http.StatusBadRequest, "invalid talkgroup id")
		return
	}
	var req talkgroupAvoidRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	d := 30 * time.Minute
	switch {
	case req.Duration != "":
		pd, err := time.ParseDuration(req.Duration)
		if err != nil || pd <= 0 {
			s.writeError(w, http.StatusBadRequest, "duration must be a positive Go duration (e.g. 30m, 2h)")
			return
		}
		d = pd
	case req.Minutes != 0:
		if req.Minutes < 0 {
			s.writeError(w, http.StatusBadRequest, "minutes must be positive")
			return
		}
		d = time.Duration(req.Minutes) * time.Minute
	}
	a := s.scanControl.AvoidTalkgroup(req.System, uint32(id), d)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "avoid": a})
}

// handleTalkgroupUnavoid clears a temporary lockout early.
func (s *Server) handleTalkgroupUnavoid(w http.ResponseWriter, r *http.Request) {
	if s.scanControl == nil {
		s.writeError(w, http.StatusServiceUnavailable, "engine not wired for scan control")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil || id == 0 {
		s.writeError(w, http.StatusBadRequest, "invalid talkgroup id")
		return
	}
	system := r.URL.Query().Get("system")
	had := s.scanControl.UnavoidTalkgroup(system, uint32(id))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "cleared": had})
}

// scannerSetModeRequest is the PATCH /api/v1/scanner body shape.
type scannerSetModeRequest struct {
	ScanMode string `json:"scan_mode"`
}

func (s *Server) handleScannerSetMode(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	var req scannerSetModeRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	if req.ScanMode == "" {
		s.writeError(w, http.StatusBadRequest, "scan_mode required")
		return
	}
	prev, err := s.scanner.SetScanMode(req.ScanMode)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
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
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	system := r.PathValue("system")
	if system == "" {
		s.writeError(w, http.StatusBadRequest, "system required")
		return
	}
	if !op(system) {
		s.writeError(w, http.StatusNotFound, "no such system")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "system": system})
}

func (s *Server) handleConvHold(w http.ResponseWriter, _ *http.Request) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	if !s.scanner.HoldConventional() {
		s.writeError(w, http.StatusServiceUnavailable, "conventional scanner not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConvResume(w http.ResponseWriter, _ *http.Request) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	if !s.scanner.ResumeConventional() {
		s.writeError(w, http.StatusServiceUnavailable, "conventional scanner not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConvDwell(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	idxStr := r.PathValue("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		s.writeError(w, http.StatusBadRequest, "invalid index")
		return
	}
	if !s.scanner.DwellConventional(idx) {
		s.writeError(w, http.StatusNotFound, "channel index out of range")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "index": idx})
}

// handleConvLockout flips the per-channel lockout flag on. Locked-
// out channels are skipped by the scanner's pickNextChannel until
// the operator unlocks them via handleConvUnlockout.
//
//	POST /api/v1/scanner/conventional/{index}/lockout
//
// Responses:
//
//	200 {"ok":true,"index":N,"locked_out":true}
//	404 if the channel index is out of range
//	503 if the scanner cockpit isn't wired
func (s *Server) handleConvLockout(w http.ResponseWriter, r *http.Request) {
	s.convLockoutOp(w, r, s.scanner.LockoutConventional, true)
}

// handleConvUnlockout flips the per-channel lockout flag off. Same
// shape as handleConvLockout.
//
//	POST /api/v1/scanner/conventional/{index}/unlockout
func (s *Server) handleConvUnlockout(w http.ResponseWriter, r *http.Request) {
	s.convLockoutOp(w, r, s.scanner.UnlockoutConventional, false)
}

// convLockoutOp is the shared mechanics for both lockout sides.
// Factors out the index parsing + 404/503 logic so the two
// handlers stay one-liners.
func (s *Server) convLockoutOp(w http.ResponseWriter, r *http.Request, op func(int) bool, newState bool) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	idxStr := r.PathValue("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		s.writeError(w, http.StatusBadRequest, "invalid index")
		return
	}
	if !op(idx) {
		s.writeError(w, http.StatusNotFound, "channel index out of range")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"index":      idx,
		"locked_out": newState,
	})
}

// handleScannerManualTune adds a VFO-style temporary channel to the
// conventional scanner and forces dwell on it. Mirrors the muscle
// memory of a traditional scanner's "FREQ" / "MAN" / "TUNE" key.
//
//	POST /api/v1/scanner/manual_tune
//	Content-Type: application/json
//	{"frequency_hz":155895000,"label":"sheriff","mode":"fm"}
//	{"frequency_hz":118700000,"label":"tower","mode":"am","squelch_cn_db":12}
//
// Responses:
//
//	200 {"ok":true,"index":N,"frequency_hz":...}
//	400 if frequency_hz is missing or out of range
//	503 if the conventional scanner isn't wired (no Voice SDR
//	    carved out for it)
func (s *Server) handleScannerManualTune(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	var req ManualTuneRequest
	if r.Body == nil || r.ContentLength == 0 {
		s.writeError(w, http.StatusBadRequest, "body required")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.FrequencyHz == 0 {
		s.writeError(w, http.StatusBadRequest, "frequency_hz required")
		return
	}
	// Sanity check: 25 MHz – 1.3 GHz is the practical RTL-SDR tuning
	// range. Looser is fine but a zero or absurd value almost
	// certainly means the operator mis-typed.
	if req.FrequencyHz < 25_000_000 || req.FrequencyHz > 1_300_000_000 {
		s.writeError(w, http.StatusBadRequest, "frequency_hz outside 25 MHz – 1.3 GHz tuning range")
		return
	}
	if req.Mode != "" && req.Mode != "fm" && req.Mode != "nfm" && req.Mode != "am" && req.Mode != "p25" {
		s.writeError(w, http.StatusBadRequest, "mode must be fm, nfm, am or p25")
		return
	}
	idx, ok := s.scanner.ManualTune(req)
	if !ok {
		s.writeError(w, http.StatusServiceUnavailable, "conventional scanner not configured")
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
//	DELETE /api/v1/scanner/manual_tune/{index}
//
// Responses:
//
//	200 {"ok":true,"index":N}
//	400 if the index can't be parsed
//	404 if the index isn't a temp channel
//	503 if the scanner isn't wired
func (s *Server) handleScannerClearManualTune(w http.ResponseWriter, r *http.Request) {
	if s.scanner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "scanner not wired")
		return
	}
	idxStr := r.PathValue("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		s.writeError(w, http.StatusBadRequest, "invalid index")
		return
	}
	if !s.scanner.ClearManualTune(idx) {
		s.writeError(w, http.StatusNotFound, "no such temporary channel")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "index": idx})
}
