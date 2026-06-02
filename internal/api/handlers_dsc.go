package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// DSCProvider is the read surface the dsc-log endpoint consumes.
// The daemon implements it on top of storage.DSCLog; tests
// substitute a fake.
type DSCProvider interface {
	RecentDSCMessages(limit int) ([]storage.DSCMessage, error)
}

// DSCMessageDTO is the JSON wire shape for the dsc-log endpoint.
// Position fields and distress-only fields stay omitted from the
// JSON when zero / empty so the routine-call wire stays compact.
type DSCMessageDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Format     string    `json:"format"`
	Category   string    `json:"category"`
	SelfMMSI   uint64    `json:"self_mmsi"`
	TargetMMSI uint64    `json:"target_mmsi,omitempty"`
	Nature     string    `json:"nature,omitempty"`
	TimeUTC    string    `json:"time_utc,omitempty"`

	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	HasPosition bool    `json:"has_position"`

	Body   string `json:"body,omitempty"`
	RawHex string `json:"raw_hex,omitempty"`
}

func dscMessageToDTO(m storage.DSCMessage) DSCMessageDTO {
	return DSCMessageDTO{
		ID:          m.ID,
		ReceivedAt:  m.ReceivedAt,
		Format:      m.Format,
		Category:    m.Category,
		SelfMMSI:    m.SelfMMSI,
		TargetMMSI:  m.TargetMMSI,
		Nature:      m.Nature,
		TimeUTC:     m.TimeUTC,
		Latitude:    m.Latitude,
		Longitude:   m.Longitude,
		HasPosition: m.HasPosition,
		Body:        m.Body,
		RawHex:      m.RawHex,
	}
}

// handleDSCMessages answers GET /api/v1/dsc/messages. Optional
// ?limit= (default 200, max 5000). 503 when the storage layer
// isn't wired (daemon started without storage.path).
func (s *Server) handleDSCMessages(w http.ResponseWriter, r *http.Request) {
	if s.dsc == nil {
		s.writeError(w, http.StatusServiceUnavailable, "dsc subsystem not enabled")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.dsc.RecentDSCMessages(limit)
	if err != nil {
		s.log.Error("api: dsc messages", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]DSCMessageDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, dscMessageToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}
