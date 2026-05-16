package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/tui"
	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// daemonWithoutHTTP builds a daemon that doesn't bind an HTTP
// listener — its HTTPListenAddr always returns "". Used to assert
// the "no HTTP API" error paths in tests that depend on the live
// listener being absent.
func daemonWithoutHTTP(t *testing.T) *Daemon {
	t.Helper()
	cfg := config.Default()
	cfg.API.HTTPAddr = "" // disabled
	cfg.API.GRPCAddr = ""
	cfg.Audio.Enabled = false
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemonWithPath(cfg, "", "test", logger)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// daemonForTest wires up a minimum daemon for in-process-TUI tests.
// It binds to a random loopback port via NewDaemonWithPath +
// returns its HTTP listen addr after the API server is up.
func daemonForTest(t *testing.T) (*Daemon, func()) {
	t.Helper()
	cfg := config.Default()
	// 127.0.0.1:0 → kernel picks a free port.
	cfg.API.HTTPAddr = "127.0.0.1:0"
	cfg.API.GRPCAddr = ""
	cfg.Audio.Enabled = false

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemonWithPath(cfg, "", "test", logger)
	if err != nil {
		t.Fatalf("daemon init: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() {
		runErr <- d.Run(ctx)
	}()

	// Wait for the listener to bind. HTTPListenAddr returns the
	// configured cfg.API.HTTPAddr ("127.0.0.1:0") immediately, so
	// poll for the actually-bound address (won't contain ":0") to
	// avoid racing against net.Listen.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		addr := d.HTTPListenAddr()
		if addr != "" && !strings.HasSuffix(addr, ":0") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	addr := d.HTTPListenAddr()
	if addr == "" || strings.HasSuffix(addr, ":0") {
		cancel()
		t.Fatalf("daemon HTTP listener never came up; addr=%q", addr)
	}

	cleanup := func() {
		cancel()
		select {
		case <-runErr:
		case <-time.After(3 * time.Second):
			t.Log("daemon did not shut down in time")
		}
	}
	return d, cleanup
}

func TestPrepareInProcessTUI_RequiresHTTPAddr(t *testing.T) {
	d := daemonWithoutHTTP(t)
	logSwap := gtlog.NewSwappableWriter(io.Discard)

	_, err := prepareInProcessTUI(d, logSwap)
	if err == nil {
		t.Fatal("expected error when HTTPListenAddr is empty")
	}
	if !strings.Contains(err.Error(), "api.http_addr") {
		t.Errorf("error should mention api.http_addr, got %q", err.Error())
	}
}

func TestPrepareInProcessTUI_RedirectsLogs(t *testing.T) {
	d, cleanup := daemonForTest(t)
	defer cleanup()

	// Capture original via a sentinel buffer.
	var originalSink strings.Builder
	logSwap := gtlog.NewSwappableWriter(&originalSink)

	setup, err := prepareInProcessTUI(d, logSwap)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.cleanup(slog.New(slog.NewTextHandler(io.Discard, nil)))

	if setup.logFile == nil {
		t.Fatal("expected logFile to be created")
	}
	// Write a marker through the swap and verify it lands in the
	// temp file (not the original sink).
	if _, err := logSwap.Write([]byte("MARKER\n")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(setup.logFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "MARKER") {
		t.Errorf("temp log should contain MARKER, got %q", string(body))
	}
	if strings.Contains(originalSink.String(), "MARKER") {
		t.Errorf("original sink should not have received MARKER while redirected; got %q", originalSink.String())
	}
}

func TestPrepareInProcessTUI_CleanupRestoresWriter(t *testing.T) {
	d, cleanup := daemonForTest(t)
	defer cleanup()

	var sink strings.Builder
	logSwap := gtlog.NewSwappableWriter(&sink)

	setup, err := prepareInProcessTUI(d, logSwap)
	if err != nil {
		t.Fatal(err)
	}
	tempPath := setup.logFile.Name()

	// Cleanup restores; subsequent writes should land back in sink.
	setup.cleanup(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := logSwap.Write([]byte("AFTER\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sink.String(), "AFTER") {
		t.Errorf("after cleanup, original sink should receive writes; got %q", sink.String())
	}
	// Temp log file is still on disk (we don't auto-delete) so
	// operators can inspect it post-mortem.
	if _, err := os.Stat(tempPath); err != nil {
		t.Errorf("temp log should still exist after cleanup; got %v", err)
	}
	_ = os.Remove(tempPath)
}

func TestPrepareInProcessTUI_ClientPointsAtDaemon(t *testing.T) {
	d, cleanup := daemonForTest(t)
	defer cleanup()

	logSwap := gtlog.NewSwappableWriter(io.Discard)
	setup, err := prepareInProcessTUI(d, logSwap)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.cleanup(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Smoke: the client should be able to round-trip /api/v1/health
	// against the bound daemon. We wait a beat for the listener to
	// settle, then call.
	hctx, hcancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer hcancel()
	if _, err := setup.cli.Health(hctx); err != nil {
		t.Errorf("client.Health: %v", err)
	}
}

// TestInProcessTUI_TeaTest exercises the actual bubbletea Update
// loop against a stub HTTP server so the message pump runs without
// a real terminal. The TUI's first tick fans out polls; we drive a
// single 'q' keystroke and verify clean exit.
func TestInProcessTUI_TeaTest(t *testing.T) {
	stub := httptest.NewServer(stubMuxForTUI())
	defer stub.Close()

	cli := client.New(stub.URL, 1*time.Second, false)
	m := tui.New(cli, tui.Options{Write: false})

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 30),
	)
	// Quit cleanly.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

// stubMuxForTUI returns a tiny http handler that satisfies every
// /api/v1 GET route the TUI's Init poll batch hits. Each route
// answers 200 + a valid-but-empty JSON body so the TUI client's
// JSON decoders don't bail. Mutation routes are not stubbed —
// the TUI shouldn't be issuing any during this test.
func stubMuxForTUI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/version":
			_, _ = w.Write([]byte(`{"version":"test"}`))
		case "/api/v1/runtime":
			_, _ = w.Write([]byte(`{"version":"test","config_path":""}`))
		case "/api/v1/systems":
			_, _ = w.Write([]byte(`{"systems":[]}`))
		case "/api/v1/talkgroups":
			_, _ = w.Write([]byte(`{"talkgroups":[]}`))
		case "/api/v1/calls/active":
			_, _ = w.Write([]byte(`{"calls":[]}`))
		case "/api/v1/calls/history":
			_, _ = w.Write([]byte(`{"calls":[]}`))
		case "/api/v1/devices":
			_, _ = w.Write([]byte(`{"devices":[]}`))
		case "/api/v1/scanner":
			_, _ = w.Write([]byte(`{}`))
		case "/api/v1/audio":
			_, _ = w.Write([]byte(`{}`))
		case "/api/v1/mutations":
			_, _ = w.Write([]byte(`{"can_mutate":false,"allow_mutations":false,"auth_mode":"disabled"}`))
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("# HELP test\n"))
		default:
			// Fall back to empty JSON so unknown routes added in
			// the future don't break this test.
			_, _ = w.Write([]byte(`{}`))
		}
	})
}
