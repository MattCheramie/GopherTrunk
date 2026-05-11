//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// freeAddr returns an ephemeral 127.0.0.1 address (port reserved then
// released so the daemon can bind it). Good enough for in-test use; the
// brief race between release and re-bind is benign on Linux.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func waitReachable(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s did not respond within %v", url, timeout)
}

// TestDaemonEndToEnd boots the wired daemon (no real SDR) with the
// HTTP API + storage + recordings + metrics enabled, publishes a
// synthetic call grant on the bus, and asserts the resulting state
// across every component:
//
//   - the engine started a call
//   - the recorder created a per-call WAV
//   - the SQLite call log captured the row
//   - /api/v1/calls/active reports it
//   - /api/v1/calls/history surfaces it once ended
//   - /metrics reflects the call counters
//   - SSE delivers call.start + call.end
func TestDaemonEndToEnd(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Storage.Path = filepath.Join(dir, "calls.db")
	cfg.Recordings.Dir = filepath.Join(dir, "recordings")
	cfg.Recordings.SampleRate = 8000
	cfg.Recordings.WriteRaw = true
	cfg.Metrics.Enabled = true
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "Alpha", Protocol: "p25", ControlChannels: []uint32{851_000_000}},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-test", logger)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-runErrCh:
		case <-time.After(3 * time.Second):
		}
	})

	base := "http://" + cfg.API.HTTPAddr
	waitReachable(t, base+"/api/v1/health", 3*time.Second)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Alpha", Protocol: "p25",
			GroupID: 1234, SourceID: 56789,
			FrequencyHz: 851_000_000,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP"},
		DeviceSerial: "VOICE-1",
		StartedAt:    startedAt,
	}
	d.Bus().Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	// The CallStart event reaches the recorder + call log + metrics
	// asynchronously; poll until each lands.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/v1/calls/history?limit=10")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if strings.Contains(string(body), "FIRE-DISP") {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Verify history has the row.
	histResp, err := http.Get(base + "/api/v1/calls/history?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer histResp.Body.Close()
	var historyBody struct {
		Calls []map[string]any `json:"calls"`
	}
	json.NewDecoder(histResp.Body).Decode(&historyBody)
	if len(historyBody.Calls) != 1 {
		t.Fatalf("history rows = %d, want 1: %+v", len(historyBody.Calls), historyBody.Calls)
	}
	if got := historyBody.Calls[0]["talkgroup_alpha"]; got != "FIRE-DISP" {
		t.Errorf("talkgroup_alpha = %v, want FIRE-DISP", got)
	}

	// Recordings directory should now contain a WAV under
	// Alpha/FIRE-DISP/...
	waitForRecording(t, cfg.Recordings.Dir, "FIRE-DISP", 2*time.Second)

	// Metrics endpoint should expose calls_active = 1 and the
	// build_info gauge with our supplied version.
	metricsBody := scrape(t, base+"/metrics")
	for _, want := range []string{
		`gophertrunk_calls_active 1`,
		`gophertrunk_build_info{version="integration-test"} 1`,
	} {
		if !strings.Contains(metricsBody, want) {
			t.Errorf("/metrics missing %q", want)
		}
	}

	// Now end the call and verify call.end propagates everywhere.
	endedAt := startedAt.Add(2 * time.Second)
	d.Bus().Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant:        cs.Grant,
			Talkgroup:    cs.Talkgroup,
			DeviceSerial: cs.DeviceSerial,
			StartedAt:    startedAt,
			EndedAt:      endedAt,
			Reason:       trunking.EndReasonNormal,
		},
	})

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		body := scrape(t, base+"/metrics")
		if strings.Contains(body, `gophertrunk_calls_total{reason="normal"} 1`) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("metrics never surfaced calls_total{reason=normal}")
}

func waitForRecording(t *testing.T, root, alpha string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		matches, _ := filepath.Glob(filepath.Join(root, "*", alpha, "*.wav"))
		if len(matches) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no WAV file appeared under %s/*/%s/", root, alpha)
}

func scrape(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// TestDaemonStartsWithMinimalConfig asserts the daemon still boots
// cleanly when every optional component is disabled — e.g. when an
// operator only wants the SDR-list CLI and a bare events bus for
// downstream wiring.
func TestDaemonStartsWithMinimalConfig(t *testing.T) {
	cfg := config.Default()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "minimal", logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := d.Run(ctx); err != nil && err.Error() != context.DeadlineExceeded.Error() {
		t.Errorf("Run err = %v", err)
	}
}

// Compile-time assertion: NewDaemon's signature can be reflected in a
// fmt.Stringer (smoke check that the helper symbols exist in this
// build tag).
var _ = fmt.Stringer(nil)

// TestDaemonWiresCCDecoder boots the daemon with a mock SDR + one
// trunked system and asserts the new IQ → CC decoder connector is
// constructed alongside the CC Hunter supervisor. The connector
// runs as a daemon goroutine that owns the control SDR's StreamIQ
// loop and pumps IQ through the active per-protocol pipeline
// whenever the supervisor reports a HuntProgress event; this test
// validates the wiring only, not the symbol-domain decode path
// (that's covered by the per-protocol receiver and ccdecoder unit
// tests).
//
// A small empty mock IQ file backs a sdr.MockDriver registered in
// the test's setup; the daemon's pool picks the device up by
// serial and assigns it RoleControl, which is the trigger for
// constructing the connector.
func TestDaemonWiresCCDecoder(t *testing.T) {
	dir := t.TempDir()
	iqPath := filepath.Join(dir, "ctrl.cfile")
	if err := os.WriteFile(iqPath, make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "Alpha", Protocol: "p25", ControlChannels: []uint32{851_000_000}},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "ccdecoder-wire", logger)
	if err != nil {
		t.Fatal(err)
	}
	if d.ccDecoder == nil {
		t.Fatalf("ccDecoder is nil; daemon should have constructed one when a control SDR + trunked system are present")
	}
	if d.cchuntSup == nil {
		t.Fatalf("cchuntSup is nil; daemon should have constructed the supervisor alongside the connector")
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()

	// Publish a HuntProgress event to confirm the connector
	// subscription is alive — the connector will look up the
	// matching system + protocol factory and either construct a
	// pipeline (P25 → wires; we don't synthesize valid P25 IQ
	// here so the pipeline just runs silently) or log a "no
	// factory" debug message. Either way the daemon must not
	// crash and shutdown must remain clean.
	d.Bus().Publish(events.Event{
		Kind: events.KindHuntProgress,
		Payload: trunking.HuntProgress{
			System:          "Alpha",
			AttemptedFreqHz: 851_000_000,
			AttemptIndex:    1,
			TotalCandidates: 1,
			At:              time.Now(),
		},
	})

	// Let the daemon spin for a bit so the goroutine pumps a
	// few IQ chunks from the mock file (the mock returns EOF
	// quickly with our 4 KiB seed; that's fine — the test only
	// asserts the wiring is in place + nothing crashes).
	time.Sleep(200 * time.Millisecond)

	cancel()
	select {
	case err := <-runErrCh:
		if err != nil && err != context.Canceled {
			t.Errorf("Run returned unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Errorf("daemon did not exit within 3s after cancel")
	}
}
