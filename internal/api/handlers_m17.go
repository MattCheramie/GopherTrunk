package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// M17Provider is the read surface the m17-log endpoint consumes. The
// daemon implements it on top of storage.M17Log; tests substitute a
// fake.
type M17Provider interface {
	RecentM17LinkSetups(limit int) ([]storage.M17LinkSetup, error)
}

// M17LinkSetupDTO is the JSON wire shape for the m17-log endpoint.
type M17LinkSetupDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Src        string    `json:"src"`
	Dst        string    `json:"dst"`
	Mode       string    `json:"mode"`
	CAN        uint8     `json:"can"`
	Meta       string    `json:"meta,omitempty"`
	CRCOK      bool      `json:"crc_ok"`
	Body       string    `json:"body,omitempty"`
}

func m17LinkSetupToDTO(l storage.M17LinkSetup) M17LinkSetupDTO {
	return M17LinkSetupDTO{
		ID:         l.ID,
		ReceivedAt: l.ReceivedAt,
		Src:        l.Src,
		Dst:        l.Dst,
		Mode:       l.Mode,
		CAN:        l.CAN,
		Meta:       l.Meta,
		CRCOK:      l.CRCOK,
		Body:       l.Body,
	}
}

// handleM17LinkSetups answers GET /api/v1/m17/linksetups. Optional
// ?limit= (default 200, max 5000). 503 when the storage layer isn't
// wired (daemon started without storage.path).
func (s *Server) handleM17LinkSetups(w http.ResponseWriter, r *http.Request) {
	if s.m17 == nil {
		s.writeError(w, http.StatusServiceUnavailable, "m17 subsystem not enabled")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.m17.RecentM17LinkSetups(limit)
	if err != nil {
		s.log.Error("api: m17 linksetups", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]M17LinkSetupDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, m17LinkSetupToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}
