package storage

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "calls.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCallLogRecordsStartAndEnd(t *testing.T) {
	db := openTestDB(t)
	bus := events.NewBus(8)
	defer bus.Close()
	cl, err := NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Alpha", Protocol: "p25",
			GroupID: 1234, SourceID: 56789,
			FrequencyHz: 851_000_000,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP"},
		DeviceSerial: "VOICE-1",
		StartedAt:    startedAt,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	// Wait for the row to land.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), HistoryFilter{Limit: 1})
		if len(rows) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	rows, _ := db.History(context.Background(), HistoryFilter{Limit: 10})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.System != "Alpha" || r.GroupID != 1234 || r.DeviceSerial != "VOICE-1" {
		t.Errorf("row = %+v", r)
	}
	if r.TalkgroupAlpha != "FIRE-DISP" {
		t.Errorf("alpha = %q", r.TalkgroupAlpha)
	}
	if !r.EndedAt.IsZero() {
		t.Errorf("EndedAt should be zero on active call: %v", r.EndedAt)
	}

	endedAt := startedAt.Add(2 * time.Second)
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant:        cs.Grant,
			Talkgroup:    cs.Talkgroup,
			DeviceSerial: cs.DeviceSerial,
			StartedAt:    startedAt,
			EndedAt:      endedAt,
			Reason:       trunking.EndReasonNormal,
		},
	})

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), HistoryFilter{OnlyEnded: true})
		if len(rows) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	rows, _ = db.History(context.Background(), HistoryFilter{Limit: 10})
	if rows[0].EndReason != "normal" {
		t.Errorf("end reason = %q, want normal", rows[0].EndReason)
	}
	if rows[0].DurationMs != 2000 {
		t.Errorf("duration = %d, want 2000", rows[0].DurationMs)
	}
	if rows[0].EndedAt.IsZero() {
		t.Errorf("EndedAt missing")
	}
}

// TestCallLogPersistsTimeslot verifies the DMR TDMA slot survives the
// CallStart → call_log round-trip, so a carrier's two concurrent calls
// (TS1 + TS2) are distinguishable in history.
func TestCallLogPersistsTimeslot(t *testing.T) {
	db := openTestDB(t)
	bus := events.NewBus(8)
	defer bus.Close()
	cl, err := NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: trunking.Grant{
			System: "DMR3", Protocol: "dmr-tier3",
			GroupID: 100, SourceID: 7, FrequencyHz: 460_000_000, Timeslot: 2,
		},
		DeviceSerial: "VOICE-2",
		StartedAt:    time.Now().UTC(),
	}})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), HistoryFilter{Limit: 1})
		if len(rows) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	rows, _ := db.History(context.Background(), HistoryFilter{Limit: 1})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Timeslot != 2 {
		t.Errorf("Timeslot = %d, want 2 (TS2)", rows[0].Timeslot)
	}
}

func TestCallLogIdempotentStart(t *testing.T) {
	db := openTestDB(t)
	bus := events.NewBus(8)
	defer bus.Close()
	cl, err := NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "X", GroupID: 1, FrequencyHz: 1},
		DeviceSerial: "Y",
		StartedAt:    startedAt,
	}
	for i := 0; i < 3; i++ {
		bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	}

	// Eventually exactly one row.
	deadline := time.Now().Add(time.Second)
	var n int
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), HistoryFilter{Limit: 10})
		n = len(rows)
		if n == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("rows = %d, want 1", n)
}

