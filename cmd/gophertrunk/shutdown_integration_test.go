//go:build integration

package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestDaemonShutdownWithStreamingClientsAttached is the regression for the
// operator-visible "Ctrl-C, then nothing happens for 30 seconds" report: their
// debug.log showed "gophertrunk shutdown initiated" and "gophertrunk shutdown
// complete" exactly 30.001 s apart with one line in between.
//
// The cause was an ordering inversion. Daemon.Close stops the HTTP server
// first, and http.Server.Shutdown waits for active non-hijacked requests
// without cancelling their contexts — so an attached SSE or live-audio
// subscriber held the window open until it expired, while the bus that would
// have ended those handlers is closed at the very end of Close.
func TestDaemonShutdownWithStreamingClientsAttached(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Storage.Path = filepath.Join(dir, "calls.db")
	cfg.Recordings.Dir = filepath.Join(dir, "recordings")
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "Alpha", Protocol: "p25", ControlChannels: []uint32{851_000_000}},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "shutdown-test", logger)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()

	base := "http://" + cfg.API.HTTPAddr
	waitReachable(t, base+"/api/v1/health", 3*time.Second)

	// Attach an SSE subscriber and read its priming comment so the handler is
	// provably inside its event loop before the daemon is asked to stop.
	resp, err := http.Get(base + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", resp.StatusCode)
	}
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatalf("read SSE preamble: %v", err)
	}

	start := time.Now()
	cancel()
	select {
	case <-runErrCh:
	case <-time.After(20 * time.Second):
		t.Fatal("daemon did not shut down within 20 s — the HTTP drain window " +
			"is being spent in full")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("shutdown took %v with an SSE client attached, want < 5s", elapsed)
	}
}
