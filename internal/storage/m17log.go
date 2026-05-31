// M17 link-setup log writer — drains KindM17LinkSetup events off the
// shared bus and writes one row per reassembled Link Setup Frame to the
// SQLite m17_log table. Mirrors dsclog.go / aprslog.go.
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

// M17LinkSetup is one persisted M17 link-setup frame.
type M17LinkSetup struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Src        string    `json:"src"`
	Dst        string    `json:"dst"`
	Mode       string    `json:"mode"` // "voice" | "data" | "voice+data" | "packet"
	CAN        uint8     `json:"can"`  // channel-access number
	Meta       string    `json:"meta"` // hex-encoded META block
	CRCOK      bool      `json:"crc_ok"`
	Body       string    `json:"body"` // display summary
}

// M17Log drains KindM17LinkSetup events until ctx cancels or the bus
// closes.
type M17Log struct {
	db        *DB
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// NewM17Log wires the log to the bus. Subscription happens at
// construction so events published before Run() begins aren't lost.
func NewM17Log(db *DB, bus *events.Bus, logger *slog.Logger) (*M17Log, error) {
	if db == nil {
		return nil, errors.New("storage/m17log: DB is required")
	}
	if bus == nil {
		return nil, errors.New("storage/m17log: events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	m := &M17Log{db: db, bus: bus, log: logger, runDone: make(chan struct{})}
	m.sub = bus.Subscribe()
	return m, nil
}

// Run drains KindM17LinkSetup events until ctx cancels or the bus closes.
func (m *M17Log) Run(ctx context.Context) error {
	defer close(m.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-m.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindM17LinkSetup {
				continue
			}
			ls, ok := ev.Payload.(M17LinkSetup)
			if !ok {
				continue
			}
			if err := m.insert(ls); err != nil {
				m.log.Error("m17log: insert failed", "err", err)
			}
		}
	}
}

func (m *M17Log) insert(ls M17LinkSetup) error {
	at := ls.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	crcOK := 0
	if ls.CRCOK {
		crcOK = 1
	}
	_, err := m.db.SQL().Exec(
		`INSERT INTO m17_log
		 (received_at, src, dst, mode, can, meta, crc_ok, body)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), ls.Src, ls.Dst, ls.Mode, ls.CAN, ls.Meta, crcOK, ls.Body,
	)
	return err
}

// Recent returns the most recent link-setup frames, newest first,
// capped at limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (m *M17Log) Recent(limit int) ([]M17LinkSetup, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := m.db.SQL().Query(
		`SELECT id, received_at, src, dst, mode, can, meta, crc_ok, body
		 FROM m17_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/m17log: query: %w", err)
	}
	defer rows.Close()
	var out []M17LinkSetup
	for rows.Next() {
		var (
			ls    M17LinkSetup
			ns    int64
			crcOK int
		)
		if err := rows.Scan(&ls.ID, &ns, &ls.Src, &ls.Dst, &ls.Mode,
			&ls.CAN, &ls.Meta, &crcOK, &ls.Body); err != nil {
			return nil, fmt.Errorf("storage/m17log: scan: %w", err)
		}
		ls.ReceivedAt = time.Unix(0, ns)
		ls.CRCOK = crcOK != 0
		out = append(out, ls)
	}
	return out, rows.Err()
}

// Close releases the bus subscription and waits for Run to drain.
func (m *M17Log) Close() error {
	m.closeOnce.Do(func() {
		m.sub.Close()
		select {
		case <-m.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
