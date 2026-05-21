package api

import "net/http"

// handleBroadcastStatus returns the outbound call-streaming subsystem's
// counters. Always 200 — when no broadcast feed is configured the
// response reports the subsystem as disabled so UIs can render a
// stable shape.
func (s *Server) handleBroadcastStatus(w http.ResponseWriter, _ *http.Request) {
	if s.broadcast == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": true,
		"stats":   s.broadcast.BroadcastStats(),
	})
}