func TestHistoryFilters(t *testing.T) {
	db := openTestDB(t)
	bus := events.NewBus(16)
	defer bus.Close()
	cl, err := NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	now := time.Now().UTC().Truncate(time.Second)
	publish := func(sys string, grp, src uint32, dt time.Duration, dev string) {
		bus.Publish(events.Event{
			Kind: events.KindCallStart,
			Payload: trunking.CallStart{
				Grant:        trunking.Grant{System: sys, GroupID: grp, SourceID: src, FrequencyHz: 1},
				DeviceSerial: dev,
				StartedAt:    now.Add(dt),
			},
		})
	}
	publish("Alpha", 100, 7, -3*time.Hour, "A")
	publish("Alpha", 200, 8, -2*time.Hour, "B")
	publish("Bravo", 100, 7, -1*time.Hour, "C")

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), HistoryFilter{Limit: 100})
		if len(rows) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	rows, err := db.History(context.Background(), HistoryFilter{System: "Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("Alpha rows = %d, want 2", len(rows))
	}

	rows, _ = db.History(context.Background(), HistoryFilter{GroupID: 100})
	if len(rows) != 2 {
		t.Errorf("group=100 rows = %d, want 2", len(rows))
	}

	// SourceID filter picks both calls from RID 7 across the two systems.
	rows, _ = db.History(context.Background(), HistoryFilter{SourceID: 7})
	if len(rows) != 2 {
		t.Errorf("source=7 rows = %d, want 2", len(rows))
	}
	for _, r := range rows {
		if r.SourceID != 7 {
			t.Errorf("source=7 returned row with SourceID=%d", r.SourceID)
		}
	}

	// Source + group together — only Alpha/100/7.
	rows, _ = db.History(context.Background(), HistoryFilter{SourceID: 7, GroupID: 100})
	if len(rows) != 2 {
		t.Errorf("source=7 group=100 rows = %d, want 2", len(rows))
	}

	rows, _ = db.History(context.Background(), HistoryFilter{Since: now.Add(-90 * time.Minute)})
	if len(rows) != 1 || rows[0].System != "Bravo" {
		t.Errorf("since rows = %+v", rows)
	}

	rows, _ = db.History(context.Background(), HistoryFilter{Limit: 1})
	if len(rows) != 1 {
		t.Errorf("limit 1 = %d", len(rows))
	}
	// Ordering is newest-first.
	if rows[0].System != "Bravo" {
		t.Errorf("newest-first: got %q, want Bravo", rows[0].System)
	}
}

// waitHistory polls db.History until pred returns true or the deadline
// passes, returning the last-read rows. Mirrors the inline poll loops the
// other tests use, factored out for the backfill cases below.
func waitHistory(t *testing.T, db *DB, f HistoryFilter, pred func([]CallRow) bool) []CallRow {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	var rows []CallRow
	for time.Now().Before(deadline) {
		rows, _ = db.History(context.Background(), f)
		if pred(rows) {
			return rows
		}
		time.Sleep(10 * time.Millisecond)
	}
	return rows
}

func runningCallLog(t *testing.T) (*DB, *events.Bus) {
	t.Helper()
	db := openTestDB(t)
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	cl, err := NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go cl.Run(ctx)
	return db, bus
}

// TestCallLogBackfillsSourceOnEnd models a P25 Phase 2 compressed grant:
// the call starts with SourceID=0 (the RID is absent on the control
// channel) and the real RID only surfaces mid-call, arriving on the bound
// grant by CallEnd. recordEnd must persist it so the RID lands in history
// (issue #696).
func TestCallLogBackfillsSourceOnEnd(t *testing.T) {
	db, bus := runningCallLog(t)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	startGrant := trunking.Grant{
		System: "P25P2", Protocol: "p25-phase2",
		GroupID: 4321, SourceID: 0, FrequencyHz: 851_000_000,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: startGrant, DeviceSerial: "VOICE-1", StartedAt: startedAt,
	}})

	rows := waitHistory(t, db, HistoryFilter{Limit: 1}, func(r []CallRow) bool { return len(r) == 1 })
	if len(rows) != 1 || rows[0].SourceID != 0 {
		t.Fatalf("start row = %+v, want one row with SourceID=0", rows)
	}

	endGrant := startGrant
	endGrant.SourceID = 778899 // backfilled mid-call on the traffic channel
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant:        endGrant,
		DeviceSerial: "VOICE-1",
		StartedAt:    startedAt,
		EndedAt:      startedAt.Add(time.Second),
		Reason:       trunking.EndReasonNormal,
	}})

	rows = waitHistory(t, db, HistoryFilter{Limit: 1}, func(r []CallRow) bool {
		return len(r) == 1 && r[0].SourceID == 778899
	})
	if len(rows) != 1 || rows[0].SourceID != 778899 {
		t.Fatalf("end row = %+v, want SourceID backfilled to 778899", rows)
	}
	// The RID is now queryable by the source filter the /rids history uses.
	rids, _ := db.History(context.Background(), HistoryFilter{SourceID: 778899})
	if len(rids) != 1 {
		t.Errorf("source filter rows = %d, want 1", len(rids))
	}
}

