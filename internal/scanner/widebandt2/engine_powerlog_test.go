package widebandt2

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// runPowerDiag builds a one-channel Engine wired to a bus, loads the
// channel power accumulator and a stub decoded-frame counter, flushes one
// diagnostics window, and returns the published ChannelPower event (if
// any).
func runPowerDiag(t *testing.T, chanIQ []complex64, decoded uint64) (events.ChannelPower, bool) {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	sub := bus.Subscribe()
	t.Cleanup(sub.Close)

	ec := &engineChannel{
		freqHz: 450_125_000, sysName: "Main_Site_1", protoTag: "p25-phase2",
		decoded: func() uint64 { return decoded },
	}
	ec.pwr.Add(chanIQ)
	e := &Engine{
		log:      slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})),
		bus:      bus,
		channels: []*engineChannel{ec},
		now:      time.Now,
	}

	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base)                                   // seeds lastDiagAt
	e.maybeLogDiagnostics(base.Add(iqpower.Window + time.Second)) // flushes

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindChannelPower {
				return ev.Payload.(events.ChannelPower), true
			}
		default:
			return events.ChannelPower{}, false
		}
	}
}

// TestPowerEventGatedOnDecodeActivity: a channel with no decode activity
// this window must not publish a ChannelPower event, even when its IQ
// power is low.
func TestPowerEventGatedOnDecodeActivity(t *testing.T) {
	if _, ok := runPowerDiag(t, quietIQ(4096), 0); ok {
		t.Fatal("published a ChannelPower event with zero decode delta; want none")
	}
}

// TestPowerEventLowPowerFlag: a decode-active channel publishes a
// ChannelPower event, and LowPower tracks whether the window's IQ power is
// below the threshold.
func TestPowerEventLowPowerFlag(t *testing.T) {
	cp, ok := runPowerDiag(t, quietIQ(4096), 3)
	if !ok {
		t.Fatal("expected a ChannelPower event for a decode-active channel, got none")
	}
	if !cp.LowPower {
		t.Errorf("quiet channel LowPower = false, want true (dbfs=%.1f)", cp.PowerDbFS)
	}
	if cp.Decoded != 3 {
		t.Errorf("Decoded = %d, want 3 (per-window delta)", cp.Decoded)
	}
	if cp.System != "Main_Site_1" || cp.Protocol != "p25-phase2" || cp.FreqHz != 450_125_000 {
		t.Errorf("identity fields = %+v", cp)
	}

	cp, ok = runPowerDiag(t, loudIQ(4096), 3)
	if !ok {
		t.Fatal("expected a ChannelPower event for a loud decode-active channel, got none")
	}
	if cp.LowPower {
		t.Errorf("loud channel LowPower = true, want false (dbfs=%.1f)", cp.PowerDbFS)
	}
}
