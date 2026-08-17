package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// IQPoint and IQFrame mirror the shapes the daemon's diag.Decimator
// produces. Defined here so the api package stays free of an import
// dependency on internal/dsp/diag; the DiagProvider interface bridges
// the two.
type IQPoint struct {
	I float32 `json:"i"`
	Q float32 `json:"q"`
}

// IQFrame is the wire shape of one decimated-IQ batch.
type IQFrame struct {
	TimestampNs  int64     `json:"ts_ns"`
	SampleRateHz uint32    `json:"sample_rate"`
	CenterHz     uint32    `json:"center_hz"`
	Points       []IQPoint `json:"points"`
	EnergyDBFS   float32   `json:"energy_dbfs"`
}

// DiagProvider is the daemon-side abstraction the diag endpoints
// consume. The daemon implements it on top of the iqtap broker
// map; tests substitute a fake.
type DiagProvider interface {
	// OpenIQStream starts a per-request decimator on the named
	// device. TargetRateSPS clamps the output rate (≤ device
	// sample_rate). offsetHz mixes a frequency offset (relative to
	// the device centre) down to baseband before decimation so an
	// off-centre channel can be viewed clear of the centre DC spike;
	// zero keeps the centre-tuned view. The provider clamps the
	// magnitude to the device's Nyquist. Returns the wire-frame
	// channel and a cleanup func the caller MUST invoke on disconnect.
	OpenIQStream(ctx context.Context, serial string, targetRateSPS uint32, offsetHz int32) (<-chan IQFrame, func(), error)
}

// handleDiagStream answers WS /api/v1/diag/iq?device=...&rate=....
//
// Frames arrive as JSON text messages. Server pings every 30 s to
// keep proxies from idling the socket. Connection stays open until
// the client disconnects or the device disappears.
func (s *Server) handleDiagStream(w http.ResponseWriter, r *http.Request) {
	if s.diag == nil {
		s.writeError(w, http.StatusServiceUnavailable, "diag subsystem not enabled")
		return
	}
	q := r.URL.Query()
	serial := q.Get("device")
	if serial == "" {
		s.writeError(w, http.StatusBadRequest, "device query parameter is required")
		return
	}
	rate := uint32(parseIntQuery(q, "rate", 2000, 100, 20000))
	// offset is the view shift in Hz relative to the SDR centre. The
	// provider re-clamps to the device's actual Nyquist; the wide
	// bound here just rejects nonsense. Signed: negative selects a
	// channel below centre.
	offset := int32(parseIntQuery(q, "offset", 0, -30_000_000, 30_000_000))

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
		s.log.Debug("api: diag WS upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	frames, cleanup, err := s.diag.OpenIQStream(ctx, serial, rate, offset)
	if err != nil {
		s.log.Debug("api: diag OpenIQStream failed", "serial", serial, "err", err)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
			websocket.CloseInternalServerErr, err.Error()))
		return
	}
	defer cleanup()

	// Drain client → server messages so close frames are processed.
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
				s.log.Debug("api: diag frame marshal", "err", err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
				return
			}
		}
	}
}
