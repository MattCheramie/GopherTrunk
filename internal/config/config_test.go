package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func u8ptr(v uint8) *uint8 { return &v }

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
		{"normalize apply_to recording ok", Config{Recordings: RecordingsConfig{Normalize: NormalizeConfig{Enabled: true, ApplyTo: "recording"}}}, false},
		{"normalize apply_to distributed ok", Config{Recordings: RecordingsConfig{Normalize: NormalizeConfig{Enabled: true, ApplyTo: "distributed"}}}, false},
		{"normalize apply_to both ok", Config{Recordings: RecordingsConfig{Normalize: NormalizeConfig{Enabled: true, ApplyTo: "both"}}}, false},
		{"normalize apply_to empty ok", Config{Recordings: RecordingsConfig{Normalize: NormalizeConfig{Enabled: true}}}, false},
		{"normalize apply_to invalid", Config{Recordings: RecordingsConfig{Normalize: NormalizeConfig{Enabled: true, ApplyTo: "stream"}}}, true},
		// recordings naming templates.
		{"filename_template ok", Config{Recordings: RecordingsConfig{FilenameTemplate: "{date}_{time}_{tg}_{freq}"}}, false},
		{"filename_template unknown token", Config{Recordings: RecordingsConfig{FilenameTemplate: "{date}_{tgid}"}}, true},
		{"filename_template with separator", Config{Recordings: RecordingsConfig{FilenameTemplate: "{tg}/{date}"}}, true},
		{"path_template ok", Config{Recordings: RecordingsConfig{PathTemplate: "{system}/{year}/{month}/{day}"}}, false},
		{"path_template unknown token", Config{Recordings: RecordingsConfig{PathTemplate: "{system}/{quarter}"}}, true},
		{"path_template absolute", Config{Recordings: RecordingsConfig{PathTemplate: "/abs/{system}"}}, true},
		{"bad sample rate", Config{SDR: SDRConfig{SampleRate: 100}}, true},
		// Wideband soapy_remote sources (issue #550): rates above the RTL
		// 3.2 MHz hardware cap are valid config, bounded at 20 MHz.
		{"wideband sample rate 10M", Config{SDR: SDRConfig{SampleRate: 10_000_000}}, false},
		{"wideband sample rate 20M", Config{SDR: SDRConfig{SampleRate: 20_000_000}}, false},
		{"sample rate above 20M", Config{SDR: SDRConfig{SampleRate: 20_000_001}}, true},
		{"bad role", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Role: "bogus"}}}}, true},
		{"bad protocol", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "lte"}}}}, true},
		{"tetra protocol", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "tetra"}}}}, false},
		{"nxdn protocol", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn"}}}}, false},
		{"missing name", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Protocol: "p25"}}}}, true},
		// per-system encrypted_calls policy (issue #711).
		{"encrypted_calls empty ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", EncryptedCalls: EncryptedCallsConfig{}}}}}, false},
		{"encrypted_calls follow ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", EncryptedCalls: EncryptedCallsConfig{Mode: "follow"}}}}}, false},
		{"encrypted_calls metadata ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", EncryptedCalls: EncryptedCallsConfig{Mode: "metadata", MetadataFollowMs: 1000}}}}}, false},
		{"encrypted_calls ignore ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", EncryptedCalls: EncryptedCallsConfig{Mode: "ignore"}}}}}, false},
		{"encrypted_calls bad mode", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", EncryptedCalls: EncryptedCallsConfig{Mode: "drop"}}}}}, true},
		{"encrypted_calls negative follow", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", EncryptedCalls: EncryptedCallsConfig{Mode: "metadata", MetadataFollowMs: -1}}}}}, true},
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
		{"oversized key", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", EncryptionKeys: []EncryptionKeyConfig{{KeyID: 1, Algorithm: "rc4", Key: strings.Repeat("ab", 33)}}}}}}, true},
		// p25_band_plan: the operator's escape hatch for sites that
		// never broadcast IDEN_UP for some channel ID (issue #345).
		{"band_plan ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 10, BaseHz: 425_262_500, SpacingHz: 6250, TxOffsetHz: 4_000_000, BandwidthHz: 12500}}}}}}, false},
		{"band_plan channel_id too high", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 16, BaseHz: 1, SpacingHz: 1}}}}}}, true},
		{"band_plan duplicate channel_id", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 3, BaseHz: 1, SpacingHz: 1}, {ChannelID: 3, BaseHz: 2, SpacingHz: 2}}}}}}, true},
		{"band_plan zero spacing", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 3, BaseHz: 1, SpacingHz: 0}}}}}}, true},
		{"band_plan zero base", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", P25BandPlan: []P25BandPlanEntryConfig{{ChannelID: 3, BaseHz: 0, SpacingHz: 1}}}}}}, true},
		// dmr_band_plan: DMR Tier III LCN→frequency resolver. Exactly one
		// of linear / table is required; without it T3 voice grants drop.
		{"dmr band_plan linear ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Linear: &DMRLinearBandPlanConfig{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1}}}}}}, false},
		{"dmr band_plan table ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Table: []DMRBandPlanTableEntryConfig{{LCN: 1, FreqHz: 866_000_000}, {LCN: 2, FreqHz: 866_025_000}}}}}}}, false},
		{"dmr band_plan both linear and table", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Linear: &DMRLinearBandPlanConfig{BaseHz: 1, SpacingHz: 1}, Table: []DMRBandPlanTableEntryConfig{{LCN: 1, FreqHz: 1}}}}}}}, true},
		{"dmr band_plan empty", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{}}}}}, true},
		{"dmr band_plan linear zero spacing", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Linear: &DMRLinearBandPlanConfig{BaseHz: 1, SpacingHz: 0}}}}}}, true},
		{"dmr band_plan linear zero base", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Linear: &DMRLinearBandPlanConfig{BaseHz: 0, SpacingHz: 1}}}}}}, true},
		{"dmr band_plan table zero freq", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Table: []DMRBandPlanTableEntryConfig{{LCN: 1, FreqHz: 0}}}}}}}, true},
		{"dmr band_plan table duplicate lcn", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRBandPlan: &DMRBandPlanConfig{Table: []DMRBandPlanTableEntryConfig{{LCN: 1, FreqHz: 1}, {LCN: 1, FreqHz: 2}}}}}}}, true},
		// color_code: conventional DMR (IPSC / linked-repeater) colour-code
		// hard filter. Valid 0..15 on dmr-tier2/dmr-tier1 only.
		{"color_code on dmr-tier2 ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2", DMRColorCode: u8ptr(12)}}}}, false},
		{"color_code on dmr-tier1 ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier1", DMRColorCode: u8ptr(0)}}}}, false},
		{"color_code out of range", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr-tier2", DMRColorCode: u8ptr(16)}}}}, true},
		{"color_code on trunked dmr rejected", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "dmr", DMRColorCode: u8ptr(1), DMRBandPlan: &DMRBandPlanConfig{Linear: &DMRLinearBandPlanConfig{BaseHz: 1, SpacingHz: 1}}}}}}, true},
		{"color_code on p25 rejected", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "p25", DMRColorCode: u8ptr(1)}}}}, true},
		{"nxdn band_plan linear ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn", NXDNBandPlan: &NXDNBandPlanConfig{Linear: &NXDNLinearBandPlanConfig{BaseHz: 461_000_000, SpacingHz: 12_500, Offset: 1}}}}}}, false},
		{"nxdn band_plan table ok", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn", NXDNBandPlan: &NXDNBandPlanConfig{Table: []NXDNBandPlanTableEntryConfig{{Channel: 1, FreqHz: 461_000_000}, {Channel: 2, FreqHz: 461_012_500}}}}}}}, false},
		{"nxdn band_plan both linear and table", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn", NXDNBandPlan: &NXDNBandPlanConfig{Linear: &NXDNLinearBandPlanConfig{BaseHz: 1, SpacingHz: 1}, Table: []NXDNBandPlanTableEntryConfig{{Channel: 1, FreqHz: 1}}}}}}}, true},
		{"nxdn band_plan empty", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn", NXDNBandPlan: &NXDNBandPlanConfig{}}}}}, true},
		{"nxdn band_plan linear zero spacing", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn", NXDNBandPlan: &NXDNBandPlanConfig{Linear: &NXDNLinearBandPlanConfig{BaseHz: 1, SpacingHz: 0}}}}}}, true},
		{"nxdn band_plan table duplicate channel", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Name: "x", Protocol: "nxdn", NXDNBandPlan: &NXDNBandPlanConfig{Table: []NXDNBandPlanTableEntryConfig{{Channel: 1, FreqHz: 1}, {Channel: 1, FreqHz: 2}}}}}}}, true},
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
		{"wideband TETRA ok", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 467_900_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 467_912_500, System: "tetra-net"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "tetra-net", Protocol: "tetra",
				ControlChannels: []uint32{467_912_500},
			}}},
		}, false},
		{"wideband mixed T3 + TETRA", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 460_000_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 459_775_000, System: "regional-t3"},
					{FrequencyHz: 460_212_500, System: "tetra-net"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{
				{Name: "regional-t3", Protocol: "dmr", ControlChannels: []uint32{459_775_000}},
				{Name: "tetra-net", Protocol: "tetra", ControlChannels: []uint32{460_212_500}},
			}},
		}, false},
		{"wideband TETRA channel not in CC list", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 467_900_000,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 467_800_000, System: "tetra-net"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "tetra-net", Protocol: "tetra",
				ControlChannels: []uint32{467_912_500}, // doesn't include 467_800_000
			}}},
		}, true},
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
		// No hard upper bound on voice_taps: a high count validates
		// (the daemon warns about CPU above 16 at tap-build time).
		{"wideband voice_taps high count allowed", Config{
			SDR: SDRConfig{SampleRate: 2_400_000, Devices: []DeviceConfig{{
				Serial: "00000010", Role: "wideband", CenterFreqHz: 851_500_000, VoiceTaps: 28,
				Channels: []DeviceChannelConfig{
					{FrequencyHz: 851_037_500, System: "regional-p25"},
				},
			}}},
			Trunking: TrunkingConfig{Systems: []SystemConfig{{
				Name: "regional-p25", Protocol: "p25",
				ControlChannels: []uint32{851_037_500},
			}}},
		}, false},
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
		// web.tabs: turn off nav items. Known keys (in any state) are
		// fine; an unknown key is rejected so typos surface at load.
		{"web tabs known ok", Config{Web: WebConfig{Tabs: map[string]bool{"pagers": false, "metrics": true}}}, false},
		{"web tabs unknown key", Config{Web: WebConfig{Tabs: map[string]bool{"pagerz": false}}}, true},
		// soapy_remote args: SoapySDR make() kwargs as "k=v,k=v" (issue #542).
		{"soapy args ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "rx_subdev_spec=A:0,antenna=RX1"}}}}, false},
		{"soapy args malformed", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "rx_subdev_spec"}}}}, true},
		// soapy_remote args must not carry SoapyRemote stream knobs: GT builds
		// the SETUP_STREAM frame itself and ignores remote:* in make() args, so
		// they belong in the dedicated stream_* keys, not args (issue #876).
		{"soapy args remote:mtu rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "remote:mtu=8000"}}}}, true},
		{"soapy args remote:window rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "remote:window=16777216"}}}}, true},
		{"soapy args remote:prot rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "remote:prot=tcp"}}}}, true},
		{"soapy args remote:mtu mixed rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "antenna=RX1,remote:mtu=8000"}}}}, true},
		// UHD frame-size make() args are legitimate and stay allowed.
		{"soapy args recv_frame_size ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "num_recv_frames=512,recv_frame_size=16384"}}}}, false},
		// soapy_remote diversity (issue #1062): "" / "mrc" ok, anything else rejected.
		{"soapy diversity mrc ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: "mrc"}}}}, false},
		{"soapy diversity empty ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: ""}}}}, false},
		{"soapy diversity bad rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: "selection"}}}}, true},
		{"soapy diversity mrc-static ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: "mrc-static"}}}}, false},
		// The escape hatch still opens two RX channels, so a per-channel antenna
		// pair must stay legal under it.
		{"soapy antennas pair ok under mrc-static", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: "mrc-static", Antennas: []string{"RX1", "RX2"}}}}}, false},
		// antennas[]: one is fine single-channel; two require mrc; >2 and empties rejected.
		{"soapy antennas single ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Antennas: []string{"RX2"}}}}}, false},
		{"soapy antennas two with mrc ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: "mrc", Antennas: []string{"RX1", "RX2"}}}}}, false},
		{"soapy antennas two without mrc rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Antennas: []string{"RX1", "RX2"}}}}}, true},
		{"soapy antennas three rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Diversity: "mrc", Antennas: []string{"RX1", "RX2", "RX3"}}}}}, true},
		{"soapy antennas empty entry rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Antennas: []string{""}}}}}, true},
		{"soapy antennas conflict with args rejected", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", Args: "antenna=RX1", Antennas: []string{"RX2"}}}}}, true},
		// soapy_remote stream_mtu: 0 = default; a real MTU is fine; out-of-range fails.
		{"soapy stream_mtu zero ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1"}}}}, false},
		{"soapy stream_mtu valid", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamMTU: 8192}}}}, false},
		{"soapy stream_mtu too small", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamMTU: 10}}}}, true},
		{"soapy stream_mtu too large", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamMTU: 1 << 21}}}}, true},
		// soapy_remote stream_window: 0 = default; a real window is fine;
		// out-of-range or below the MTU fails.
		{"soapy stream_window zero ok", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1"}}}}, false},
		{"soapy stream_window valid", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamWindow: 16 << 20}}}}, false},
		{"soapy stream_window too small", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamWindow: 1024}}}}, true},
		{"soapy stream_window too large", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamWindow: 512 << 20}}}}, true},
		{"soapy stream_window below mtu", Config{SDR: SDRConfig{SoapyRemote: []SoapyRemoteConfig{{Addr: "h:1", StreamMTU: 1 << 20, StreamWindow: 1 << 16}}}}, true},
		// ka9q_radio: addr + ssrc required; role/encoding/channels constrained.
		{"ka9q ok", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{Addr: "hf.local", SSRC: 162550}}}}, false},
		{"ka9q ok full", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{Addr: "239.1.2.3:5006", SSRC: 1, Role: "control", Encoding: "f32le", Channels: 2}}}}, false},
		{"ka9q missing addr", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{SSRC: 1}}}}, true},
		{"ka9q missing ssrc", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{Addr: "hf.local"}}}}, true},
		{"ka9q bad role", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{Addr: "hf.local", SSRC: 1, Role: "nope"}}}}, true},
		{"ka9q bad encoding", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{Addr: "hf.local", SSRC: 1, Encoding: "opus"}}}}, true},
		{"ka9q bad channels", Config{SDR: SDRConfig{Ka9qRadio: []Ka9qRadioConfig{{Addr: "hf.local", SSRC: 1, Channels: 3}}}}, true},
		{"ka9q serial collides", Config{SDR: SDRConfig{Devices: []DeviceConfig{{Serial: "x"}}, Ka9qRadio: []Ka9qRadioConfig{{Addr: "hf.local", SSRC: 1, Serial: "x"}}}}, true},
		// baseband.auto_record: event-triggered raw-IQ capture.
		{"auto_record disabled ignores fields", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Format: "bogus"}}}, false},
		{"auto_record concurrent ok", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, OnConcurrentCalls: 2}}}, false},
		{"auto_record manual-only ok", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8}}}, false},
		{"auto_record cs16 ok", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 4, Format: "cs16", OnEncrypted: true}}}, false},
		{"auto_record cooldown ok", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 4, Cooldown: "5s", OnEmergency: true}}}, false},
		{"auto_record missing dir", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Seconds: 8, OnNoVoiceDevice: true}}}, true},
		{"auto_record zero seconds", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", OnConcurrentCalls: 2}}}, true},
		{"auto_record bad format", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, Format: "flac", OnEncrypted: true}}}, true},
		{"auto_record tap ddc ok", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, Tap: "ddc", OnConcurrentCalls: 2}}}, false},
		{"auto_record tap wideband ok", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, Tap: "wideband"}}}, false},
		{"auto_record bad tap", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, Tap: "narrowband"}}}, true},
		{"auto_record bad cooldown", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, Cooldown: "soon", OnEncrypted: true}}}, true},
		{"auto_record negative concurrent", Config{Baseband: BasebandConfig{AutoRecord: BasebandAutoRecordConfig{Enabled: true, Dir: "iq", Seconds: 8, OnConcurrentCalls: -1}}}, true},
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

