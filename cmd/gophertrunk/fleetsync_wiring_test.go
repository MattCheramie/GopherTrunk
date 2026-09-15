package main

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestDaemonWiresFleetSyncChannels pins the fleetsync.channels → receiver
// construction path (#1184): a well-formed entry yields a live receiver
// slot, a malformed one keeps its (nil) slot for stable indexing and
// surfaces a startup warning instead of failing the daemon — the same
// contract the MDC1200 / DSC / APRS lists honour.
func TestDaemonWiresFleetSyncChannels(t *testing.T) {
	cfg := config.Default()
	cfg.API.HTTPAddr = ""
	cfg.SDR.Devices = nil
	cfg.SDR.RTLTCP = []config.RTLTCPConfig{{Addr: "127.0.0.1:1", Serial: "fs-rig"}}
	cfg.FleetSync.Channels = []config.FleetSyncChannelConfig{
		{Serial: "fs-rig", FrequencyHz: 462_562_500},
		{Serial: "fs-rig", FrequencyHz: 462_562_500, BaudHz: 2400, DropBadCRC: true},
		{Serial: "", FrequencyHz: 462_562_500}, // missing serial → skipped
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "fleetsync-wiring-test", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	if got := len(d.fleetsyncReceivers); got != 3 {
		t.Fatalf("fleetsyncReceivers = %d slots, want 3 (one per config entry)", got)
	}
	if d.fleetsyncReceivers[0] == nil || d.fleetsyncReceivers[1] == nil {
		t.Errorf("well-formed entries must construct receivers: %v", d.fleetsyncReceivers[:2])
	}
	if d.fleetsyncReceivers[1].BaudHz() != 2400 {
		t.Errorf("BaudHz not threaded through: got %d", d.fleetsyncReceivers[1].BaudHz())
	}
	if d.fleetsyncReceivers[0].BaudHz() != 1200 {
		t.Errorf("default BaudHz = %d, want 1200", d.fleetsyncReceivers[0].BaudHz())
	}
	if d.fleetsyncReceivers[2] != nil {
		t.Errorf("entry without a serial must be skipped (nil slot)")
	}
	if d.fleetsyncSpecs[0].serial != "fs-rig" || d.fleetsyncSpecs[0].freq != 462_562_500 {
		t.Errorf("spec[0] = %+v", d.fleetsyncSpecs[0])
	}
	var warned bool
	for _, w := range d.StartupWarnings() {
		if strings.Contains(w, "fleetsync.channels") && strings.Contains(w, "skipped") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("malformed fleetsync.channels entry produced no startup warning: %v", d.StartupWarnings())
	}
}

// TestDaemonOpensFleetSyncLogWithStorage pins the storage → REST half: with
// storage.path set the daemon owns a FleetSyncLog (so the /fleetsync panel
// and GET /api/v1/fleetsync/messages have a backing table); without it the
// preflight names fleetsync among the decoders that need storage.path.
func TestDaemonOpensFleetSyncLogWithStorage(t *testing.T) {
	cfg := config.Default()
	cfg.API.HTTPAddr = ""
	cfg.SDR.Devices = nil
	cfg.Storage = config.StorageConfig{Path: filepath.Join(t.TempDir(), "calls.db")}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "fleetsync-storage-test", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)
	if d.fleetsyncLog == nil {
		t.Fatal("storage.path set but fleetsyncLog is nil")
	}
	if rows, err := d.fleetsyncLog.Recent(5); err != nil || len(rows) != 0 {
		t.Errorf("fresh fleetsync_log Recent = %v, %v; want empty, nil", rows, err)
	}

	noStore := config.Config{
		FleetSync: config.FleetSyncConfig{Channels: []config.FleetSyncChannelConfig{
			{Serial: "fs-rig", FrequencyHz: 462_562_500},
		}},
	}
	warnings, err := preflight(noStore)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	var named bool
	for _, w := range warnings {
		if strings.Contains(w, "storage.path is empty") && strings.Contains(w, "fleetsync") {
			named = true
		}
	}
	if !named {
		t.Errorf("preflight must name fleetsync among the decoders needing storage.path: %v", warnings)
	}
}
