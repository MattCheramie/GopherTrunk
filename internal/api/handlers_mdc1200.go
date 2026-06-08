package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// MDC1200Provider is the read surface the mdc1200-log endpoint
// consumes. The daemon implements it on top of storage.MDC1200Log;
// tests substitute a fake.
type MDC1200Provider interface {
	RecentMDC1200Messages(limit int) ([]storage.MDC1200Message, error)
}

// MDC1200MessageDTO is the JSON wire shape for the mdc1200-log
// endpoint. The operation label and raw-hex stay omitted when empty so
// the wire stays compact for the common PTT-ID burst.
type MDC1200MessageDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Op         uint8     `json:"op"`
	Arg        uint8     `json:"arg"`
	UnitID     uint16    `json:"unit_id"`
	Operation  string    `json:"operation,omitempty"`
	Body       string    `json:"body,omitempty"`
	RawHex     string    `json:"raw_hex,omitempty"`
	CRCOK      bool      `json:"crc_ok"`
}

func mdc1200MessageToDTO(m storage.MDC1200Message) MDC1200MessageDTO {
	return MDC1200MessageDTO{
		ID:         m.ID,
		ReceivedAt: m.ReceivedAt,
		Op:         m.Op,
		Arg:        m.Arg,
		UnitID:     m.UnitID,
		Operation:  m.Operation,
		Body:       m.Body,
		RawHex:     m.RawHex,
		CRCOK:      m.CRCOK,
	}
}

// handleMDC1200Messages answers GET /api/v1/mdc1200/messages. Optional
// ?limit= (default 200, max 5000). 503 when the storage layer isn't
// wired (daemon started without storage.path).
func (s *Server) handleMDC1200Messages(w http.ResponseWriter, r *http.Request) {
	if s.mdc1200 == nil {
		s.writeError(w, http.StatusServiceUnavailable, "mdc1200 subsystem not enabled (set storage.path in config to persist and view decoded messages)")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.mdc1200.RecentMDC1200Messages(limit)
	if err != nil {
		s.log.Error("api: mdc1200 messages", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]MDC1200MessageDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, mdc1200MessageToDTO(m))
	}
	writeJSON(w, http.StatusOK, out)
}
