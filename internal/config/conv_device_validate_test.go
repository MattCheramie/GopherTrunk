package config

import (
	"strings"
	"testing"
)

// TestValidateConvChannelDevice pins the scanner.conventional[].device rule
// (issue #1260): a channel may name a voice SDR (or one validation cannot see,
// such as an unlisted or network source — the daemon checks those against the
// live pool), but never a control or wideband SDR, which the scanner would
// retune out from under the CC decoder or the channelizer.
func TestValidateConvChannelDevice(t *testing.T) {
	errText := func(c Config) string {
		var b strings.Builder
		for _, err := range c.validateScanner() {
			b.WriteString(err.Error())
			b.WriteString("\n")
		}
		return b.String()
	}
	base := func(dev string) Config {
		c := Config{}
		c.SDR.Devices = []DeviceConfig{
			{Serial: "CTRL", Role: "control"},
			{Serial: "WB", Role: "wideband"},
			{Serial: "AIR", Role: "voice"},
		}
		c.Scanner.Conventional = []ConvChannelConfig{
			{Label: "A", FrequencyHz: 126_400_000, Mode: "am", Device: dev},
		}
		return c
	}
	for _, dev := range []string{"", "AIR", "not-listed"} {
		if got := errText(base(dev)); got != "" {
			t.Errorf("device %q: errors = %q, want none", dev, got)
		}
	}
	for _, dev := range []string{"CTRL", "WB", " CTRL "} {
		if got := errText(base(dev)); !strings.Contains(got, "needs a role: voice SDR") {
			t.Errorf("device %q: errors = %q, want a role error", dev, got)
		}
	}
}
