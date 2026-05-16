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
