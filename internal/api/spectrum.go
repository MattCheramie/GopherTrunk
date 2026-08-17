package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// SpectrumDevice is the per-SDR descriptor returned by
// GET /api/v1/spectrum/devices. Mirrors the proto shape but stays
// JSON-self-contained for the same reason SDRStatus does.
type SpectrumDevice struct {
	Serial       string `json:"serial"`
	Driver       string `json:"driver"`
	Product      string `json:"product,omitempty"`
	Role         string `json:"role"`
	CenterHz     uint32 `json:"center_hz"`
	SampleRateHz uint32 `json:"sample_rate_hz"`
	// P25Modulation is the configured P25 Phase 1 demod mode of the
	// system this SDR is decoding ("c4fm" or "cqpsk"), resolved by
	// matching the device's tuning against the configured systems. Lets
	// the web symbol/constellation panels auto-select the right mode
	// instead of asking the operator to pick it. Empty when no P25
	// Phase 1 system matches (e.g. a DMR-only device).
	P25Modulation string `json:"p25_modulation,omitempty"`
	// SymbolProto is the WS /api/v1/diag/symbols receiver selector for
	// the system this SDR is decoding — "p25-c4fm", "p25-cqpsk",
	// "tetra" or "dmr" — or empty when no configured system matches.
	// P25Modulation only ever describes P25 Phase 1 systems, so on a
	// TETRA/DMO/DMR rig the web panels' "Auto" mode fell back to
	// p25-c4fm and demodulated the carrier with the wrong receiver;
	// this field is protocol-aware so they pick the right one. Kept
	// separate from P25Modulation, which older clients still read.
	SymbolProto string `json:"symbol_proto,omitempty"`
	// ControlChannelHz is the configured control-channel frequency that
	// falls inside this SDR's passband (the one closest to centre when
	// several are in band), or 0 when none matches. The web Plots panels
	// rest their view on it so they default off the centre DC spike onto a
	// decodable channel instead of the useless SDR centre. (#557)
	ControlChannelHz uint32 `json:"control_channel_hz,omitempty"`
}

// SpectrumFrame is the wire shape of one frame on the WS stream.
type SpectrumFrame struct {
	TimestampNs  int64     `json:"ts_ns"`
	CenterHz     uint32    `json:"center_hz"`
	SampleRateHz uint32    `json:"sample_rate_hz"`
	Bins         []float32 `json:"bins"`
}

// SpectrumProvider is the daemon-side abstraction the api package
// consumes. The daemon (cmd/gophertrunk) implements it on top of the
// iqtap broker map; tests can substitute a fake. Kept narrow so the
// api package stays free of dependencies on internal/sdr.
type SpectrumProvider interface {
	// Devices returns the list of SDRs that can be streamed.
	Devices() []SpectrumDevice
	// OpenStream starts a per-request producer for the given device.
	// FFTSize must be a positive power of two; fps caps the frame
	// rate (zero picks a default). Returns an output channel that
	// closes when ctx cancels or the device disappears, and a
	// cleanup func the caller MUST invoke on disconnect.
	OpenStream(ctx context.Context, serial string, fftSize int, fps float64) (<-chan SpectrumFrame, func(), error)
	// Tune programs the named SDR's centre frequency in Hz. Used by
	// the web panel's click-to-tune handler. Returns an error if the
	// serial isn't known or the underlying device rejects the value.
	Tune(serial string, centerHz uint32) error
}

// TuneRequest is the body shape POST'd to /api/v1/spectrum/devices/{serial}/tune.
type TuneRequest struct {
	CenterHz uint32 `json:"center_hz"`
}

// handleSpectrumDevices answers GET /api/v1/spectrum/devices.
func (s *Server) handleSpectrumDevices(w http.ResponseWriter, r *http.Request) {
	if s.spectrum == nil {
		s.writeError(w, http.StatusServiceUnavailable, "spectrum subsystem not enabled")
		return
	}
	writeJSON(w, http.StatusOK, s.spectrum.Devices())
}

