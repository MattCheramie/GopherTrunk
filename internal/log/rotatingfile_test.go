package log

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileAppendsAndRotates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "debug.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// A pre-existing file is appended to, not truncated: a restart must
	// not wipe the lines that explain why it was restarted.
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRotatingFile(path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.Write([]byte("new line\n")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "old line\nnew line\n" {
		t.Fatalf("after append: %q", got)
	}

	// Pushing past 1 MB rotates the full file to .1 and starts fresh.
	big := []byte(strings.Repeat("x", 1<<20) + "\n")
	if _, err := r.Write(big); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatalf("rotated file missing: %v", err)
	}
	if string(old) != "old line\nnew line\n" {
		t.Fatalf("rotated file = %q", old[:min(len(old), 40)])
	}
	cur, _ := os.ReadFile(path)
	if len(cur) != len(big) {
		t.Fatalf("current file len %d, want %d", len(cur), len(big))
	}
}

func TestRotatingFileCreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "debug.log")
	r, err := OpenRotatingFile(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
	// Writes after Close are discarded, not errors.
	if n, err := r.Write([]byte("x")); n != 1 || err != nil {
		t.Fatalf("write after close = %d, %v", n, err)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("stderr gone") }

// TestTeeLogFileSurvivesRedirectAndBrokenStderr pins the two cases log.file
// exists for (#1242): the in-process TUI redirects stderr to a tempfile,
// and a detached Windows console can make stderr writes fail. In both the
// configured file must still get every line.
func TestTeeLogFileSurvivesRedirectAndBrokenStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	r, err := OpenRotatingFile(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	logger, sw := NewWithSwapTee("info", "text", r)

	logger.Info("composer: p25p1 transmission quality", "src", 1234567)

	var tui strings.Builder
	sw.Redirect(&tui)
	logger.Warn("under the tui")
	sw.Restore()

	sw.Redirect(failWriter{})
	logger.Warn("stderr is broken")
	sw.Restore()

	logger.Debug("below level")

	got, _ := os.ReadFile(path)
	for _, want := range []string{"p25p1 transmission quality", "src=1234567", "under the tui", "stderr is broken"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("log file missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(string(got), "below level") {
		t.Errorf("log file holds a record below the configured level")
	}
	if !strings.Contains(tui.String(), "under the tui") {
		t.Errorf("redirected stderr did not get the TUI-time line")
	}
}
