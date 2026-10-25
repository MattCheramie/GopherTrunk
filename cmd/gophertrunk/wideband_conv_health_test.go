package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
)

// The health classification never reads RF activity (issue #1288 §12): only
// a stopped stream is unhealthy, a parked pump is "suspended", and a tap that
// has not delivered yet is "no_iq".
func TestWbConvHealthClassification(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		h    widebandt2.ChannelIQHealth
		ok   bool
		want string
	}{
		{"no engine", widebandt2.ChannelIQHealth{}, false, api.ConvHealthNoIQ},
		{"never delivered", widebandt2.ChannelIQHealth{}, true, api.ConvHealthNoIQ},
		{"fresh", widebandt2.ChannelIQHealth{LastIQ: now.Add(-20 * time.Millisecond)}, true, api.ConvHealthOK},
		{"just under bound", widebandt2.ChannelIQHealth{LastIQ: now.Add(-api.WidebandConvStallAfter + time.Millisecond)}, true, api.ConvHealthOK},
		{"stalled", widebandt2.ChannelIQHealth{LastIQ: now.Add(-api.WidebandConvStallAfter)}, true, api.ConvHealthStalled},
		{"suspended beats stale", widebandt2.ChannelIQHealth{LastIQ: now.Add(-time.Minute), Suspended: true}, true, api.ConvHealthSuspended},
	}
	for _, c := range cases {
		if got := wbConvHealth(c.h, c.ok, now); got != c.want {
			t.Errorf("%s: health = %q, want %q", c.name, got, c.want)
		}
	}
}

// The engine stamps a wideband conventional tap's IQ flow, and the cockpit
// reports it: no_iq before the pump runs, ok while chunks arrive, stalled
// once the stream has been silent past the bound — on a channel nobody
// transmitted on, so idle RF never reads as a fault.
func TestWidebandConvTapReportsIQHealth(t *testing.T) {
	const (
		rate     = 960_000.0
		centerHz = 460_500_000
		chunk    = 9_600
	)
	chHz := uint32(centerHz - 200_000)
	iq := make([]complex64, int(rate*0.3)) // 0.3 s of digital silence: no RF activity
	var chunks [][]complex64
	for i := 0; i+chunk <= len(iq); i += chunk {
		chunks = append(chunks, iq[i:i+chunk])
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()
	eng, err := widebandt2.New(widebandt2.Options{
		Log: log, Bus: bus,
		Device:       &pacedWidebandDevice{chunks: chunks, period: 5 * time.Millisecond},
		SampleRateHz: uint32(rate), CenterFreqHz: centerHz,
		Channels: widebandConvTapChannels([]config.ConvChannelConfig{{FrequencyHz: chHz, Mode: "nfm"}}),
		Serial:   "WB1",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := widebandt2.NewChannelVoiceSource(func() *widebandt2.Engine { return eng }, chHz, wbConvSerial("WB1", chHz))
	h, ok := src.IQHealth()
	if got := wbConvHealth(h, ok, time.Now()); got != api.ConvHealthNoIQ {
		t.Fatalf("before the pump runs: health = %q, want no_iq", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = eng.Run(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if h, ok = src.IQHealth(); ok && !h.LastIQ.IsZero() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if h.LastIQ.IsZero() {
		t.Fatal("tap never stamped its IQ flow")
	}
	if got := wbConvHealth(h, ok, time.Now()); got != api.ConvHealthOK {
		t.Fatalf("while chunks arrive: health = %q, want ok", got)
	}
	// The paced device stops after 0.3 s; past the bound the same snapshot
	// must read stalled.
	if got := wbConvHealth(h, ok, h.LastIQ.Add(api.WidebandConvStallAfter)); got != api.ConvHealthStalled {
		t.Fatalf("after the stream stops: health = %q, want stalled", got)
	}
}

func TestConventionalChannelCountsStalled(t *testing.T) {
	var st api.ScannerStatus
	st.Conventional.WidebandChannels = []api.WidebandConvChannelDTO{
		{Health: api.ConvHealthOK, Active: true},
		{Health: api.ConvHealthStalled},
		{Health: api.ConvHealthSuspended},
	}
	counts := conventionalChannelCounts(st)
	wb := counts[1]
	if wb.Kind != "wideband" || wb.Configured != 3 || wb.Active != 1 || wb.Stalled != 1 {
		t.Fatalf("wideband counts = %+v, want configured 3 active 1 stalled 1", wb)
	}
}
