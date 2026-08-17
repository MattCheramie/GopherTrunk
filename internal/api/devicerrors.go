package api

import (
	"errors"
	"net/http"
)

// ErrUnknownDevice is what a diagnostic provider returns when the requested SDR
// serial is not one this daemon has. Providers wrap it with %w so a handler can
// tell "you asked for a device that does not exist" apart from "that device
// exists but the stream could not start", which are a 404 and a 500
// respectively.
//
// This distinction is not cosmetic. The symbol/spectrum/diag WebSocket handlers
// used to complete the upgrade first and only then resolve the serial, so an
// unknown device produced a successful 101 handshake followed immediately by a
// 1011 close. Browser clients reset their reconnect backoff on `onopen`, so a
// handshake that always succeeds means the backoff never grows: a single stale
// tab left open across a daemon restart with different hardware reconnected two
// to four times a second indefinitely, and the daemon logged a line each time.
// Resolving the device BEFORE the upgrade turns that into a plain HTTP 404 the
// client can actually back off from — and act on.
var ErrUnknownDevice = errors.New("api: unknown SDR serial")

// rejectUnknownDevice answers 404 and reports true when serial is not a device
// this daemon has. Call it BEFORE upgrading a diagnostic WebSocket, so a stale
// client gets an HTTP status it can back off from instead of a handshake that
// succeeds and closes.
//
// The authority is SpectrumProvider.Devices(), which enumerates the same broker
// map every diagnostic provider looks the serial up in. When no spectrum
// provider is wired there is nothing to check against, so this reports false and
// the per-provider lookup remains the backstop — the point is to catch the
// common case cheaply, not to duplicate every provider's own validation.
func (s *Server) rejectUnknownDevice(w http.ResponseWriter, serial string) bool {
	if s.spectrum == nil {
		return false
	}
	for _, d := range s.spectrum.Devices() {
		if d.Serial == serial {
			return false
		}
	}
	s.writeError(w, http.StatusNotFound, "unknown SDR serial "+serial+
		" — this daemon is not running that device; re-read GET /api/v1/spectrum/devices")
	return true
}
