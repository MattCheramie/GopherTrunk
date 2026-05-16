package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// preflight runs after config.Load and before NewDaemon. It exists
// to convert "silently goes bad at runtime" classes of failure into
// "fails loudly with a clear message before the launcher / TUI ever
// fires" — directory creation, TLS file readability, talkgroup CSV
// existence checks.
//
// Returns a slice of non-fatal warning strings (surfaced via
// Daemon.startupWarnings) and an error for fatal misconfiguration
// (unwritable directories, missing TLS files when TLS is enabled).
func preflight(cfg config.Config) ([]string, error) {
	var warnings []string

	// Auto-create directories the daemon writes into. The
	// recorder / storage / cc-cache subsystems each do this lazily,
	// but doing it up-front gives the operator a single clear error
	// instead of a runtime warning that lands somewhere in the log.
	dirs := []struct {
		label string
		path  string
	}{
		{"recordings.dir", cfg.Recordings.Dir},
		{"storage.path (parent)", parentDir(cfg.Storage.Path)},
		{"storage.cc_cache_file (parent)", parentDir(cfg.Storage.CCCacheFile)},
	}
	for _, d := range dirs {
		if d.path == "" {
			continue
		}
		if err := os.MkdirAll(d.path, 0o755); err != nil {
			return warnings, fmt.Errorf("preflight: %s: mkdir %q: %w", d.label, d.path, err)
		}
	}

	// TLS cert/key pre-flight. Both must be set together (the
	// existing server validates the XOR case at construction). Here
	// we additionally verify the files are readable + parse as a
	// valid X.509 keypair so a typo / mode 0o600-from-another-user
	// surfaces as `preflight: tls_cert …` instead of an opaque
	// goroutine error after the listener has already bound.
	if cfg.API.TLSCert != "" && cfg.API.TLSKey != "" {
		if _, err := tls.LoadX509KeyPair(cfg.API.TLSCert, cfg.API.TLSKey); err != nil {
			return warnings, fmt.Errorf("preflight: tls cert/key (%s, %s): %w",
				cfg.API.TLSCert, cfg.API.TLSKey, err)
		}
	}

	// Talkgroup CSV existence — non-fatal: the daemon happily runs
	// with no alpha-tag database, just without the operator-friendly
	// labels.
	for _, sys := range cfg.Trunking.Systems {
		if sys.TalkgroupFile == "" {
			continue
		}
		info, err := os.Stat(sys.TalkgroupFile)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("trunking.systems[%s].talkgroup_file: cannot stat %q (%v) — calls on this system will have no alpha tags",
					sys.Name, sys.TalkgroupFile, err))
			continue
		}
		if info.IsDir() {
			warnings = append(warnings,
				fmt.Sprintf("trunking.systems[%s].talkgroup_file: %q is a directory; expected a CSV file",
					sys.Name, sys.TalkgroupFile))
		}
	}

	return warnings, nil
}

// parentDir returns the directory containing path, or "" if path
// itself is empty. Tolerates absolute and relative paths.
func parentDir(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}
