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

// TestDaemonActivateReload_SwitchesFile confirms ActivateReload re-points the
// daemon at a different config file: the writer/path follow and the
// hot-reloadable diff is applied against the new file's values.
func TestDaemonActivateReload_SwitchesFile(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.yaml")
	pathB := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(pathA, []byte(reloadInitialYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(reloadUpdatedYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(pathA)
	if err != nil {
		t.Fatalf("load a: %v", err)
	}
	d, err := NewDaemonWithPath(cfg, pathA, "test", slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	applied, restartReq, err := d.ActivateReload(pathB)
	if err != nil {
		t.Fatalf("activate reload: %v", err)
	}
	if !contains(applied, "scanner.scan_mode") || !contains(applied, "audio.volume") {
		t.Errorf("expected scan_mode + audio.volume applied, got %v", applied)
	}
	if !contains(restartReq, "recordings.dir") {
		t.Errorf("expected recordings.dir restart-required, got %v", restartReq)
	}
	// The daemon now tracks b.yaml as its active config + writer.
	if d.cfgPath != pathB {
		t.Errorf("cfgPath = %q want %q", d.cfgPath, pathB)
	}
	if d.Path() != pathB {
		t.Errorf("Path() = %q want %q", d.Path(), pathB)
	}
	if d.cfg.Scanner.ScanMode != "list" {
		t.Errorf("in-memory scan_mode = %q want list", d.cfg.Scanner.ScanMode)
	}
}

func TestBuildRestartArgs(t *testing.T) {
	cases := []struct {
		name string
		orig []string
		want []string
	}{
		{
			name: "appends when absent",
			orig: []string{"gophertrunk", "run"},
			want: []string{"gophertrunk", "run", "-config", "/new.yaml"},
		},
		{
			name: "replaces -config value form",
			orig: []string{"gophertrunk", "run", "-config", "/old.yaml", "-web"},
			want: []string{"gophertrunk", "run", "-web", "-config", "/new.yaml"},
		},
		{
			name: "replaces --config=eq form",
			orig: []string{"gophertrunk", "--config=/old.yaml"},
			want: []string{"gophertrunk", "-config", "/new.yaml"},
		},
		{
			name: "bare invocation",
			orig: []string{"gophertrunk"},
			want: []string{"gophertrunk", "-config", "/new.yaml"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRestartArgs(tc.orig, "/new.yaml")
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Errorf("buildRestartArgs(%v) = %v, want %v", tc.orig, got, tc.want)
			}
		})
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
