package main

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// noise_squelch reaches the scanner's channel (#1329): on when the key is
// absent or true, off only when the config says false.
func TestConvChannelFromConfigNoiseSquelch(t *testing.T) {
	on, off := true, false
	for _, tc := range []struct {
		name    string
		v       *bool
		wantOff bool
	}{
		{"unset", nil, false},
		{"true", &on, false},
		{"false", &off, true},
	} {
		ch := convChannelFromConfig(config.ConvChannelConfig{
			Label: "PMR446 ch1", FrequencyHz: 446_006_250, Mode: "nfm", SquelchDbFS: -60, NoiseSquelch: tc.v,
		}, 0x80000000)
		if ch.NoiseSquelchOff != tc.wantOff {
			t.Errorf("noise_squelch %s: NoiseSquelchOff = %v, want %v", tc.name, ch.NoiseSquelchOff, tc.wantOff)
		}
	}
}
