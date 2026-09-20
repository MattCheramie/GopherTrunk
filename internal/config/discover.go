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
// Candidate directories (in order). Each GopherTrunk root is scanned
// at its top level AND in its config/ subfolder (the data-root layout
// the installer lays down, config.yaml at <DataRoot>/config/config.yaml):
//   - <os.UserConfigDir()>/GopherTrunk and .../GopherTrunk/config
//     (%APPDATA%\GopherTrunk on Windows, ~/.config/GopherTrunk on
//     Linux, ~/Library/Application Support/GopherTrunk on macOS).
//   - <os.UserConfigDir()>/gophertrunk and ~/.config/gophertrunk — the
//     lowercase XDG-style path the install docs use (issue #836).
//   - <UserHomeDir>/Documents/GopherTrunk and .../GopherTrunk/config
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

// CandidateDirs returns the directories config discovery scans, in
// precedence order (see DiscoverWith). Exported so the web Config
// Builder can list and constrain saves to the same set of locations the
// daemon auto-discovers from.
func CandidateDirs() []string { return candidateDirs() }

// DirConfigFiles returns the *.yaml + *.yml files in dir, sorted
// lexically. Exported for the web Config Builder's file browser; an
// unreadable / missing dir yields an empty slice.
func DirConfigFiles(dir string) []string { return dirConfigFiles(dir) }

// candidateDirs returns the directories DiscoverWith will scan, in
// precedence order. Factored out so tests can assert the order
// without touching the filesystem.
//
// Each GopherTrunk root is scanned both at its top level (the legacy
// layout, config.yaml directly in the root) and in its config/
// subfolder (the data-root layout the installer lays down, where
// config.yaml lives at <DataRoot>/config/config.yaml). The top-level
// entry is listed first so an old-style config still wins if both
// exist.
//
// Both the platform-conventional CamelCase name (%APPDATA%\GopherTrunk
// on Windows, ~/Library/Application Support/GopherTrunk on macOS,
// ~/.config/GopherTrunk on Linux) AND the lowercase XDG-style name
// ~/.config/gophertrunk are scanned. The lowercase name is what the
// install docs (install-linux.md, install-macos.md, downloads.md) tell
// operators to create by hand, and what Linux CLI tools conventionally
// use — but os.UserConfigDir() only yields the CamelCase root, so a
// config placed at the documented ~/.config/gophertrunk/config.yaml was
// never discovered and only worked when the daemon happened to run from
// that directory (the cwd fallback). Issue #836. On macOS the documented
// ~/.config path is not os.UserConfigDir() at all, so it is scanned
// explicitly from $HOME. Duplicate paths (e.g. on Linux, where
// os.UserConfigDir() already is ~/.config, and on case-insensitive
// filesystems) are collapsed so a directory is never scanned twice.
func candidateDirs() []string {
	var out []string
	seen := make(map[string]bool)
	add := func(root string) {
		for _, d := range []string{root, filepath.Join(root, "config")} {
			if !seen[d] {
				seen[d] = true
				out = append(out, d)
			}
		}
	}
	if dir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(dir, "GopherTrunk"))
		add(filepath.Join(dir, "gophertrunk"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		// The docs point Linux and macOS at ~/.config/gophertrunk, but
		// os.UserConfigDir() is ~/Library/Application Support on macOS, so
		// scan the documented XDG path explicitly too (a no-op dedup on
		// Linux, where it equals the lowercase entry above).
		add(filepath.Join(home, ".config", "gophertrunk"))
		add(filepath.Join(home, "Documents", "GopherTrunk"))
	}
	if !seen["."] {
		out = append(out, ".")
	}
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
