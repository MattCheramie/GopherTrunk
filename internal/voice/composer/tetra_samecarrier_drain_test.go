package composer

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// blockingRawSink is a PCMSink + rawFrameSink whose WriteRawFrame blocks in the
// first call until release is closed, so a test can observe that a call's tail
// frame is still being written and assert the chain has not finalized yet.
type blockingRawSink struct {
	entered     chan struct{}
	release     chan struct{}
	onceEntered sync.Once
	writes      atomic.Int64
}

func (s *blockingRawSink) WritePCM(string, []int16) error { return nil }

func (s *blockingRawSink) WriteRawFrame(_ string, _ []byte) error {
	s.onceEntered.Do(func() { close(s.entered) })
	<-s.release
	s.writes.Add(1)
	return nil
}

func waitForOwner(t *testing.T, d *tetraSlotDemux, marker uint8) *tetraSlotOwner {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		d.mu.Lock()
		o := d.owners[marker]
		d.mu.Unlock()
		if o != nil {
			return o
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("owner never registered with the demux")
	return nil
}

// TestTETRASameCarrierChainWaitsForOwnerDrain pins part (B) of the trailing-frame
// fix: a same-carrier TETRA chain must not close its done channel (which the
// composer takes as "all frames written" before signalling the recorder to
// finalize) until its owner worker has drained the bursts queued at call end and
// written their speech frames. A blocking sink holds the tail write open so the
// test can prove done stays open while the write is in flight.
func TestTETRASameCarrierChainWaitsForOwnerDrain(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sink := &blockingRawSink{entered: make(chan struct{}), release: make(chan struct{})}
	c := &Composer{
		log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus:        bus,
		sink:       sink,
		hangtime:   time.Hour, // never let the boundary tracker end the call itself
		touchEvery: time.Hour,
	}
	d := &tetraSlotDemux{
		c:      c,
		key:    "k",
		owners: make(map[uint8]*tetraSlotOwner),
		done:   make(chan struct{}),
	}

	marker := uint8(tetra.DLUsageTraffic)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go c.runTETRASameCarrierChain(ctx, d, "serial-1", 1, 1, marker, done)

	// Queue a CRC-valid TCH/S burst so the worker has a tail job to drain at cancel.
	o := waitForOwner(t, d, marker)
	frame := tetra.EncodeTCHS(make([]byte, 137), make([]byte, 137))
	o.enqueue(frame, nil)

	cancel() // call end (control-channel D-RELEASE analogue)

	// The worker drains the queued burst and blocks writing its first speech frame.
	select {
	case <-sink.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("owner worker did not process the queued tail burst after cancel")
	}

	// While the tail write is in flight, done MUST stay open — otherwise the
	// composer would signal finalize and the recorder would delete the session
	// out from under the write (the dropped trailing-frames bug).
	select {
	case <-done:
		t.Fatal("chain done closed before the owner worker finished draining the tail")
	case <-time.After(50 * time.Millisecond):
	}

	// Let the write complete; the worker exits and only then does done close.
	close(sink.release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("chain done never closed after the drain completed")
	}
	if got := sink.writes.Load(); got < 1 {
		t.Fatalf("tail frames not written: %d", got)
	}
}
