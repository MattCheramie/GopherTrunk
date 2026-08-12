// APRS log writer — drains KindAPRSPacket events off the shared bus
// and writes one row per decoded packet to the SQLite aprs_log
// table. Mirrors pagerlog.go and locationlog.go in structure.
package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// APRSPacket is one persisted decoded APRS packet. Mirrors the
// aprs.Packet shape with the AX.25 envelope flattened to a
// callsign-string triple (src, dst, path) and the sub-payload
// summarised into a single "body" field for compact rendering.
// Latitude / Longitude are populated only for position-bearing
// types and stay 0 otherwise.
type APRSPacket struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Src        string    `json:"src"`
	Dst        string    `json:"dst"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	Body       string    `json:"body"`
	Latitude   float64   `json:"latitude,omitempty"`
	Longitude  float64   `json:"longitude,omitempty"`
	RawInfo    string    `json:"raw_info"`
	FCSOK      bool      `json:"fcs_ok"`
}

// APRSLog drains KindAPRSPacket events to the aprs_log table. The bus
// subscription / drain / close lifecycle is the shared eventLog[APRSPacket];
// this type adds only the APRS-specific insert and Recent.
type APRSLog struct {
	*eventLog[APRSPacket]
	db *DB
}

// NewAPRSLog wires the log to the bus. Subscription happens at
// construction so events published before Run() begins aren't lost.
func NewAPRSLog(db *DB, bus *events.Bus, logger *slog.Logger) (*APRSLog, error) {
	if db == nil {
		return nil, errors.New("storage/aprslog: DB is required")
	}
	a := &APRSLog{db: db}
	el, err := newEventLog[APRSPacket](bus, logger, events.KindAPRSPacket, "aprslog", a.insert)
	if err != nil {
		return nil, err
	}
	a.eventLog = el
	return a, nil
}

func (a *APRSLog) insert(p APRSPacket) error {
	at := p.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	fcsOK := 0
	if p.FCSOK {
		fcsOK = 1
	}
	_, err := a.db.SQL().Exec(
		`INSERT INTO aprs_log
		 (received_at, src, dst, path, type, body, latitude, longitude, raw_info, fcs_ok)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), p.Src, p.Dst, p.Path, p.Type, p.Body,
		p.Latitude, p.Longitude, p.RawInfo, fcsOK,
	)
	return err
}

// Recent returns the most recent packets, newest first, capped at
// limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (a *APRSLog) Recent(limit int) ([]APRSPacket, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := a.db.SQL().Query(
		`SELECT id, received_at, src, dst, path, type, body,
		        latitude, longitude, raw_info, fcs_ok
		 FROM aprs_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/aprslog: query: %w", err)
	}
	defer rows.Close()
	var out []APRSPacket
	for rows.Next() {
		var (
			p     APRSPacket
			ns    int64
			fcsOK int
		)
		if err := rows.Scan(&p.ID, &ns, &p.Src, &p.Dst, &p.Path,
			&p.Type, &p.Body, &p.Latitude, &p.Longitude, &p.RawInfo, &fcsOK); err != nil {
			return nil, fmt.Errorf("storage/aprslog: scan: %w", err)
		}
		p.ReceivedAt = time.Unix(0, ns)
		p.FCSOK = fcsOK != 0
		out = append(out, p)
	}
	return out, rows.Err()
}
