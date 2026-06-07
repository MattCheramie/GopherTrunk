package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// PagerProvider is the read surface the pager-log endpoint
// consumes. The daemon implements it on top of
// storage.PagerLog; tests substitute a fake.
type PagerProvider interface {
	RecentPagerMessages(limit int) ([]storage.PagerMessage, error)
}

// PagerMessageDTO is the JSON wire shape for the pager-log endpoint.
type PagerMessageDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Protocol   string    `json:"protocol"`
	RIC        uint32    `json:"ric"`
	Func       uint8     `json:"func"`
	Encoding   string    `json:"encoding"`
	Body       string    `json:"body"`
	Corrected  int       `json:"corrected"`
}

func pagerMessageToDTO(m storage.PagerMessage) PagerMessageDTO {
	proto := m.Protocol
	if proto == "" {
		proto = "pocsag"
	}
	return PagerMessageDTO{
		ID:         m.ID,
		ReceivedAt: m.ReceivedAt,
		Protocol:   proto,
		RIC:        m.RIC,
		Func:       m.Func,
		Encoding:   m.Encoding,
		Body:       m.Body,
		Corrected:  m.Corrected,
	}
}

// handlePagerMessages answers GET /api/v1/pager/messages.
// Optional ?limit= (default 200, max 5000).
func (s *Server) handlePagerMessages(w http.ResponseWriter, r *http.Request) {
	if s.pager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "pager subsystem not enabled (set storage.path in config to persist and view decoded messages)")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.pager.RecentPagerMessages(limit)
	if err != nil {
		s.log.Error("api: pager messages", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]PagerMessageDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, pagerMessageToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}
