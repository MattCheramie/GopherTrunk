//go:build !windows

package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

const sighupInitialYAML = `log:
  level: info
audio:
  enabled: false
  volume: 0.5
scanner:
  scan_mode: all
`

const sighupUpdatedYAML = `log:
  level: warn
audio:
  enabled: false
  volume: 0.99
scanner:
  scan_mode: list
`

// TestSIGHUP_TriggersReload verifies the actual signal-driven
// reload path that operators exercise post-`$EDITOR`. The test
// writes a config.yaml, builds a daemon against it, starts the
// SIGHUP watcher, rewrites the file, sends a real SIGHUP to the
// current process, and verifies the in-memory config picks up the
// edit. Replaces the "manual smoke: kill -HUP after $EDITOR" check
// item with a CI-runnable equivalent.
func TestSIGHUP_TriggersReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(sighupInitialYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load initial: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemonWithPath(cfg, path, "test", logger)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}
	if d.Cfg().Scanner.ScanMode != "all" {
		t.Fatalf("pre-reload scan_mode = %q want all", d.Cfg().Scanner.ScanMode)
	}

	// Stand up the SIGHUP watcher in a goroutine, with a context
	// we'll cancel on test exit.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchReloadSignal(ctx, d, logger)

	// Rewrite the config file to simulate "operator just saved
	// $EDITOR".
	if err := os.WriteFile(path, []byte(sighupUpdatedYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Deliver SIGHUP to the current process; the watcher above
	// catches it and triggers Reload().
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("kill: %v", err)
	}

	// Poll for the in-memory cfg to flip — the reload is async
	// inside the goroutine. Use Cfg() so the read is serialised
	// against Reload's write lock and -race stays happy.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if d.Cfg().Scanner.ScanMode == "list" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := d.Cfg().Scanner.ScanMode; got != "list" {
		t.Errorf("post-SIGHUP scan_mode = %q want list (reload did not run)", got)
	}
}

// safeBuffer is a minimal thread-safe io.Writer + reader so the
// test goroutine can scan for log output produced by a watcher
// goroutine without tripping -race. Wraps a single mutex.
type safeBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestSIGHUP_BadConfigDoesNotCrash verifies that a malformed
// config.yaml at SIGHUP-time produces an error log entry but
// leaves the running daemon's state untouched (so the operator
// can fix the file + re-HUP).
func TestSIGHUP_BadConfigDoesNotCrash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(sighupInitialYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	captured := &safeBuffer{}
	logger := slog.New(slog.NewTextHandler(captured, nil))
	d, err := NewDaemonWithPath(cfg, path, "test", logger)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchReloadSignal(ctx, d, logger)

	// Stomp on the file with broken YAML.
	if err := os.WriteFile(path, []byte("log: [not valid yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to log the failure.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(captured.String(), "sighup: reload failed") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(captured.String(), "sighup: reload failed") {
		t.Errorf("expected reload-failed log entry; captured: %q", captured.String())
	}
	// Pre-reload state should still be intact — the bad reload
	// must not have mutated d.cfg.
	if got := d.Cfg().Scanner.ScanMode; got != "all" {
		t.Errorf("scan_mode mutated after failed reload: %q", got)
	}
}
