package config

import "testing"

// #1184: tone.dcs_polarity selects the DCS sense that opens the gate.
func TestValidateConvChannelDCSPolarity(t *testing.T) {
	base := ConvChannelConfig{FrequencyHz: 447_100_000, Tone: ConvToneConfig{Mode: "dcs", DCSCode: "025"}}
	for _, p := range []string{"", "normal", "N", "inverted", "i", "both"} {
		ch := base
		ch.Tone.DCSPolarity = p
		if err := validateConvChannel(0, ch); err != nil {
			t.Errorf("dcs_polarity %q rejected: %v", p, err)
		}
	}
	ch := base
	ch.Tone.DCSPolarity = "reverse"
	if err := validateConvChannel(0, ch); err == nil {
		t.Error(`dcs_polarity "reverse" accepted`)
	}
}
