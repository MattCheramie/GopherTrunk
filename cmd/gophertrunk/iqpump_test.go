package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// failStreamDevice is an sdr.Device whose StreamIQ always errors, used to
// exercise the claim-release path in openSingleChannelIQ.
type failStreamDevice struct{ serial string }

func (f *failStreamDevice) Info() sdr.Info             { return sdr.Info{Driver: "fail", Serial: f.serial} }
func (f *failStreamDevice) SetCenterFreq(uint32) error { return nil }
func (f *failStreamDevice) SetSampleRate(uint32) error { return nil }
func (f *failStreamDevice) SetGain(int) error          { return nil }
func (f *failStreamDevice) SetPPM(int) error           { return nil }
func (f *failStreamDevice) SetBiasTee(bool) error      { return nil }
func (f *failStreamDevice) Close() error               { return nil }
func (f *failStreamDevice) StreamIQ(context.Context) (<-chan []complex64, error) {
	return nil, errors.New("fail: StreamIQ unavailable")
}

// TestOpenSingleChannelIQDrivesPrimary locks the issue #547 fix: a
// single-channel decoder on a dongle with no trunking/wideband primary must
// itself drive the broker's StreamIQ pump so the broker actually fans IQ out
// to its subscribers. Before the fix the decoder only Subscribe()d and the
// broker — never pumped — delivered nothing.
func TestOpenSingleChannelIQDrivesPrimary(t *testing.T) {
	dev := newStreamingFake("dev-primary")
	br := iqtap.New(dev, 0, nil)
	d := &Daemon{log: slog.New(slog.DiscardHandler), iqPrimary: map[string]bool{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, cleanup, err := d.openSingleChannelIQ(ctx, br, "dev-primary")
	if err != nil {
		t.Fatalf("openSingleChannelIQ: %v", err)
	}
	defer cleanup()

	// The first caller becomes the primary: StreamIQ is now driving the
	// broker, so Stats reports Streaming and IQ actually flows to the
	// returned channel.
	if !br.Stats().Streaming {
		t.Fatal("broker not Streaming after first openSingleChannelIQ — pump not driven (issue #547)")
	}
	select {
	case chunk, ok := <-ch:
		if !ok {
			t.Fatal("primary IQ channel closed prematurely")
		}
		if len(chunk) == 0 {
			t.Fatal("primary delivered an empty chunk")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no IQ delivered to the primary decoder within 2s")
	}
}

// TestOpenSingleChannelIQSecondConsumerSubscribes verifies that once a serial
// has a primary, a later consumer on the same serial Subscribes rather than
// opening a second (conflicting) StreamIQ.
func TestOpenSingleChannelIQSecondConsumerSubscribes(t *testing.T) {
	dev := newStreamingFake("dev-shared")
	br := iqtap.New(dev, 0, nil)
	d := &Daemon{log: slog.New(slog.DiscardHandler), iqPrimary: map[string]bool{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First consumer: primary. No subscribers registered yet.
	_, cleanup1, err := d.openSingleChannelIQ(ctx, br, "dev-shared")
	if err != nil {
		t.Fatalf("first openSingleChannelIQ: %v", err)
	}
	defer cleanup1()
	if got := br.Stats().Subscribers; got != 0 {
		t.Fatalf("after primary: Subscribers = %d, want 0", got)
	}

	// Second consumer on the same serial: must Subscribe (one subscriber),
	// not open another stream.
	ch2, cleanup2, err := d.openSingleChannelIQ(ctx, br, "dev-shared")
	if err != nil {
		t.Fatalf("second openSingleChannelIQ: %v", err)
	}
	defer cleanup2()
	if got := br.Stats().Subscribers; got != 1 {
		t.Fatalf("after second consumer: Subscribers = %d, want 1 (should Subscribe, not re-StreamIQ)", got)
	}
	// The subscriber still receives fanned-out IQ because the primary pump
	// is running.
	select {
	case chunk, ok := <-ch2:
		if !ok || len(chunk) == 0 {
			t.Fatal("subscriber received no usable IQ")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no IQ fanned out to the second consumer within 2s")
	}
}

// TestOpenSingleChannelIQRespectsExistingPrimary verifies that a serial
// already owned by a trunking/wideband primary (pre-seeded in iqPrimary)
// makes the single-channel decoder Subscribe instead of opening a second
// StreamIQ on the same dongle.
func TestOpenSingleChannelIQRespectsExistingPrimary(t *testing.T) {
	dev := newStreamingFake("dev-trunk")
	br := iqtap.New(dev, 0, nil)
	d := &Daemon{
		log:       slog.New(slog.DiscardHandler),
		iqPrimary: map[string]bool{"dev-trunk": true}, // CC/wideband already owns it
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, cleanup, err := d.openSingleChannelIQ(ctx, br, "dev-trunk")
	if err != nil {
		t.Fatalf("openSingleChannelIQ: %v", err)
	}
	defer cleanup()

	if got := br.Stats().Subscribers; got != 1 {
		t.Fatalf("Subscribers = %d, want 1 (must Subscribe when a primary already owns the serial)", got)
	}
	if br.Stats().Streaming {
		t.Fatal("single-channel decoder opened its own StreamIQ on a dongle already owned by a primary")
	}
}

// TestOpenSingleChannelIQReleasesClaimOnError verifies that a failed StreamIQ
// releases the primary claim so a later consumer (or retry) can drive the
// pump instead of the serial being wedged forever on a dead primary.
func TestOpenSingleChannelIQReleasesClaimOnError(t *testing.T) {
	br := iqtap.New(&failStreamDevice{serial: "dev-fail"}, 0, nil)
	d := &Daemon{log: slog.New(slog.DiscardHandler), iqPrimary: map[string]bool{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, _, err := d.openSingleChannelIQ(ctx, br, "dev-fail"); err == nil {
		t.Fatal("openSingleChannelIQ: want error from failing StreamIQ, got nil")
	}

	d.iqPrimaryMu.Lock()
	claimed := d.iqPrimary["dev-fail"]
	d.iqPrimaryMu.Unlock()
	if claimed {
		t.Fatal("primary claim not released after StreamIQ error — serial is wedged")
	}
}
