package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// APRSProvider is the read surface the aprs-log endpoint consumes.
// The daemon implements it on top of storage.APRSLog; tests
// substitute a fake.
type APRSProvider interface {
	RecentAPRSPackets(limit int) ([]storage.APRSPacket, error)
}

// APRSPacketDTO is the JSON wire shape for the aprs-log endpoint.
type APRSPacketDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Src        string    `json:"src"`
	Dst        string    `json:"dst"`
	Path       string    `json:"path,omitempty"`
	Type       string    `json:"type"`
	Body       string    `json:"body,omitempty"`
	Latitude   float64   `json:"latitude,omitempty"`
	Longitude  float64   `json:"longitude,omitempty"`
	RawInfo    string    `json:"raw_info,omitempty"`
	FCSOK      bool      `json:"fcs_ok"`
}

func aprsPacketToDTO(p storage.APRSPacket) APRSPacketDTO {
	return APRSPacketDTO{
		ID:         p.ID,
		ReceivedAt: p.ReceivedAt,
		Src:        p.Src,
		Dst:        p.Dst,
		Path:       p.Path,
		Type:       p.Type,
		Body:       p.Body,
		Latitude:   p.Latitude,
		Longitude:  p.Longitude,
		RawInfo:    p.RawInfo,
		FCSOK:      p.FCSOK,
	}
}

// handleAPRSPackets answers GET /api/v1/aprs/packets. Optional
// ?limit= (default 200, max 5000).
func (s *Server) handleAPRSPackets(w http.ResponseWriter, r *http.Request) {
	if s.aprs == nil {
		s.writeError(w, http.StatusServiceUnavailable, "aprs subsystem not enabled (set storage.path in config to persist and view decoded messages)")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.aprs.RecentAPRSPackets(limit)
	if err != nil {
		s.log.Error("api: aprs packets", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]APRSPacketDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, aprsPacketToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}
