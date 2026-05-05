package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Log      LogConfig      `yaml:"log"`
	SDR      SDRConfig      `yaml:"sdr"`
	Trunking TrunkingConfig `yaml:"trunking"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type SDRConfig struct {
	SampleRate uint32          `yaml:"sample_rate"`
	Devices    []DeviceConfig  `yaml:"devices"`
}

type DeviceConfig struct {
	Serial string `yaml:"serial"`
	Role   string `yaml:"role"`
	PPM    int    `yaml:"ppm"`
	Gain   string `yaml:"gain"`
}

type TrunkingConfig struct {
	Systems []SystemConfig `yaml:"systems"`
}

type SystemConfig struct {
	Name             string   `yaml:"name"`
	Protocol         string   `yaml:"protocol"`
	ControlChannels  []uint32 `yaml:"control_channels"`
	TalkgroupFile    string   `yaml:"talkgroup_file"`
}

func Default() Config {
	return Config{
		Log: LogConfig{Level: "info", Format: "text"},
		SDR: SDRConfig{SampleRate: 2_400_000},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.SDR.SampleRate != 0 && (c.SDR.SampleRate < 225_000 || c.SDR.SampleRate > 3_200_000) {
		return errors.New("sdr.sample_rate must be between 225 kHz and 3.2 MHz")
	}
	for i, d := range c.SDR.Devices {
		switch d.Role {
		case "", "control", "voice", "auto":
		default:
			return fmt.Errorf("sdr.devices[%d]: role must be control|voice|auto", i)
		}
	}
	for i, s := range c.Trunking.Systems {
		if s.Name == "" {
			return fmt.Errorf("trunking.systems[%d]: name required", i)
		}
		switch s.Protocol {
		case "p25", "dmr", "nxdn":
		default:
			return fmt.Errorf("trunking.systems[%d]: protocol must be p25|dmr|nxdn", i)
		}
	}
	return nil
}
