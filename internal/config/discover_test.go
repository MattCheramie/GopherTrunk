package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestDiscover_EnvVarWins covers the highest-precedence branch:
// GOPHERTRUNK_CONFIG is returned verbatim, even if the referenced
// file doesn't exist yet (Load surfaces the missing-file error to
// the operator). Pick must NOT be called when the env var is set.
func TestDiscover_EnvVarWins(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "/no/such/path/config.yaml")
	called := false
	got, err := DiscoverWith(DiscoverOptions{
		Pick: func(paths []string) (string, error) {
			called = true
			return "", errors.New("should not be called")
		},
	})
	if err != nil {
		t.Fatalf("DiscoverWith: %v", err)
	}
	if got != "/no/such/path/config.yaml" {
		t.Errorf("Discover() = %q, want verbatim env-var value", got)
	}
	if called {
		t.Errorf("Pick was invoked despite GOPHERTRUNK_CONFIG being set")
	}
}

// TestDiscover_SingleFile creates one config in the first candidate
// directory and asserts it's returned without Pick being called.
func TestDiscover_SingleFile(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	dir := redirectUserDirs(t)
	cfgDir, _ := os.UserConfigDir()
	target := filepath.Join(cfgDir, "GopherTrunk", "config.yaml")
	mustWrite(t, target, "log:\n")

	called := false
	got, err := DiscoverWith(DiscoverOptions{
		Pick: func(paths []string) (string, error) { called = true; return "", nil },
	})
	if err != nil {
		t.Fatalf("DiscoverWith: %v", err)
	}
	if got != target {
		t.Errorf("Discover() = %q, want %q (dir=%s)", got, target, dir)
	}
	if called {
		t.Errorf("Pick called for a single-file directory")
	}
}

// TestDiscover_MultipleFiles_CallsPick is the headline test: when
// two configs share a directory, Pick is called with all matches
// (sorted) and its return value is what Discover returns.
func TestDiscover_MultipleFiles_CallsPick(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	redirectUserDirs(t)
	cfgDir, _ := os.UserConfigDir()
	root := filepath.Join(cfgDir, "GopherTrunk")
	mustWrite(t, filepath.Join(root, "config.yaml"), "log:\n")
	mustWrite(t, filepath.Join(root, "prod.yaml"), "log:\n")
	mustWrite(t, filepath.Join(root, "test.yml"), "log:\n")

	wantPaths := []string{
		filepath.Join(root, "config.yaml"),
		filepath.Join(root, "prod.yaml"),
		filepath.Join(root, "test.yml"),
	}

	var seen []string
	got, err := DiscoverWith(DiscoverOptions{
		Pick: func(paths []string) (string, error) {
			seen = paths
			return paths[1], nil // pretend the operator picked "prod.yaml"
		},
	})
	if err != nil {
		t.Fatalf("DiscoverWith: %v", err)
	}
	if !reflect.DeepEqual(seen, wantPaths) {
		t.Errorf("Pick paths = %v, want %v", seen, wantPaths)
	}
	if got != wantPaths[1] {
		t.Errorf("Discover() = %q, want %q", got, wantPaths[1])
	}
}

// TestDiscover_MultipleFiles_NoPick_PicksFirst preserves the
// existing behaviour of Discover (no Pick) — picks the first match
// rather than blowing up. Important for tests / non-interactive
// callers.
func TestDiscover_MultipleFiles_NoPick_PicksFirst(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	redirectUserDirs(t)
	cfgDir, _ := os.UserConfigDir()
	root := filepath.Join(cfgDir, "GopherTrunk")
	mustWrite(t, filepath.Join(root, "config.yaml"), "log:\n")
	mustWrite(t, filepath.Join(root, "prod.yaml"), "log:\n")

	got := Discover()
	want := filepath.Join(root, "config.yaml") // lexically first
	if got != want {
		t.Errorf("Discover() = %q, want %q", got, want)
	}
}

// TestDiscover_PickError_Propagates checks that an interactive
// picker returning an error (e.g. operator typed an out-of-range
// number) aborts discovery instead of silently falling through.
func TestDiscover_PickError_Propagates(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	redirectUserDirs(t)
	cfgDir, _ := os.UserConfigDir()
	root := filepath.Join(cfgDir, "GopherTrunk")
	mustWrite(t, filepath.Join(root, "a.yaml"), "log:\n")
	mustWrite(t, filepath.Join(root, "b.yaml"), "log:\n")

	want := errors.New("invalid selection")
	_, err := DiscoverWith(DiscoverOptions{
		Pick: func(paths []string) (string, error) { return "", want },
	})
	if !errors.Is(err, want) {
		t.Errorf("DiscoverWith err = %v, want %v", err, want)
	}
}

// TestDiscover_NothingFound returns empty when no candidate exists
// and the env var isn't set — Load interprets "" as "use defaults".
func TestDiscover_NothingFound(t *testing.T) {
	t.Setenv("GOPHERTRUNK_CONFIG", "")
	redirectUserDirs(t)
	t.Chdir(t.TempDir())

	if got := Discover(); got != "" {
		t.Errorf("Discover() = %q, want empty (no candidates exist)", got)
	}
}

// TestCandidateDirs_OrderIncludesCwdLast pins the precedence the
// installer + docs rely on: UserConfigDir, then Documents, then cwd.
func TestCandidateDirs_OrderIncludesCwdLast(t *testing.T) {
	got := candidateDirs()
	if len(got) == 0 {
		t.Fatalf("candidateDirs() returned no entries")
	}
	if got[len(got)-1] != "." {
		t.Errorf("last candidate = %q, want %q (cwd entry must be last)", got[len(got)-1], ".")
	}
}

// redirectUserDirs points os.UserConfigDir / os.UserHomeDir at a
// fresh tempdir for the duration of the test, returning the
// tempdir for path-building. Sets every relevant env var so the
// test stays cross-platform.
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
