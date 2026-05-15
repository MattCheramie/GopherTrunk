package config

import (
	"os"
	"path/filepath"
)

// Discover walks the standard config-file search precedence and
// returns the first path that exists, or "" if none does. Used by
// the daemon when `gophertrunk run` is launched without an explicit
// -config flag.
//
// Precedence:
//  1. $GOPHERTRUNK_CONFIG (env var; used verbatim, no existence check
//     — operators set this deliberately so a missing file should
//     surface as a clear Load error, not a silent fallback).
//  2. <os.UserConfigDir()>/GopherTrunk/config.yaml
//     (%APPDATA%\GopherTrunk on Windows, ~/.config/GopherTrunk on
//     Linux, ~/Library/Application Support/GopherTrunk on macOS).
//  3. <UserHomeDir>/Documents/GopherTrunk/config.yaml
//     (the Windows installer's suggested default — operators who
//     accept that default get the daemon to find it without any
//     extra flag wiring).
//  4. ./config.yaml (cwd; legacy / dev convenience).
//
// An empty return is the loader's contract for "use defaults".
func Discover() string {
	if p := os.Getenv("GOPHERTRUNK_CONFIG"); p != "" {
		return p
	}
	for _, c := range discoverCandidates() {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// discoverCandidates returns the on-disk paths Discover will stat
// in order. Factored out so tests can assert the search order
// without touching the filesystem.
func discoverCandidates() []string {
	var out []string
	if dir, err := os.UserConfigDir(); err == nil {
		out = append(out, filepath.Join(dir, "GopherTrunk", "config.yaml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, "Documents", "GopherTrunk", "config.yaml"))
	}
	out = append(out, "config.yaml")
	return out
}