// handleSpectrumTune answers POST /api/v1/spectrum/devices/{serial}/tune.
// Body: {"center_hz": <uint32>}. Gated like every other mutation
// route — auth / allow-mutations checks run first.
//
// Daemon-internal "tune the SDR" surface for the web Spectrum
// panel's click-to-tune handler. External clients should use the
// rigctld TCP server instead — it speaks the standard Hamlib wire
// protocol against the same broker.
func (s *Server) handleSpectrumTune(w http.ResponseWriter, r *http.Request) {
	if s.spectrum == nil {
		s.writeError(w, http.StatusServiceUnavailable, "spectrum subsystem not enabled")
		return
	}
	serial := r.PathValue("serial")
	if serial == "" {
		s.writeError(w, http.StatusBadRequest, "serial path parameter is required")
		return
	}
	var body TuneRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.CenterHz == 0 {
		s.writeError(w, http.StatusBadRequest, "center_hz is required and must be > 0")
		return
	}
	if err := s.spectrum.Tune(serial, body.CenterHz); err != nil {
		s.log.Debug("api: spectrum tune", "serial", serial, "hz", body.CenterHz, "err", err)
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSpectrumStream answers WS /api/v1/spectrum/stream?device=...&fps=...&bins=....
//
// Frames arrive as JSON text messages; the connection stays open until
// the client disconnects, the device disappears, or the server
// shuts down. Server pings every 30 s to keep idle proxies from
// reaping the socket — same pattern as handleWS.
func (s *Server) handleSpectrumStream(w http.ResponseWriter, r *http.Request) {
	if s.spectrum == nil {
		s.writeError(w, http.StatusServiceUnavailable, "spectrum subsystem not enabled")
		return
	}

	q := r.URL.Query()
	serial := q.Get("device")
	if serial == "" {
		s.writeError(w, http.StatusBadRequest, "device query parameter is required")
		return
	}
	bins := parseIntQuery(q, "bins", 4096, 64, 16384)
	if bins&(bins-1) != 0 {
		s.writeError(w, http.StatusBadRequest, "bins must be a power of two")
		return
	}
	fps := parseFloatQuery(q, "fps", 10, 1, 30)

	upgrader := wsUpgrader
	if s.cors.enabled() {
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
	// Resolve the device BEFORE the upgrade. An unknown serial must be an HTTP
	// status, not a successful handshake followed by a close: browser clients
	// reset their reconnect backoff on `onopen`, so a handshake that always
	// succeeds makes the backoff dead code and a stale tab reconnects forever.
	if s.rejectUnknownDevice(w, serial) {
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Debug("api: spectrum WS upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	frames, cleanup, err := s.spectrum.OpenStream(ctx, serial, bins, fps)
	if err != nil {
		s.log.Debug("api: spectrum OpenStream failed", "serial", serial, "err", err)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
			websocket.CloseInternalServerErr, err.Error()))
		return
	}
	defer cleanup()

	// Drain client → server messages so close frames are processed
	// quickly.
	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				cancel()
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case f, ok := <-frames:
			if !ok {
				return
			}
			body, err := json.Marshal(f)
			if err != nil {
				s.log.Debug("api: spectrum frame marshal", "err", err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
				return
			}
		}
	}
}

// parseIntQuery returns the named query value as an int clamped to
// [lo, hi]. Missing or unparseable values fall back to def.
func parseIntQuery(q map[string][]string, name string, def, lo, hi int) int {
	v, ok := q[name]
	if !ok || len(v) == 0 || v[0] == "" {
		return def
	}
	n, err := strconv.Atoi(v[0])
	if err != nil {
		return def
	}
	if n < lo {
		n = lo
	}
	if n > hi {
		n = hi
	}
	return n
}

func parseFloatQuery(q map[string][]string, name string, def, lo, hi float64) float64 {
	v, ok := q[name]
	if !ok || len(v) == 0 || v[0] == "" {
		return def
	}
	n, err := strconv.ParseFloat(v[0], 64)
	if err != nil {
		return def
	}
	if n < lo {
		n = lo
	}
	if n > hi {
		n = hi
	}
	return n
}
