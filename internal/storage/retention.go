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

// Retention deletes old data on a schedule:
//   1. call_log rows with started_at older than CallRowMaxAge.
//   2. WAV / raw files under FilesRoot whose modification time is older
//      than FilesMaxAge.
//
// File deletion is opt-in by setting FilesRoot; an empty value skips
// the filesystem sweep. The sweeper is idempotent and safe to run
// concurrently with the call-log writer (SQLite serialises).
type Retention struct {
	db        *DB
	log       *slog.Logger
	files     string
	dbAge     time.Duration
	filesAge  time.Duration
	interval  time.Duration
}

// RetentionOptions configure a Retention sweeper.
type RetentionOptions struct {
	DB *DB
	// FilesRoot is the directory the voice recorder writes WAV / raw
	// files under. Empty disables the filesystem sweep.
	FilesRoot string
	// CallRowMaxAge: rows with started_at older than this are deleted.
	// Zero (the default) disables row deletion.
	CallRowMaxAge time.Duration
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
