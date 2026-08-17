package api

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// startServer boots a Server on a free port and returns its base URL plus the
// cancel func for the ctx Run is bound to. Unlike mkServer it hands back the
// Run error channel so a test can time how long shutdown actually takes.
func startServer(t *testing.T, opts ServerOptions) (base string, cancel func(), done <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	opts.Addr = addr

	s, err := NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelCtx := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return "http://" + addr, cancelCtx, errCh
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancelCtx()
	t.Fatalf("server didn't start on %s", addr)
	return "", nil, nil
}

// http.Server.Shutdown waits for active non-hijacked requests and does NOT
// cancel their r.Context(). An SSE subscriber therefore keeps the server
// "busy" until the shutdown context expires — the daemon's whole teardown sat
// on that window because the bus that would end the stream is closed long
// after the HTTP server is stopped. The server has to tell its own streaming
// handlers to leave.
func TestShutdownDoesNotWaitOutTheWindowForAnSSEClient(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	base, cancel, done := startServer(t, ServerOptions{Bus: bus})

	req, err := http.NewRequest(http.MethodGet, base+"/api/v1/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", resp.StatusCode)
	}
	// Read the priming comment so the handler is provably inside its loop
	// before we ask the server to stop.
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatalf("read SSE preamble: %v", err)
	}

	start := time.Now()
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return within 10 s of ctx cancel — the shutdown " +
			"window is being spent in full")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("shutdown took %v with one SSE client attached, want < 2s", elapsed)
	}
}

// Close() is the daemon's path (Daemon.Close calls httpAPI.Close), distinct
// from Run's ctx-cancel path. Both go through shutdown(), and both raced into
// a mutex held for the whole window.
func TestCloseDoesNotWaitOutTheWindowForAnSSEClient(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	s, err := NewServer(ServerOptions{Addr: addr, Bus: bus})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Run(context.Background()) }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("tcp", addr); err == nil {
			conn.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	resp, err := http.Get("http://" + addr + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatalf("read SSE preamble: %v", err)
	}

	closed := make(chan error, 1)
	start := time.Now()
	go func() { closed <- s.Close() }()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not return within 10 s")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Close took %v with one SSE client attached, want < 2s", elapsed)
	}
}
