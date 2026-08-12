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
	// ridAlias resolves a source radio ID to its human alias/name (from the
	// operator RID catalogue + live-decoded talker aliases). Optional — nil
	// leaves source_alpha unresolved. Set via SetRIDResolver.
	ridAlias func(radioID uint32) string
}

// SetRIDResolver installs the source-radio alias resolver used to stamp
// source_alpha onto call records — "who was talking", not just the RID.
// Safe to call once before Run. A nil fn (or nil resolver) leaves
// source_alpha NULL.
func (c *CallLog) SetRIDResolver(fn func(radioID uint32) string) { c.ridAlias = fn }

// sourceAlpha resolves the source radio's alias, or "" when no resolver is
// set or the radio is unknown / anonymous (id 0).
func (c *CallLog) sourceAlpha(radioID uint32) string {
	if c.ridAlias == nil || radioID == 0 {
		return ""
	}
	return c.ridAlias(radioID)
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
			case events.KindCallComplete:
				// The recorder finished writing the WAV; stamp its path onto the
				// row so History can offer playback. Fired after CallEnd, keyed by
				// the same (device_serial, started_at).
				if cc, ok := ev.Payload.(trunking.CallComplete); ok {
					if err := c.recordComplete(cc); err != nil {
						c.log.Warn("calllog: update recording path failed", "err", err, "device", cc.DeviceSerial)
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
    encrypted, algorithm_id, key_id, emergency, data_call, individual, timeslot, priority,
    device_serial, started_at, talkgroup_alpha, source_alpha
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := c.db.sql.Exec(q,
		cs.Grant.System, cs.Grant.Protocol, cs.Grant.GroupID, cs.Grant.SourceID, cs.Grant.FrequencyHz,
		boolToInt(cs.Grant.Encrypted), cs.Grant.AlgorithmID, cs.Grant.KeyID,
		boolToInt(cs.Grant.Emergency), boolToInt(cs.Grant.DataCall), boolToInt(cs.Grant.Individual),
		cs.Grant.Timeslot, cs.Grant.Priority,
		cs.DeviceSerial, cs.StartedAt.UnixNano(),
		alpha, c.sourceAlpha(cs.Grant.SourceID),
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
	// legitimate start-time value when the end grant is still zero (source_id,
	// algorithm_id, key_id, priority), and the CASE only ever latches
	// encrypted / emergency / individual on (never off). A DMR Tier III / P25
	// Phase 2 grant carries no Service Options, so the emergency + priority bits
	// arrive only on the traffic channel (UpdateSource) and must be persisted
	// here, the same way encrypted already is. individual latches on too: a
	// TETRA unit-to-unit destination cannot be flagged on first sighting, so a
	// call can be recognised as individual after its start row is written.
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
       emergency    = CASE WHEN ? != 0 THEN 1 ELSE emergency END,
       individual   = CASE WHEN ? != 0 THEN 1 ELSE individual END,
       algorithm_id = COALESCE(NULLIF(?, 0), algorithm_id),
       key_id       = COALESCE(NULLIF(?, 0), key_id),
       priority     = COALESCE(NULLIF(?, 0), priority),
       source_alpha = COALESCE(NULLIF(?, ''), source_alpha),
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
		boolToInt(ce.Grant.Emergency),
		boolToInt(ce.Grant.Individual),
		ce.Grant.AlgorithmID,
		ce.Grant.KeyID,
		ce.Grant.Priority,
		// Re-resolve from the end grant's SourceID: a compressed grant's RID is
		// backfilled mid-call, so the source alias may only be resolvable now.
		c.sourceAlpha(ce.Grant.SourceID),
		sig,
		evm,
		snr,
		ce.DeviceSerial, ce.StartedAt.UnixNano(),
	)
	return err
}

// recordComplete stamps the finished recording's path onto the matching call
// row. Keyed by (device_serial, started_at) exactly like recordEnd. Uses
// COALESCE(NULLIF(?, ''), recording_path) so an empty path never clobbers a
// value already written, mirroring the never-downgrade discipline elsewhere in
// this file. A CallComplete with no matching row (recording without a persisted
// call, e.g. replay tooling) updates nothing and is not an error.
func (c *CallLog) recordComplete(cc trunking.CallComplete) error {
	if cc.AudioPath == "" {
		return nil
	}
	const q = `
UPDATE call_log
   SET recording_path = COALESCE(NULLIF(?, ''), recording_path)
 WHERE device_serial = ? AND started_at = ?`
	_, err := c.db.sql.Exec(q, cc.AudioPath, cc.DeviceSerial, cc.StartedAt.UnixNano())
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
	// Individual is true for a unit-to-unit / private call, where GroupID is a
	// target radio address rather than a talkgroup. Omitted for group calls.
	Individual bool `json:"individual,omitempty"`
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
	// SourceAlpha is the resolved alias/name of the source radio (RID), from
	// the operator RID catalogue + live-decoded talker aliases; "" when
	// unresolved.
	SourceAlpha string `json:"source_alpha,omitempty"`
	// SignalDbFS is the call's mean received channel power in dBFS
	// (channel power, not calibrated RSSI or SNR). nil when unmeasured.
	SignalDbFS *float64 `json:"signal_dbfs,omitempty"`
	// EVMPct / SNRDb are the call's demod quality — RMS error-vector
	// magnitude (%) and estimated symbol SNR (dB). nil when unmeasured
	// (only P25 Phase 1 chains measure them). These ARE the demod-quality
	// figures to compare against another decoder, unlike SignalDbFS.
	EVMPct *float64 `json:"evm_pct,omitempty"`
	SNRDb  *float64 `json:"snr_db,omitempty"`
	// HasRecording is true when a finished recording is on disk for this call
	// (recording_path is set). The path itself is deliberately not serialised —
	// the UI plays via GET /api/v1/calls/{id}/audio, not a filesystem path.
	HasRecording bool `json:"has_recording,omitempty"`
}

// History queries the call_log with the supplied filter, newest-first.
func (d *DB) History(ctx context.Context, f HistoryFilter) ([]CallRow, error) {
	q := `SELECT id, system, protocol, group_id, source_id, frequency_hz,
	             encrypted, algorithm_id, key_id, emergency, data_call, individual, timeslot, priority,
	             device_serial, started_at, ended_at, duration_ms,
	             end_reason, talkgroup_alpha, source_alpha, signal_dbfs, evm_pct, snr_db,
	             recording_path
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
		var alpha, srcAlpha, recPath sql.NullString
		var sig, evm, snr sql.NullFloat64
		var enc, emer, data, indiv, algID, keyID, slot, prio int
		if err := rows.Scan(
			&r.ID, &r.System, &r.Protocol, &r.GroupID, &r.SourceID, &r.FrequencyHz,
			&enc, &algID, &keyID, &emer, &data, &indiv, &slot, &prio, &r.DeviceSerial,
			&startNs, &endNs, &durMs, &reason, &alpha, &srcAlpha, &sig, &evm, &snr,
			&recPath,
		); err != nil {
			return nil, err
		}
		r.HasRecording = recPath.Valid && recPath.String != ""
		r.Encrypted = enc != 0
		r.AlgorithmID = uint8(algID)
		r.KeyID = uint16(keyID)
		r.Emergency = emer != 0
		r.DataCall = data != 0
		r.Individual = indiv != 0
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
		if srcAlpha.Valid {
			r.SourceAlpha = srcAlpha.String
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

// RecordingPathByID returns the on-disk recording path for one call row, or
// "" when the call has no recording (or the id is unknown). The audio endpoint
// resolves a call id to a file this way so the filesystem path is never exposed
// to the client.
func (d *DB) RecordingPathByID(ctx context.Context, id int64) (string, error) {
	var p sql.NullString
	err := d.sql.QueryRowContext(ctx,
		`SELECT recording_path FROM call_log WHERE id = ?`, id).Scan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("storage: recording path for call %d: %w", id, err)
	}
	if !p.Valid {
		return "", nil
	}
	return p.String, nil
}

// RIDSummary is one aggregated per-radio (source_id) row derived from the
// persisted call_log. It is the durable, all-time view of a radio the Radio IDs
// list falls back to when the in-memory affiliation tracker has swept the radio
// out of its 30-minute idle window (issue: RID list "reset after each CC lock").
type RIDSummary struct {
	SourceID uint32
	System   string // system of the most recent call
	Protocol string // protocol of the most recent call
	// LastTalkgroup is the group_id of the most recent GROUP call (individual=0)
	// this radio was heard on; 0 when the radio has only ever made individual /
	// unit-to-unit calls (whose group_id is a target radio, not a talkgroup).
	LastTalkgroup uint32
	CallCount     uint64
	FirstSeen     time.Time
	LastSeen      time.Time
	// SourceAlpha is the resolved alias from the most recent call, "" if none.
	SourceAlpha string
}

// RIDSummaryFilter narrows RIDSummaries.
type RIDSummaryFilter struct {
	System   string // "" = all systems
	SourceID uint32 // 0 = all radios (filters call_log.source_id)
	Limit    int    // <= 0 = all radios
}

// RIDSummaries aggregates the call_log by source_id (radio ID) into a durable
// per-radio view: total call count, first/last-seen, and the system, protocol,
// alias and last GROUP talkgroup of the most recent call. Rows with source_id 0
// (unknown source) are excluded. Ordered by most-recently-seen first.
//
// This is the persisted basis for the Radio IDs list, so an observed radio no
// longer collapses out of the list once the live affiliation tracker's 30-minute
// idle TTL sweeps it — the list reflects every radio seen over the daemon's
// lifetime and survives control-channel re-locks and restarts.
func (d *DB) RIDSummaries(ctx context.Context, f RIDSummaryFilter) ([]RIDSummary, error) {
	// agg supplies count/first/last per radio; the outer join pulls the most
	// recent call's system/protocol/alias by matching its started_at, and a
	// correlated subquery picks the last GROUP call's talkgroup (skipping
	// individual calls, whose group_id is a target radio rather than a TG). The
	// optional system / single-source filters appear once per block; args are
	// bound in the order the blocks appear in the statement text (correlated
	// subquery c2, then agg, then outer c).
	filt := func(col string) (string, []any) {
		frag := ""
		var a []any
		if f.System != "" {
			frag += " AND " + col + "system = ?"
			a = append(a, f.System)
		}
		if f.SourceID != 0 {
			frag += " AND " + col + "source_id = ?"
			a = append(a, f.SourceID)
		}
		return frag, a
	}
	c2Frag, c2Args := filt("c2.")
	aggFrag, aggArgs := filt("")
	outerFrag, outerArgs := filt("c.")
	q := `SELECT c.source_id, agg.cnt, agg.first_seen, agg.last_seen,
	             c.system, c.protocol, c.source_alpha,
	             (SELECT c2.group_id FROM call_log c2
	                WHERE c2.source_id = c.source_id AND c2.individual = 0` + c2Frag + `
	                ORDER BY c2.started_at DESC LIMIT 1) AS last_tg
	      FROM call_log c
	      JOIN (SELECT source_id, COUNT(*) AS cnt,
	                   MIN(started_at) AS first_seen, MAX(started_at) AS last_seen
	              FROM call_log
	              WHERE source_id != 0` + aggFrag + `
	              GROUP BY source_id) agg
	        ON agg.source_id = c.source_id AND agg.last_seen = c.started_at
	      WHERE c.source_id != 0` + outerFrag + `
	      GROUP BY c.source_id
	      ORDER BY agg.last_seen DESC`
	args := []any{}
	args = append(args, c2Args...)
	args = append(args, aggArgs...)
	args = append(args, outerArgs...)
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit)
	}

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: query rid summaries: %w", err)
	}
	defer rows.Close()
	var out []RIDSummary
	for rows.Next() {
		var s RIDSummary
		var firstNs, lastNs int64
		var srcAlpha sql.NullString
		var lastTG sql.NullInt64
		if err := rows.Scan(&s.SourceID, &s.CallCount, &firstNs, &lastNs,
			&s.System, &s.Protocol, &srcAlpha, &lastTG); err != nil {
			return nil, err
		}
		s.FirstSeen = time.Unix(0, firstNs).UTC()
		s.LastSeen = time.Unix(0, lastNs).UTC()
		if srcAlpha.Valid {
			s.SourceAlpha = srcAlpha.String
		}
		if lastTG.Valid {
			s.LastTalkgroup = uint32(lastTG.Int64)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
