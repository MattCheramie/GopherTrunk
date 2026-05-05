package storage

import (
	"context"
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
	publish := func(sys string, grp uint32, dt time.Duration, dev string) {
		bus.Publish(events.Event{
			Kind: events.KindCallStart,
			Payload: trunking.CallStart{
				Grant:        trunking.Grant{System: sys, GroupID: grp, FrequencyHz: 1},
				DeviceSerial: dev,
				StartedAt:    now.Add(dt),
			},
		})
	}
	publish("Alpha", 100, -3*time.Hour, "A")
	publish("Alpha", 200, -2*time.Hour, "B")
	publish("Bravo", 100, -1*time.Hour, "C")

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