// TestSoapyRemoteDeviceArgs covers the device-args config block (issue #542):
// the "k=v,k=v" Args string merges with the Driver shorthand into make() kwargs.
func TestNormalizeApplyToRouting(t *testing.T) {
	cases := []struct {
		applyTo           string
		enabled           bool
		wantRec, wantDist bool
	}{
		{"", true, true, false},
		{"recording", true, true, false},
		{"distributed", true, false, true},
		{"both", true, true, true},
		{"recording", false, false, false}, // disabled overrides
		{"both", false, false, false},
	}
	for _, tc := range cases {
		n := NormalizeConfig{Enabled: tc.enabled, ApplyTo: tc.applyTo}
		if got := n.AppliesToRecording(); got != tc.wantRec {
			t.Errorf("apply_to=%q enabled=%v: AppliesToRecording=%v want %v", tc.applyTo, tc.enabled, got, tc.wantRec)
		}
		if got := n.AppliesToDistributed(); got != tc.wantDist {
			t.Errorf("apply_to=%q enabled=%v: AppliesToDistributed=%v want %v", tc.applyTo, tc.enabled, got, tc.wantDist)
		}
	}
}

func TestSoapyRemoteDeviceArgs(t *testing.T) {
	cases := []struct {
		name    string
		cfg     SoapyRemoteConfig
		want    map[string]string
		wantErr bool
	}{
		{"empty", SoapyRemoteConfig{}, nil, false},
		{"driver only", SoapyRemoteConfig{Driver: "uhd"}, map[string]string{"driver": "uhd"}, false},
		{"args only", SoapyRemoteConfig{Args: "antenna=RX1"}, map[string]string{"antenna": "RX1"}, false},
		{
			"driver and args merge",
			SoapyRemoteConfig{Driver: "uhd", Args: "rx_subdev_spec=A:0,antenna=RX1"},
			map[string]string{"driver": "uhd", "rx_subdev_spec": "A:0", "antenna": "RX1"},
			false,
		},
		{
			"explicit driver in args wins",
			SoapyRemoteConfig{Driver: "uhd", Args: "driver=lime"},
			map[string]string{"driver": "lime"},
			false,
		},
		{
			"whitespace trimmed and empty segments skipped",
			SoapyRemoteConfig{Args: " antenna = RX1 , , gain = 30 "},
			map[string]string{"antenna": "RX1", "gain": "30"},
			false,
		},
		{"malformed missing equals", SoapyRemoteConfig{Args: "rx_subdev_spec"}, nil, true},
		{"malformed empty key", SoapyRemoteConfig{Args: "=value"}, nil, true},
		{
			"master clock injected as Hz string",
			SoapyRemoteConfig{Driver: "uhd", MasterClockHz: 61_440_000},
			map[string]string{"driver": "uhd", "master_clock_rate": "61440000"},
			false,
		},
		{
			"explicit master_clock_rate in args wins",
			SoapyRemoteConfig{MasterClockHz: 61_440_000, Args: "master_clock_rate=30720000"},
			map[string]string{"master_clock_rate": "30720000"},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.cfg.DeviceArgs()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("DeviceArgs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWebConfigHiddenTabs(t *testing.T) {
	w := WebConfig{Tabs: map[string]bool{
		"pagers":  false,
		"metrics": false,
		"aprs":    true, // explicitly visible — not hidden
	}}
	got := w.HiddenTabs()
	want := []string{"metrics", "pagers"} // sorted, value==false only
	if len(got) != len(want) {
		t.Fatalf("HiddenTabs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("HiddenTabs() = %v, want %v", got, want)
		}
	}
	// Nil/empty map hides nothing.
	if h := (WebConfig{}).HiddenTabs(); len(h) != 0 {
		t.Fatalf("empty WebConfig.HiddenTabs() = %v, want none", h)
	}
}

func TestWebConfigIDBase(t *testing.T) {
	// Default (unset) and any unrecognised value fall back to hex.
	for _, in := range []string{"", "hex", "HEX", "bogus"} {
		if got := (WebConfig{IDBase: in}).IDBaseOrDefault(); got != "hex" {
			t.Errorf("IDBaseOrDefault(%q) = %q, want hex", in, got)
		}
	}
	if got := (WebConfig{IDBase: "dec"}).IDBaseOrDefault(); got != "dec" {
		t.Errorf("IDBaseOrDefault(dec) = %q, want dec", got)
	}

	// validateWeb accepts hex/dec/empty and rejects anything else.
	for _, ok := range []string{"", "hex", "dec"} {
		if errs := (Config{Web: WebConfig{IDBase: ok}}).validateWeb(); len(errs) != 0 {
			t.Errorf("validateWeb id_base=%q errored: %v", ok, errs)
		}
	}
	if errs := (Config{Web: WebConfig{IDBase: "octal"}}).validateWeb(); len(errs) == 0 {
		t.Error("validateWeb id_base=octal: want error, got none")
	}
}

func writeFile(path, data string) error {
	return writeFileImpl(path, []byte(data))
}
