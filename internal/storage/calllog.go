package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// CallLog persists trunking calls to the SQLite call_log table by
// subscribing to events.KindCallStart and events.KindCallEnd on the
// shared events bus.
//
// Rows are keyed by (device_serial, started_at). On CallStart we INSERT
// with a NULL ended_at; on CallEnd we UPDATE the matching row with the
// ended_at, duration, and end-reason. The unique index in the schema
// keeps duplicate-start events idempotent.
type CallLog struct {
	db        *DB
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// NewCallLog wires the call log to the bus. It subscribes immediately
// so callers can publish events before Run is called.
func NewCallLog(db *DB, bus *events.Bus, logger *slog.Logger) (*CallLog, error) {
	if db == nil {
		return nil, errors.New("storage/calllog: DB is required")
	}
	if bus == nil {
		return nil, errors.New("storage/calllog: events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	cl := &CallLog{
		db:      db,
		bus:     bus,
		log:     logger,
		runDone: make(chan struct{}),
	}
	cl.sub = bus.Subscribe()
	return cl, nil
}

// Run drains call.start / call.end events until ctx cancels.
func (c *CallLog) Run(ctx context.Context) error {
	defer close(c.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-c.sub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindCallStart:
				if cs, ok := ev.Payload.(trunking.CallStart); ok {
					if err := c.recordStart(cs); err != nil {
						c.log.Warn("calllog: insert start failed", "err", err, "device", cs.DeviceSerial)
					}
				}
			case events.KindCallEnd:
				if ce, ok := ev.Payload.(trunking.CallEnd); ok {
					if err := c.recordEnd(ce); err != nil {
						c.log.Warn("calllog: update end failed", "err", err, "device", ce.DeviceSerial)
					}
				}
			}
		}
	}
}

