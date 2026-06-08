package main

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// newPagingTestDaemon builds the minimal Daemon buildWidebandPagingGroup
// needs: a live event bus (handed to each receiver) and a warnings slice
// (addWarning just appends). No SDR pool / brokers are required at the
// construction stage under test.
func newPagingTestDaemon() *Daemon {
	return &Daemon{bus: events.NewBus(64)}
}

// TestBuildWidebandPagingGroupAutoCenter covers the headline use case
// from the feature request: FLEX on 153.0250 MHz + POCSAG on 153.3500 MHz
// grouped onto one dongle. With no explicit center the daemon should pick
// the midpoint and place each tap at its signed offset from it.
func TestBuildWidebandPagingGroupAutoCenter(t *testing.T) {
	d := newPagingTestDaemon()
	wg := config.PagingWidebandConfig{
		Serial: "pager-sdr",
		Channels: []config.PagingWidebandChannel{
			{Protocol: "flex", FrequencyHz: 153_025_000},
			{Protocol: "pocsag", FrequencyHz: 153_350_000, BaudHz: 1200},
		},
	}

	group := d.buildWidebandPagingGroup(wg, 2_048_000, slog.Default())
	if group == nil {
		t.Fatalf("buildWidebandPagingGroup returned nil; warnings=%v", d.StartupWarnings())
	}
	if len(d.StartupWarnings()) != 0 {
		t.Fatalf("unexpected warnings: %v", d.StartupWarnings())
	}

	// Midpoint of 153.025 and 153.350 MHz.
	const wantCenter = 153_187_500
	if group.centerFreq != wantCenter {
		t.Fatalf("center = %d Hz, want %d Hz", group.centerFreq, wantCenter)
	}
	if len(group.channels) != 2 {
		t.Fatalf("channels = %d, want 2", len(group.channels))
	}

	flex := group.channels[0]
	if flex.flex == nil || flex.pocsag != nil {
		t.Fatalf("channel 0 should be FLEX, got pocsag=%v flex=%v", flex.pocsag, flex.flex)
	}
	if flex.offsetHz != float64(153_025_000-wantCenter) {
		t.Fatalf("flex offset = %.0f Hz, want %d Hz", flex.offsetHz, 153_025_000-wantCenter)
	}

	poc := group.channels[1]
	if poc.pocsag == nil || poc.flex != nil {
		t.Fatalf("channel 1 should be POCSAG, got pocsag=%v flex=%v", poc.pocsag, poc.flex)
	}
	if poc.offsetHz != float64(153_350_000-wantCenter) {
		t.Fatalf("pocsag offset = %.0f Hz, want %d Hz", poc.offsetHz, 153_350_000-wantCenter)
	}
}

// TestBuildWidebandPagingGroupExplicitCenter verifies an operator-supplied
// center is honoured verbatim and offsets are taken relative to it.
func TestBuildWidebandPagingGroupExplicitCenter(t *testing.T) {
	d := newPagingTestDaemon()
	wg := config.PagingWidebandConfig{
		Serial:       "pager-sdr",
		CenterFreqHz: 153_200_000,
		Channels: []config.PagingWidebandChannel{
			{Protocol: "flex", FrequencyHz: 153_025_000},
		},
	}

	group := d.buildWidebandPagingGroup(wg, 2_048_000, slog.Default())
	if group == nil {
		t.Fatalf("buildWidebandPagingGroup returned nil; warnings=%v", d.StartupWarnings())
	}
	if group.centerFreq != 153_200_000 {
		t.Fatalf("center = %d Hz, want 153200000 Hz", group.centerFreq)
	}
	if group.channels[0].offsetHz != float64(153_025_000-153_200_000) {
		t.Fatalf("offset = %.0f Hz, want %d Hz", group.channels[0].offsetHz, 153_025_000-153_200_000)
	}
}

// TestBuildWidebandPagingGroupUnknownProtocol drops the bad channel with a
// warning but keeps the group alive on the remaining valid channel.
func TestBuildWidebandPagingGroupUnknownProtocol(t *testing.T) {
	d := newPagingTestDaemon()
	wg := config.PagingWidebandConfig{
		Serial: "pager-sdr",
		Channels: []config.PagingWidebandChannel{
			{Protocol: "morse", FrequencyHz: 153_025_000},
			{Protocol: "pocsag", FrequencyHz: 153_350_000},
		},
	}

	group := d.buildWidebandPagingGroup(wg, 2_048_000, slog.Default())
	if group == nil {
		t.Fatalf("group should survive one bad channel; warnings=%v", d.StartupWarnings())
	}
	if len(group.channels) != 1 || group.channels[0].pocsag == nil {
		t.Fatalf("expected one surviving POCSAG channel, got %+v", group.channels)
	}
	if len(d.StartupWarnings()) != 1 {
		t.Fatalf("want 1 warning for the bad protocol, got %v", d.StartupWarnings())
	}
}

// TestBuildWidebandPagingGroupRejectsEmpty drops groups with no serial or
// no channels entirely (nil return, warning recorded).
func TestBuildWidebandPagingGroupRejectsEmpty(t *testing.T) {
	d := newPagingTestDaemon()
	if g := d.buildWidebandPagingGroup(config.PagingWidebandConfig{}, 2_048_000, slog.Default()); g != nil {
		t.Fatalf("empty group should return nil, got %+v", g)
	}
	if len(d.StartupWarnings()) != 1 {
		t.Fatalf("want 1 warning for the empty group, got %v", d.StartupWarnings())
	}
}
