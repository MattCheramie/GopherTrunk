package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// A call recorded in per-transmission grouping rolls a file at every over, so
// a reply that lands inside the hangtime shares the call row with the radio
// that keyed before it. call_log.source_id keeps only the LAST talker, which
// attributed the first over's audio to the wrong radio; each segment now
// carries the radio heard in its own file.
func TestCallRecordingsCarryPerOverSource(t *testing.T) {
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

	const (
		firstTalker  = uint32(3200460)
		secondTalker = uint32(315529)
	)
	callStart := time.Now().UTC().Truncate(time.Microsecond)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "R13", Protocol: "p25", GroupID: 30508, SourceID: firstTalker,
			FrequencyHz: 167_850_000,
		},
		DeviceSerial: "wb:test0001:tap-0",
		StartedAt:    callStart,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	waitFor := func(cond func() bool) bool {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if cond() {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		return cond()
	}
	var id int64
	if !waitFor(func() bool {
		rows, _ := db.History(context.Background(), HistoryFilter{Limit: 1})
		if len(rows) == 1 {
			id = rows[0].ID
			return true
		}
		return false
	}) {
		t.Fatal("call row never landed")
	}

	// Over 1: the radio the grant named. Over 2: a different radio keyed
	// inside the hangtime, so the recorder rolled the file and the session's
	// grant carries the new talker.
	seg0 := cs.Grant
	seg1 := cs.Grant
	seg1.SourceID = secondTalker
	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{
		Grant: seg0, DeviceSerial: cs.DeviceSerial,
		StartedAt: callStart, EndedAt: callStart.Add(5 * time.Second),
		CallStartedAt: callStart, Segment: 0, AudioPath: "/rec/over0.wav",
	}})
	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{
		Grant: seg1, DeviceSerial: cs.DeviceSerial,
		StartedAt: callStart.Add(5 * time.Second), EndedAt: callStart.Add(8 * time.Second),
		CallStartedAt: callStart, Segment: 1, AudioPath: "/rec/over1.wav",
	}})

	var segs []RecordingSegment
	if !waitFor(func() bool {
		segs, _ = db.RecordingSegmentsByID(context.Background(), id)
		return len(segs) == 2
	}) {
		t.Fatalf("RecordingSegmentsByID = %d segments, want 2", len(segs))
	}
	if segs[0].SourceID != firstTalker {
		t.Errorf("segment 0 source = %d, want %d (the radio that opened the call)", segs[0].SourceID, firstTalker)
	}
	if segs[1].SourceID != secondTalker {
		t.Errorf("segment 1 source = %d, want %d (the radio that replied)", segs[1].SourceID, secondTalker)
	}

	var feed []FinalizedRecording
	if !waitFor(func() bool {
		feed, _ = db.RecordingFeed(context.Background(), 0, 10)
		return len(feed) == 2
	}) {
		t.Fatalf("RecordingFeed = %d rows, want 2", len(feed))
	}
	if feed[0].SourceID != firstTalker || feed[1].SourceID != secondTalker {
		t.Errorf("feed sources = %d, %d; want %d, %d",
			feed[0].SourceID, feed[1].SourceID, firstTalker, secondTalker)
	}
}

// A database created before call_recordings carried the per-segment talker
// gains the column on open; its existing rows read back as 0 (unknown) and
// the feed still lists them.
func TestCallRecordingsMigratesSourceColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL().Exec(`DROP TABLE call_recordings`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL().Exec(`CREATE TABLE call_recordings (
		id INTEGER PRIMARY KEY AUTOINCREMENT, call_id INTEGER NOT NULL,
		seq INTEGER NOT NULL, path TEXT NOT NULL, started_at INTEGER NOT NULL,
		ended_at INTEGER, iq_lost_chunks INTEGER NOT NULL DEFAULT 0,
		UNIQUE(call_id, path))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL().Exec(`INSERT INTO call_recordings (call_id, seq, path, started_at) VALUES (1, 0, '/r/a.wav', 1)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	db, err = Open(path)
	if err != nil {
		t.Fatalf("reopen legacy db: %v", err)
	}
	defer db.Close()
	if _, err := db.SQL().Exec(`INSERT INTO call_recordings (call_id, seq, path, started_at, source_id) VALUES (2, 0, '/r/b.wav', 2, 315529)`); err != nil {
		t.Fatalf("insert after migration: %v", err)
	}
	recs, err := db.RecordingFeed(context.Background(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || recs[0].SourceID != 0 || recs[1].SourceID != 315529 {
		t.Fatalf("feed after migration = %+v, want the legacy row 0 and the new one 315529", recs)
	}
}
