package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// touchFile creates a file with a modification time of `mtime`.
func touchFile(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestRetentionDeletesOldRows(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	insert := func(id, ageHours int) {
		t.Helper()
		_, err := db.sql.Exec(
			`INSERT INTO call_log (system, group_id, device_serial, started_at) VALUES (?, ?, ?, ?)`,
			"X", id, "DEV", now.Add(-time.Duration(ageHours)*time.Hour).UnixNano(),
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(1, 25)
	insert(2, 10)
	insert(3, 1)

	r, err := NewRetention(RetentionOptions{DB: db, CallRowMaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	r.SweepOnce(context.Background())

	rows, _ := db.History(context.Background(), HistoryFilter{Limit: 10})
	if len(rows) != 2 {
		t.Errorf("rows after sweep = %d, want 2 (only 25h-old should be deleted)", len(rows))
	}
	for _, r := range rows {
		if r.GroupID == 1 {
			t.Errorf("25h-old row should have been deleted")
		}
	}
}

func TestRetentionDeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	touchFile(t, filepath.Join(dir, "Alpha", "100", "old.wav"), now.Add(-48*time.Hour))
	touchFile(t, filepath.Join(dir, "Alpha", "100", "old.raw"), now.Add(-48*time.Hour))
	touchFile(t, filepath.Join(dir, "Alpha", "100", "fresh.wav"), now)
	touchFile(t, filepath.Join(dir, "Alpha", "100", "config.yaml"), now.Add(-48*time.Hour)) // not a recording

	r, err := NewRetention(RetentionOptions{
		FilesRoot:   dir,
		FilesMaxAge: 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.SweepOnce(context.Background())

	cases := map[string]bool{
		filepath.Join(dir, "Alpha", "100", "old.wav"):     false,
		filepath.Join(dir, "Alpha", "100", "old.raw"):     false,
		filepath.Join(dir, "Alpha", "100", "fresh.wav"):   true,
		filepath.Join(dir, "Alpha", "100", "config.yaml"): true, // preserved
	}
	for path, shouldExist := range cases {
		_, err := os.Stat(path)
		if shouldExist && err != nil {
			t.Errorf("%s should exist: %v", path, err)
		}
		if !shouldExist && err == nil {
			t.Errorf("%s should be deleted", path)
		}
	}
}

func TestRetentionRequiresAtLeastOne(t *testing.T) {
	if _, err := NewRetention(RetentionOptions{}); err == nil {
		t.Error("expected error when neither DB nor FilesRoot configured")
	}
}

func TestRetentionRunStopsOnContextCancel(t *testing.T) {
	r, _ := NewRetention(RetentionOptions{
		DB:            openTestDB(t),
		CallRowMaxAge: time.Hour,
		Interval:      10 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := r.Run(ctx); err != context.DeadlineExceeded {
		t.Errorf("Run err = %v, want DeadlineExceeded", err)
	}
}
