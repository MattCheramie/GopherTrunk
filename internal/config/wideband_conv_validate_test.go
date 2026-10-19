package config

import (
	"strings"
	"testing"
)

// TestValidateWidebandConventional pins the sdr.devices[].conventional rules
// (issue #1288): analog FM channels in the dongle's band, with no system,
// accepted on their own; anything the never-retuned shared front end cannot
// honour rejected with the channel's own path in the error.
func TestValidateWidebandConventional(t *testing.T) {
	dev := func(chs ...ConvChannelConfig) DeviceConfig {
		return DeviceConfig{Serial: "WB", Role: "wideband", CenterFreqHz: 460_500_000, Conventional: chs}
	}
	ok := []ConvChannelConfig{
		{Label: "Fire", FrequencyHz: 460_100_000, Mode: "nfm"},
		{Label: "EMS", FrequencyHz: 460_900_000, Mode: "fm", Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: 123}},
		{Label: "Ops", FrequencyHz: 460_300_000, Decoders: []string{"mdc1200"}},
	}
	if err := validateWidebandDevice(0, dev(ok...), 2_400_000, nil); err != nil {
		t.Fatalf("valid conventional-only wideband device rejected: %v", err)
	}
	for name, tc := range map[string]struct {
		d    DeviceConfig
		want string
	}{
		"no channels at all":  {DeviceConfig{Serial: "WB", Role: "wideband", CenterFreqHz: 460_500_000}, "at least one channel"},
		"am not yet":          {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Mode: "am"}), "conventional[0]: mode \"am\" is not supported"},
		"digital not yet":     {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Mode: "dmr"}), "not supported on a wideband dongle"},
		"per-channel gain":    {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Gain: "200"}), "gain does not apply"},
		"device":              {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Device: "X"}), "device does not apply"},
		"out of band":         {dev(ConvChannelConfig{FrequencyHz: 462_000_000}), "usable band"},
		"duplicate":           {dev(ConvChannelConfig{FrequencyHz: 460_100_000}, ConvChannelConfig{FrequencyHz: 460_100_000}), "duplicate frequency_hz"},
		"shared field checks": {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: 20}}), "sdr.devices[0].conventional[0].tone.ctcss_hz"},
	} {
		err := validateWidebandDevice(0, tc.d, 2_400_000, nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, tc.want)
		}
	}
	// A conventional channel on a trunked channel's frequency would put two
	// taps on one carrier.
	d := dev(ConvChannelConfig{FrequencyHz: 460_100_000})
	d.Channels = []DeviceChannelConfig{{FrequencyHz: 460_100_000, System: "T2"}}
	err := validateWidebandDevice(0, d, 2_400_000, []SystemConfig{{Name: "T2", Protocol: "dmr-tier2"}})
	if err == nil || !strings.Contains(err.Error(), "also sdr.devices[0].channels[0]") {
		t.Errorf("conventional channel on a channels: frequency: err = %v", err)
	}
}

// TestValidateWidebandConventionalTalkgroupIDs: wideband conventional
// channels key synthetic calls by talkgroup ID alongside the scan list, so
// a collision either way is rejected, and their positional IDs never meet
// the scan list's (manual-tune channels extend 0x80000000|i past the list).
func TestValidateWidebandConventionalTalkgroupIDs(t *testing.T) {
	base := func() Config {
		c := Config{}
		c.Scanner.Conventional = []ConvChannelConfig{{FrequencyHz: 155_000_000}, {FrequencyHz: 155_100_000, TalkgroupID: 77}}
		c.SDR.Devices = []DeviceConfig{
			{Serial: "WB1", Role: "wideband", Conventional: []ConvChannelConfig{{FrequencyHz: 460_100_000}}},
			{Serial: "WB2", Role: "wideband", Conventional: []ConvChannelConfig{{FrequencyHz: 461_100_000}}},
		}
		return c
	}
	if errs := base().validateScanner(); len(errs) != 0 {
		t.Fatalf("positional IDs collide: %v", errs)
	}
	c := base()
	c.SDR.Devices[1].Conventional[0].TalkgroupID = 77
	if errs := c.validateScanner(); len(errs) != 1 || !strings.Contains(errs[0].Error(), "collides with scanner.conventional[1]") {
		t.Errorf("collision with the scan list: %v", errs)
	}
	c = base()
	c.SDR.Devices[1].Conventional[0].TalkgroupID = WidebandConvPositionalBase // WB1's positional ID
	if errs := c.validateScanner(); len(errs) != 1 || !strings.Contains(errs[0].Error(), "collides with sdr.devices[0].conventional[0]") {
		t.Errorf("collision between wideband channels: %v", errs)
	}
}
