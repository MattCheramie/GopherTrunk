// Package terms embeds the GopherTrunk Terms of Service and records the
// operator's acknowledgment of them.
//
// The canonical document is TERMS_OF_SERVICE.md at the repository root;
// the copy embedded here must stay byte-identical to it (pinned by
// TestEmbeddedTermsMatchCanonicalDocument). Acknowledgment is recorded
// locally as a small marker file under the operator's user config
// directory — the same location the Windows installer writes after its
// Terms of Service wizard page — and nothing is ever sent anywhere.
package terms

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Version is the terms-of-service revision this binary ships. Bump it
// whenever TERMS_OF_SERVICE.md changes materially; operators are then
// asked to re-acknowledge on the next run. Keep in sync with the
// TermsVersion define in installer/windows/gophertrunk.iss, which
// records the same marker on behalf of the Windows installer.
const Version = 1

// Text is the full Terms of Service document.
//
//go:embed TERMS_OF_SERVICE.md
var Text string

// EnvVar names the environment variable that accepts the terms for
// unattended installs (services, containers, CI).
const EnvVar = "GOPHERTRUNK_ACCEPT_TERMS"

// markerName is the acceptance-record filename. The Windows installer
// writes the same file (see installer/windows/gophertrunk.iss).
const markerName = "terms-accepted"

// DefaultDir returns the directory the acceptance marker lives in:
// <os.UserConfigDir()>/GopherTrunk (%AppData%\GopherTrunk on Windows,
// ~/.config/GopherTrunk on Linux), falling back to ~/.gophertrunk when
// the platform config dir cannot be resolved.
func DefaultDir() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "GopherTrunk"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("terms: no user config or home directory: %w", err)
	}
	return filepath.Join(home, ".gophertrunk"), nil
}

// MarkerPath returns the acceptance-marker path under dir.
func MarkerPath(dir string) string { return filepath.Join(dir, markerName) }

// AcceptedVersion parses the marker file under dir and returns the
// terms version it records, or 0 when there is no (readable, parseable)
// marker. A marker from an older terms revision therefore reads as
// "accepted, but not the current version".
func AcceptedVersion(dir string) int {
	b, err := os.ReadFile(MarkerPath(dir))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "version="); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// IsAccepted reports whether the current terms Version has been
// acknowledged under dir.
func IsAccepted(dir string) bool { return AcceptedVersion(dir) >= Version }

// Record writes the acceptance marker under dir, creating the directory
// if needed. via names how acceptance happened ("interactive", "env",
// "command", "windows-installer") for the operator's own reference.
func Record(dir, via string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("terms: %w", err)
	}
	body := fmt.Sprintf(
		"# GopherTrunk terms-of-service acceptance record (see TERMS_OF_SERVICE.md).\n"+
			"version=%d\naccepted_at=%s\naccepted_via=%s\n",
		Version, time.Now().UTC().Format(time.RFC3339), via)
	if err := os.WriteFile(MarkerPath(dir), []byte(body), 0o644); err != nil {
		return fmt.Errorf("terms: %w", err)
	}
	return nil
}

// EnvAccepted reports whether GOPHERTRUNK_ACCEPT_TERMS is set to an
// affirmative value, the unattended-install acceptance path.
func EnvAccepted() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvVar))) {
	case "1", "true", "yes", "y":
		return true
	}
	return false
}
