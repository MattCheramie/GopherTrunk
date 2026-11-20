package config

import (
	"strings"
	"testing"
)

// TestValidateSystemRequiresControlChannels pins that config validation
// rejects every trunking system the daemon refuses to start on (#1379
// follow-up). The daemon checks each system with trunking.System.Validate,
// which requires at least one control channel within 25–1300 MHz; config
// validation did not, so `config validate`, `doctor` and the web config
// builder passed a config whose daemon then failed at start-up with
// "trunking: at least one control_channel frequency is required".
func TestValidateSystemRequiresControlChannels(t *testing.T) {
	cases := []struct {
		name    string
		ccs     []uint32
		wantErr string // "" = valid
	}{
		{"none", nil, "control_channels"},
		{"empty list", []uint32{}, "control_channels"},
		{"below 25 MHz", []uint32{24_999_999}, "control_channels[0]"},
		{"above 1300 MHz", []uint32{851_000_000, 1_300_000_001}, "control_channels[1]"},
		{"zero entry", []uint32{0}, "control_channels[0]"},
		{"valid", []uint32{851_012_500}, ""},
		{"valid edges", []uint32{25_000_000, 1_300_000_000}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			cfg.Trunking.Systems = []SystemConfig{
				{Name: "site", Protocol: "p25", ControlChannels: tc.ccs},
			}
			errs := cfg.ValidateSection("trunking")
			if tc.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("valid control_channels %v rejected: %v", tc.ccs, errs)
				}
				return
			}
			if len(errs) == 0 {
				t.Fatalf("control_channels %v accepted; the daemon refuses to start on it", tc.ccs)
			}
			if !strings.Contains(errs[0].Error(), tc.wantErr) {
				t.Fatalf("error %q does not name %q", errs[0], tc.wantErr)
			}
			if !strings.Contains(errs[0].Error(), "trunking.systems[0]") {
				t.Fatalf("error %q does not name the system's path", errs[0])
			}
		})
	}
}

// TestValidateSystemWidebandDMRNeedsNoControlChannels pins the exemption
// the daemon has always applied (EffectiveControlChannels): a conventional
// DMR system whose carriers are channels of a role: wideband device needs
// no control_channels list, and must not be rejected by config validation.
// A trunked system on the same device still needs its list.
func TestValidateSystemWidebandDMRNeedsNoControlChannels(t *testing.T) {
	wideband := func(system string) SDRConfig {
		return SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
			Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
			Channels: []DeviceChannelConfig{
				{FrequencyHz: 453_125_000, System: system},
				{FrequencyHz: 453_775_000, System: system},
			},
		}}}
	}
	for _, proto := range []string{"dmr-tier2", "dmr-tier1"} {
		cfg := Default()
		cfg.SDR = wideband("regional")
		cfg.Trunking.Systems = []SystemConfig{{Name: "regional", Protocol: proto}}
		if errs := cfg.ValidateSection("trunking"); len(errs) != 0 {
			t.Fatalf("%s on wideband channels without control_channels rejected: %v", proto, errs)
		}
		got := cfg.Trunking.Systems[0].EffectiveControlChannels(cfg.SDR.Devices)
		if len(got) != 2 || got[0] != 453_125_000 || got[1] != 453_775_000 {
			t.Fatalf("%s: effective control channels = %v, want the two wideband carriers", proto, got)
		}
	}
	// No wideband channel names it: nothing to derive, so rejected.
	cfg := Default()
	cfg.SDR = wideband("other")
	cfg.Trunking.Systems = []SystemConfig{{Name: "regional", Protocol: "dmr-tier2"}}
	if errs := cfg.ValidateSection("trunking"); len(errs) == 0 {
		t.Fatalf("dmr-tier2 with no control_channels and no wideband channel accepted")
	}
	// A trunked protocol is never derived, even with wideband channels.
	cfg = Default()
	cfg.SDR = wideband("regional")
	cfg.Trunking.Systems = []SystemConfig{{Name: "regional", Protocol: "p25"}}
	if errs := cfg.ValidateSection("trunking"); len(errs) == 0 {
		t.Fatalf("p25 with no control_channels accepted because of wideband channels")
	}
}
