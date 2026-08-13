// DSC log writer — drains KindDSCMessage events off the shared bus
// and writes one row per decoded sequence to the SQLite dsc_log
// table. Mirrors aprslog.go / vessellog.go / pagerlog.go.
package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// DSCMessage is one persisted decoded DSC sequence.
type DSCMessage struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Format     string    `json:"format"`   // "distress" | "all-ships" | "individual" | "group" | ...
	Category   string    `json:"category"` // "distress" | "urgency" | "safety" | "routine"
	SelfMMSI   uint64    `json:"self_mmsi"`
	TargetMMSI uint64    `json:"target_mmsi,omitempty"`
	Nature     string    `json:"nature,omitempty"`   // distress nature ("fire", "sinking", ...)
	TimeUTC    string    `json:"time_utc,omitempty"` // HH:MM, distress only

	// Position fields — populated only on distress alerts that
	// included a position field with a non-sentinel value.
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	HasPosition bool    `json:"has_position"`

	Body   string `json:"body"`    // type-specific summary
	RawHex string `json:"raw_hex"` // hex-encoded 7-bit symbol stream
}

// DSCLog drains KindDSCMessage events until ctx cancels or the bus
// closes.
type DSCLog struct {
	*eventLog[DSCMessage]
	db *DB
}

// NewDSCLog wires the log to the bus. Subscription happens at
// construction so events published before Run() begins aren't lost.
func NewDSCLog(db *DB, bus *events.Bus, logger *slog.Logger) (*DSCLog, error) {
	if db == nil {
		return nil, errors.New("storage/dsclog: DB is required")
	}
	d := &DSCLog{db: db}
	el, err := newEventLog[DSCMessage](bus, logger, events.KindDSCMessage, "dsclog", d.insert)
	if err != nil {
		return nil, err
	}
	d.eventLog = el
	return d, nil
}

func (d *DSCLog) insert(m DSCMessage) error {
	at := m.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	hasPos := 0
	if m.HasPosition {
		hasPos = 1
	}
	_, err := d.db.SQL().Exec(
		`INSERT INTO dsc_log
		 (received_at, format, category, self_mmsi, target_mmsi,
		  nature, time_utc, latitude, longitude, has_position,
		  body, raw_hex)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), m.Format, m.Category, m.SelfMMSI, m.TargetMMSI,
		m.Nature, m.TimeUTC, m.Latitude, m.Longitude, hasPos,
		m.Body, m.RawHex,
	)
	return err
}

// Recent returns the most recent messages, newest first, capped at
// limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (d *DSCLog) Recent(limit int) ([]DSCMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := d.db.SQL().Query(
		`SELECT id, received_at, format, category, self_mmsi,
		        target_mmsi, nature, time_utc, latitude, longitude,
		        has_position, body, raw_hex
		 FROM dsc_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/dsclog: query: %w", err)
	}
	defer rows.Close()
	var out []DSCMessage
	for rows.Next() {
		var (
			m      DSCMessage
			ns     int64
			hasPos int
		)
		if err := rows.Scan(&m.ID, &ns, &m.Format, &m.Category,
			&m.SelfMMSI, &m.TargetMMSI, &m.Nature, &m.TimeUTC,
			&m.Latitude, &m.Longitude, &hasPos, &m.Body, &m.RawHex); err != nil {
			return nil, fmt.Errorf("storage/dsclog: scan: %w", err)
		}
		m.ReceivedAt = time.Unix(0, ns)
		m.HasPosition = hasPos != 0
		out = append(out, m)
	}
	return out, rows.Err()
}
