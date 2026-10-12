package config

import "testing"

// scanner.conventional[].gain takes the sdr.devices gain forms (#1239).
func TestValidateConvChannelGain(t *testing.T) {
	for _, g := range []string{"", "auto", "AUTO", "280", "28.0", "28,0"} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 155_000_000, Gain: g}); err != nil {
			t.Errorf("gain %q rejected: %v", g, err)
		}
	}
	for _, g := range []string{"loud", "28dB"} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 155_000_000, Gain: g}); err == nil {
			t.Errorf("gain %q accepted", g)
		}
	}
}

func TestValidateConvChannelAudioGain(t *testing.T) {
	for _, g := range []float64{0, 6, -12.5, 30, -30} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 155_752_500, Mode: "p25", AudioGainDB: g}); err != nil {
			t.Errorf("audio_gain_db %v rejected: %v", g, err)
		}
	}
	for _, g := range []float64{30.5, -31, 60} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 155_752_500, Mode: "p25", AudioGainDB: g}); err == nil {
			t.Errorf("audio_gain_db %v accepted, want an error (±30 dB)", g)
		}
	}
}
