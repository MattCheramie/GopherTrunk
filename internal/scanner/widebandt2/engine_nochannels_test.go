package widebandt2

import (
	"context"
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestEngineWithNoChannelsStreamsForSubscribers pins the sdr.groups member
// that carries no channel: with AllowNoChannels it tunes the dongle and
// drains its stream (so a broker's subscribers get IQ) without per-channel
// work, and still reports a dead stream for the daemon's reacquire loop.
func TestEngineWithNoChannelsStreamsForSubscribers(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	chunks := make([][]complex64, 8)
	for i := range chunks {
		chunks[i] = make([]complex64, 4096)
	}
	dev := newMockDevice(chunks)
	eng, err := New(Options{Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 853_000_000,
		AllowNoChannels: true, Group: "south", Serial: "M2"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if eng.Group() != "south" || eng.CenterFreqHz() != 853_000_000 {
		t.Errorf("Group/CenterFreqHz = %q/%d", eng.Group(), eng.CenterFreqHz())
	}
	err = eng.Run(context.Background())
	if !errors.Is(err, ErrIQStreamClosed) {
		t.Fatalf("Run = %v, want ErrIQStreamClosed once the stream ends", err)
	}
	if got := dev.centerFreqHz.Load(); got != 853_000_000 {
		t.Errorf("dongle tuned to %d", got)
	}
	if len(dev.chunkCh) != 0 {
		t.Errorf("%d chunks left undrained", len(dev.chunkCh))
	}
}
