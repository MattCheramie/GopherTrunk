package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// MixerFrame is the wire shape of one channel-baseband spectrum snapshot
// for the web Mixer plot (OP25's Raw Mixer / Tuned Mixer tabs). It
// mirrors symbolscope.MixerFrame so the api package stays free of a
// dependency on the DSP package; the MixerProvider interface bridges the
// two.
//
// RawBins is the FFT of the channelized baseband as the receiver sees it
// (the carrier sits at the residual offset); TunedBins is the same window
// re-mixed by the receiver's carrier-offset estimate so a locked loop
// centres the carrier. Both are dBFS, FFT-shifted (Bins[0] is the lowest
// frequency, Bins[n/2] is DC), equal length.
type MixerFrame struct {
	TimestampNs     int64     `json:"ts_ns"`
	CenterHz        uint32    `json:"center_hz"`
	SampleRateHz    uint32    `json:"sample_rate_hz"`
	OffsetHz        int32     `json:"offset_hz"`
	CarrierOffsetHz float64   `json:"carrier_offset_hz"`
	RawBins         []float32 `json:"raw_bins"`
	TunedBins       []float32 `json:"tuned_bins"`
}

// MixerProvider is the daemon-side abstraction the mixer endpoint
// consumes. The daemon implements it on top of the iqtap broker map +
// internal/scanner/symbolscope; tests substitute a fake.
type MixerProvider interface {
	// OpenMixerStream starts a per-request symbol-recovery engine on the
	// named device with the Mixer FFT taps enabled. proto selects the
	// receiver ("p25-c4fm" / "p25-cqpsk"); offsetHz tunes an off-centre
	// channel down to baseband before channelizing. Returns the wire-frame
	// channel and a cleanup func the caller MUST invoke on disconnect.
	OpenMixerStream(ctx context.Context, serial, proto string, offsetHz int32) (<-chan MixerFrame, func(), error)
}

// handleMixerStream answers WS /api/v1/diag/mixer?device=...&proto=...&offset=....
//
// Frames arrive as JSON text messages. Server pings every 30 s to keep
// proxies from idling the socket. Connection stays open until the client
// disconnects or the device disappears. Mirrors handleSymbolStream.
func (s *Server) handleMixerStream(w http.ResponseWriter, r *http.Request) {
	if s.mixer == nil {
		s.writeError(w, http.StatusServiceUnavailable, "mixer subsystem not enabled")
		return
	}
	q := r.URL.Query()
	serial := q.Get("device")
	if serial == "" {
		s.writeError(w, http.StatusBadRequest, "device query parameter is required")
		return
	}
	proto := q.Get("proto")
	if proto == "" {
		proto = "p25-c4fm"
	}
	if !symbolProtos[proto] {
		s.writeError(w, http.StatusBadRequest, "unsupported proto (want p25-c4fm or p25-cqpsk)")
		return
	}
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
		s.log.Debug("api: mixer WS upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	frames, cleanup, err := s.mixer.OpenMixerStream(ctx, serial, proto, offset)
	if err != nil {
		s.log.Debug("api: mixer OpenMixerStream failed", "serial", serial, "err", err)
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
				s.log.Debug("api: mixer frame marshal", "err", err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
				return
			}
		}
	}
}
