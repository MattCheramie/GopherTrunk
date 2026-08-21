package configbuilder

import (
	"os"
	"path/filepath"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/pathutil"
)

// ExpandConfigPath resolves ~, $VAR/${VAR}, and %VAR% references in a
// config-file path (relocated from the old CLI wizard so the terminal Config
// Builder accepts the same operator-typed paths). Unknown vars are preserved.
// It delegates to pathutil.Expand, the shared leaf helper the daemon's config
// loader uses for the same resolution.
func ExpandConfigPath(p string) string {
	return pathutil.Expand(p)
}

// DefaultConfigPath picks a sensible default config.yaml location:
//  1. $GOPHERTRUNK_CONFIG (the Windows installer sets this);
//  2. an existing config the daemon would auto-discover — so editing
//     goes to the file actually in use;
//  3. ./config.yaml when the cwd is writable;
//  4. <os.UserConfigDir()>/GopherTrunk/config.yaml otherwise.
//
// Step 2 is what keeps `gophertrunk config` in sync with the daemon
// (issue #1120). Without it the two disagree: on a fresh Windows
// install the daemon discovers the installer-seeded
// <Documents>\GopherTrunk\config\config.yaml, while this builder
// defaulted straight to %APPDATA%\GopherTrunk\config.yaml — so a
// freshly-built config was written somewhere the daemon never loaded
// it. Reusing config.Discover here resolves the builder's default to
// the same file the daemon reads on startup.
func DefaultConfigPath() string {
	if p := os.Getenv("GOPHERTRUNK_CONFIG"); p != "" {
		return p
	}
	if p := config.Discover(); p != "" {
		return p
	}
	if cwdWritable() {
		return "./config.yaml"
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "GopherTrunk", "config.yaml")
	}
	return "./config.yaml"
}

func cwdWritable() bool {
	f, err := os.CreateTemp(".", ".gophertrunk-cfg-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
