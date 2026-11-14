package config

import "testing"

// scanner.conventional[].squelch_open_ms is 0..1000 ms (#1321 follow-up).
func TestValidateConvChannelSquelchOpenMs(t *testing.T) {
	for _, ms := range []int{0, 1, 60, 1000} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 155_000_000, SquelchOpenMs: ms}); err != nil {
			t.Errorf("squelch_open_ms %d rejected: %v", ms, err)
		}
	}
	for _, ms := range []int{-1, 1001, 5000} {
		if err := validateConvChannel(0, ConvChannelConfig{FrequencyHz: 155_000_000, SquelchOpenMs: ms}); err == nil {
			t.Errorf("squelch_open_ms %d accepted, want 0..1000", ms)
		}
	}
}
