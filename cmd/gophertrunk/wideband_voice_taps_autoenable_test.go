package main

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// multiFakeDriver enumerates several streamingFakeDevices so a test can open a
// pool with more than one role (e.g. a wideband dongle plus a physical voice
// SDR) without real hardware.
type multiFakeDriver struct{ devs []*streamingFakeDevice }

func (m *multiFakeDriver) Name() string { return "fakemulti" }

func (m *multiFakeDriver) Enumerate() ([]sdr.Info, error) {
	infos := make([]sdr.Info, len(m.devs))
	for i, dv := range m.devs {
		infos[i] = sdr.Info{Driver: "fakemulti", Index: i, Serial: dv.serial, Gains: []int{0}}
	}
	return infos, nil
}

func (m *multiFakeDriver) Open(idx int) (sdr.Device, error) { return m.devs[idx], nil }

// openPool builds a Daemon with a pool opened from the given hints, wires the
// IQ brokers, and returns it ready for buildVirtualVoiceTuners.
func openPool(t *testing.T, cfg config.Config, hints []sdr.Hint) *Daemon {
	t.Helper()
	log := slog.New(slog.DiscardHandler)
	d := &Daemon{pool: sdr.NewPool(log)}
	if err := d.pool.OpenWith(sdr.PoolOpenOptions{
		SampleRateHz: cfg.SDR.SampleRate,
		Hints:        hints,
		Strict:       true,
	}); err != nil {
		t.Fatalf("pool open: %v", err)
	}
	d.wrapIQBrokers(cfg, log)
	if err := d.buildVirtualVoiceTuners(cfg, log); err != nil {
		t.Fatalf("buildVirtualVoiceTuners: %v", err)
	}
	return d
}

// TestWidebandAutoEnablesVoiceTapsWithoutPhysicalVoice pins the fix for the DMR
// "TGs/RIDs show but no audio and no recordings" foot-gun: a single role:
// wideband dongle with voice_taps unset (0) decodes control metadata but binds
// no voice tuner, so every grant is dropped. When trunking is configured and no
// physical role: voice SDR exists, the daemon must auto-enable virtual voice
// taps so the voice pool is non-empty and voice actually decodes.
func TestWidebandAutoEnablesVoiceTapsWithoutPhysicalVoice(t *testing.T) {
	const (
		wbSerial   = "wb-autoenable"
		centerHz   = 453_500_000
		sampleRate = 2_400_000
	)

	systemCfg := config.SDRConfig{
		SampleRate: sampleRate,
		Devices: []config.DeviceConfig{{
			Serial:       wbSerial,
			Role:         "wideband",
			CenterFreqHz: centerHz,
			// VoiceTaps intentionally left 0 (unset) — the whole point.
		}},
	}
	withSystems := config.Config{
		SDR: systemCfg,
		Trunking: config.TrunkingConfig{Systems: []config.SystemConfig{{
			Name:            "regional-dmr-t2",
			Protocol:        "dmr-tier2",
			ControlChannels: []uint32{453_125_000},
		}}},
	}

	t.Run("wideband-only trunking auto-enables taps", func(t *testing.T) {
		sdr.Register(&multiFakeDriver{devs: []*streamingFakeDevice{newStreamingFake(wbSerial)}})
		d := openPool(t, withSystems, []sdr.Hint{{Serial: wbSerial, Role: sdr.RoleWideband}})
		if got := len(d.collectVoiceDevices()); got != autoVoiceTaps {
			t.Fatalf("collectVoiceDevices = %d, want %d (auto-enabled taps for wideband-only DMR)", got, autoVoiceTaps)
		}
	})

	t.Run("no trunking systems does not auto-enable", func(t *testing.T) {
		sdr.Register(&multiFakeDriver{devs: []*streamingFakeDevice{newStreamingFake(wbSerial)}})
		noSystems := config.Config{SDR: systemCfg} // Trunking.Systems empty
		d := openPool(t, noSystems, []sdr.Hint{{Serial: wbSerial, Role: sdr.RoleWideband}})
		if got := len(d.collectVoiceDevices()); got != 0 {
			t.Fatalf("collectVoiceDevices = %d, want 0 (no trunking systems => no auto-enable)", got)
		}
	})

	t.Run("physical voice SDR present does not auto-enable", func(t *testing.T) {
		const voiceSerial = "voice-autoenable"
		sdr.Register(&multiFakeDriver{devs: []*streamingFakeDevice{
			newStreamingFake(wbSerial),
			newStreamingFake(voiceSerial),
		}})
		cfg := withSystems
		cfg.SDR.Devices = append([]config.DeviceConfig{}, systemCfg.Devices...)
		cfg.SDR.Devices = append(cfg.SDR.Devices, config.DeviceConfig{Serial: voiceSerial, Role: "voice"})
		d := openPool(t, cfg, []sdr.Hint{
			{Serial: wbSerial, Role: sdr.RoleWideband},
			{Serial: voiceSerial, Role: sdr.RoleVoice},
		})
		// A physical voice SDR carries the grants, so no virtual taps are added;
		// the pool has exactly the one physical voice device.
		if got := len(d.virtualVoiceTuners); got != 0 {
			t.Fatalf("virtualVoiceTuners = %d, want 0 (physical voice SDR present => no auto-enable)", got)
		}
		if got := len(d.collectVoiceDevices()); got != 1 {
			t.Fatalf("collectVoiceDevices = %d, want 1 (only the physical voice SDR)", got)
		}
	})
}