// TestCallLogBackfillsEncryptionOnEnd models encryption that only resolves
// mid-call (P25 Phase 1 LDU2 / Phase 2 EncryptionSync): the start grant
// carries no ALGID/KID, and recordEnd must persist the values that landed
// on the bound grant.
func TestCallLogBackfillsEncryptionOnEnd(t *testing.T) {
	db, bus := runningCallLog(t)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	startGrant := trunking.Grant{
		System: "P25P1", Protocol: "p25", GroupID: 100, SourceID: 42,
		FrequencyHz: 851_000_000, Encrypted: false,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: startGrant, DeviceSerial: "VOICE-2", StartedAt: startedAt,
	}})
	waitHistory(t, db, HistoryFilter{Limit: 1}, func(r []CallRow) bool { return len(r) == 1 })

	endGrant := startGrant
	endGrant.Encrypted = true
	endGrant.AlgorithmID = 0x84 // AES-256
	endGrant.KeyID = 7
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant:        endGrant,
		DeviceSerial: "VOICE-2",
		StartedAt:    startedAt,
		EndedAt:      startedAt.Add(time.Second),
		Reason:       trunking.EndReasonNormal,
	}})

	rows := waitHistory(t, db, HistoryFilter{Limit: 1}, func(r []CallRow) bool {
		return len(r) == 1 && r[0].Encrypted
	})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if !r.Encrypted || r.AlgorithmID != 0x84 || r.KeyID != 7 {
		t.Errorf("row = {enc:%v alg:%#x key:%d}, want {true 0x84 7}",
			r.Encrypted, r.AlgorithmID, r.KeyID)
	}
}

// TestCallLogEndDoesNotDowngrade confirms the never-downgrade guards: a
// known start-time RID survives a zero-source end grant, and a call that
// started encrypted stays encrypted even if the end grant reports clear.
func TestCallLogEndDoesNotDowngrade(t *testing.T) {
	db, bus := runningCallLog(t)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	startGrant := trunking.Grant{
		System: "Alpha", Protocol: "p25", GroupID: 1, SourceID: 5150,
		FrequencyHz: 851_000_000, Encrypted: true, AlgorithmID: 0x84, KeyID: 3,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: startGrant, DeviceSerial: "VOICE-3", StartedAt: startedAt,
	}})
	waitHistory(t, db, HistoryFilter{Limit: 1}, func(r []CallRow) bool { return len(r) == 1 })

	// End grant arrives blank (a later compressed-form update) — must not
	// clobber the known identity.
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant:        trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000},
		DeviceSerial: "VOICE-3",
		StartedAt:    startedAt,
		EndedAt:      startedAt.Add(time.Second),
		Reason:       trunking.EndReasonNormal,
	}})

	rows := waitHistory(t, db, HistoryFilter{OnlyEnded: true}, func(r []CallRow) bool { return len(r) == 1 })
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.SourceID != 5150 {
		t.Errorf("SourceID = %d, want 5150 preserved", r.SourceID)
	}
	if !r.Encrypted || r.AlgorithmID != 0x84 || r.KeyID != 3 {
		t.Errorf("identity downgraded: {enc:%v alg:%#x key:%d}, want {true 0x84 3}",
			r.Encrypted, r.AlgorithmID, r.KeyID)
	}
}

// TestCallLogRecordsEncryptedCallRID is the issue #696 decoupling
// regression: the call log writes the RID + encryption flag for an
// encrypted call regardless of any recording policy. The call log never
// consults the recorder or recordings.skip_encrypted — that option only
// gates WAV/raw file writing — so an encrypted call's identity always
// reaches history.
func TestCallLogRecordsEncryptedCallRID(t *testing.T) {
	db, bus := runningCallLog(t)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: trunking.Grant{
			System: "Secure", Protocol: "p25", GroupID: 911, SourceID: 13579,
			FrequencyHz: 851_000_000, Encrypted: true, AlgorithmID: 0x84, KeyID: 1,
		},
		DeviceSerial: "VOICE-4",
		StartedAt:    startedAt,
	}})

	rows := waitHistory(t, db, HistoryFilter{SourceID: 13579}, func(r []CallRow) bool { return len(r) == 1 })
	if len(rows) != 1 {
		t.Fatalf("encrypted-call rows by RID = %d, want 1 (history identity must be recording-independent)", len(rows))
	}
	if !rows[0].Encrypted {
		t.Errorf("encrypted flag not recorded for RID %d", rows[0].SourceID)
	}
}

