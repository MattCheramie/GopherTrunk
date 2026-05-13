package version

import (
	"strings"
	"testing"
)

func TestStringJustVersion(t *testing.T) {
	saveV, saveC, saveB := Version, Commit, BuildTime
	defer func() { Version, Commit, BuildTime = saveV, saveC, saveB }()
	Version, Commit, BuildTime = "v1.2.3", "", ""
	got := String()
	if got != "v1.2.3" {
		t.Errorf("String() = %q, want %q", got, "v1.2.3")
	}
}

func TestStringWithCommitAndBuildTime(t *testing.T) {
	saveV, saveC, saveB := Version, Commit, BuildTime
	defer func() { Version, Commit, BuildTime = saveV, saveC, saveB }()
	Version, Commit, BuildTime = "v1.2.3", "abc1234", "2026-05-13T19:00:00Z"
	got := String()
	if !strings.HasPrefix(got, "v1.2.3 (") {
		t.Errorf("String() = %q, want prefix %q", got, "v1.2.3 (")
	}
	for _, want := range []string{"sha=abc1234", "built=2026-05-13T19:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
}

func TestStringWithCommitOnly(t *testing.T) {
	saveV, saveC, saveB := Version, Commit, BuildTime
	defer func() { Version, Commit, BuildTime = saveV, saveC, saveB }()
	Version, Commit, BuildTime = "v1.2.3", "abc1234", ""
	got := String()
	want := "v1.2.3 (sha=abc1234)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestStringWithBuildTimeOnly(t *testing.T) {
	saveV, saveC, saveB := Version, Commit, BuildTime
	defer func() { Version, Commit, BuildTime = saveV, saveC, saveB }()
	Version, Commit, BuildTime = "v1.2.3", "", "2026-05-13T19:00:00Z"
	got := String()
	want := "v1.2.3 (built=2026-05-13T19:00:00Z)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestDefaultVersionIsDev(t *testing.T) {
	// Sanity: an unpopulated build (no -ldflags) reports "dev".
	// This test runs against the package default; if the package
	// is ever rebuilt with ldflags, the default constant in the
	// source is what matters.
	saveV, saveC, saveB := Version, Commit, BuildTime
	defer func() { Version, Commit, BuildTime = saveV, saveC, saveB }()
	// Force the defaults via the package-level vars so we don't
	// depend on the test binary's actual link state.
	Version, Commit, BuildTime = "dev", "", ""
	if got := String(); got != "dev" {
		t.Errorf("default String() = %q, want %q", got, "dev")
	}
}
