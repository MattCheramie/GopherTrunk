package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// buildSameCarrierDaemon constructs a daemon for a single role:control system on
// one carrier and returns it, so the same-carrier voice-tap registration can be
// asserted at construction time (no SDR I/O required).
func buildSameCarrierDaemon(t *testing.T, protocol string) *Daemon {
	t.Helper()
	cfg := config.Default()
	cfg.SDR.SampleRate = 48_000
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: "mock-00", Role: "control"}}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "OneCarrier", Protocol: protocol, ControlChannels: []uint32{460_500_000}},
	}
	// Leave no artifacts on disk.
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = ""
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "test-samecarrier", logger)
	if err != nil {
		t.Fatalf("NewDaemon(%s): %v", protocol, err)
	}
	return d
}

// TestConventionalDMRRegistersSameCarrierVoiceTaps is the #1036 regression: a
// conventional DMR (Tier II) repeater carries voice on the SAME carrier its Voice
// LC Headers decode on, so the daemon must register same-carrier voice taps for it
// — otherwise a grant on the repeater carrier has no voice source, is dropped, and
// nothing records. Before the fix the same-carrier tap was gated to TETRA only, so
// this registered 0 taps.
func TestConventionalDMRRegistersSameCarrierVoiceTaps(t *testing.T) {
	for _, proto := range []string{"dmr-tier2", "dmr-tier1"} {
		t.Run(proto, func(t *testing.T) {
			d := buildSameCarrierDaemon(t, proto)
			if got, want := len(d.ccVoiceSources), dmrSameCarrierTaps; got != want {
				t.Fatalf("%s registered %d same-carrier voice taps, want %d — a grant on the repeater carrier would have no voice source and never record (#1036)",
					proto, got, want)
			}
		})
	}
}

// TestTETRARegistersSameCarrierVoiceTaps pins that TETRA still registers its four
// per-timeslot taps (unchanged by the #1036 generalization).
func TestTETRARegistersSameCarrierVoiceTaps(t *testing.T) {
	d := buildSameCarrierDaemon(t, "tetra")
	if got, want := len(d.ccVoiceSources), tetraSameCarrierTaps; got != want {
		t.Fatalf("tetra registered %d same-carrier voice taps, want %d", got, want)
	}
}

// TestTrunkedDMRRegistersNoSameCarrierVoiceTaps pins that trunked DMR Tier III —
// whose voice hops to a separate traffic channel — registers NO same-carrier tap,
// so the "no voice SDR" diagnostic isn't masked for it.
func TestTrunkedDMRRegistersNoSameCarrierVoiceTaps(t *testing.T) {
	d := buildSameCarrierDaemon(t, "dmr")
	if got := len(d.ccVoiceSources); got != 0 {
		t.Fatalf("trunked DMR Tier III registered %d same-carrier voice taps, want 0", got)
	}
}
