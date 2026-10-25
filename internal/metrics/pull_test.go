package metrics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRegisterConventionalChannels(t *testing.T) {
	m, err := New(nil, nil, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	active := 1
	if err := m.RegisterConventionalChannels(func() []ConventionalChannelCounts {
		return []ConventionalChannelCounts{{Kind: "scan", Configured: 4, Active: 0}, {Kind: "wideband", Configured: 8, Active: active}}
	}); err != nil {
		t.Fatal(err)
	}
	want := `
# HELP gophertrunk_conventional_channels_active Conventional channels whose squelch (and tone / NAC / call gate) is open now, by kind.
# TYPE gophertrunk_conventional_channels_active gauge
gophertrunk_conventional_channels_active{kind="scan"} 0
gophertrunk_conventional_channels_active{kind="wideband"} 1
# HELP gophertrunk_conventional_channels_configured Conventional channels configured, by kind (scan = the sequential scan list, wideband = continuous channels on a wideband dongle).
# TYPE gophertrunk_conventional_channels_configured gauge
gophertrunk_conventional_channels_configured{kind="scan"} 4
gophertrunk_conventional_channels_configured{kind="wideband"} 8
`
	if err := testutil.GatherAndCompare(m.Registry(), strings.NewReader(want),
		"gophertrunk_conventional_channels_active", "gophertrunk_conventional_channels_configured"); err != nil {
		t.Error(err)
	}
	// Read at scrape time: a change shows on the next scrape.
	active = 3
	if err := testutil.GatherAndCompare(m.Registry(), strings.NewReader(strings.Replace(want,
		`gophertrunk_conventional_channels_active{kind="wideband"} 1`, `gophertrunk_conventional_channels_active{kind="wideband"} 3`, 1)),
		"gophertrunk_conventional_channels_active", "gophertrunk_conventional_channels_configured"); err != nil {
		t.Error(err)
	}
}

func TestRegisterDelivery(t *testing.T) {
	m, err := New(nil, nil, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.RegisterDelivery(func() DeliveryCounts {
		return DeliveryCounts{
			Sent:    map[string]int{"webhook": 10, "rdioscanner": 7},
			Failed:  map[string]int{"webhook": 2},
			Dropped: 1,
			Queued:  3,
		}
	}); err != nil {
		t.Fatal(err)
	}
	want := `
# HELP gophertrunk_delivery_dropped_total Items dropped before delivery because a delivery queue was full.
# TYPE gophertrunk_delivery_dropped_total counter
gophertrunk_delivery_dropped_total 1
# HELP gophertrunk_delivery_failures_total Items an outbound feed gave up on after its retries, by backend.
# TYPE gophertrunk_delivery_failures_total counter
gophertrunk_delivery_failures_total{backend="rdioscanner"} 0
gophertrunk_delivery_failures_total{backend="webhook"} 2
# HELP gophertrunk_delivery_queue_depth Items waiting in the delivery queues.
# TYPE gophertrunk_delivery_queue_depth gauge
gophertrunk_delivery_queue_depth 3
# HELP gophertrunk_delivery_sent_total Items delivered to an outbound feed, by backend: completed calls, or grants for a grant_webhook:<name> backend.
# TYPE gophertrunk_delivery_sent_total counter
gophertrunk_delivery_sent_total{backend="rdioscanner"} 7
gophertrunk_delivery_sent_total{backend="webhook"} 10
`
	if err := testutil.GatherAndCompare(m.Registry(), strings.NewReader(want),
		"gophertrunk_delivery_dropped_total", "gophertrunk_delivery_failures_total",
		"gophertrunk_delivery_queue_depth", "gophertrunk_delivery_sent_total"); err != nil {
		t.Error(err)
	}
}

// Close used to wait for Run unconditionally, so a daemon built with metrics
// on and closed without running (as a test does) hung forever in Close.
func TestCloseWithoutRunReturns(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, err := New(bus, nil, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = m.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked waiting for a Run that never started")
	}
}

// With Run going, Close still waits for it to drain.
func TestCloseWaitsForRun(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, err := New(bus, nil, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan struct{})
	go func() { _ = m.Run(context.Background()); close(ran) }()
	for !m.running.Load() {
		time.Sleep(time.Millisecond)
	}
	_ = m.Close()
	select {
	case <-ran:
	default:
		t.Fatal("Close returned before Run finished")
	}
}
