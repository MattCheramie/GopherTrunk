package widebandt2

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildFixedGainEngine constructs a multi-site wideband engine with the given
// gain string and returns any captured startup WARN about a fixed gain. New()
// emits the WARN at construction time, so the engine is never Run.
func buildFixedGainEngine(t *testing.T, gain string, channels []ChannelConfig) (capturedRecord, bool) {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)

	_, err := New(Options{
		Log:          slog.New(handler),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: 2_400_000,
		CenterFreqHz: 460_000_000,
		Channels:     channels,
		Systems:      []trunking.System{t2System("dmr-simplex")},
		Gain:         gain,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "pinned to a fixed gain") {
			return r, true
		}
	}
	return capturedRecord{}, false
}

var twoSiteChannels = []ChannelConfig{
	{FrequencyHz: 459_900_000, SystemName: "dmr-simplex"},
	{FrequencyHz: 460_100_000, SystemName: "dmr-simplex"},
}

// TestFixedGainMultiChannelWarns: two sites on one wideband dongle pinned to a
// fixed gain draws the startup WARN pointing the operator at gain: auto (issue
// #749). A single fixed gain can only be right for one site.
func TestFixedGainMultiChannelWarns(t *testing.T) {
	r, ok := buildFixedGainEngine(t, "600", twoSiteChannels)
	if !ok {
		t.Fatal("expected a fixed-gain WARN for a multi-channel wideband device, got none")
	}
	if !strings.Contains(r.msg, "gain: auto") {
		t.Errorf("WARN msg = %q, want the gain: auto (AGC) remedy", r.msg)
	}
	// The value is surfaced in dB so the operator can confirm it's the gain
	// they set (600 tenths = 60.0 dB).
	if g, ok := r.attrs["gain_db"].(string); !ok || g != "60.0" {
		t.Errorf("gain_db attr = %v, want %q", r.attrs["gain_db"], "60.0")
	}
	if c, ok := r.attrs["channels"].(int64); !ok || c != 2 {
		t.Errorf("channels attr = %v, want 2", r.attrs["channels"])
	}
}

// TestFixedGainDecimalWarns: a decimal fixed gain ("49.6") is a fixed value too
// and still draws the WARN.
func TestFixedGainDecimalWarns(t *testing.T) {
	r, ok := buildFixedGainEngine(t, "49.6", twoSiteChannels)
	if !ok {
		t.Fatal("expected a fixed-gain WARN for a decimal fixed gain, got none")
	}
	if g, ok := r.attrs["gain_db"].(string); !ok || g != "49.6" {
		t.Errorf("gain_db attr = %v, want %q", r.attrs["gain_db"], "49.6")
	}
}

// TestAutoGainMultiChannelNoWarn: gain: auto (AGC) is the recommended setting,
// so it must NOT draw the WARN. Empty and "auto" (any case) both mean AGC.
func TestAutoGainMultiChannelNoWarn(t *testing.T) {
	for _, gain := range []string{"", "auto", "AUTO", "  ", "notanumber"} {
		if _, ok := buildFixedGainEngine(t, gain, twoSiteChannels); ok {
			t.Errorf("gain %q drew a fixed-gain WARN; want none (AGC / unparseable is not a fixed gain)", gain)
		}
	}
}

// TestFixedGainSingleChannelNoWarn: a single-site wideband device pinned to a
// fixed gain is fine — one gain for one site is exactly right, so no WARN.
func TestFixedGainSingleChannelNoWarn(t *testing.T) {
	one := []ChannelConfig{{FrequencyHz: 460_000_000, SystemName: "dmr-simplex"}}
	if _, ok := buildFixedGainEngine(t, "600", one); ok {
		t.Error("single-channel fixed-gain device drew a fixed-gain WARN; want none")
	}
}
