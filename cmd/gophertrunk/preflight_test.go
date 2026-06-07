package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

func TestPreflightCreatesDirs(t *testing.T) {
	root := t.TempDir()
	recDir := filepath.Join(root, "rec")
	dbPath := filepath.Join(root, "db", "calls.sqlite")
	cachePath := filepath.Join(root, "cache", "cc.json")

	cfg := config.Config{
		Recordings: config.RecordingsConfig{Dir: recDir},
		Storage:    config.StorageConfig{Path: dbPath, CCCacheFile: cachePath},
	}
	warnings, err := preflight(cfg)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	for _, p := range []string{recDir, filepath.Dir(dbPath), filepath.Dir(cachePath)} {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", p)
		}
	}
}

func TestPreflightTalkgroupWarning(t *testing.T) {
	cfg := config.Config{
		Trunking: config.TrunkingConfig{
			Systems: []config.SystemConfig{{
				Name:          "X",
				Protocol:      "p25",
				TalkgroupFile: "/definitely/does/not/exist.csv",
			}},
		},
	}
	warnings, err := preflight(cfg)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning for the missing talkgroup file")
	}
	if !strings.Contains(warnings[0], "talkgroup_file") {
		t.Errorf("warning %q missing 'talkgroup_file' tag", warnings[0])
	}
}

func TestPreflightTalkgroupEmptyWarning(t *testing.T) {
	// A talkgroup CSV that exists but is empty (zero bytes) must
	// surface an actionable preflight warning rather than letting the
	// daemon trip over a cryptic csv-header EOF at load time.
	empty := filepath.Join(t.TempDir(), "talkgroups.csv")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	cfg := config.Config{
		Trunking: config.TrunkingConfig{
			Systems: []config.SystemConfig{{
				Name:          "X",
				Protocol:      "p25",
				TalkgroupFile: empty,
			}},
		},
	}
	warnings, err := preflight(cfg)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning for the empty talkgroup file")
	}
	if !strings.Contains(warnings[0], "empty") {
		t.Errorf("warning %q should mention the file is empty", warnings[0])
	}
}

func TestPreflightStorageDecoderWarning(t *testing.T) {
	// A message decoder (POCSAG here) configured without storage.path
	// must surface a preflight warning naming the storage dependency —
	// otherwise the decoder runs but pager/messages returns 503 with no
	// hint why (issue #565).
	cfg := config.Config{
		Paging: config.PagingConfig{
			POCSAG: []config.PagingPOCSAGConfig{{
				Serial:      "antenna-pi",
				FrequencyHz: 152_007_500,
			}},
		},
	}
	warnings, err := preflight(cfg)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning for paging configured without storage.path")
	}
	w := warnings[0]
	if !strings.Contains(w, "storage.path") {
		t.Errorf("warning %q missing 'storage.path' tag", w)
	}
	if !strings.Contains(w, "paging") {
		t.Errorf("warning %q should name the 'paging' decoder", w)
	}

	// With storage.path set, the warning must not fire.
	cfg.Storage = config.StorageConfig{Path: filepath.Join(t.TempDir(), "calls.db")}
	warnings, err = preflight(cfg)
	if err != nil {
		t.Fatalf("preflight (with storage): %v", err)
	}
	for _, w := range warnings {
		if strings.Contains(w, "storage.path is empty") {
			t.Errorf("unexpected storage warning when storage.path is set: %q", w)
		}
	}
}

func TestPreflightTLSMissing(t *testing.T) {
	cfg := config.Config{
		API: config.APIConfig{
			TLSCert: "/definitely/does/not/exist.crt",
			TLSKey:  "/definitely/does/not/exist.key",
		},
	}
	_, err := preflight(cfg)
	if err == nil {
		t.Fatal("expected preflight to fail with missing TLS files")
	}
	if !strings.Contains(err.Error(), "tls") {
		t.Errorf("error %q should mention tls", err.Error())
	}
}
