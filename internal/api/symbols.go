package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// SymbolFrame is the wire shape of one batch of recovered symbols for
// the web "Symbol" scope. It mirrors symbolscope.Frame so the api
// package stays free of a dependency on the DSP package; the
// SymbolProvider interface bridges the two.
//
// Soft is the pre-slicer soft waveform (empty when the demod path has no
// soft tap, e.g. P25 CQPSK); Dibits are the sliced decisions (0..3 when
// IsBits is false, 0..1 when true). When Soft is non-empty it is aligned
// index-for-index with Dibits.
//
// SymI/SymQ carry the complex symbol-decision points (the true
// constellation) on the linear/CQPSK path; empty on C4FM. When non-empty
// they are aligned index-for-index with Dibits.
type SymbolFrame struct {
	TimestampNs  int64     `json:"ts_ns"`
	SymbolRateHz float64   `json:"symbol_rate_hz"`
	CenterHz     uint32    `json:"center_hz"`
	OffsetHz     int32     `json:"offset_hz"`
	Soft         []float32 `json:"soft"`
	SymI         []float32 `json:"sym_i"`
	SymQ         []float32 `json:"sym_q"`
	Dibits       []uint8   `json:"dibits"`
	IsBits       bool      `json:"is_bits"`
	BaseIdx      int       `json:"base_idx"`
}

// SymbolProvider is the daemon-side abstraction the symbol endpoint
// consumes. The daemon implements it on top of the iqtap broker map +
// internal/scanner/symbolscope; tests substitute a fake.
type SymbolProvider interface {
	// OpenSymbolStream starts a per-request symbol-recovery engine on
	// the named device. proto selects the receiver ("p25-c4fm" /
	// "p25-cqpsk"); offsetHz tunes an off-centre channel down to
	// baseband before channelizing. Returns the wire-frame channel and
	// a cleanup func the caller MUST invoke on disconnect.
	OpenSymbolStream(ctx context.Context, serial, proto string, offsetHz int32) (<-chan SymbolFrame, func(), error)
}

// symbolProtos is the set of receiver selectors the symbol scope
// accepts today. Phase 1 ships P25 Phase 1 (C4FM soft+dibit, CQPSK
// dibit-only); more land as the per-receiver soft taps do.
var symbolProtos = map[string]bool{
	"p25-c4fm":  true,
	"p25-cqpsk": true,
}

// handleSymbolStream answers WS /api/v1/diag/symbols?device=...&proto=...&offset=....
//
// Frames arrive as JSON text messages. Server pings every 30 s to keep
// proxies from idling the socket. Connection stays open until the client
// disconnects or the device disappears. Mirrors handleDiagStream.
func (s *Server) handleSymbolStream(w http.ResponseWriter, r *http.Request) {
	if s.symbols == nil {
		s.writeError(w, http.StatusServiceUnavailable, "symbol subsystem not enabled")
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Debug("api: symbol WS upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	frames, cleanup, err := s.symbols.OpenSymbolStream(ctx, serial, proto, offset)
	if err != nil {
		s.log.Debug("api: symbol OpenSymbolStream failed", "serial", serial, "err", err)
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
				s.log.Debug("api: symbol frame marshal", "err", err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
				return
			}
		}
	}
}
