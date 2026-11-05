package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RotatingFile is an io.Writer that appends to a file and rotates it to
// "<path>.1" once it would grow past a size cap. It backs log.file, the
// on-disk copy of the daemon's diagnostic (slog) output.
//
// A write never fails the logger: if the file cannot be reopened after a
// rotation the writer goes quiet rather than erroring every log call.
type RotatingFile struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	f        *os.File
	size     int64
}

// OpenRotatingFile opens (appending to) path, creating its directory.
// maxSizeMB <= 0 means the default of 16 MB.
func OpenRotatingFile(path string, maxSizeMB int) (*RotatingFile, error) {
	if path == "" {
		return nil, errors.New("log: rotating file path is required")
	}
	if maxSizeMB <= 0 {
		maxSizeMB = 16
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("log: create dir for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log: open %s: %w", path, err)
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("log: stat %s: %w", path, err)
	}
	return &RotatingFile{
		path:     path,
		maxBytes: int64(maxSizeMB) << 20,
		f:        f,
		size:     st.Size(),
	}, nil
}

// Path reports the file being written.
func (r *RotatingFile) Path() string { return r.path }

// Write appends p, rotating first if p would push the file past the cap.
// It always reports len(p) written so a full disk cannot break logging.
func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return len(p), nil
	}
	if r.size > 0 && r.size+int64(len(p)) > r.maxBytes {
		r.rotate()
		if r.f == nil {
			return len(p), nil
		}
	}
	n, _ := r.f.Write(p)
	r.size += int64(n)
	return len(p), nil
}

// rotate renames the current file to "<path>.1" and starts a fresh one.
// Caller holds r.mu.
func (r *RotatingFile) rotate() {
	r.f.Close()
	_ = os.Rename(r.path, r.path+".1")
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		r.f = nil
		return
	}
	r.f = f
	r.size = 0
}

// Close closes the file. Later writes are discarded.
func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}
