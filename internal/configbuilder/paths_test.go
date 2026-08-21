package configbuilder

import (
	"os"
	"path/filepath"
	"testing"
)

// redirectUserDirs points os.UserConfigDir / os.UserHomeDir at a fresh
// tempdir for the duration of the test, returning the tempdir. Mirrors
// the helper in internal/config's discover_test so these builder tests
// stay hermetic and cross-platform.
func redirectUserDirs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestDefaultConfigPath_HonorsEnvVar keeps the highest-precedence rule:
// $GOPHERTRUNK_CONFIG wins verbatim, even over a discoverable config.
func TestDefaultConfigPath_HonorsEnvVar(t *testing.T) {
	redirectUserDirs(t)
	want := filepath.Join(t.TempDir(), "explicit.yaml")
	t.Setenv("GOPHERTRUNK_CONFIG", want)

	if got := DefaultConfigPath(); got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q (env var must win)", got, want)
	}
}

// TestDefaultConfigPath_PrefersDiscoveredConfig is the #1120 regression.
// The daemon auto-discovers a config the installer seeded under the
// data-root config/ subfolder (<Documents>\GopherTrunk\config on
// Windows). The builder must default to that same file rather than a
// second, un-loaded location — otherwise a freshly-built config lands
// where the daemon never reads it.
//
// Fails against the old DefaultConfigPath (which skipped straight to the
// cwd / UserConfigDir heuristic and returned a different path); passes
// once Discover is consulted first.
func TestDefaultConfigPath_PrefersDiscoveredConfig(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	home := redirectUserDirs(t)
	// Run from a writable, config-free cwd so the old code's
	// "./config.yaml when cwd is writable" branch would fire and
	// produce a path that is NOT the discovered file.
	t.Chdir(t.TempDir())

	seeded := filepath.Join(home, "Documents", "GopherTrunk", "config", "config.yaml")
	mustWrite(t, seeded, "log:\n")

	got := DefaultConfigPath()
	if got != seeded {
		t.Errorf("DefaultConfigPath() = %q, want %q (must default to the config the daemon discovers)", got, seeded)
	}
}

// TestDefaultConfigPath_FallsBackWhenNothingDiscovered keeps the
// no-config behaviour: with nothing to discover and a writable cwd, the
// builder still offers ./config.yaml as the create-here default.
func TestDefaultConfigPath_FallsBackWhenNothingDiscovered(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	redirectUserDirs(t)
	t.Chdir(t.TempDir())

	if got := DefaultConfigPath(); got != "./config.yaml" {
		t.Errorf("DefaultConfigPath() = %q, want %q (writable cwd, nothing discovered)", got, "./config.yaml")
	}
}
