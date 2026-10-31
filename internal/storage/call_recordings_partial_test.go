package storage

import (
	"context"
	"path/filepath"
	"testing"
)

// A database created before call_recordings carried the partial-recording
// marker (#1288) gains the column on open; its existing rows read back as
// complete (0 lost) and the feed still lists them.
func TestCallRecordingsMigratesPartialColumn(t *testing.T) {
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
		ended_at INTEGER, UNIQUE(call_id, path))`); err != nil {
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
	if _, err := db.SQL().Exec(`INSERT INTO call_recordings (call_id, seq, path, started_at, iq_lost_chunks) VALUES (2, 0, '/r/b.wav', 2, 7)`); err != nil {
		t.Fatalf("insert after migration: %v", err)
	}
	recs, err := db.RecordingFeed(context.Background(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || recs[0].IQLostChunks != 0 || recs[1].IQLostChunks != 7 {
		t.Fatalf("feed after migration = %+v, want legacy row complete and the new one 7 lost", recs)
	}
}
