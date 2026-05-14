package api

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// wsUpgrader trades cross-origin policy for usability — GopherTrunk is
// meant to run on a private network or localhost, and the API is
// read-only. Operators that expose the API publicly should put it
// behind a reverse proxy that enforces origin policy.
//
// When api.cors.allowed_origins is configured, the upgrader's
// CheckOrigin is overridden per-request in handleWS so the same
// allow-list governs both REST CORS and the WS upgrade.
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// handleWS bridges the events bus to a WebSocket connection. Each event
// is sent as a single JSON text frame (the same EventDTO shape as SSE).
// Clients should treat the connection as one-way (server → client); the
// server pings every 30 s to keep proxies from idling the socket.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := wsUpgrader
	if s.cors.enabled() {
		// When CORS is configured, gate the WS upgrade by the same
		// origin allow-list rather than the wide-open default. An
		// empty Origin header (curl, websocat, server-to-server) is
		// allowed through — non-browser clients can't be spoofed via
		// CSRF the way browser pages can.
		cors := s.cors
		upgrader.CheckOrigin = func(req *http.Request) bool {
			origin := req.Header.Get("Origin")
			if origin == "" {
				return true
			}
			_, ok := cors.originAllowed(origin)
			return ok
		}
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrader writes its own error response.
		s.log.Debug("api: WS upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	sub := s.bus.Subscribe()
	defer sub.Close()

	// Drain any client messages so close frames are processed quickly.
	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case ev, ok := <-sub.C:
			if !ok {
				return
			}
			dto := eventToDTO(ev)
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(dto); err != nil {
				s.log.Debug("api: WS write failed", "err", err)
				return
			}
		}
	}
}
