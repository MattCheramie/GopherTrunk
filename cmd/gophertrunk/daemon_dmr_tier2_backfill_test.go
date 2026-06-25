package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestSampleDMRTier2MultiRepeaterLoads pins the shipped many-repeater sample:
// it must pass config validation and construct a daemon whose conventional-DMR
// system has its control channels backfilled from the wideband channels (one
// per repeater, across all three dongles), so the sample's "no control_channels"
// stance stays honest.
func TestSampleDMRTier2MultiRepeaterLoads(t *testing.T) {
	const path = "../../samples/dmr-tier2-multi-repeater/config.yaml"
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load(%s): %v", path, err)
	}
	// The sample points storage / recordings at paths under its own folder;
	// blank them so constructing the daemon here leaves no artifacts in the
	// repo. We only care that the config validates and the system is built.
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = ""
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "test", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	var found bool
	for _, s := range d.systems {
		if s.Name != "regional-dmr" {
			continue
		}
		found = true
		// 3 dongles × 8 repeaters, all distinct frequencies.
		if len(s.ControlChannels) != 24 {
			t.Fatalf("regional-dmr derived ControlChannels = %d, want 24", len(s.ControlChannels))
		}
	}
	if !found {
		t.Fatal("regional-dmr system not constructed from the sample config")
	}
}

// TestWidebandChannelsForSystem covers the pure backfill helper: it returns the
// config-order, de-duplicated union of a system's wideband channel frequencies
// across all devices, skipping non-wideband devices and other systems' channels.
func TestWidebandChannelsForSystem(t *testing.T) {
	ch := func(freq uint32, system string) config.DeviceChannelConfig {
		return config.DeviceChannelConfig{FrequencyHz: freq, System: system}
	}
	devices := []config.DeviceConfig{
		{Serial: "RTL-A", Role: "wideband", Channels: []config.DeviceChannelConfig{
			ch(453_125_000, "regional-dmr"), ch(453_275_000, "regional-dmr"),
		}},
		{Serial: "RTL-B", Role: "wideband", Channels: []config.DeviceChannelConfig{
			ch(453_275_000, "regional-dmr"), // duplicate across devices → once
			ch(454_100_000, "regional-dmr"),
			ch(460_000_000, "other-system"), // different system → excluded
		}},
		{Serial: "RTL-C", Role: "control", Channels: []config.DeviceChannelConfig{
			ch(453_900_000, "regional-dmr"), // not a wideband device → excluded
		}},
	}

	got := widebandChannelsForSystem(devices, "regional-dmr")
	want := []uint32{453_125_000, 453_275_000, 454_100_000}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (config order, deduped)", got, want)
		}
	}

	if got := widebandChannelsForSystem(devices, "no-such-system"); len(got) != 0 {
		t.Fatalf("unknown system: got %v, want empty", got)
	}
	if got := widebandChannelsForSystem(nil, "regional-dmr"); len(got) != 0 {
		t.Fatalf("nil devices: got %v, want empty", got)
	}
}