// Close releases the bus subscription and waits for Run to drain.
func (c *CallLog) Close() error {
	c.closeOnce.Do(func() {
		c.sub.Close()
		select {
		case <-c.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}

func (c *CallLog) recordStart(cs trunking.CallStart) error {
	alpha := ""
	if cs.Talkgroup != nil {
		alpha = cs.Talkgroup.AlphaTag
	}
	const q = `
INSERT OR REPLACE INTO call_log (
    system, protocol, group_id, source_id, frequency_hz,
    encrypted, algorithm_id, key_id, emergency, data_call, timeslot, priority,
    device_serial, started_at, talkgroup_alpha
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := c.db.sql.Exec(q,
		cs.Grant.System, cs.Grant.Protocol, cs.Grant.GroupID, cs.Grant.SourceID, cs.Grant.FrequencyHz,
		boolToInt(cs.Grant.Encrypted), cs.Grant.AlgorithmID, cs.Grant.KeyID,
		boolToInt(cs.Grant.Emergency), boolToInt(cs.Grant.DataCall), cs.Grant.Timeslot, cs.Grant.Priority,
		cs.DeviceSerial, cs.StartedAt.UnixNano(),
		alpha,
	)
	return err
}

func (c *CallLog) recordEnd(ce trunking.CallEnd) error {
	// Persist the call's final identity alongside the end timestamp. On
	// P25 Phase 2 compressed grants the SourceID (RID) starts at 0 and is
	// only backfilled mid-call on the traffic channel; ALGID/KID likewise
	// surface mid-call (P25 Phase 1 LDU2, Phase 2 EncryptionSync). The
	// engine backfills these onto the bound grant, so ce.Grant carries the
	// real values here — without this UPDATE the history row would keep the
	// grant-time placeholders (source_id=0, algorithm_id=0) and the RID
	// would never appear in call history (issue #696).
	//
	// The guards mirror the voice pool's never-downgrade semantics
	// (UpdateSource / UpdateEncryption): COALESCE(NULLIF(?, 0), col) keeps a
	// legitimate start-time value when the end grant is still zero, and the
	// CASE only ever upgrades encrypted (the spec defines no mid-call
	// decryption).
	// signal_dbfs uses COALESCE(?, signal_dbfs) with a NULL-valued bind
	// when the call carried no measurement, so a non-composer end (watchdog
	// timeout, preemption, shutdown) never clobbers a value an earlier
	// composer stamp may have written. nil → NULL → keeps the existing cell.
	var sig sql.NullFloat64
	if ce.SignalDbFS != nil {
		sig = sql.NullFloat64{Float64: *ce.SignalDbFS, Valid: true}
	}
	// evm_pct / snr_db use the same COALESCE(?, col) never-clobber discipline
	// as signal_dbfs: a non-composer end (or a non-Phase-1 chain) carries no
	// measurement, so a NULL bind keeps whatever an earlier composer stamp wrote.
	var evm, snr sql.NullFloat64
	if ce.EVMPct != nil {
		evm = sql.NullFloat64{Float64: *ce.EVMPct, Valid: true}
	}
	if ce.SNRDb != nil {
		snr = sql.NullFloat64{Float64: *ce.SNRDb, Valid: true}
	}
	const q = `
UPDATE call_log
   SET ended_at = ?,
       duration_ms = ?,
       end_reason = ?,
       source_id    = COALESCE(NULLIF(?, 0), source_id),
       encrypted    = CASE WHEN ? != 0 THEN 1 ELSE encrypted END,
       algorithm_id = COALESCE(NULLIF(?, 0), algorithm_id),
       key_id       = COALESCE(NULLIF(?, 0), key_id),
       signal_dbfs  = COALESCE(?, signal_dbfs),
       evm_pct      = COALESCE(?, evm_pct),
       snr_db       = COALESCE(?, snr_db)
 WHERE device_serial = ? AND started_at = ?`
	_, err := c.db.sql.Exec(q,
		ce.EndedAt.UnixNano(),
		ce.Duration().Milliseconds(),
		ce.Reason.String(),
		ce.Grant.SourceID,
		boolToInt(ce.Grant.Encrypted),
		ce.Grant.AlgorithmID,
		ce.Grant.KeyID,
		sig,
		evm,
		snr,
		ce.DeviceSerial, ce.StartedAt.UnixNano(),
	)
	return err
}

// HistoryFilter narrows a History query.
type HistoryFilter struct {
	System    string
	GroupID   uint32 // 0 = no filter (filters call_log.group_id)
	SourceID  uint32 // 0 = no filter (filters call_log.source_id — RID)
	Since     time.Time
	Until     time.Time
	Limit     int
	OnlyEnded bool
}

// CallRow is one row from the call_log table.
type CallRow struct {
	ID          int64  `json:"id"`
	System      string `json:"system"`
	Protocol    string `json:"protocol"`
	GroupID     uint32 `json:"group_id"`
	SourceID    uint32 `json:"source_id"`
	FrequencyHz uint32 `json:"frequency_hz"`
	Encrypted   bool   `json:"encrypted"`
	AlgorithmID uint8  `json:"algorithm_id"`
	KeyID       uint16 `json:"key_id"`
	Emergency   bool   `json:"emergency"`
	DataCall    bool   `json:"data_call"`
	// Timeslot is the 1-based DMR TDMA slot (0 = n/a, 1 = TS1, 2 = TS2).
	Timeslot uint8 `json:"timeslot,omitempty"`
	// Priority is the call's on-air signalled priority (service-options low
	// 3 bits, 0 = none/lowest). Distinct from a talkgroup's configured priority.
	Priority       uint8     `json:"priority,omitempty"`
	DeviceSerial   string    `json:"device_serial"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at,omitempty"` // zero if call still active
	DurationMs     int64     `json:"duration_ms,omitempty"`
	EndReason      string    `json:"end_reason,omitempty"`
	TalkgroupAlpha string    `json:"talkgroup_alpha,omitempty"`
	// SignalDbFS is the call's mean received channel power in dBFS
	// (channel power, not calibrated RSSI or SNR). nil when unmeasured.
	SignalDbFS *float64 `json:"signal_dbfs,omitempty"`
	// EVMPct / SNRDb are the call's demod quality — RMS error-vector
	// magnitude (%) and estimated symbol SNR (dB). nil when unmeasured
	// (only P25 Phase 1 chains measure them). These ARE the demod-quality
	// figures to compare against another decoder, unlike SignalDbFS.
	EVMPct *float64 `json:"evm_pct,omitempty"`
	SNRDb  *float64 `json:"snr_db,omitempty"`
}

// History queries the call_log with the supplied filter, newest-first.
func (d *DB) History(ctx context.Context, f HistoryFilter) ([]CallRow, error) {
	q := `SELECT id, system, protocol, group_id, source_id, frequency_hz,
	             encrypted, algorithm_id, key_id, emergency, data_call, timeslot, priority,
	             device_serial, started_at, ended_at, duration_ms,
	             end_reason, talkgroup_alpha, signal_dbfs, evm_pct, snr_db
	      FROM call_log WHERE 1=1`
	args := []any{}
	if f.System != "" {
		q += " AND system = ?"
		args = append(args, f.System)
	}
	if f.GroupID != 0 {
		q += " AND group_id = ?"
		args = append(args, f.GroupID)
	}
	if f.SourceID != 0 {
		q += " AND source_id = ?"
		args = append(args, f.SourceID)
	}
	if !f.Since.IsZero() {
		q += " AND started_at >= ?"
		args = append(args, f.Since.UnixNano())
	}
	if !f.Until.IsZero() {
		q += " AND started_at < ?"
		args = append(args, f.Until.UnixNano())
	}
	if f.OnlyEnded {
		q += " AND ended_at IS NOT NULL"
	}
	q += " ORDER BY started_at DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit)
	}

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: query history: %w", err)
	}
	defer rows.Close()
	var out []CallRow
	for rows.Next() {
		var r CallRow
		var startNs int64
		var endNs sql.NullInt64
		var durMs sql.NullInt64
		var reason sql.NullString
		var alpha sql.NullString
		var sig, evm, snr sql.NullFloat64
		var enc, emer, data, algID, keyID, slot, prio int
		if err := rows.Scan(
			&r.ID, &r.System, &r.Protocol, &r.GroupID, &r.SourceID, &r.FrequencyHz,
			&enc, &algID, &keyID, &emer, &data, &slot, &prio, &r.DeviceSerial,
			&startNs, &endNs, &durMs, &reason, &alpha, &sig, &evm, &snr,
		); err != nil {
			return nil, err
		}
		r.Encrypted = enc != 0
		r.AlgorithmID = uint8(algID)
		r.KeyID = uint16(keyID)
		r.Emergency = emer != 0
		r.DataCall = data != 0
		r.Timeslot = uint8(slot)
		r.Priority = uint8(prio)
		r.StartedAt = time.Unix(0, startNs).UTC()
		if endNs.Valid {
			r.EndedAt = time.Unix(0, endNs.Int64).UTC()
		}
		if durMs.Valid {
			r.DurationMs = durMs.Int64
		}
		if reason.Valid {
			r.EndReason = reason.String
		}
		if alpha.Valid {
			r.TalkgroupAlpha = alpha.String
		}
		if sig.Valid {
			v := sig.Float64
			r.SignalDbFS = &v
		}
		if evm.Valid {
			v := evm.Float64
			r.EVMPct = &v
		}
		if snr.Valid {
			v := snr.Float64
			r.SNRDb = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
