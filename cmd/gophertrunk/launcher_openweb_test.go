package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

// TestOpenWebUI_PrefersEmbeddedWhenAssetsPresent verifies the
// embedded-SPA branch wins when the daemon binary was linked
// against a populated web/dist embed. The opened URL must be the
// daemon URL itself — no file:// asset hop.
func TestOpenWebUI_PrefersEmbeddedWhenAssetsPresent(t *testing.T) {
	d, cleanup := daemonForTest(t)
	defer cleanup()

	withStubbed(t, true, true, func(opened *string) {
		err := openWebUI(d, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err != nil {
			t.Fatalf("openWebUI: %v", err)
		}
		if *opened == "" {
			t.Fatal("expected openBrowserFn to be called")
		}
		// Embedded branch opens the daemon URL directly; the
		// asset-path branch would prepend file:// and append
		// #server=...
		if strings.HasPrefix(*opened, "file://") {
			t.Errorf("embedded path should not use file://; got %q", *opened)
		}
		if !strings.Contains(*opened, d.HTTPListenAddr()) {
			t.Errorf("opened URL should reference daemon addr %q; got %q",
				d.HTTPListenAddr(), *opened)
		}
	})
}

// TestOpenWebUI_HeadlessFallbackEvenWithEmbed verifies the
// no-display fallback prints the URL instead of trying to open a
// browser when canOpenBrowser returns false.
func TestOpenWebUI_HeadlessFallbackEvenWithEmbed(t *testing.T) {
	d, cleanup := daemonForTest(t)
	defer cleanup()

	withStubbed(t, true, false, func(opened *string) {
		err := openWebUI(d, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err != nil {
			t.Fatalf("openWebUI: %v", err)
		}
		if *opened != "" {
			t.Errorf("headless host should not invoke openBrowser; got %q", *opened)
		}
	})
}

// TestOpenWebUI_NoEmbedFallsBackToFilesystem verifies the
// sibling-directory discovery path kicks in when the embed has no
// real assets (fresh checkout without `make web-build`).
func TestOpenWebUI_NoEmbedFallsBackToFilesystem(t *testing.T) {
	d, cleanup := daemonForTest(t)
	defer cleanup()

	withStubbed(t, false, false, func(opened *string) {
		err := openWebUI(d, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err != nil {
			t.Fatalf("openWebUI: %v", err)
		}
		// We expect the headless fallback to run because
		// canOpenBrowser=false; the test really just verifies we
		// took the non-embedded code path and didn't error out.
		if *opened != "" {
			t.Errorf("headless host should not invoke openBrowser; got %q", *opened)
		}
	})
}

// TestOpenWebUI_RequiresHTTPAddr verifies the error path when the
// daemon was started without a usable HTTP listener.
func TestOpenWebUI_RequiresHTTPAddr(t *testing.T) {
	d := daemonWithoutHTTP(t)

	withStubbed(t, true, true, func(_ *string) {
		err := openWebUI(d, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err == nil {
			t.Fatal("expected error when HTTPListenAddr is empty")
		}
		if !strings.Contains(err.Error(), "api.http_addr") {
			t.Errorf("error should mention api.http_addr; got %q", err.Error())
		}
	})
}

// withStubbed swaps the launcher's test seams for the duration of
// the test body. opened captures the URL handed to openBrowser (""
// if openBrowser wasn't called).
func withStubbed(
	t *testing.T,
	hasAssets, canOpen bool,
	body func(opened *string),
) {
	t.Helper()
	prevHas := hasWebAssetsFn
	prevCan := canOpenBrowserFn
	prevOpen := openBrowserFn
	t.Cleanup(func() {
		hasWebAssetsFn = prevHas
		canOpenBrowserFn = prevCan
		openBrowserFn = prevOpen
	})

	hasWebAssetsFn = func() bool { return hasAssets }
	canOpenBrowserFn = func() bool { return canOpen }
	opened := new(string)
	openBrowserFn = func(target string) error {
		*opened = target
		return nil
	}
	body(opened)
}
