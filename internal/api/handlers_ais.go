package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// AISProvider is the read surface the ais-log endpoint consumes.
// The daemon implements it on top of storage.VesselLog; tests
// substitute a fake.
type AISProvider interface {
	RecentAISMessages(limit int) ([]storage.AISMessage, error)
}

// AISMessageDTO is the JSON wire shape for the ais-log endpoint.
// The Position fields and the static-data fields are omitted from
// the JSON when zero/empty so the wire stays compact for the
// position-only common case.
type AISMessageDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	MMSI       uint32    `json:"mmsi"`
	Type       string    `json:"type"`
	Body       string    `json:"body,omitempty"`

	Latitude         float64 `json:"latitude,omitempty"`
	Longitude        float64 `json:"longitude,omitempty"`
	SpeedOverGround  float64 `json:"sog,omitempty"`
	CourseOverGround float64 `json:"cog,omitempty"`
	Heading          int     `json:"heading,omitempty"`
	HasPosition      bool    `json:"has_position"`

	VesselName  string `json:"vessel_name,omitempty"`
	Callsign    string `json:"callsign,omitempty"`
	Destination string `json:"destination,omitempty"`
	ShipType    int    `json:"ship_type,omitempty"`
	IMO         uint32 `json:"imo,omitempty"`

	RawHex string `json:"raw_hex,omitempty"`
	FCSOK  bool   `json:"fcs_ok"`
}

func aisMessageToDTO(m storage.AISMessage) AISMessageDTO {
	return AISMessageDTO{
		ID:               m.ID,
		ReceivedAt:       m.ReceivedAt,
		MMSI:             m.MMSI,
		Type:             m.Type,
		Body:             m.Body,
		Latitude:         m.Latitude,
		Longitude:        m.Longitude,
		SpeedOverGround:  m.SpeedOverGround,
		CourseOverGround: m.CourseOverGround,
		Heading:          m.Heading,
		HasPosition:      m.HasPosition,
		VesselName:       m.VesselName,
		Callsign:         m.Callsign,
		Destination:      m.Destination,
		ShipType:         m.ShipType,
		IMO:              m.IMO,
		RawHex:           m.RawHex,
		FCSOK:            m.FCSOK,
	}
}

// handleAISMessages answers GET /api/v1/ais/vessels. Optional
// ?limit= (default 200, max 5000). 503 when the storage layer
// isn't wired (daemon started without storage.path).
func (s *Server) handleAISMessages(w http.ResponseWriter, r *http.Request) {
	if s.ais == nil {
		s.writeError(w, http.StatusServiceUnavailable, "ais subsystem not enabled (set storage.path in config to persist and view decoded messages)")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.ais.RecentAISMessages(limit)
	if err != nil {
		s.log.Error("api: ais messages", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]AISMessageDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, aisMessageToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}
