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

// scanner.conventional[].nac / p25_phase1_demod_mode (#1262): hex NAC and a
// known demod on mode p25 only.
func TestValidateConvChannelNACAndDemod(t *testing.T) {
	p25 := func(nac, demod string) ConvChannelConfig {
		return ConvChannelConfig{FrequencyHz: 155_752_500, Mode: "p25", NAC: nac, P25Phase1DemodMode: demod}
	}
	for _, ch := range []ConvChannelConfig{p25("293", ""), p25("0x293", "cqpsk"), p25("F7E", "c4fm"), p25("", "LSM")} {
		if err := validateConvChannel(0, ch); err != nil {
			t.Errorf("nac %q demod %q rejected: %v", ch.NAC, ch.P25Phase1DemodMode, err)
		}
	}
	bad := []ConvChannelConfig{
		p25("1000", ""), p25("29G", ""), p25("0x", ""), p25("", "gmsk"),
		{FrequencyHz: 155_000_000, Mode: "fm", NAC: "293"},
		{FrequencyHz: 155_000_000, Mode: "nfm", P25Phase1DemodMode: "cqpsk"},
	}
	for _, ch := range bad {
		if err := validateConvChannel(0, ch); err == nil {
			t.Errorf("mode %q nac %q demod %q accepted", ch.Mode, ch.NAC, ch.P25Phase1DemodMode)
		}
	}
}
