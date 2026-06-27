// Package progress provides the shared progress-logging and resumable
// checkpoint helpers used by every cryptolab tool: throttled structured
// logging for high-volume search loops, an append-only JSONL survivor log,
// and an atomically-written JSON checkpoint so multi-hour runs survive a
// kill and resume where they left off.
package progress

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Throttle rate-limits high-volume progress lines. A search loop that
// visits millions of candidates calls Tick every iteration but only emits
// a log line every Every iterations or every Interval, whichever comes
// first — plus an unconditional line whenever the caller reports a new best.
type Throttle struct {
	Logger   *slog.Logger
	Msg      string
	Every    int           // emit at most once per this many ticks (0 = no count gate)
	Interval time.Duration // emit at most once per this duration (0 = no time gate)

	n       int
	lastN   int
	lastLog time.Time
}

// Tick advances the counter and emits a throttled line carrying args.
func (t *Throttle) Tick(args ...any) {
	t.n++
	now := time.Now()
	countDue := t.Every > 0 && t.n-t.lastN >= t.Every
	timeDue := t.Interval > 0 && now.Sub(t.lastLog) >= t.Interval
	if countDue || timeDue {
		t.emit(now, args...)
	}
}

// Best forces an immediate line (a new best candidate is always worth
// logging regardless of throttle state).
func (t *Throttle) Best(args ...any) { t.emit(time.Now(), args...) }

func (t *Throttle) emit(now time.Time, args ...any) {
	if t.Logger != nil {
		full := append([]any{"count", t.n}, args...)
		t.Logger.Info(t.Msg, full...)
	}
	t.lastN = t.n
	t.lastLog = now
}

// JSONL is an append-only newline-delimited JSON writer for survivor logs.
type JSONL struct {
	f   *os.File
	enc *json.Encoder
}

// OpenJSONL creates/truncates a .jsonl file at dir/name. A nil JSONL (when
// dir is empty) is safe to Append to and Close — it no-ops.
func OpenJSONL(dir, name string) (*JSONL, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	return &JSONL{f: f, enc: json.NewEncoder(f)}, nil
}

// Append writes one JSON record. Safe on a nil receiver.
func (j *JSONL) Append(v any) error {
	if j == nil {
		return nil
	}
	return j.enc.Encode(v)
}

// Close flushes and closes the file. Safe on a nil receiver.
func (j *JSONL) Close() error {
	if j == nil {
		return nil
	}
	return j.f.Close()
}

// SaveCheckpoint atomically writes v as pretty JSON to path (temp file in
// the same directory + rename), so a killed process never leaves a
// half-written checkpoint.
func SaveCheckpoint(path string, v any) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ckpt-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// LoadCheckpoint reads a checkpoint into v. A missing file is not an error
// (found is false), so callers can treat "no checkpoint" as "start fresh".
func LoadCheckpoint(path string, v any) (found bool, err error) {
	if path == "" {
		return false, nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return false, err
	}
	return true, nil
}
