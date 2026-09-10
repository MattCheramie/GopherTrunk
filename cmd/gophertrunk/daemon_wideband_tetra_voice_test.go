package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// registerMockDongle registers one mock SDR ("mock-00") backed by a short IQ
// file so pool.Open enumerates a device for the daemon under test; it is
// never streamed.
func registerMockDongle(t *testing.T) {
	t.Helper()
	iqPath := filepath.Join(t.TempDir(), "wb.cfile")
	if err := os.WriteFile(iqPath, make([]byte, 4096), 0o644); err != nil {
		t.Fatalf("write mock IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})
}

// buildWidebandDaemon constructs a daemon whose one (mock) dongle is a role:
// wideband device hosting the given systems' control channels, so the
// wideband same-carrier voice-tap registration can be asserted at
// construction time without SDR I/O.
func buildWidebandDaemon(t *testing.T, systems []config.SystemConfig, channels []config.DeviceChannelConfig) *Daemon {
	t.Helper()
	registerMockDongle(t)
	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{{
		Serial: "mock-00", Role: "wideband", CenterFreqHz: 467_900_000, Channels: channels,
	}}
	cfg.API.HTTPAddr = ""
	cfg.Trunking.Systems = systems
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = ""
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "test-wideband", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	return d
}

// TestWidebandTETRARegistersSameCarrierVoiceTaps pins the 10 Sep "two TETRA
// systems on one X310" fix: every TETRA control channel hosted on a role:
// wideband device gets its own four per-timeslot same-carrier voice taps
// (each bound to that channel's frequency), so a grant on either CC carrier
// follows on the channel's own 144 kHz tap through the shared per-carrier
// demux — before this the wideband path had no TETRA same-carrier source at
// all and only the single-tuner hunter (which camps on the first system
// that locks) could serve TETRA voice.
func TestWidebandTETRARegistersSameCarrierVoiceTaps(t *testing.T) {
	d := buildWidebandDaemon(t,
		[]config.SystemConfig{
			{Name: "250_013", Protocol: "tetra", ControlChannels: []uint32{467_912_500}},
			{Name: "250_208", Protocol: "tetra", ControlChannels: []uint32{467_875_000}},
		},
		[]config.DeviceChannelConfig{
			{FrequencyHz: 467_912_500, System: "250_013"},
			{FrequencyHz: 467_875_000, System: "250_208"},
		})
	if got, want := len(d.wbVoiceSources), 2*tetraSameCarrierTaps; got != want {
		t.Fatalf("registered %d wideband same-carrier taps, want %d (4 per TETRA channel)", got, want)
	}
	perFreq := map[uint32]int{}
	serials := map[string]bool{}
	for _, s := range d.wbVoiceSources {
		perFreq[s.FrequencyHz()]++
		if serials[s.Serial()] {
			t.Errorf("duplicate tap serial %q", s.Serial())
		}
		serials[s.Serial()] = true
	}
	if perFreq[467_912_500] != tetraSameCarrierTaps || perFreq[467_875_000] != tetraSameCarrierTaps {
		t.Errorf("taps per carrier = %v, want %d each", perFreq, tetraSameCarrierTaps)
	}
	// Every system is wideband-hosted, so the serial hunter has nothing to
	// multiplex and must not warn about sharing one control SDR.
	for _, w := range d.startupWarnings {
		if strings.Contains(w, "share ONE control SDR") {
			t.Errorf("unexpected single-tuner warning on a fully wideband-hosted config: %s", w)
		}
	}
	// The taps are voice devices the pool can bind and IQ sources the
	// composer can resolve.
	found := 0
	for _, v := range d.collectVoiceDevices() {
		if strings.Contains(v.Serial, "same-carrier:467912500") {
			found++
		}
	}
	if found != tetraSameCarrierTaps {
		t.Errorf("voice pool carries %d taps for 467.9125 MHz, want %d", found, tetraSameCarrierTaps)
	}
	if m := d.virtualVoiceMap(); m == nil || m[d.wbVoiceSources[0].Serial()] == nil {
		t.Error("wideband same-carrier taps missing from the composer's virtual voice map")
	}
}

// TestWidebandDMRRegistersNoSameCarrierVoiceTaps: conventional DMR on a
// wideband dongle keeps the on-air-verified VirtualTuner path; no wideband
// same-carrier taps are registered for it.
func TestWidebandDMRRegistersNoSameCarrierVoiceTaps(t *testing.T) {
	d := buildWidebandDaemon(t,
		[]config.SystemConfig{{Name: "Fire", Protocol: "dmr-tier2", ControlChannels: []uint32{467_912_500}}},
		[]config.DeviceChannelConfig{{FrequencyHz: 467_912_500, System: "Fire"}})
	if got := len(d.wbVoiceSources); got != 0 {
		t.Fatalf("DMR Tier II wideband channel registered %d same-carrier taps, want 0", got)
	}
}

// TestTwoSystemsOnOneControlSDRWarns pins the operator-facing diagnosis of
// the 10 Sep two-TETRA-system report: two systems on a single role: auto/
// control tuner are time-multiplexed (the second is never hunted while the
// first stays locked), and the daemon now says so at startup — and flags
// the `channels:` plan such a config typically carries on the non-wideband
// device, which was silently ignored.
func TestTwoSystemsOnOneControlSDRWarns(t *testing.T) {
	registerMockDongle(t)
	cfg := config.Default()
	cfg.SDR.SampleRate = 200_000
	cfg.SDR.Devices = []config.DeviceConfig{{
		Serial: "mock-00", Role: "auto", VoiceTaps: 4,
		Channels: []config.DeviceChannelConfig{{FrequencyHz: 467_912_500}, {FrequencyHz: 467_875_000}},
	}}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "250_013", Protocol: "tetra", ControlChannels: []uint32{467_912_500}},
		{Name: "250_208", Protocol: "tetra", ControlChannels: []uint32{467_875_000}},
	}
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = ""
	cfg.API.HTTPAddr = ""
	d, err := NewDaemon(cfg, "test-two-systems", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	var shared, dead bool
	for _, w := range d.startupWarnings {
		if strings.Contains(w, "share ONE control SDR") && strings.Contains(w, "250_208") {
			shared = true
		}
		if strings.Contains(w, "being IGNORED") && strings.Contains(w, "mock-00") {
			dead = true
		}
	}
	if !shared {
		t.Errorf("no single-control-SDR warning for two systems; warnings: %v", d.startupWarnings)
	}
	if !dead {
		t.Errorf("no dead-config warning for channels:/voice_taps on a role: auto device; warnings: %v", d.startupWarnings)
	}
	if got := len(d.wbVoiceSources); got != 0 {
		t.Errorf("non-wideband device registered %d wideband taps", got)
	}
}