// TestDMRTier2ControlChannelBackfill locks the conventional-DMR ergonomics
// change: a `dmr-tier2` / `dmr-tier1` system may omit `control_channels` and
// have them derived from the `role: wideband` channels assigned to it, so each
// repeater carrier is listed exactly once (in sdr.devices[].channels) instead
// of twice. Without the backfill, trunking.System.Validate() rejects the
// empty list with "at least one control_channel frequency is required" and
// NewDaemon fails — that is the red state this test pins.
func TestDMRTier2ControlChannelBackfill(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// baseCfg returns a Default() config with the API listener and storage
	// disabled so NewDaemon constructs a hermetic, side-effect-free daemon.
	baseCfg := func() config.Config {
		cfg := config.Default()
		cfg.API.HTTPAddr = ""
		return cfg
	}
	wbDevice := func(serial, system string, freqs ...uint32) config.DeviceConfig {
		d := config.DeviceConfig{Serial: serial, Role: "wideband", CenterFreqHz: 453_500_000}
		for _, f := range freqs {
			d.Channels = append(d.Channels, config.DeviceChannelConfig{FrequencyHz: f, System: system})
		}
		return d
	}
	findSystem := func(t *testing.T, d *Daemon, name string) trunking.System {
		t.Helper()
		for _, s := range d.systems {
			if s.Name == name {
				return s
			}
		}
		t.Fatalf("system %q not found among %d constructed systems", name, len(d.systems))
		return trunking.System{}
	}
	sameFreqs := func(got []uint32, want ...uint32) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range want {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}

	t.Run("backfill from a single wideband device", func(t *testing.T) {
		freqs := []uint32{453_125_000, 453_275_000, 453_775_000, 454_100_000}
		cfg := baseCfg()
		cfg.SDR.Devices = []config.DeviceConfig{wbDevice("RTL-A", "regional-dmr", freqs...)}
		cfg.Trunking.Systems = []config.SystemConfig{
			{Name: "regional-dmr", Protocol: "dmr-tier2"}, // no control_channels
		}
		d, err := NewDaemon(cfg, "test", logger)
		if err != nil {
			t.Fatalf("NewDaemon: %v (control_channels should be derived from wideband channels)", err)
		}
		sys := findSystem(t, d, "regional-dmr")
		if !sameFreqs(sys.ControlChannels, freqs...) {
			t.Fatalf("derived ControlChannels = %v, want %v (config order preserved)", sys.ControlChannels, freqs)
		}
	})

	t.Run("union across two wideband devices, deduped and ordered", func(t *testing.T) {
		cfg := baseCfg()
		cfg.SDR.Devices = []config.DeviceConfig{
			wbDevice("RTL-A", "regional-dmr", 453_125_000, 453_275_000),
			// 453_275_000 repeated across devices must appear once.
			wbDevice("RTL-B", "regional-dmr", 453_275_000, 454_100_000),
		}
		cfg.Trunking.Systems = []config.SystemConfig{
			{Name: "regional-dmr", Protocol: "dmr-tier2"},
		}
		d, err := NewDaemon(cfg, "test", logger)
		if err != nil {
			t.Fatalf("NewDaemon: %v", err)
		}
		sys := findSystem(t, d, "regional-dmr")
		if !sameFreqs(sys.ControlChannels, 453_125_000, 453_275_000, 454_100_000) {
			t.Fatalf("union ControlChannels = %v, want [453125000 453275000 454100000]", sys.ControlChannels)
		}
	})

	t.Run("explicit control_channels are left untouched", func(t *testing.T) {
		cfg := baseCfg()
		cfg.SDR.Devices = []config.DeviceConfig{
			wbDevice("RTL-A", "regional-dmr", 453_125_000, 453_275_000),
		}
		cfg.Trunking.Systems = []config.SystemConfig{
			// Explicit list differs from the channel freqs: must win.
			{Name: "regional-dmr", Protocol: "dmr-tier2", ControlChannels: []uint32{460_000_000}},
		}
		d, err := NewDaemon(cfg, "test", logger)
		if err != nil {
			t.Fatalf("NewDaemon: %v", err)
		}
		sys := findSystem(t, d, "regional-dmr")
		if !sameFreqs(sys.ControlChannels, 460_000_000) {
			t.Fatalf("ControlChannels = %v, want [460000000] (explicit list must not be overwritten)", sys.ControlChannels)
		}
	})

	t.Run("conventional DMR with no channels and no CCs still errors", func(t *testing.T) {
		cfg := baseCfg()
		// Wideband channels reference a DIFFERENT system, so there is nothing
		// to derive for regional-dmr.
		cfg.SDR.Devices = []config.DeviceConfig{wbDevice("RTL-A", "some-other-system", 453_125_000)}
		cfg.Trunking.Systems = []config.SystemConfig{
			{Name: "regional-dmr", Protocol: "dmr-tier2"},
			{Name: "some-other-system", Protocol: "dmr-tier2"},
		}
		if _, err := NewDaemon(cfg, "test", logger); err == nil {
			t.Fatal("expected an error: regional-dmr has no control_channels and no wideband channels")
		}
	})

	t.Run("dmr-tier1 direct mode is also backfilled", func(t *testing.T) {
		cfg := baseCfg()
		cfg.SDR.Devices = []config.DeviceConfig{wbDevice("RTL-A", "dmr-dm", 446_500_000)}
		cfg.Trunking.Systems = []config.SystemConfig{
			{Name: "dmr-dm", Protocol: "dmr-tier1"},
		}
		d, err := NewDaemon(cfg, "test", logger)
		if err != nil {
			t.Fatalf("NewDaemon: %v", err)
		}
		sys := findSystem(t, d, "dmr-dm")
		if !sameFreqs(sys.ControlChannels, 446_500_000) {
			t.Fatalf("derived ControlChannels = %v, want [446500000]", sys.ControlChannels)
		}
	})

	t.Run("trunked DMR Tier III is not backfilled", func(t *testing.T) {
		cfg := baseCfg()
		// A wideband channel hosts the Tier III control channel, but the
		// backfill must NOT apply to trunked protocols — they require an
		// explicit control_channels list.
		cfg.SDR.Devices = []config.DeviceConfig{wbDevice("RTL-A", "t3-system", 851_037_500)}
		cfg.Trunking.Systems = []config.SystemConfig{
			{Name: "t3-system", Protocol: "dmr"}, // Tier III, no control_channels
		}
		if _, err := NewDaemon(cfg, "test", logger); err == nil {
			t.Fatal("expected an error: dmr Tier III must declare control_channels explicitly")
		}
	})
}
