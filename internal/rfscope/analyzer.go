package rfscope

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

// Input is the read-only material an Analyzer works from. The segmented bursts
// already live on the Scene; Input adds the run-level parameters and a lazy
// accessor back to each burst's decimated baseband IQ, so a long capture is
// never held whole in memory.
type Input struct {
	// SampleRateHz is the capture's IQ sample rate.
	SampleRateHz float64
	// Window is the observation window in seconds (the scene's time span).
	Window float64
	// Log is the run logger; never nil (Run substitutes a discard logger).
	Log *slog.Logger
	// BurstIQ returns the decimated baseband IQ of one burst and its rate, or an
	// error if the source can no longer be read. Analyzers that only need the
	// burst metadata (timeline, timing) ignore it; classifiers/identifiers call
	// it on demand.
	BurstIQ func(b Burst) (iq []complex64, rateHz float64, err error)
}

// Analyzer is one named analysis pass over a Scene. Analyzers register
// themselves from init() (mirroring the cryptolab Tool registry and the
// ccdecoder pipeline factories), so adding one is a new file plus a blank import
// in the CLI wiring. DependsOn names analyzers that must run first; Run
// topologically sorts on it.
type Analyzer interface {
	Name() string
	Synopsis() string
	DependsOn() []string
	Analyze(ctx context.Context, sc *Scene, in *Input) error
}

var (
	analyzersMu sync.RWMutex
	analyzers   = map[string]Analyzer{}
)

// Register adds an analyzer to the global registry. It panics on a duplicate
// name, which can only happen from a programming error at init time.
func Register(a Analyzer) {
	analyzersMu.Lock()
	defer analyzersMu.Unlock()
	name := a.Name()
	if _, dup := analyzers[name]; dup {
		panic(fmt.Sprintf("rfscope: analyzer %q registered twice", name))
	}
	analyzers[name] = a
}

// Lookup returns the analyzer registered under name.
func Lookup(name string) (Analyzer, bool) {
	analyzersMu.RLock()
	defer analyzersMu.RUnlock()
	a, ok := analyzers[name]
	return a, ok
}

// Analyzers returns every registered analyzer sorted by name.
func Analyzers() []Analyzer {
	analyzersMu.RLock()
	defer analyzersMu.RUnlock()
	out := make([]Analyzer, 0, len(analyzers))
	for _, a := range analyzers {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Run resolves the requested analyzers (empty names ⇒ all registered) plus their
// transitive dependencies, orders them with a topological sort over DependsOn,
// and runs each Analyze in turn against sc. It returns an error on an unknown
// analyzer name, a dependency cycle, or the first Analyze failure.
func Run(ctx context.Context, sc *Scene, in *Input, names ...string) error {
	analyzersMu.RLock()
	reg := make(map[string]Analyzer, len(analyzers))
	for k, v := range analyzers {
		reg[k] = v
	}
	analyzersMu.RUnlock()

	order, err := topoOrder(reg, names)
	if err != nil {
		return err
	}
	if in != nil && in.Log == nil {
		in.Log = slog.New(slog.NewTextHandler(discard{}, nil))
	}
	for _, a := range order {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := a.Analyze(ctx, sc, in); err != nil {
			return fmt.Errorf("rfscope: analyzer %q: %w", a.Name(), err)
		}
	}
	return nil
}

// topoOrder is the pure ordering logic, separated from the global registry so it
// is unit-testable with a constructed map. It expands the requested set with
// transitive DependsOn, then returns a deterministic dependency-first ordering,
// erroring on an unknown name or a cycle.
func topoOrder(reg map[string]Analyzer, names []string) ([]Analyzer, error) {
	want := map[string]bool{}
	if len(names) == 0 {
		for n := range reg {
			want[n] = true
		}
	} else {
		for _, n := range names {
			if _, ok := reg[n]; !ok {
				return nil, fmt.Errorf("rfscope: unknown analyzer %q", n)
			}
			want[n] = true
		}
	}

	// Pull in transitive dependencies until the set is stable.
	for {
		added := false
		for n := range want {
			for _, d := range reg[n].DependsOn() {
				if _, ok := reg[d]; !ok {
					return nil, fmt.Errorf("rfscope: analyzer %q depends on unknown %q", n, d)
				}
				if !want[d] {
					want[d] = true
					added = true
				}
			}
		}
		if !added {
			break
		}
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	state := map[string]int{}
	var order []Analyzer
	var visit func(n string) error
	visit = func(n string) error {
		switch state[n] {
		case gray:
			return fmt.Errorf("rfscope: dependency cycle through %q", n)
		case black:
			return nil
		}
		state[n] = gray
		deps := append([]string(nil), reg[n].DependsOn()...)
		sort.Strings(deps)
		for _, d := range deps {
			if want[d] {
				if err := visit(d); err != nil {
					return err
				}
			}
		}
		state[n] = black
		order = append(order, reg[n])
		return nil
	}

	keys := make([]string, 0, len(want))
	for n := range want {
		keys = append(keys, n)
	}
	sort.Strings(keys)
	for _, n := range keys {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// discard is an io.Writer sink for the fallback logger.
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
