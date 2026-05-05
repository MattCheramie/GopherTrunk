package config

import (
	"path/filepath"
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
		{"missing name", Config{Trunking: TrunkingConfig{Systems: []SystemConfig{{Protocol: "p25"}}}}, true},
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
