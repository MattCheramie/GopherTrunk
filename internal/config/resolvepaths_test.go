package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadResolvesRelativePaths verifies that config-relative path fields
// are anchored to the directory holding config.yaml, while absolute and
// empty fields pass through untouched.
func TestLoadResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "config.yaml")

	abs := filepath.Join(dir, "elsewhere", "calls.db")
	yaml := `
storage:
  path: "../data/calls.db"
  cc_cache_file: "` + filepath.ToSlash(abs) + `"
recordings:
  dir: "../recordings"
log:
  message_log:
    enabled: true
    path: "../logs/messages.log"
  power_log:
    enabled: true
    path: "../logs/power.log"
baseband:
  record:
    - serial: "00000001"
      dir: "../iq/"
trunking:
  systems:
    - name: TestSystem
      protocol: p25
      control_channels: [851000000]
      talkgroup_file: "../config/talkgroups.csv"
`
	if err := writeFile(path, yaml); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := func(rel string) string { return filepath.Join(cfgDir, rel) }
	cases := []struct{ name, got, want string }{
		{"storage.path", cfg.Storage.Path, want("../data/calls.db")},
		{"recordings.dir", cfg.Recordings.Dir, want("../recordings")},
		{"message_log.path", cfg.Log.MessageLog.Path, want("../logs/messages.log")},
		{"power_log.path", cfg.Log.PowerLog.Path, want("../logs/power.log")},
		{"baseband.record[0].dir", cfg.Baseband.Record[0].Dir, want("../iq")},
		{"talkgroup_file", cfg.Trunking.Systems[0].TalkgroupFile, want("../config/talkgroups.csv")},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}

	// Absolute path passes through unchanged.
	if cfg.Storage.CCCacheFile != abs {
		t.Errorf("cc_cache_file = %q, want %q (absolute, unchanged)", cfg.Storage.CCCacheFile, abs)
	}
	// Empty field stays empty (token_file unset).
	if cfg.API.Auth.TokenFile != "" {
		t.Errorf("token_file = %q, want empty", cfg.API.Auth.TokenFile)
	}
}

// TestLoadExpandsEnvThenAbsolute verifies an env var expanding to an
// absolute path is NOT re-anchored to the config dir.
func TestLoadExpandsEnvThenAbsolute(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "rootdata")
	t.Setenv("GOPHERTRUNK_TEST_ROOT", root)
	path := filepath.Join(cfgDir, "config.yaml")
	yaml := `
recordings:
  dir: "${GOPHERTRUNK_TEST_ROOT}/recordings"
`
	if err := writeFile(path, yaml); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(root, "recordings")
	if cfg.Recordings.Dir != want {
		t.Errorf("recordings.dir = %q, want %q", cfg.Recordings.Dir, want)
	}
}
