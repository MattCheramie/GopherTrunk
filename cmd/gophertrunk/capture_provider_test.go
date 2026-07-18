package main

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// idleDevice is a minimal sdr.Device whose StreamIQ is never called by the
// capture path: the broker only fans IQ out to subscribers while a primary
// StreamIQ session is active, so a broker with no primary stream (modelling a
// control SDR whose scanner is mid-hunt / in control-channel backoff) delivers
// nothing to a capture subscriber.
type idleDevice struct{ rateOnlyDevice }

// TestCaptureProviderTimesOutWhenBrokerIdle pins the fix for the "Capture from
// tuner hangs forever" report: when the tuner is not streaming (no primary
// StreamIQ session — e.g. a scan looping trying → hunt failed → backoff), a
// capture must return a bounded error instead of blocking the request forever.
//
// Before the timer-arm fix the select in captureProvider.CaptureStream blocked
// on sub.C with no deadline arm, so this test would hang until the -timeout kills
// the run.
func TestCaptureProviderTimesOutWhenBrokerIdle(t *testing.T) {
	br := iqtap.New(idleDevice{}, 0, nil)
	if err := br.SetSampleRate(48_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	// No StreamIQ session is ever started, so the broker never delivers a
	// chunk to a capture subscriber.

	p := &captureProvider{brokers: map[string]*iqtap.Broker{"dev-a": br}}

	// The capture's internal safety deadline is seconds+5s; give it generous
	// headroom before declaring a hang so the test is not flaky under load.
	const seconds = 1
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, _, err := p.CaptureStream(context.Background(), "dev-a", seconds, func([]complex64) error { return nil })
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Capture returned nil error for an idle (non-streaming) tuner; want a bounded timeout error")
		}
		if elapsed := time.Since(start); elapsed > (seconds+5)*time.Second+2*time.Second {
			t.Errorf("Capture took %v; expected it to bail near the seconds+5s safety deadline", elapsed)
		}
	case <-time.After((seconds+5)*time.Second + 5*time.Second):
		t.Fatal("Capture blocked past its safety deadline — the hang is not fixed")
	}
}
