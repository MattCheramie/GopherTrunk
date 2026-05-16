package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPickLaunchMode(t *testing.T) {
	cases := []struct {
		name        string
		tui         bool
		web         bool
		headless    bool
		want        launchMode
		expectError bool
	}{
		{"no flags → auto", false, false, false, launchAuto, false},
		{"-tui", true, false, false, launchTUI, false},
		{"-web", false, true, false, launchWeb, false},
		{"-headless", false, false, true, launchHeadless, false},
		{"-tui + -web → error", true, true, false, launchAuto, true},
		{"-tui + -headless → error", true, false, true, launchAuto, true},
		{"-web + -headless → error", false, true, true, launchAuto, true},
		{"all three → error", true, true, true, launchAuto, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pickLaunchMode(tc.tui, tc.web, tc.headless)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error, got mode=%v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got mode=%v want %v", got, tc.want)
			}
		})
	}
}

func TestLaunchModeFlag(t *testing.T) {
	cases := map[launchMode]string{
		launchAuto:     "auto",
		launchTUI:      "tui",
		launchWeb:      "web",
		launchHeadless: "headless",
	}
	for m, want := range cases {
		if got := launchModeFlag(m); got != want {
			t.Errorf("launchModeFlag(%v) = %q want %q", m, got, want)
		}
	}
}

func TestNormaliseServerURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{":8080", "http://localhost:8080"},
		{"0.0.0.0:8080", "http://localhost:8080"},
		{"127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"[::]:8080", "http://localhost:8080"},
		{"192.168.1.5:8080", "http://192.168.1.5:8080"},
	}
	for _, tc := range cases {
		if got := normaliseServerURL(tc.in); got != tc.want {
			t.Errorf("normaliseServerURL(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildWebTargetURL(t *testing.T) {
	asset := filepath.Join(t.TempDir(), "index.html")
	out := buildWebTargetURL(asset, "http://127.0.0.1:8080")
	if !strings.HasPrefix(out, "file://") {
		t.Fatalf("expected file:// prefix, got %q", out)
	}
	if !strings.Contains(out, "#server=http://127.0.0.1:8080") {
		t.Fatalf("expected server hash, got %q", out)
	}
}

func TestBuildWebTargetURL_NoAsset(t *testing.T) {
	out := buildWebTargetURL("", "http://127.0.0.1:8080")
	if out != "http://127.0.0.1:8080" {
		t.Fatalf("got %q; want passthrough to the server URL when no assets found", out)
	}
}

func TestFindWebAssets_Empty(t *testing.T) {
	// With nothing in canonical locations during tests, findWebAssets
	// can either return "" (no assets) or the dev-tree web/dist if
	// the user happens to have built it. Both are valid; the
	// invariant we test is "no panic, returns a string".
	_ = findWebAssets()
}
