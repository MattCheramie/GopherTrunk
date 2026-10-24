package config

import (
	"strings"
	"testing"
)

// TestValidateWidebandConventional pins the sdr.devices[].conventional rules
// (issue #1288): analog FM and AM channels in the dongle's band, with no system,
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
		{Label: "Air", FrequencyHz: 460_700_000, Mode: "am", SquelchCNDb: 15, Decoders: []string{"acars"}},
	}
	if err := validateWidebandDevice(0, dev(ok...), 2_400_000, nil); err != nil {
		t.Fatalf("valid conventional-only wideband device rejected: %v", err)
	}
	for name, tc := range map[string]struct {
		d    DeviceConfig
		want string
	}{
		"no channels at all":  {DeviceConfig{Serial: "WB", Role: "wideband", CenterFreqHz: 460_500_000}, "at least one channel"},
		"digital not yet":     {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Mode: "dmr"}), "conventional[0]: mode \"dmr\" is not supported on a wideband dongle"},
		"per-channel gain":    {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Gain: "200"}), "gain does not apply"},
		"device":              {dev(ConvChannelConfig{FrequencyHz: 460_100_000, Device: "X"}), "device does not apply"},
		"out of band":         {dev(ConvChannelConfig{FrequencyHz: 462_000_000}), "usable band"},
		"duplicate untoned":   {dev(ConvChannelConfig{FrequencyHz: 460_100_000}, ConvChannelConfig{FrequencyHz: 460_100_000}), "each need a tone"},
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

// TestValidateWidebandConventionalSharedFrequency pins issue #1288 §5:
// several logical channels may share one frequency when their tone gates
// tell every transmission apart, and anything that would let one
// transmission open two of them (or decode its data twice) is refused.
func TestValidateWidebandConventionalSharedFrequency(t *testing.T) {
	const hz = 460_100_000
	ctcss := func(label string, tone float64) ConvChannelConfig {
		return ConvChannelConfig{Label: label, FrequencyHz: hz, Mode: "nfm", Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: tone}}
	}
	dcs := func(label, code, pol string) ConvChannelConfig {
		return ConvChannelConfig{Label: label, FrequencyHz: hz, Mode: "nfm",
			Tone: ConvToneConfig{Mode: "dcs", DCSCode: code, DCSPolarity: pol}}
	}
	dev := func(chs ...ConvChannelConfig) DeviceConfig {
		return DeviceConfig{Serial: "WB", Role: "wideband", CenterFreqHz: 460_500_000, Conventional: chs}
	}
	withDecoders := func(c ConvChannelConfig, d ...string) ConvChannelConfig { c.Decoders = d; return c }

	for name, d := range map[string]DeviceConfig{
		"two CTCSS":               dev(ctcss("A", 100), ctcss("B", 141.3)),
		"adjacent EIA tones":      dev(ctcss("A", 159.8), ctcss("B", 162.2)),
		"CTCSS and DCS":           dev(ctcss("A", 100), dcs("B", "023", "")),
		"023N and 023I":           dev(dcs("A", "023", "normal"), dcs("B", "023", "inverted")),
		"three channels":          dev(ctcss("A", 100), ctcss("B", 141.3), dcs("C", "754", "")),
		"decoders on one":         dev(withDecoders(ctcss("A", 100), "mdc1200"), ctcss("B", 141.3)),
		"default mode matches fm": dev(ConvChannelConfig{FrequencyHz: hz, Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: 100}}, ConvChannelConfig{FrequencyHz: hz, Mode: "fm", Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: 141.3}}),
	} {
		if err := validateWidebandDevice(0, d, 2_400_000, nil); err != nil {
			t.Errorf("%s: rejected: %v", name, err)
		}
	}

	for name, tc := range map[string]struct {
		d    DeviceConfig
		want string
	}{
		"one untoned":  {dev(ctcss("A", 100), ConvChannelConfig{FrequencyHz: hz, Mode: "nfm"}), "conventional[1]: frequency_hz 460100000 is also conventional[0]; channels sharing a frequency each need a tone"},
		"same CTCSS":   {dev(ctcss("A", 123), ctcss("B", 123)), "conventional[1]: tone CTCSS 123.0 Hz cannot be told apart from conventional[0]'s CTCSS 123.0 Hz"},
		"150.0/151.4":  {dev(ctcss("A", 150.0), ctcss("B", 151.4)), "cannot be told apart"},
		"DCS alias":    {dev(dcs("A", "023", ""), dcs("B", "047", "inverted")), "tone DCS 047I cannot be told apart from conventional[0]'s DCS 023N"},
		"DCS both":     {dev(dcs("A", "023", "both"), dcs("B", "023", "inverted")), "cannot be told apart"},
		"mode differs": {dev(ctcss("A", 100), ConvChannelConfig{FrequencyHz: hz, Mode: "am", Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: 141.3}}), "mode \"am\" differs from conventional[0]"},
		"decoders twice": {dev(withDecoders(ctcss("A", 100), "mdc1200"), withDecoders(ctcss("B", 141.3), "fleetsync")),
			"conventional[1]: decoders are also set on conventional[0]"},
		"third overlaps first": {dev(ctcss("A", 100), ctcss("B", 141.3), ctcss("C", 100)), "conventional[2]: tone CTCSS 100.0 Hz cannot be told apart from conventional[0]"},
	} {
		err := validateWidebandDevice(0, tc.d, 2_400_000, nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, tc.want)
		}
	}
}
