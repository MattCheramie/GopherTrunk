package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// scanner.conventional[].noise_squelch is an fm / nfm setting (#1329): any
// value is fine there, and naming it on another mode is a mistake worth
// stopping at load, as nac is on a non-p25 channel.
func TestValidateConvChannelNoiseSquelch(t *testing.T) {
	on, off := true, false
	for _, mode := range []string{"", "fm", "nfm"} {
		for _, v := range []*bool{nil, &on, &off} {
			if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 446_006_250, Mode: mode, NoiseSquelch: v}); err != nil {
				t.Errorf("mode %q noise_squelch %v rejected: %v", mode, v, err)
			}
		}
	}
	for _, mode := range []string{"am", "p25", "dmr", "nxdn", "nxdn48", "dstar", "ysf"} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 446_006_250, Mode: mode}); err != nil {
			t.Errorf("mode %q without noise_squelch rejected: %v", mode, err)
		}
		for _, v := range []*bool{&on, &off} {
			err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 446_006_250, Mode: mode, NoiseSquelch: v})
			if err == nil || !strings.Contains(err.Error(), "noise_squelch") {
				t.Errorf("mode %q noise_squelch %v: err = %v, want a noise_squelch error", mode, *v, err)
			}
		}
	}
}

// The key reads as a tri-state: absent stays unset (on), and both values
// survive a round trip.
func TestConvChannelNoiseSquelchYAML(t *testing.T) {
	for _, tc := range []struct {
		yaml string
		want *bool
	}{
		{"frequency_hz: 446006250\nmode: nfm\n", nil},
		{"frequency_hz: 446006250\nmode: nfm\nnoise_squelch: true\n", ptrBool(true)},
		{"frequency_hz: 446006250\nmode: nfm\nnoise_squelch: false\n", ptrBool(false)},
	} {
		var ch ConvChannelConfig
		if err := yaml.Unmarshal([]byte(tc.yaml), &ch); err != nil {
			t.Fatalf("%q: %v", tc.yaml, err)
		}
		switch {
		case tc.want == nil && ch.NoiseSquelch != nil:
			t.Errorf("%q: noise_squelch = %v, want unset", tc.yaml, *ch.NoiseSquelch)
		case tc.want != nil && (ch.NoiseSquelch == nil || *ch.NoiseSquelch != *tc.want):
			t.Errorf("%q: noise_squelch = %v, want %v", tc.yaml, ch.NoiseSquelch, *tc.want)
		}
		out, err := yaml.Marshal(ch)
		if err != nil {
			t.Fatal(err)
		}
		if has := strings.Contains(string(out), "noise_squelch"); has != (tc.want != nil) {
			t.Errorf("%q: marshalled config mentions noise_squelch = %v, want %v:\n%s", tc.yaml, has, tc.want != nil, out)
		}
	}
}

func ptrBool(v bool) *bool { return &v }
