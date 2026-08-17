package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// DibitArray marshals a dibit stream as a JSON number array. Go's
// encoding/json renders []byte/[]uint8 as a base64 string, which the web
// client (expecting number[]) silently drops — this forces the array
// form. Unmarshal uses the default []uint8 path (which accepts a JSON
// number array), so round-trips still work.
type DibitArray []uint8

// MarshalJSON emits the dibits as [n,n,...]; a nil/empty slice becomes []
// so the field is always an array, never null, on the wire.
func (d DibitArray) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return []byte("[]"), nil
	}
	buf := make([]byte, 0, len(d)*3+1)
	buf = append(buf, '[')
	for i, v := range d {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendUint(buf, uint64(v), 10)
	}
	buf = append(buf, ']')
	return buf, nil
}

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
	// EyeSoft is a bounded window of oversampled, AGC-scaled matched-
	// filter output (EyeSPS samples per symbol) for the C4FM eye diagram;
	// empty on CQPSK. Fold it over EyeSPS to render the 4-level eye.
	EyeSoft []float32  `json:"eye_soft"`
	EyeSPS  int        `json:"eye_sps"`
	Dibits  DibitArray `json:"dibits"`
	IsBits  bool       `json:"is_bits"`
	BaseIdx int        `json:"base_idx"`

	// Receiver-state metrics for the Tuning panel (see symbolscope.Frame).
	CarrierOffsetHz float64 `json:"carrier_offset_hz"`
	AGCLevel        float64 `json:"agc_level"`
	AGCTarget       float64 `json:"agc_target"`
	ClockMu         float64 `json:"clock_mu"`
	ClockSPS        float64 `json:"clock_sps"`
	CMAError        float64 `json:"cma_error"`
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
	"tetra":     true,
	"dmr":       true,
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
		s.writeError(w, http.StatusBadRequest, "unsupported proto (want p25-c4fm, p25-cqpsk, dmr, or tetra)")
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
