package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleYAML = `# top of file comment kept
log:
  level: info
  format: text

api:
  http_addr: "127.0.0.1:8080"
  # auth section omitted — daemon defaults apply

audio:
  enabled: false
  volume: 0.8

# trailing comment kept
`

func writeTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(sampleYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPatchApply(t *testing.T) {
	cfg := Config{}
	level := "debug"
	volume := float32(0.5)
	enabled := true
	p := Patch{LogLevel: &level, AudioVolume: &volume, AudioEnabled: &enabled}
	out := p.Apply(cfg)
	if out.Log.Level != "debug" {
		t.Errorf("Log.Level = %q want debug", out.Log.Level)
	}
	if out.Audio.Volume != 0.5 {
		t.Errorf("Audio.Volume = %v want 0.5", out.Audio.Volume)
	}
	if !out.Audio.Enabled {
		t.Errorf("Audio.Enabled = false want true")
	}
}

func TestWriterWritePatchPreservesComments(t *testing.T) {
	path := writeTempConfig(t)
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	level := "debug"
	volume := float32(0.42)
	enabled := true
	merged, err := w.WritePatch(Patch{
		LogLevel:     &level,
		AudioVolume:  &volume,
		AudioEnabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Log.Level != "debug" {
		t.Errorf("merged.Log.Level = %q want debug", merged.Log.Level)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "# top of file comment kept") {
		t.Errorf("expected top-of-file comment preserved, got:\n%s", body)
	}
	if !strings.Contains(body, "# trailing comment kept") {
		t.Errorf("expected trailing comment preserved, got:\n%s", body)
	}
	if !strings.Contains(body, "level: debug") {
		t.Errorf("expected updated log.level, got:\n%s", body)
	}
	if !strings.Contains(body, "volume: 0.42") {
		t.Errorf("expected updated audio.volume, got:\n%s", body)
	}
	if !strings.Contains(body, "enabled: true") {
		t.Errorf("expected updated audio.enabled, got:\n%s", body)
	}
}

func TestWriterRejectsInvalidPatch(t *testing.T) {
	path := writeTempConfig(t)
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	bad := "list-or-all-only"
	if _, err := w.WritePatch(Patch{ScannerScanMode: &bad}); err == nil {
		t.Fatal("expected validation error for invalid scan_mode")
	}
}

func TestWriterMtimeGuard(t *testing.T) {
	path := writeTempConfig(t)
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	// External edit while the writer is "running": rewrite the file
	// with a clearly different mtime.
	bumpFutureMtime(t, path)

	level := "warn"
	if _, err := w.WritePatch(Patch{LogLevel: &level}); err == nil {
		t.Fatal("expected mtime guard to reject the write")
	} else if !strings.Contains(err.Error(), "modified externally") {
		t.Errorf("expected 'modified externally' in error, got %v", err)
	}
}

// bumpFutureMtime rewrites the file with a +1 minute mtime so the
// writer's stored mtime differs.
func bumpFutureMtime(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	future := info.ModTime().Add(60_000_000_000) // 1 minute in ns
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

func TestNewWriterMissingFile(t *testing.T) {
	if _, err := NewWriter(filepath.Join(t.TempDir(), "does-not-exist.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestNewWriterEmptyPath(t *testing.T) {
	w, err := NewWriter("")
	if err != nil {
		t.Fatalf("expected nil-writer / nil-error for empty path, got err=%v", err)
	}
	if w != nil {
		t.Fatalf("expected nil writer for empty path, got %v", w)
	}
}
