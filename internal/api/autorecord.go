package api

import "net/http"

// AutoRecordProvider actuates a manual raw-IQ auto-record capture on operator
// request. The daemon implements it over its iqAutoRecorder; a nil provider on
// the Server keeps the route returning 503.
type AutoRecordProvider interface {
	// Trigger fires one manual capture of the control SDR, bypassing the
	// automatic cooldown. The write happens asynchronously, so this returns
	// immediately.
	Trigger()
}

// handleSiglabAutoRecordTrigger fires a manual event-driven IQ capture.
//
// POST /api/v1/siglab/autorecord/trigger
//
// The capture is written to the configured baseband.auto_record.dir with a
// self-describing filename + metadata sidecar (same as the automatic
// triggers). The response is 202 Accepted because the write is asynchronous —
// the file appears in the auto-record directory shortly after.
func (s *Server) handleSiglabAutoRecordTrigger(w http.ResponseWriter, _ *http.Request) {
	if s.autoRecord == nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"siglab: auto-record not available (baseband.auto_record disabled or no control SDR)")
		return
	}
	s.autoRecord.Trigger()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "triggered"})
}
