package config

import "testing"

// scanner.conventional[].mode accepts am (issue #1219); squelch_cn_db must
// not be negative.
func TestValidateConvChannelAM(t *testing.T) {
	ok := ConvChannelConfig{FrequencyHz: 118_700_000, Mode: "am", SquelchCNDb: 12}
	if err := validateConvChannel(0, ok); err != nil {
		t.Errorf("mode am rejected: %v", err)
	}
	for _, bad := range []ConvChannelConfig{
		{FrequencyHz: 118_700_000, Mode: "usb"},
		{FrequencyHz: 118_700_000, Mode: "am", SquelchCNDb: -1},
	} {
		if err := validateConvChannel(0, bad); err == nil {
			t.Errorf("%+v accepted", bad)
		}
	}
}

// scanner.conventional[].mode accepts p25 (issue #1239), but not the
// analog-only tone gate or data decoders on it.
func TestValidateConvChannelP25(t *testing.T) {
	ok := ConvChannelConfig{FrequencyHz: 155_752_500, Mode: "p25", SquelchDbFS: -50}
	if err := validateConvChannel(0, ok); err != nil {
		t.Errorf("mode p25 rejected: %v", err)
	}
	for _, bad := range []ConvChannelConfig{
		{FrequencyHz: 155_752_500, Mode: "p25", Tone: ConvToneConfig{Mode: "ctcss", CTCSSHz: 100}},
		{FrequencyHz: 155_752_500, Mode: "p25", Decoders: []string{"mdc1200"}},
	} {
		if err := validateConvChannel(0, bad); err == nil {
			t.Errorf("%+v accepted", bad)
		}
	}
}
