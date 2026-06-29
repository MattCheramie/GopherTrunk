package rfscope

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeAnalyzer is a test double recording the order it ran in.
type fakeAnalyzer struct {
	name    string
	deps    []string
	runErr  error
	ranInto *[]string
}

func (f fakeAnalyzer) Name() string        { return f.name }
func (f fakeAnalyzer) Synopsis() string    { return "fake " + f.name }
func (f fakeAnalyzer) DependsOn() []string { return f.deps }
func (f fakeAnalyzer) Analyze(_ context.Context, _ *Scene, _ *Input) error {
	if f.ranInto != nil {
		*f.ranInto = append(*f.ranInto, f.name)
	}
	return f.runErr
}

func regOf(as ...Analyzer) map[string]Analyzer {
	m := make(map[string]Analyzer, len(as))
	for _, a := range as {
		m[a.Name()] = a
	}
	return m
}

func names(as []Analyzer) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Name()
	}
	return out
}

func TestTopoOrderDependencyFirst(t *testing.T) {
	t.Parallel()
	// topology depends on segment; hierarchy depends on segment.
	reg := regOf(
		fakeAnalyzer{name: "segment"},
		fakeAnalyzer{name: "hierarchy", deps: []string{"segment"}},
		fakeAnalyzer{name: "topology", deps: []string{"segment"}},
	)
	order, err := topoOrder(reg, nil)
	if err != nil {
		t.Fatalf("topoOrder: %v", err)
	}
	pos := map[string]int{}
	for i, n := range names(order) {
		pos[n] = i
	}
	if pos["segment"] > pos["hierarchy"] || pos["segment"] > pos["topology"] {
		t.Fatalf("segment must precede its dependents: %v", names(order))
	}
	if len(order) != 3 {
		t.Fatalf("want all 3 analyzers, got %v", names(order))
	}
}

func TestTopoOrderPullsTransitiveDeps(t *testing.T) {
	t.Parallel()
	// Requesting only "conv" must still run its transitive deps.
	reg := regOf(
		fakeAnalyzer{name: "segment"},
		fakeAnalyzer{name: "emitters", deps: []string{"segment"}},
		fakeAnalyzer{name: "conv", deps: []string{"emitters"}},
	)
	order, err := topoOrder(reg, []string{"conv"})
	if err != nil {
		t.Fatalf("topoOrder: %v", err)
	}
	got := names(order)
	if len(got) != 3 || got[0] != "segment" || got[1] != "emitters" || got[2] != "conv" {
		t.Fatalf("transitive deps not pulled/ordered: %v", got)
	}
}

func TestTopoOrderDetectsCycle(t *testing.T) {
	t.Parallel()
	reg := regOf(
		fakeAnalyzer{name: "a", deps: []string{"b"}},
		fakeAnalyzer{name: "b", deps: []string{"a"}},
	)
	if _, err := topoOrder(reg, nil); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("want cycle error, got %v", err)
	}
}

func TestTopoOrderUnknownName(t *testing.T) {
	t.Parallel()
	reg := regOf(fakeAnalyzer{name: "segment"})
	if _, err := topoOrder(reg, []string{"nope"}); err == nil {
		t.Fatalf("want unknown-analyzer error")
	}
	// Unknown dependency is also an error.
	reg2 := regOf(fakeAnalyzer{name: "x", deps: []string{"ghost"}})
	if _, err := topoOrder(reg2, nil); err == nil {
		t.Fatalf("want unknown-dependency error")
	}
}

func TestRunPropagatesAnalyzeError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	var ran []string
	reg := regOf(
		fakeAnalyzer{name: "ok", ranInto: &ran},
		fakeAnalyzer{name: "bad", deps: []string{"ok"}, runErr: boom, ranInto: &ran},
	)
	order, err := topoOrder(reg, nil)
	if err != nil {
		t.Fatalf("topoOrder: %v", err)
	}
	// Drive Analyze directly in resolved order to mirror Run's loop.
	var runErr error
	for _, a := range order {
		if runErr = a.Analyze(context.Background(), &Scene{}, &Input{}); runErr != nil {
			break
		}
	}
	if !errors.Is(runErr, boom) {
		t.Fatalf("want boom, got %v", runErr)
	}
	if len(ran) != 2 || ran[0] != "ok" {
		t.Fatalf("ok should run before bad: %v", ran)
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	t.Parallel()
	// Run against the real global registry: with no analyzers registered yet in
	// this package's tests, an all-analyzers Run is a no-op and must succeed.
	if err := Run(context.Background(), &Scene{}, &Input{}); err != nil {
		t.Fatalf("empty Run: %v", err)
	}
}
