package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	for i, s := range cfg.SDR.SoapyRemote {
		if s.DiversityCapture == "" {
			continue
		}
		// The value is a filename prefix; a trailing separator means the
		// operator named the directory itself (the branch recorder then
		// generates a timestamped basename inside it).
		dir := parentDir(s.DiversityCapture)
		if strings.HasSuffix(s.DiversityCapture, string(os.PathSeparator)) || strings.HasSuffix(s.DiversityCapture, "/") {
			dir = filepath.Clean(s.DiversityCapture)
		}
		dirs = append(dirs, struct {
			label string
			path  string
		}{fmt.Sprintf("sdr.soapy_remote[%d].diversity_capture (parent)", i), dir})
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
			continue
		}
		if info.Size() == 0 {
			warnings = append(warnings,
				fmt.Sprintf("trunking.systems[%s].talkgroup_file: %q is empty — calls on this system will have no alpha tags",
					sys.Name, sys.TalkgroupFile))
		}
	}

	// RID alias file existence — same non-fatal stat-only check as the
	// talkgroup file. Empty / missing leaves operators without RID
	// aliases on this system; live observations still surface via the
	// affiliation tracker.
	for _, sys := range cfg.Trunking.Systems {
		if sys.RIDAliasFile == "" {
			continue
		}
		info, err := os.Stat(sys.RIDAliasFile)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("trunking.systems[%s].rid_alias_file: cannot stat %q (%v) — radios on this system will have no operator aliases",
					sys.Name, sys.RIDAliasFile, err))
			continue
		}
		if info.IsDir() {
			warnings = append(warnings,
				fmt.Sprintf("trunking.systems[%s].rid_alias_file: %q is a directory; expected a CSV or JSON file",
					sys.Name, sys.RIDAliasFile))
			continue
		}
		if info.Size() == 0 {
			warnings = append(warnings,
				fmt.Sprintf("trunking.systems[%s].rid_alias_file: %q is empty — radios on this system will have no operator aliases",
					sys.Name, sys.RIDAliasFile))
		}
	}

	// Message decoders that only surface their output through the
	// SQLite-backed logs (pager / aprs / ais / dsc / mdc1200 / m17). With
	// storage.path empty the daemon skips storage init entirely, so these
	// decoders still run but their logs are never wired: the REST endpoints
	// return 503 and the web panels stay empty. Operators hit this as a
	// silent failure (see issue #565 — POCSAG configured, pager/messages 503
	// until storage.path was added), so name the dependency loudly here.
	if cfg.Storage.Path == "" {
		var needs []string
		if len(cfg.Paging.POCSAG) > 0 || len(cfg.Paging.FLEX) > 0 {
			needs = append(needs, "paging")
		}
		if len(cfg.APRS.Channels) > 0 {
			needs = append(needs, "aprs")
		}
		if len(cfg.AIS.Channels) > 0 {
			needs = append(needs, "ais")
		}
		if len(cfg.DSC.Channels) > 0 {
			needs = append(needs, "dsc")
		}
		if len(cfg.MDC1200.Channels) > 0 {
			needs = append(needs, "mdc1200")
		}
		if len(cfg.M17.Channels) > 0 {
			needs = append(needs, "m17")
		}
		if len(needs) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"storage.path is empty but these decoders need it to surface decoded "+
					"messages: %s — they will run, but their REST endpoints (e.g. "+
					"/api/v1/pager/messages) return 503 and the web panels stay empty. "+
					"Set storage.path to enable persistence.",
				strings.Join(needs, ", ")))
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
