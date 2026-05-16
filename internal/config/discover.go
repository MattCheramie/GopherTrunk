package config

import (
	"os"
	"path/filepath"
	"sort"
)

// DiscoverOptions tunes DiscoverWith. Pick is invoked when the
// chosen candidate directory contains more than one config file so
// the caller (typically the CLI) can prompt the operator. Pick
// always receives at least two paths; when nil, the first lexical
// match wins silently.
type DiscoverOptions struct {
	Pick func(paths []string) (string, error)
}

// Discover finds the daemon's config file using the standard
// precedence and returns it (or "" when none exists). Equivalent to
// DiscoverWith(DiscoverOptions{}): when multiple files share a
// directory, the first lexical match wins. Callers that want to
// prompt the operator should use DiscoverWith.
func Discover() string {
	p, _ := DiscoverWith(DiscoverOptions{})
	return p
}

// DiscoverWith walks the standard precedence and returns the
// resolved config path (or "" when none exists). Steps:
//
//  1. $GOPHERTRUNK_CONFIG — used verbatim, no existence check (an
//     operator who sets the var should see a clear Load error if
//     the file is missing, not a silent fallback to a different
//     config).
//  2. The first candidate directory containing one or more
//     *.yaml / *.yml files. Within that directory:
//     - 1 file → use it.
//     - 2+ files → call opts.Pick; if nil, take the first.
//
// Candidate directories (in order):
//   - <os.UserConfigDir()>/GopherTrunk
//     (%APPDATA%\GopherTrunk on Windows, ~/.config/GopherTrunk on
//     Linux, ~/Library/Application Support/GopherTrunk on macOS).
//   - <UserHomeDir>/Documents/GopherTrunk
//     (the Windows installer's default — operators who accept it
//     get auto-discovery without setting any env var).
//   - the current working directory.
//
// Pick returning an error aborts discovery; the caller should
// surface the error rather than fall back to a default.
func DiscoverWith(opts DiscoverOptions) (string, error) {
	if p := os.Getenv("GOPHERTRUNK_CONFIG"); p != "" {
		return p, nil
	}
	for _, dir := range candidateDirs() {
		matches := dirConfigFiles(dir)
		switch {
		case len(matches) == 0:
			continue
		case len(matches) == 1:
			return matches[0], nil
		case opts.Pick != nil:
			return opts.Pick(matches)
		default:
			return matches[0], nil
		}
	}
	return "", nil
}

// candidateDirs returns the directories DiscoverWith will scan, in
// precedence order. Factored out so tests can assert the order
// without touching the filesystem.
func candidateDirs() []string {
	var out []string
	if dir, err := os.UserConfigDir(); err == nil {
		out = append(out, filepath.Join(dir, "GopherTrunk"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, "Documents", "GopherTrunk"))
	}
	out = append(out, ".")
	return out
}

// dirConfigFiles returns the *.yaml + *.yml files in dir, sorted
// lexically. An unreadable / missing dir yields an empty slice so
// the caller can keep walking the precedence list.
func dirConfigFiles(dir string) []string {
	var out []string
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		m, err := filepath.Glob(filepath.Join(dir, pattern))
		if err == nil {
			out = append(out, m...)
		}
	}
	sort.Strings(out)
	return out
}
