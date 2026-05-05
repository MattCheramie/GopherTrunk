package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// handleSSE is a Server-Sent Events stream of every event published on the
// internal bus, JSON-encoded per the EventDTO envelope.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)

	// Send a comment line so curl shows progress immediately.
	_, _ = w.Write([]byte(": gophertrunk events stream\n\n"))
	flusher.Flush()

	sub := s.bus.Subscribe()
	defer sub.Close()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-sub.C:
			if !ok {
				return
			}
			dto := eventToDTO(ev)
			payload, err := json.Marshal(dto)
			if err != nil {
				s.log.Warn("api: SSE encode failed", "kind", ev.Kind, "err", err)
				continue
			}
			// SSE allows the optional "event:" line; clients can dispatch
			// per kind via addEventListener.
			fmt.Fprintf(w, "event: %s\n", sanitizeForHeader(string(ev.Kind)))
			// Encode payload across CRLF in case it contains newlines.
			for _, line := range strings.Split(string(payload), "\n") {
				fmt.Fprintf(w, "data: %s\n", line)
			}
			fmt.Fprint(w, "\n")
			flusher.Flush()
		}
	}
}

func sanitizeForHeader(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r == '\r' || r == '\n' {
			continue
		}
		out = append(out, byte(r))
	}
	return string(out)
}

// eventToDTO converts an internal events.Event to the wire envelope. The
// payload is mapped to a JSON-friendly DTO when the kind is recognised;
// otherwise the raw payload is passed through (useful for debugging
// future event kinds before the API is updated).
func eventToDTO(ev events.Event) EventDTO {
	dto := EventDTO{Kind: string(ev.Kind), Timestamp: ev.Timestamp}
	switch p := ev.Payload.(type) {
	case trunking.CallStart:
		dto.Payload = callStartToDTO(p)
	case trunking.CallEnd:
		dto.Payload = callEndToDTO(p)
	case trunking.Grant:
		dto.Payload = grantToDTO(p)
	default:
		dto.Payload = p
	}
	return dto
}
