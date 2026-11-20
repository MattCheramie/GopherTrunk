package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Two control channels of one P25 system on a wideband dongle centred on
// 467.900 MHz at 2.4 MS/s: one 12.5 kHz from the centre, one 400 kHz away.
// Both sit in the passband, so the symbol panels' choice between them is
// decided by which one counts as locked.
const (
	lockedTestCentreHz = 467_900_000
	lockedTestRateHz   = 2_400_000
	lockedTestNearCC   = 467_912_500
	lockedTestFarCC    = 468_300_000
)

// buildWidebandDaemonWithAPI is buildWidebandDaemon with the HTTP API
// built, so the test reads the spectrum devices the web console reads,
// through the daemon's own wiring.
func buildWidebandDaemonWithAPI(t *testing.T) *Daemon {
	t.Helper()
	iqPath := filepath.Join(t.TempDir(), "wb.cfile")
	if err := os.WriteFile(iqPath, make([]byte, 4096), 0o644); err != nil {
		t.Fatalf("write mock IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})
	cfg := config.Default()
	cfg.SDR.SampleRate = lockedTestRateHz
	cfg.SDR.Devices = []config.DeviceConfig{{
		Serial: "mock-00", Role: "wideband", CenterFreqHz: lockedTestCentreHz,
		Channels: []config.DeviceChannelConfig{
			{FrequencyHz: lockedTestNearCC, System: "metro"},
			{FrequencyHz: lockedTestFarCC, System: "metro"},
		},
	}}
	cfg.Trunking.Systems = []config.SystemConfig{{
		Name: "metro", Protocol: "p25", ControlChannels: []uint32{lockedTestNearCC, lockedTestFarCC},
	}}
	cfg.API.HTTPAddr = "127.0.0.1:0"
	cfg.API.GRPCAddr = ""
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = ""
	d, err := NewDaemon(cfg, "test-locked-cc", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if d.httpAPI == nil {
		t.Fatal("daemon built no HTTP API")
	}
	br := d.iqBrokers["mock-00"]
	if br == nil {
		t.Fatal("no IQ broker for the wideband dongle")
	}
	if err := br.SetCenterFreq(lockedTestCentreHz); err != nil {
		t.Fatal(err)
	}
	if err := br.SetSampleRate(lockedTestRateHz); err != nil {
		t.Fatal(err)
	}
	return d
}

// spectrumControlChannel serves the daemon's API and returns
// control_channel_hz for the wideband dongle from GET
// /api/v1/spectrum/devices.
func spectrumControlChannel(t *testing.T, d *Daemon, ctx context.Context) uint32 {
	t.Helper()
	srv := d.httpAPI
	go func() { _ = srv.Run(ctx) }()
	deadline := time.Now().Add(2 * time.Second)
	for srv.BoundAddr() == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if srv.BoundAddr() == "" {
		t.Fatal("API server never bound")
	}
	resp, err := http.Get("http://" + srv.BoundAddr() + "/api/v1/spectrum/devices")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/spectrum/devices: %s", resp.Status)
	}
	var devices []struct {
		Serial           string `json:"serial"`
		ControlChannelHz uint32 `json:"control_channel_hz"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatal(err)
	}
	for _, dev := range devices {
		if dev.Serial == "mock-00" {
			return dev.ControlChannelHz
		}
	}
	t.Fatalf("mock-00 missing from %+v", devices)
	return 0
}

// TestSymbolPanelsIgnoreAControlChannelThatStoppedDecoding pins the review
// finding on PR #1384. The site table behind GET /api/v1/sites never drops
// an entry, so as a "locked" source it holds every control channel that
// decoded once since start-up. A control channel that decoded earlier and
// has since gone quiet (a fade, a site off the air, a hunter that moved on)
// kept the symbol panels on it for the rest of the run, even with the
// channel decoding now nearer the tuner, the dead-air symptom #1384 fixes.
//
// Here the far channel decoded once and nothing decodes now. It must not
// count as locked: the panels fall back to the configured choice, as they do
// before any lock.
func TestSymbolPanelsIgnoreAControlChannelThatStoppedDecoding(t *testing.T) {
	d := buildWidebandDaemonWithAPI(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.siteTracker.Run(ctx) }()

	// What the far channel's decoder published while it decoded.
	d.bus.Publish(events.Event{Kind: events.KindSiteUpdate, Payload: trunking.SiteUpdate{
		System: "metro", RFSSID: 1, SiteID: 2, ControlChannelHz: lockedTestFarCC, At: time.Now(),
	}})
	deadline := time.Now().Add(2 * time.Second)
	for {
		sites := d.siteTracker.Snapshot()
		if len(sites) == 1 && sites[0].ControlChannelHz == lockedTestFarCC {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("site tracker never recorded the far channel: %+v", sites)
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got := spectrumControlChannel(t, d, ctx); got != lockedTestNearCC {
		t.Fatalf("control_channel_hz = %d, want the configured choice %d: %d decoded once and "+
			"has stopped, so it is not locked", got, lockedTestNearCC, lockedTestFarCC)
	}
}
