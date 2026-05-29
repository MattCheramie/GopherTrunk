package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("default log level = %q, want info", cfg.Log.Level)
	}
	if cfg.SDR.SampleRate != 2_400_000 {
		t.Errorf("default sample rate = %d, want 2400000", cfg.SDR.SampleRate)
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	yaml := `
log:
  level: debug
  format: json
sdr:
  sample_rate: 2400000
  devices:
    - serial: "00000001"
      role: control
      ppm: -2
trunking:
  systems:
    - name: TestSystem
      protocol: p25
      control_channels: [851000000, 852000000]
`
	if err := writeFile(path, yaml); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("level = %q", cfg.Log.Level)
	}
	if len(cfg.SDR.Devices) != 1 || cfg.SDR.Devices[0].Role != "control" {
		t.Errorf("devices = %+v", cfg.SDR.Devices)
	}
	if len(cfg.Trunking.Systems) != 1 || cfg.Trunking.Systems[0].Protocol != "p25" {
		t.Errorf("systems = %+v", cfg.Trunking.Systems)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"ok", Default(), false},
		{"bad sample rate", Config{SDR: SDRConfig{SampleRate: 100}}, true},
		{"bad role", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Role: "bogus"}}}}, true},
		{"bad protocol", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "lte"}}}}, true},
		{"tetra protocol", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "tetra"}}}}, false},
		{"nxdn protocol", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn"}}}}, false},
		{"missing name", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Protocol: "p25"}}}}, true},
		{"rc4 key ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rc4", Key: "0123456789"}}}}}}, false},
		{"arc4 alias ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "ARC4", Key: "0x AB CD EF"}}}}}}, false},
		{"missing algorithm", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Key: "abcd"}}}}}}, true},
		{"aes not supported", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "aes", Key: "abcd"}}}}}}, true},
		{"unknown algorithm", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rot13", Key: "abcd"}}}}}}, true},
		{"bad hex key", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rc4", Key: "xyz"}}}}}}, true},
		{"empty key", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rc4", Key: ""}}}}}}, true},
		{"duplicate key_id", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rc4", Key: "ab"}, {KeyID: 1, Algorithm: "rc4", Key: "cd"}}}}}}, true},
		{"duplicate sdr serial", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Serial: "00000006", Role: "control"}, {Serial: "00000006", Role: "voice"}}}}, true},
		{"distinct sdr serials ok", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Serial: "00000001", Role: "control"}, {Serial: "00000002", Role: "voice"}}}}, false},
		{"empty sdr serials ok", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Role: "control"}, {Role: "voice"}}}}, false},
		// sdr.ignore_serials: a top-level opt-out list for dongles
		// the daemon should discover-but-skip, e.g. when an HF+ is
		// reserved for OpenWebRX+ on the same host.
		{"ignore_serials happy path", Config{SDR: SDRConfig{IgnoreSerials: []string{"AIRSPYHF SN:foo"}}}, false},
		{"ignore_serials with trim ok", Config{SDR: SDRConfig{IgnoreSerials: []string{"  AIRSPYHF SN:foo  "}}}, false},
		{"ignore_serials empty entry", Config{SDR: SDRConfig{IgnoreSerials: []string{""}}}, true},
		{"ignore_serials whitespace entry", Config{SDR: SDRConfig{IgnoreSerials: []string{"   "}}}, true},
		{"ignore_serials duplicate within list", Config{SDR: SDRConfig{IgnoreSerials: []string{"X", "X"}}}, true},
		{"ignore_serials duplicate after trim", Config{SDR: SDRConfig{IgnoreSerials: []string{"X", "  X "}}}, true},
		{"ignore_serials overlaps devices", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Serial: "00000011", Role: "voice"}}, IgnoreSerials: []string{"00000011"}}}, true},
		{"ignore_serials overlaps rtl_tcp", Config{SDR: SDRConfig{RTLTCP: []RTLTCPConfig{{Addr: "127.0.0.1:1234", Serial: "rtltcp-x", Role: "voice"}}, IgnoreSerials: []string{"rtltcp-x"}}}, true},
		{"oversized key", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rc4", Key: strings.Repeat("ab", 33)}}}}}}, true},
		// p25_band_plan: the operator's escape hatch for sites that
		// never broadcast IDEN_UP for some channel ID (issue #345).
		{"band_plan ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 10, BaseHz: 425_262_500, SpacingHz: 6250, TxOffsetHz: 4_000_000, BandwidthHz: 12500}}}}}}, false},
		{"band_plan channel_id too high", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 16, BaseHz: 1, SpacingHz: 1}}}}}}, true},
		{"band_plan duplicate channel_id", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 3, BaseHz: 1, SpacingHz: 1}, {ChannelID: 3, BaseHz: 2, SpacingHz: 2}}}}}}, true},
		{"band_plan zero spacing", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 3, BaseHz: 1, SpacingHz: 0}}}}}}, true},
		{"band_plan zero base", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 3, BaseHz: 0, SpacingHz: 1}}}}}}, true},
		// wideband role: pin a dongle to a centre frequency and list
		// per-repeater carriers inside its IQ bandwidth. Stage 2 added
		// DMR Tier II conventional; Stage 3 added DMR Tier III trunked
		// control channel.
		{"wideband T2 ok", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "regional-t2"},
					{FrequencyHz: 453_775_000, System: "regional-t2"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "regional-t2", Protocol: "dmr-tier2"}}},
		}, false},
		{"wideband T3 ok", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_775_000, System: "regional-t3"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-t3", Protocol: "dmr",
				ControlChannels: []uint32{453_775_000},
			}}},
		}, false},
		{"wideband T3 channel not in CC list", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "regional-t3"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-t3", Protocol: "dmr",
				ControlChannels: []uint32{453_775_000}, // doesn't include 453_125_000
			}}},
		}, true},
		{"wideband mixed T2 + T3", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "regional-t2"},
					{FrequencyHz: 453_775_000, System: "regional-t3"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{
				{Name: "regional-t2", Protocol: "dmr-tier2"},
				{Name: "regional-t3", Protocol: "dmr", ControlChannels: []uint32{453_775_000}},
			}},
		}, false},
		{"wideband missing serial", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "x"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2"}}},
		}, true},
		{"wideband missing center", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband",
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "x"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2"}}},
		}, true},
		{"wideband missing channels", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
			}}},
		}, true},
		{"wideband channel out of band", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 460_000_000, System: "x"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2"}}},
		}, true},
		{"wideband unknown system", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "nope"},
				},
			}}},
		}, true},
		// P25 Phase 1 wideband CC tap (parallel to the T3 case above):
		// channel must sit on one of the system's declared
		// control_channels because a Phase 1 trunked control channel
		// IS one.
		{"wideband P25 Phase 1 ok", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_037_500, System: "regional-p25"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p25", Protocol: "p25",
				ControlChannels: []uint32{851_037_500},
			}}},
		}, false},
		{"wideband P25 Phase 1 channel not in CC list", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_125_000, System: "regional-p25"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p25", Protocol: "p25",
				ControlChannels: []uint32{851_037_500},
			}}},
		}, true},
		{"wideband P25 Phase 2 ok", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_006_250, System: "regional-p2"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p2", Protocol: "p25-phase2",
				ControlChannels: []uint32{851_006_250},
			}}},
		}, false},
		// voice_taps: virtual voice DDC tap count per wideband
		// dongle. 0 disables (default); 1-8 allocates that many.
		{"wideband voice_taps in range", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000, VoiceTaps: 4,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_037_500, System: "regional-p25"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p25", Protocol: "p25",
				ControlChannels: []uint32{851_037_500},
			}}},
		}, false},
		{"wideband voice_taps too large", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000, VoiceTaps: 99,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_037_500, System: "regional-p25"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p25", Protocol: "p25",
				ControlChannels: []uint32{851_037_500},
			}}},
		}, true},
		{"wideband voice_taps negative", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000, VoiceTaps: -1,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_037_500, System: "regional-p25"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p25", Protocol: "p25",
				ControlChannels: []uint32{851_037_500},
			}}},
		}, true},
		// Non-trunked protocols are still rejected — wideband doesn't
		// host NXDN / EDACS / etc. yet.
		{"wideband unsupported protocol", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "nxdn-sys"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "nxdn-sys", Protocol: "nxdn"}}},
		}, true},
		{"wideband duplicate frequency", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "x"},
					{FrequencyHz: 453_125_000, System: "x"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2"}}},
		}, true},
		{"wideband bad strategy", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 453_500_000, TunerStrategy: "magic",
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 453_125_000, System: "x"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2"}}},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func writeFile(path, data string) error {
	return writeFileImpl(path, []byte(data))
}
