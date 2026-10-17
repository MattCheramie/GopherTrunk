package config

import (
	"strings"
	"testing"
)

// sdr.devices[].sample_rate (issue #1260) is accepted only where every
// consumer honours it: a role: voice SDR that a scan-list channel names with
// `device:`, i.e. one the conventional scanner owns outright. Anywhere else a
// mismatched rate would decode nothing, so it must be rejected loudly.
func TestValidateDeviceSampleRate(t *testing.T) {
	base := func() Config {
		c := Config{}
		c.SDR.SampleRate = 2_400_000
		c.SDR.Devices = []DeviceConfig{
			{Serial: "CTRL", Role: "control"},
			{Serial: "RTL", Role: "voice"},
			{Serial: "HF", Role: "voice", SampleRate: 912_000},
		}
		c.Scanner.Conventional = []ConvChannelConfig{
			{Label: "Air", FrequencyHz: 126_400_000, Mode: "am", Device: "HF"},
			{Label: "Fire", FrequencyHz: 154_310_000, Mode: "fm"},
		}
		return c
	}
	check := func(name string, c Config, idx int, want string) {
		t.Helper()
		err := validateDeviceSampleRate(idx, c.SDR.Devices[idx], c)
		switch {
		case want == "" && err != nil:
			t.Errorf("%s: unexpected error %v", name, err)
		case want != "" && (err == nil || !strings.Contains(err.Error(), want)):
			t.Errorf("%s: error = %v, want one containing %q", name, err, want)
		}
	}

	check("dedicated scanner SDR", base(), 2, "")
	check("no override", base(), 1, "")

	c := base()
	c.Scanner.Conventional[0].Device = ""
	check("voice SDR no channel names", c, 2, "needs a scanner.conventional channel")

	c = base()
	c.SDR.Devices[0].SampleRate = 1_200_000
	check("control SDR", c, 0, "only on a role: voice SDR")

	c = base()
	c.SDR.InputSampleRate = 4_800_000
	check("with input_sample_rate", c, 2, "sdr.input_sample_rate")

	c = base()
	c.SDR.Devices[2].SampleRate = 100_000
	check("below the floor", c, 2, "outside 200 kHz..20 MHz")

	// The whole SDR section reports it too, and dc_avoid_offset_hz is
	// bounded by the device's own rate, not the shared one.
	c = base()
	c.SDR.Devices[2].DCAvoidOffsetHz = 500_000 // < 2.4 MS/s / 2, but ≥ 912 kS/s / 2
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "dc_avoid_offset_hz (500000) must be < sample_rate/2 (456000)") {
		t.Errorf("dc_avoid_offset_hz was not bounded by the device's own sample_rate: %v", err)
	}
}
