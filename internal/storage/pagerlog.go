// PagerLog persists POCSAG (and eventually FLEX) pager messages to
// the SQLite pager_log table by subscribing to events.KindPagerMessage
// on the shared bus. The decoder pipeline publishes one event per
// fully-reassembled page; this consumer writes a row and the web
// panel queries the recent rows via /api/v1/pager/messages.
package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// PagerMessage is one persisted page. Mirrors the pocsag.Page shape
// but with the "encoding" enum widened to a string for storage +
// JSON convenience.
type PagerMessage struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Protocol   string    `json:"protocol"` // "pocsag" | "flex"
	RIC        uint32    `json:"ric"`
	Func       uint8     `json:"func"` // 0..3 = A..D
	Encoding   string    `json:"encoding"`
	Body       string    `json:"body"`
	Corrected  int       `json:"corrected"`
}

// PagerLog drains KindPagerMessage off the bus and writes one row
// per page. Mirrors LocationLog's Run/Close lifecycle.
type PagerLog struct {
	db        *DB
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// NewPagerLog wires the log to the bus. The bus subscription happens
// at construction time so the decoder can publish before Run begins
// without losing events.
func NewPagerLog(db *DB, bus *events.Bus, logger *slog.Logger) (*PagerLog, error) {
	if db == nil {
		return nil, errors.New("storage/pagerlog: DB is required")
	}
	if bus == nil {
		return nil, errors.New("storage/pagerlog: events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	p := &PagerLog{db: db, bus: bus, log: logger, runDone: make(chan struct{})}
	p.sub = bus.Subscribe()
	return p, nil
}

// Run drains KindPagerMessage events until ctx cancels or the bus
// closes.
func (p *PagerLog) Run(ctx context.Context) error {
	defer close(p.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-p.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindPagerMessage {
				continue
			}
			msg, ok := ev.Payload.(PagerMessage)
			if !ok {
				continue
			}
			if err := p.insert(msg); err != nil {
				p.log.Error("pagerlog: insert failed", "err", err)
			}
		}
	}
}

func (p *PagerLog) insert(m PagerMessage) error {
	at := m.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	proto := m.Protocol
	if proto == "" {
		proto = "pocsag"
	}
	_, err := p.db.SQL().Exec(
		`INSERT INTO pager_log
		 (received_at, protocol, ric, func, encoding, body, corrected)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), proto, m.RIC, m.Func, m.Encoding, m.Body, m.Corrected,
	)
	return err
}

// Recent returns the most recent pages, newest first, capped at
// limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (p *PagerLog) Recent(limit int) ([]PagerMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := p.db.SQL().Query(
		`SELECT id, received_at, protocol, ric, func, encoding, body, corrected
		 FROM pager_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/pagerlog: query: %w", err)
	}
	defer rows.Close()
	var out []PagerMessage
	for rows.Next() {
		var (
			m  PagerMessage
			ns int64
		)
		if err := rows.Scan(&m.ID, &ns, &m.Protocol, &m.RIC, &m.Func, &m.Encoding,
			&m.Body, &m.Corrected); err != nil {
			return nil, fmt.Errorf("storage/pagerlog: scan: %w", err)
		}
		m.ReceivedAt = time.Unix(0, ns)
		out = append(out, m)
	}
	return out, rows.Err()
}

// Close releases the bus subscription and waits for Run to drain.
func (p *PagerLog) Close() error {
	p.closeOnce.Do(func() {
		p.sub.Close()
		select {
		case <-p.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
