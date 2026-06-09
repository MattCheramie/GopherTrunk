// Package pathutil holds filesystem-path helpers shared across the
// daemon without pulling in any other internal package, so packages
// like internal/config and internal/configbuilder can both depend on
// it without an import cycle.
package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// Expand resolves ~, $VAR/${VAR}, and %VAR% references in a path.
// Unknown variables are preserved verbatim. It does NOT make the path
// absolute — callers that want config-relative resolution should test
// filepath.IsAbs on the result and join against a base directory
// themselves.
func Expand(p string) string {
	switch {
	case p == "~":
		if home, err := os.UserHomeDir(); err == nil {
			p = home
		}
	case strings.HasPrefix(p, "~/"), strings.HasPrefix(p, `~\`):
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	p = os.ExpandEnv(p) // $VAR, ${VAR}
	return expandWindowsEnv(p)
}

// expandWindowsEnv replaces %VAR% references with their env values (Go's
// os.ExpandEnv only handles POSIX-style $VAR). Unknown vars are preserved.
func expandWindowsEnv(p string) string {
	var b strings.Builder
	for {
		i := strings.IndexByte(p, '%')
		if i < 0 {
			b.WriteString(p)
			return b.String()
		}
		b.WriteString(p[:i])
		rest := p[i+1:]
		j := strings.IndexByte(rest, '%')
		if j < 0 {
			b.WriteByte('%')
			b.WriteString(rest)
			return b.String()
		}
		name := rest[:j]
		if val, ok := os.LookupEnv(name); ok {
			b.WriteString(val)
		} else {
			b.WriteByte('%')
			b.WriteString(name)
			b.WriteByte('%')
		}
		p = rest[j+1:]
	}
}