// TestCallLogPersistsSignalDbFS verifies the composer-measured received
// channel power (dBFS) survives the CallEnd → call_log round-trip, and that
// a call ended with no measurement reads back nil — "unset" must be distinct
// from a legitimate 0 dBFS reading.
func TestCallLogPersistsSignalDbFS(t *testing.T) {
	db, bus := runningCallLog(t)

	f64 := func(v float64) *float64 { return &v }
	startedAt := time.Now().UTC().Truncate(time.Microsecond)

	// Call 1 carries a measured signal level on CallEnd.
	g1 := trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 1, SourceID: 10, FrequencyHz: 851_000_000}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: g1, DeviceSerial: "VOICE-1", StartedAt: startedAt,
	}})
	waitHistory(t, db, HistoryFilter{Limit: 1}, func(r []CallRow) bool { return len(r) == 1 })
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: g1, DeviceSerial: "VOICE-1", StartedAt: startedAt,
		EndedAt: startedAt.Add(time.Second), Reason: trunking.EndReasonNormal,
		SignalDbFS: f64(-42.5),
	}})

	rows := waitHistory(t, db, HistoryFilter{System: "Alpha", GroupID: 1, OnlyEnded: true}, func(r []CallRow) bool {
		return len(r) == 1 && r[0].SignalDbFS != nil
	})
	if len(rows) != 1 || rows[0].SignalDbFS == nil {
		t.Fatalf("call-1 row = %+v, want SignalDbFS populated", rows)
	}
	if got := *rows[0].SignalDbFS; math.Abs(got+42.5) > 0.01 {
		t.Errorf("SignalDbFS = %v, want -42.5", got)
	}

	// Call 2 ends with no measurement → column stays NULL → reads nil.
	started2 := startedAt.Add(time.Minute)
	g2 := trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 2, SourceID: 20, FrequencyHz: 851_000_000}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: g2, DeviceSerial: "VOICE-2", StartedAt: started2,
	}})
	waitHistory(t, db, HistoryFilter{System: "Alpha", GroupID: 2}, func(r []CallRow) bool { return len(r) == 1 })
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: g2, DeviceSerial: "VOICE-2", StartedAt: started2,
		EndedAt: started2.Add(time.Second), Reason: trunking.EndReasonTimeout,
		// SignalDbFS deliberately nil (watchdog/preemption-style end).
	}})
	rows = waitHistory(t, db, HistoryFilter{System: "Alpha", GroupID: 2, OnlyEnded: true}, func(r []CallRow) bool {
		return len(r) == 1
	})
	if len(rows) != 1 {
		t.Fatalf("call-2 rows = %d, want 1", len(rows))
	}
	if rows[0].SignalDbFS != nil {
		t.Errorf("unmeasured call SignalDbFS = %v, want nil (unset ≠ 0)", *rows[0].SignalDbFS)
	}
}

func TestOpenRejectsEmpty(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestOpenInMemory(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.History(context.Background(), HistoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %d, want 0", len(rows))
	}
}

// TestMigrationAddsEncryptionColumns builds a pre-#276 call_log table
// (no algorithm_id / key_id), then reopens it through Open and confirms
// the migration adds the columns and old rows still read back.
func TestMigrationAddsEncryptionColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	const legacy = `
CREATE TABLE call_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    system          TEXT    NOT NULL,
    protocol        TEXT    NOT NULL DEFAULT '',
    group_id        INTEGER NOT NULL,
    source_id       INTEGER NOT NULL DEFAULT 0,
    frequency_hz    INTEGER NOT NULL DEFAULT 0,
    encrypted       INTEGER NOT NULL DEFAULT 0,
    emergency       INTEGER NOT NULL DEFAULT 0,
    data_call       INTEGER NOT NULL DEFAULT 0,
    device_serial   TEXT    NOT NULL,
    started_at      INTEGER NOT NULL,
    ended_at        INTEGER,
    duration_ms     INTEGER,
    end_reason      TEXT,
    talkgroup_alpha TEXT
);`
	if _, err := raw.Exec(legacy); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO call_log (system, group_id, device_serial, started_at) VALUES (?, ?, ?, ?)`,
		"Legacy-Sys", 100, "dev0", int64(1000),
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open (migrate): %v", err)
	}
	defer db.Close()

	rows, err := db.History(context.Background(), HistoryFilter{})
	if err != nil {
		t.Fatalf("History after migration: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].System != "Legacy-Sys" {
		t.Errorf("system = %q, want Legacy-Sys", rows[0].System)
	}
	if rows[0].AlgorithmID != 0 || rows[0].KeyID != 0 {
		t.Errorf("migrated row: algorithm_id=%d key_id=%d, want 0/0",
			rows[0].AlgorithmID, rows[0].KeyID)
	}
	if rows[0].Timeslot != 0 {
		t.Errorf("migrated row: timeslot=%d, want 0 (column added with default)", rows[0].Timeslot)
	}
	if rows[0].SignalDbFS != nil {
		t.Errorf("migrated row: signal_dbfs=%v, want nil (nullable column added, old row NULL)", *rows[0].SignalDbFS)
	}

	// Reopening must be idempotent — the columns now exist.
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	db2.Close()
}
