package main

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestCchuntSystems locks issue #749: control-channel frequencies hosted on a
// `role: wideband` device (decoded in parallel by widebandt2) must be removed
// from the sequential cchunt supervisor's hunt set, and a system whose control
// channels are ALL wideband-hosted must be dropped entirely.
func TestCchuntSystems(t *testing.T) {
	// helper: find a system by name in a result slice.
	find := func(systems []trunking.System, name string) (trunking.System, bool) {
		for _, s := range systems {
			if s.Name == name {
				return s, true
			}
		}
		return trunking.System{}, false
	}
	wbDevice := func(serial string, ccs ...uint32) config.DeviceConfig {
		d := config.DeviceConfig{Serial: serial, Role: "wideband", CenterFreqHz: 420_900_000}
		for _, f := range ccs {
			d.Channels = append(d.Channels, config.DeviceChannelConfig{FrequencyHz: f, System: "MMR"})
		}
		return d
	}

	t.Run("all CCs wideband-hosted drops the system", func(t *testing.T) {
		systems := []trunking.System{{
			Name:            "MMR",
			Protocol:        trunking.ProtocolP25,
			ControlChannels: []uint32{420_012_500, 420_037_500, 420_087_500, 421_837_500},
		}}
		dev := wbDevice("AIRSPY", 420_012_500, 420_037_500, 420_087_500, 421_837_500)
		got := cchuntSystems(systems, []config.DeviceConfig{dev})
		if _, ok := find(got, "MMR"); ok {
			t.Fatalf("MMR should be dropped (all 4 CCs wideband-hosted); got %d systems", len(got))
		}
	})

	t.Run("partial leaves only non-wideband CCs and does not mutate input", func(t *testing.T) {
		systems := []trunking.System{{
			Name:            "MMR",
			Protocol:        trunking.ProtocolP25,
			ControlChannels: []uint32{420_012_500, 420_037_500, 420_087_500, 421_837_500},
		}}
		dev := wbDevice("AIRSPY", 420_012_500, 420_037_500)
		got := cchuntSystems(systems, []config.DeviceConfig{dev})
		mmr, ok := find(got, "MMR")
		if !ok {
			t.Fatal("MMR should survive (2 of 4 CCs still need hunting)")
		}
		want := []uint32{420_087_500, 421_837_500}
		if len(mmr.ControlChannels) != len(want) {
			t.Fatalf("filtered CCs = %v, want %v", mmr.ControlChannels, want)
		}
		for i := range want {
			if mmr.ControlChannels[i] != want[i] {
				t.Fatalf("filtered CCs = %v, want %v", mmr.ControlChannels, want)
			}
		}
		// Aliasing guard: the input slice must be untouched (it is shared with
		// the config-backed backing array elsewhere in the daemon).
		if got := len(systems[0].ControlChannels); got != 4 {
			t.Fatalf("input ControlChannels was mutated: len = %d, want 4", got)
		}
	})

	t.Run("non-control wideband channel changes nothing", func(t *testing.T) {
		systems := []trunking.System{{
			Name:            "MMR",
			Protocol:        trunking.ProtocolP25,
			ControlChannels: []uint32{420_012_500, 420_037_500},
		}}
		// A wideband channel frequency that is NOT one of MMR's control_channels.
		dev := config.DeviceConfig{
			Serial: "AIRSPY", Role: "wideband", CenterFreqHz: 420_900_000,
			Channels: []config.DeviceChannelConfig{{FrequencyHz: 420_500_000, System: "MMR"}},
		}
		got := cchuntSystems(systems, []config.DeviceConfig{dev})
		mmr, ok := find(got, "MMR")
		if !ok || len(mmr.ControlChannels) != 2 {
			t.Fatalf("MMR CCs = %v, want both retained", mmr.ControlChannels)
		}
	})

	t.Run("no wideband device passes through unchanged", func(t *testing.T) {
		systems := []trunking.System{{
			Name:            "MMR",
			Protocol:        trunking.ProtocolP25,
			ControlChannels: []uint32{420_012_500, 420_037_500},
		}}
		dev := config.DeviceConfig{Serial: "RTL", Role: "control"}
		got := cchuntSystems(systems, []config.DeviceConfig{dev})
		mmr, ok := find(got, "MMR")
		if !ok || len(mmr.ControlChannels) != 2 {
			t.Fatalf("MMR CCs = %v, want unchanged", mmr.ControlChannels)
		}
	})

	t.Run("two wideband devices union to cover all CCs", func(t *testing.T) {
		systems := []trunking.System{{
			Name:            "MMR",
			Protocol:        trunking.ProtocolP25,
			ControlChannels: []uint32{420_012_500, 420_037_500, 420_087_500, 421_837_500},
		}}
		devA := wbDevice("AIRSPY-A", 420_012_500, 420_037_500)
		devB := wbDevice("AIRSPY-B", 420_087_500, 421_837_500)
		got := cchuntSystems(systems, []config.DeviceConfig{devA, devB})
		if _, ok := find(got, "MMR"); ok {
			t.Fatal("MMR should be dropped (union of two wideband devices covers all CCs)")
		}
	})

	t.Run("same frequency on a different system is not filtered", func(t *testing.T) {
		const shared = 420_012_500
		systems := []trunking.System{
			{Name: "MMR", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{shared}},
			{Name: "OTHER", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{shared}},
		}
		// Only MMR's frequency is wideband-hosted.
		dev := wbDevice("AIRSPY", shared)
		got := cchuntSystems(systems, []config.DeviceConfig{dev})
		if _, ok := find(got, "MMR"); ok {
			t.Error("MMR should be dropped (its only CC is wideband-hosted)")
		}
		other, ok := find(got, "OTHER")
		if !ok || len(other.ControlChannels) != 1 || other.ControlChannels[0] != shared {
			t.Errorf("OTHER must keep its CC %d (same freq, different system); got %v", shared, other.ControlChannels)
		}
	})
}
