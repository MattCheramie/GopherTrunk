package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// FleetSyncProvider is the read surface the fleetsync-log endpoint
// consumes. The daemon implements it on top of storage.FleetSyncLog;
// tests substitute a fake.
type FleetSyncProvider interface {
	RecentFleetSyncMessages(limit int) ([]storage.FleetSyncMessage, error)
}

// FleetSyncMessageDTO is the JSON wire shape for the fleetsync-log
// endpoint. raw_hex and body stay omitted when empty so the wire stays
// compact.
type FleetSyncMessageDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Fleet      int       `json:"fleet"`
	Unit       int       `json:"unit"`
	FS2        bool      `json:"fs2"`
	Body       string    `json:"body,omitempty"`
	RawHex     string    `json:"raw_hex,omitempty"`
	CRCOK      bool      `json:"crc_ok"`
	// Serial / FrequencyHz: the fleetsync.channels receiver that decoded
	// the burst (#1184); omitted for rows logged before they were recorded.
	Serial      string `json:"serial,omitempty"`
	FrequencyHz uint32 `json:"frequency_hz,omitempty"`
}

func fleetSyncMessageToDTO(m storage.FleetSyncMessage) FleetSyncMessageDTO {
	return FleetSyncMessageDTO{
		ID:          m.ID,
		ReceivedAt:  m.ReceivedAt,
		Fleet:       m.Fleet,
		Unit:        m.Unit,
		FS2:         m.IsFS2,
		Body:        m.Body,
		RawHex:      m.RawHex,
		CRCOK:       m.CRCOK,
		Serial:      m.Serial,
		FrequencyHz: m.FrequencyHz,
	}
}

// handleFleetSyncMessages answers GET /api/v1/fleetsync/messages.
// Optional ?limit= (default 200, max 5000). 503 when the storage layer
// isn't wired (daemon started without storage.path).
func (s *Server) handleFleetSyncMessages(w http.ResponseWriter, r *http.Request) {
	if s.fleetsync == nil {
		s.writeError(w, http.StatusServiceUnavailable, "fleetsync subsystem not enabled (set storage.path in config to persist and view decoded messages)")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.fleetsync.RecentFleetSyncMessages(limit)
	if err != nil {
		s.log.Error("api: fleetsync messages", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]FleetSyncMessageDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, fleetSyncMessageToDTO(m))
	}
	writeJSON(w, http.StatusOK, out)
}
