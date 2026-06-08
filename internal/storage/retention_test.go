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

// TestRetentionLogSweepEveryTable verifies the received_at delete path
// is valid SQL for every table in decoderLogTables plus the special
// location_log (reported_at) path — a typo or a table missing its
// timestamp column would surface here. Run against empty tables so it
// doesn't depend on each table's NOT NULL column set.
func TestRetentionLogSweepEveryTable(t *testing.T) {
	db := openTestDB(t)
	r, err := NewRetention(RetentionOptions{DB: db, LogRowMaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range decoderLogTables {
		if _, err := r.deleteOldLogRows(context.Background(), table); err != nil {
			t.Errorf("deleteOldLogRows(%q): %v", table, err)
		}
	}
	if _, err := r.deleteOldByColumn(context.Background(), "location_log", "reported_at"); err != nil {
		t.Errorf("deleteOldByColumn(location_log, reported_at): %v", err)
	}
}

// TestRetentionDeletesOldLogRows checks the age cutoff end-to-end
// through SweepOnce on a representative table (m17_log).
func TestRetentionDeletesOldLogRows(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	insert := func(ageHours int, src string) {
		t.Helper()
		_, err := db.sql.Exec(
			`INSERT INTO m17_log (received_at, src, dst) VALUES (?, ?, ?)`,
			now.Add(-time.Duration(ageHours)*time.Hour).UnixNano(), src, "M17-USA")
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(48, "OLD")
	insert(1, "FRESH")

	r, err := NewRetention(RetentionOptions{DB: db, LogRowMaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	r.SweepOnce(context.Background())

	var n int
	var src string
	if err := db.sql.QueryRow(`SELECT COUNT(*), COALESCE(MIN(src), '') FROM m17_log`).Scan(&n, &src); err != nil {
		t.Fatal(err)
	}
	if n != 1 || src != "FRESH" {
		t.Errorf("after sweep: %d rows (src=%q), want 1 (FRESH)", n, src)
	}
}

// TestRetentionLogSweepDisabledByDefault confirms a zero LogRowMaxAge
// leaves decoder-log rows untouched even when CallRowMaxAge is set.
func TestRetentionLogSweepDisabledByDefault(t *testing.T) {
	db := openTestDB(t)
	old := time.Now().Add(-100 * 24 * time.Hour).UnixNano()
	if _, err := db.sql.Exec(`INSERT INTO m17_log (received_at, src, dst) VALUES (?, ?, ?)`,
		old, "AB1CDE", "M17-USA"); err != nil {
		t.Fatal(err)
	}
	r, _ := NewRetention(RetentionOptions{DB: db, CallRowMaxAge: time.Hour})
	r.SweepOnce(context.Background())
	var n int
	if err := db.sql.QueryRow(`SELECT COUNT(*) FROM m17_log`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("m17_log: %d rows, want 1 (log sweep should be disabled)", n)
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
