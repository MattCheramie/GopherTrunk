package widebandt2

import (
	"log/slog"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestEngineConventionalChannelsNeedNoSystem: an engine whose channels are all
// tap-only conventional FM (issue #1288) builds without a trunking-system
// table, gives each channel a 48 kHz fan-out, and still refuses a trunked
// channel whose system is missing.
func TestEngineConventionalChannelsNeedNoSystem(t *testing.T) {
	bus := events.NewBus(1)
	defer bus.Close()
	opts := Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Bus:          bus,
		Device:       newMockDevice(nil),
		SampleRateHz: 2_400_000,
		CenterFreqHz: 460_500_000,
		Channels: []ChannelConfig{
			{FrequencyHz: 460_100_000, Conventional: true},
			{FrequencyHz: 460_900_000, Conventional: true},
		},
	}
	e, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, hz := range []uint32{460_100_000, 460_900_000} {
		if tag := e.ChannelProtocolTags()[hz]; tag != conventionalProtoTag {
			t.Errorf("%d Hz tagged %q, want %q", hz, tag, conventionalProtoTag)
		}
		if r := e.ChannelRateHz(hz); math.Abs(r-narrowbandRateHz) > 1 {
			t.Errorf("%d Hz tap rate %.1f, want ~%.0f", hz, r, narrowbandRateHz)
		}
		if _, unsub, ok := e.SubscribeChannelIQ(hz); !ok {
			t.Errorf("%d Hz: no IQ fan-out", hz)
		} else {
			unsub()
		}
	}
	opts.Channels = append(opts.Channels, ChannelConfig{FrequencyHz: 460_300_000, SystemName: "missing"})
	if _, err := New(opts); err == nil {
		t.Error("a trunked channel without a systems table was accepted")
	}
}

// TestLowPowerConventionalFMNeverWarns: an analog conventional channel is
// silent between transmissions and decodes nothing, so its idle windows must
// not raise the "iq power very low" WARN at all — unlike a never-decoded DMR
// carrier, which is warned about once.
func TestLowPowerConventionalFMNeverWarns(t *testing.T) {
	ec := &engineChannel{freqHz: 460_100_000, sysName: "conventional", protoTag: conventionalProtoTag, conventional: true}
	quiet := make([]complex64, 4096)
	for i := range quiet {
		quiet[i] = complex(1e-5, 0)
	}
	loud := make([]complex64, 4096)
	for i := range loud {
		loud[i] = complex(0.3, 0)
	}
	if n := countLowPowerWarns(t, ec, 5, quiet, loud); n != 0 {
		t.Errorf("idle conventional FM channel raised %d low-power WARNs, want 0", n)
	}
}
