package configbuilder

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandConfigPath resolves ~, $VAR/${VAR}, and %VAR% references in a
// config-file path (relocated from the old CLI wizard so the terminal Config
// Builder accepts the same operator-typed paths). Unknown vars are preserved.
func ExpandConfigPath(p string) string {
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

// DefaultConfigPath picks a sensible default config.yaml location:
//  1. $GOPHERTRUNK_CONFIG (the Windows installer sets this);
//  2. ./config.yaml when the cwd is writable;
//  3. <os.UserConfigDir()>/GopherTrunk/config.yaml otherwise.
func DefaultConfigPath() string {
	if p := os.Getenv("GOPHERTRUNK_CONFIG"); p != "" {
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
