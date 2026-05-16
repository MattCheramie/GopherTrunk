package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

const reloadInitialYAML = `log:
  level: info
audio:
  enabled: true
  volume: 0.8
scanner:
  scan_mode: all
`

const reloadUpdatedYAML = `log:
  level: warn
audio:
  enabled: true
  volume: 0.42
scanner:
  scan_mode: list
recordings:
  dir: /tmp/recs
`

func TestDaemonReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(reloadInitialYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load initial: %v", err)
	}
	d, err := NewDaemonWithPath(cfg, path, "test", slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	// Rewrite config.yaml with new values.
	if err := os.WriteFile(path, []byte(reloadUpdatedYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := d.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !strings.Contains(summary, "scanner.scan_mode") {
		t.Errorf("expected scan_mode in applied list, got %q", summary)
	}
	if !strings.Contains(summary, "audio.volume") {
		t.Errorf("expected audio.volume in applied list, got %q", summary)
	}
	if !strings.Contains(summary, "recordings.dir") {
		t.Errorf("expected recordings.dir in restart-required, got %q", summary)
	}
	// In-memory cfg should now reflect the updated file.
	if d.cfg.Scanner.ScanMode != "list" {
		t.Errorf("in-memory scan_mode = %q want list", d.cfg.Scanner.ScanMode)
	}
}

func TestDaemonReload_NoConfigPath(t *testing.T) {
	cfg := config.Default()
	d, err := NewDaemonWithPath(cfg, "", "test", slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Reload(); err == nil {
		t.Fatal("expected error for daemon without config file")
	}
}
