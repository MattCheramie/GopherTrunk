package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LocationLog persists geographic fixes to the SQLite location_log
// table by subscribing to events.KindLocation on the shared bus.
type LocationLog struct {
	db        *DB
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// LocationRow is one persisted fix, returned by Recent.
type LocationRow struct {
	ID         int64     `json:"id"`
	System     string    `json:"system"`
	Protocol   string    `json:"protocol"`
	RadioID    uint32    `json:"radio_id"`
	Talkgroup  uint32    `json:"talkgroup"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	SpeedKnots float64   `json:"speed_knots"`
	HeadingDeg float64   `json:"heading_deg"`
	ReportedAt time.Time `json:"reported_at"`
}

// NewLocationLog wires the location log to the bus. It subscribes
// immediately so callers can publish before Run starts.
func NewLocationLog(db *DB, bus *events.Bus, logger *slog.Logger) (*LocationLog, error) {
	if db == nil {
		return nil, errors.New("storage/locationlog: DB is required")
	}
	if bus == nil {
		return nil, errors.New("storage/locationlog: events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	l := &LocationLog{db: db, bus: bus, log: logger, runDone: make(chan struct{})}
	l.sub = bus.Subscribe()
	return l, nil
}

// Run drains KindLocation events until ctx cancels or the bus closes.
func (l *LocationLog) Run(ctx context.Context) error {
	defer close(l.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-l.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindLocation {
				continue
			}
			loc, ok := ev.Payload.(trunking.Location)
			if !ok {
				continue
			}
			if err := l.insert(loc); err != nil {
				l.log.Error("locationlog: insert failed", "err", err)
			}
		}
	}
}

func (l *LocationLog) insert(loc trunking.Location) error {
	at := loc.At
	if at.IsZero() {
		at = time.Now()
	}
	_, err := l.db.SQL().Exec(
		`INSERT INTO location_log
		 (system, protocol, radio_id, talkgroup, latitude, longitude,
		  speed_knots, heading_deg, reported_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		loc.System, loc.Protocol, loc.RadioID, loc.Talkgroup,
		loc.Latitude, loc.Longitude, loc.SpeedKnots, loc.HeadingDeg,
		at.UnixNano())
	return err
}

// Recent returns the most recent fixes, newest first, capped at limit.
func (l *LocationLog) Recent(limit int) ([]LocationRow, error) {
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	rows, err := l.db.SQL().Query(
		`SELECT id, system, protocol, radio_id, talkgroup, latitude,
		        longitude, speed_knots, heading_deg, reported_at
		 FROM location_log ORDER BY reported_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/locationlog: query: %w", err)
	}
	defer rows.Close()
	var out []LocationRow
	for rows.Next() {
		var r LocationRow
		var ns int64
		if err := rows.Scan(&r.ID, &r.System, &r.Protocol, &r.RadioID,
			&r.Talkgroup, &r.Latitude, &r.Longitude, &r.SpeedKnots,
			&r.HeadingDeg, &ns); err != nil {
			return nil, fmt.Errorf("storage/locationlog: scan: %w", err)
		}
		r.ReportedAt = time.Unix(0, ns)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Close releases the bus subscription and waits for Run to drain.
func (l *LocationLog) Close() error {
	l.closeOnce.Do(func() {
		l.sub.Close()
		select {
		case <-l.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
