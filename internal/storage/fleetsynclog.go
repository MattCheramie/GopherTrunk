// FleetSync log writer — drains KindFleetSyncMessage events off the
// shared bus and writes one row per decoded ANI burst to the SQLite
// fleetsync_log table. Mirrors mdc1200log.go.
package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// FleetSyncMessage is one persisted decoded Kenwood FleetSync ANI burst.
type FleetSyncMessage struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Fleet      int       `json:"fleet"`   // transmitting radio's fleet number
	Unit       int       `json:"unit"`    // transmitting radio's unit ID
	IsFS2      bool      `json:"fs2"`     // decoded via the FleetSync II ECC path
	CRCOK      bool      `json:"crc_ok"`  // the block check validated
	RawHex     string    `json:"raw_hex"` // hex of the two recovered 32-bit words
	Body       string    `json:"body"`    // one-line summary
	// Serial and FrequencyHz name the receiver that decoded the burst —
	// the fleetsync.channels entry's SDR and channel — so an operator
	// running several channels can tell which one produced an ID (#1184).
	Serial      string `json:"serial"`
	FrequencyHz uint32 `json:"frequency_hz"`
}

// FleetSyncLog drains KindFleetSyncMessage events until ctx cancels or
// the bus closes.
type FleetSyncLog struct {
	*eventLog[FleetSyncMessage]
	db *DB
}

// NewFleetSyncLog wires the log to the bus. Subscription happens at
// construction so events published before Run() begins aren't lost.
func NewFleetSyncLog(db *DB, bus *events.Bus, logger *slog.Logger) (*FleetSyncLog, error) {
	if db == nil {
		return nil, errors.New("storage/fleetsynclog: DB is required")
	}
	f := &FleetSyncLog{db: db}
	el, err := newEventLog[FleetSyncMessage](bus, logger, events.KindFleetSyncMessage, "fleetsynclog", f.insert)
	if err != nil {
		return nil, err
	}
	f.eventLog = el
	return f, nil
}

func (f *FleetSyncLog) insert(msg FleetSyncMessage) error {
	at := msg.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	fs2, crcOK := 0, 0
	if msg.IsFS2 {
		fs2 = 1
	}
	if msg.CRCOK {
		crcOK = 1
	}
	_, err := f.db.SQL().Exec(
		`INSERT INTO fleetsync_log
		 (received_at, fleet, unit, fs2, body, raw_hex, crc_ok, serial, frequency_hz)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), msg.Fleet, msg.Unit, fs2, msg.Body, msg.RawHex, crcOK,
		msg.Serial, int64(msg.FrequencyHz),
	)
	return err
}

// Recent returns the most recent bursts, newest first, capped at
// limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (f *FleetSyncLog) Recent(limit int) ([]FleetSyncMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := f.db.SQL().Query(
		`SELECT id, received_at, fleet, unit, fs2, body, raw_hex, crc_ok, serial, frequency_hz
		 FROM fleetsync_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/fleetsynclog: query: %w", err)
	}
	defer rows.Close()
	var out []FleetSyncMessage
	for rows.Next() {
		var (
			msg   FleetSyncMessage
			ns    int64
			fs2   int
			crcOK int
			freq  int64
		)
		if err := rows.Scan(&msg.ID, &ns, &msg.Fleet, &msg.Unit, &fs2,
			&msg.Body, &msg.RawHex, &crcOK, &msg.Serial, &freq); err != nil {
			return nil, fmt.Errorf("storage/fleetsynclog: scan: %w", err)
		}
		msg.ReceivedAt = time.Unix(0, ns)
		msg.FrequencyHz = uint32(freq)
		msg.IsFS2 = fs2 != 0
		msg.CRCOK = crcOK != 0
		out = append(out, msg)
	}
	return out, rows.Err()
}
