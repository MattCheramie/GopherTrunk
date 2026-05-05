package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB is a thin wrapper over *sql.DB that lets the call-log + retention
// helpers share a typed handle. The schema is migrated on Open.
type DB struct {
	sql *sql.DB
}

// Open creates (or opens) a SQLite database at path and applies the
// embedded schema migrations. The path's parent directory is created
// if missing.
//
// `:memory:` and the standard "file:..." DSN forms are passed through
// to the driver — useful for tests.
func Open(path string) (*DB, error) {
	if path == "" {
		return nil, errors.New("storage: db path is required")
	}
	if path != ":memory:" && !looksLikeDSN(path) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("storage: mkdir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	// SQLite is single-writer; cap to one connection so the call-log
	// writer doesn't fight the retention sweeper.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}
	d := &DB{sql: db}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// Close releases the connection.
func (d *DB) Close() error { return d.sql.Close() }

// SQL returns the underlying *sql.DB. Exposed so tests and future
// integrations (an /api/v1/calls/history handler, etc.) can run their
// own queries without adding a method here for every shape.
func (d *DB) SQL() *sql.DB { return d.sql }

func looksLikeDSN(s string) bool {
	return len(s) >= 5 && s[:5] == "file:"
}

const schema = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS call_log (
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
    started_at      INTEGER NOT NULL,  -- unix nanoseconds
    ended_at        INTEGER,
    duration_ms     INTEGER,
    end_reason      TEXT,
    talkgroup_alpha TEXT
);

CREATE INDEX IF NOT EXISTS idx_call_log_started ON call_log(started_at);
CREATE INDEX IF NOT EXISTS idx_call_log_system  ON call_log(system, started_at);
CREATE INDEX IF NOT EXISTS idx_call_log_group   ON call_log(group_id, started_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_call_log_active ON call_log(device_serial, started_at);
`

func (d *DB) migrate() error {
	_, err := d.sql.Exec(schema)
	if err != nil {
		return fmt.Errorf("storage: migrate: %w", err)
	}
	// Stamp v1; future migrations check this row before running.
	_, _ = d.sql.Exec(`INSERT OR IGNORE INTO schema_version(version) VALUES (1)`)
	return nil
}
