package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDiscover_EnvVarWins covers the highest-precedence branch:
// GOPHERTRUNK_CONFIG is returned verbatim, even if the referenced
// file doesn't exist yet (Load surfaces the missing-file error to
// the operator).
func TestDiscover_EnvVarWins(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "/no/such/path/config.yaml")
	if got := Discover(); got != "/no/such/path/config.yaml" {
		t.Errorf("Discover() = %q, want verbatim env-var value", got)
	}
}

// TestDiscover_StatFallback creates a real file at the UserConfigDir
// candidate, then asserts Discover picks it up when the env var is
// unset. Uses t.Setenv on the platform-appropriate var that
// os.UserConfigDir() consults so the test doesn't depend on the host's
// real config dir.
func TestDiscover_StatFallback(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	dir := t.TempDir()

	// Redirect os.UserConfigDir() to the tempdir. On Linux it reads
	// XDG_CONFIG_HOME; on macOS HOME; on Windows APPDATA. Setting all
	// three is harmless and keeps the test cross-platform.
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir) // macOS UserConfigDir = $HOME/Library/Application Support

	want := filepath.Join(dir, "GopherTrunk", "config.yaml")
	// macOS path differs; compute it dynamically rather than guessing.
	if cd, err := os.UserConfigDir(); err == nil {
		want = filepath.Join(cd, "GopherTrunk", "config.yaml")
	}
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(want, []byte("log:\n"), 0o644); err != nil {
		t.Fatalf("write seed config: %v", err)
	}

	if got := Discover(); got != want {
		t.Errorf("Discover() = %q, want %q", got, want)
	}
}

// TestDiscover_NothingFound returns empty when no candidate exists
// and the env var isn't set — Load interprets "" as "use defaults".
func TestDiscover_NothingFound(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	// Point every UserConfigDir / UserHomeDir source at an empty
	// tempdir so the stat loop finds nothing. cwd "./config.yaml"
	// still gets checked, so we also chdir into the empty dir.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	if got := Discover(); got != "" {
		t.Errorf("Discover() = %q, want empty (no candidates exist)", got)
	}
}

// TestDiscoverCandidates_OrderIncludesCwdLast pins the precedence the
// installer + docs rely on: UserConfigDir, then Documents, then cwd.
func TestDiscoverCandidates_OrderIncludesCwdLast(t *testing.T) {
	got := discoverCandidates()
	if len(got) == 0 {
		t.Fatalf("discoverCandidates() returned no entries")
	}
	if got[len(got)-1] != "config.yaml" {
		t.Errorf("last candidate = %q, want %q (cwd entry must be last)", got[len(got)-1], "config.yaml")
	}
}
