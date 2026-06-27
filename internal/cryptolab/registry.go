package cryptolab

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Tool is one named research tool in the toolkit — e.g. "alias", "brute",
// "lfsr". A tool owns one or more Modes. Tools register themselves from an
// init() (mirroring the SDR and vocoder registries), so adding a tool is a
// new package plus a blank import in the CLI wiring.
type Tool interface {
	// Name is the verb used on the command line (`cryptolab <name>`).
	Name() string
	// Synopsis is a one-line description for help output.
	Synopsis() string
	// Modes are the runnable operations under this tool. A tool with a
	// single mode may be invoked without naming the mode.
	Modes() []Mode
}

// Mode is one runnable operation within a Tool. It parses its own
// flag-style arguments so each mode owns its knobs, the same way the
// siglab subcommands do.
type Mode interface {
	Name() string
	Synopsis() string
	// Run parses mode-specific args and executes against env. It returns a
	// structured Result for uniform export; a nil Result is allowed for
	// modes whose only output is the live progress log.
	Run(ctx context.Context, args []string, env Env) (*Result, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Tool{}
)

// Register adds a tool to the global registry. It panics on a duplicate
// name, which can only happen from a programming error at init time.
func Register(t Tool) {
	registryMu.Lock()
	defer registryMu.Unlock()
	name := t.Name()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("cryptolab: tool %q registered twice", name))
	}
	registry[name] = t
}

// Tools returns every registered tool sorted by name.
func Tools() []Tool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Tool, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Lookup returns the tool registered under name.
func Lookup(name string) (Tool, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := registry[name]
	return t, ok
}

// LookupMode resolves a tool and one of its modes. When the tool has a
// single mode, modeName may be empty to select it.
func LookupMode(toolName, modeName string) (Tool, Mode, error) {
	t, ok := Lookup(toolName)
	if !ok {
		return nil, nil, fmt.Errorf("unknown tool %q", toolName)
	}
	modes := t.Modes()
	if modeName == "" {
		if len(modes) == 1 {
			return t, modes[0], nil
		}
		return t, nil, fmt.Errorf("tool %q needs a mode: one of %s", toolName, modeNames(modes))
	}
	for _, m := range modes {
		if m.Name() == modeName {
			return t, m, nil
		}
	}
	return t, nil, fmt.Errorf("tool %q has no mode %q (have %s)", toolName, modeName, modeNames(modes))
}

func modeNames(modes []Mode) string {
	names := make([]string, len(modes))
	for i, m := range modes {
		names[i] = m.Name()
	}
	sort.Strings(names)
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
