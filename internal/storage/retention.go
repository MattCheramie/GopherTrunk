package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// decoderLogTables are the append-only decoder log tables keyed by a
// received_at (unix-nanosecond) column. The sweeper deletes rows older
// than LogRowMaxAge from each. call_log is swept separately because its
// timestamp column is started_at.
var decoderLogTables = []string{
	"pager_log",
	"aprs_log",
	"vessel_log",
	"dsc_log",
	"aircraft_log",
	"mdc1200_log",
	"m17_log",
	"location_log",
}

// Retention deletes old data on a schedule:
//  1. call_log rows with started_at older than CallRowMaxAge.
//  2. decoder log-table rows (pager_log, aprs_log, vessel_log, dsc_log,
//     aircraft_log, mdc1200_log, m17_log, location_log) with
//     received_at older than LogRowMaxAge.
//  3. WAV / raw files under FilesRoot whose modification time is older
//     than FilesMaxAge.
//
// File deletion is opt-in by setting FilesRoot; an empty value skips
// the filesystem sweep. The sweeper is idempotent and safe to run
// concurrently with the log writers (SQLite serialises).
type Retention struct {
	db       *DB
	log      *slog.Logger
	files    string
	dbAge    time.Duration
	logAge   time.Duration
	filesAge time.Duration
	interval time.Duration
}

// RetentionOptions configure a Retention sweeper.
type RetentionOptions struct {
	DB *DB
	// FilesRoot is the directory the voice recorder writes WAV / raw
	// files under. Empty disables the filesystem sweep.
	FilesRoot string
	// CallRowMaxAge: call_log rows with started_at older than this are
	// deleted. Zero (the default) disables call-row deletion.
	CallRowMaxAge time.Duration
	// LogRowMaxAge: decoder log-table rows (pager_log, aprs_log,
	// vessel_log, dsc_log, aircraft_log, mdc1200_log, m17_log,
	// location_log) with received_at older than this are deleted. Zero
	// (the default) disables decoder-log deletion.
	LogRowMaxAge time.Duration
	// FilesMaxAge: files older than this (mtime) are deleted. Zero
	// disables file deletion.
	FilesMaxAge time.Duration
	// Interval between sweeps. Default 1 h.
	Interval time.Duration
	Log      *slog.Logger
}

func NewRetention(opts RetentionOptions) (*Retention, error) {
	if opts.DB == nil && opts.FilesRoot == "" {
		return nil, errors.New("storage/retention: at least one of DB or FilesRoot is required")
	}
	if opts.Interval <= 0 {
		opts.Interval = time.Hour
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	return &Retention{
		db:       opts.DB,
		log:      log,
		files:    opts.FilesRoot,
		dbAge:    opts.CallRowMaxAge,
		logAge:   opts.LogRowMaxAge,
		filesAge: opts.FilesMaxAge,
		interval: opts.Interval,
	}, nil
}

// Run sweeps once at startup and then every Interval until ctx cancels.
func (r *Retention) Run(ctx context.Context) error {
	r.SweepOnce(ctx)
	tick := time.NewTicker(r.interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			r.SweepOnce(ctx)
		}
	}
}

// SweepOnce runs the configured deletions. Errors are logged and
// swallowed so a transient FS or DB problem doesn't kill the loop.
func (r *Retention) SweepOnce(ctx context.Context) {
	if r.db != nil && r.dbAge > 0 {
		if n, err := r.deleteOldRows(ctx); err != nil {
			r.log.Warn("retention: db sweep failed", "err", err)
		} else if n > 0 {
			r.log.Info("retention: deleted call rows", "count", n)
		}
	}
	if r.db != nil && r.logAge > 0 {
		for _, table := range decoderLogTables {
			n, err := r.deleteOldLogRows(ctx, table)
			if err != nil {
				r.log.Warn("retention: log sweep failed", "table", table, "err", err)
			} else if n > 0 {
				r.log.Info("retention: deleted log rows", "table", table, "count", n)
			}
		}
	}
	if r.files != "" && r.filesAge > 0 {
		if n, err := r.deleteOldFiles(); err != nil {
			r.log.Warn("retention: file sweep failed", "err", err)
		} else if n > 0 {
			r.log.Info("retention: deleted recordings", "count", n)
		}
	}
}

func (r *Retention) deleteOldRows(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-r.dbAge).UnixNano()
	res, err := r.db.sql.ExecContext(ctx, `DELETE FROM call_log WHERE started_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// deleteOldLogRows deletes rows older than logAge from one decoder log
// table (keyed by its received_at column). The table name comes from
// the fixed decoderLogTables allow-list, never from user input, so the
// string interpolation is safe.
func (r *Retention) deleteOldLogRows(ctx context.Context, table string) (int64, error) {
	cutoff := time.Now().Add(-r.logAge).UnixNano()
	res, err := r.db.sql.ExecContext(ctx,
		`DELETE FROM `+table+` WHERE received_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Retention) deleteOldFiles() (int, error) {
	cutoff := time.Now().Add(-r.filesAge)
	deleted := 0
	err := filepath.Walk(r.files, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // tolerate transient FS errors
		}
		if info.IsDir() {
			return nil
		}
		// Only delete recording artifacts so we don't accidentally clobber
		// configs or talkgroup CSVs the operator parked nearby.
		if !isRecordingArtifact(path) {
			return nil
		}
		if info.ModTime().After(cutoff) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			r.log.Warn("retention: rm failed", "path", path, "err", err)
			return nil
		}
		deleted++
		return nil
	})
	if err != nil {
		return deleted, fmt.Errorf("walk %s: %w", r.files, err)
	}
	return deleted, nil
}

func isRecordingArtifact(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav", ".raw":
		return true
	}
	return false
}
