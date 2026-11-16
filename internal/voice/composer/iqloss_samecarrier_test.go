package composer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// sameCarrierLossySource is a control-carrier voice tap as the composer sees
// it: several taps (serials) share one carrier key, and the stream reports
// its IQ loss.
type sameCarrierLossySource struct {
	*lossySource
}

func (s *sameCarrierLossySource) CarrierKey() string { return "cc-carrier-1363" }

// perCallSink keeps each call's loss reader (iqLossTrackerSink).
type perCallSink struct {
	recordingSink
	mu    sync.Mutex
	reads map[uint64]func() uint64
}

func (s *perCallSink) TrackIQLoss(_ string, callID uint64, read func() uint64) {
	s.mu.Lock()
	if s.reads == nil {
		s.reads = map[uint64]func() uint64{}
	}
	s.reads[callID] = read
	s.mu.Unlock()
}

func (s *perCallSink) lostFor(t *testing.T, callID uint64) uint64 {
	t.Helper()
	s.mu.Lock()
	read := s.reads[callID]
	s.mu.Unlock()
	if read == nil {
		t.Fatalf("call %d: the recorder was given no IQ-loss reader, so its recording can never be marked partial", callID)
	}
	return read()
}

func tetraSameCarrierCall(serial string, callID uint64, marker uint8) (trunking.CallStart, trunking.CallEnd) {
	g := trunking.Grant{System: "tetra-net", Protocol: "tetra", GroupID: 1000 + uint32(callID), CallID: callID,
		Timeslot: marker - 3, TETRAUsageMarker: marker}
	now := time.Now()
	return trunking.CallStart{Grant: g, DeviceSerial: serial, StartedAt: now},
		trunking.CallEnd{Grant: g, DeviceSerial: serial, StartedAt: now, EndedAt: now.Add(time.Second), Reason: trunking.EndReasonNormal}
}

func newSameCarrierComposer(t *testing.T, sink PCMSink) (*Composer, *sameCarrierLossySource, context.Context) {
	t.Helper()
	src := &sameCarrierLossySource{&lossySource{fakeSource: newFakeSource()}}
	bus := events.NewBus(16)
	t.Cleanup(bus.Close)
	c, err := New(Options{
		Bus: bus,
		Devices: &fakeDevices{src: map[string]IQSource{
			"cc:same-carrier:1": src,
			"cc:same-carrier:2": src,
		}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  144_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		c.cancelAll()
	})
	return c, src, ctx
}

// TestTETRASameCarrierCallIsChargedItsIQLoss: TETRA calls on a control
// carrier share one demux stream and open none of their own, so until #1363
// no loss reader reached the recorder and a call recorded across dropped IQ
// was announced complete. Each call is now charged the shared stream's loss
// during its own span: not what was lost before it began, and for two calls
// at once, each its own overlap.
func TestTETRASameCarrierCallIsChargedItsIQLoss(t *testing.T) {
	sink := &perCallSink{}
	c, src, ctx := newSameCarrierComposer(t, sink)

	csA, ceA := tetraSameCarrierCall("cc:same-carrier:1", 1, 4)
	c.handleStart(ctx, csA) // creates the carrier's demux
	src.lost.Add(3)         // lost while only A is up
	if got := sink.lostFor(t, 1); got != 3 {
		t.Fatalf("call A mid-call loss = %d, want 3", got)
	}

	csB, ceB := tetraSameCarrierCall("cc:same-carrier:2", 2, 5)
	c.handleStart(ctx, csB)
	if got := sink.lostFor(t, 2); got != 0 {
		t.Fatalf("call B was charged %d chunks lost before it began, want 0", got)
	}
	src.lost.Add(2) // lost while both are up
	c.handleEnd(ceA)
	src.lost.Add(4) // lost while only B is up
	c.handleEnd(ceB)

	// A's reader keeps running after its end (the recorder stops reading at
	// its last file boundary), so compare at the instants that matter: B
	// overlapped 2 + 4, and A had seen 3 + 2 when it ended.
	if got := sink.lostFor(t, 2); got != 6 {
		t.Fatalf("call B loss = %d, want 6 (2 shared with A + 4 alone)", got)
	}

	// A third call after all of it starts clean.
	csC, ceC := tetraSameCarrierCall("cc:same-carrier:1", 3, 4)
	c.handleStart(ctx, csC)
	if got := sink.lostFor(t, 3); got != 0 {
		t.Fatalf("call C was charged %d chunks lost before it began, want 0", got)
	}
	c.handleEnd(ceC)
}

// A sink that only takes the end-of-call total (no per-file tracking) gets
// the same-carrier call's loss before the drain signal, like any other call.
func TestTETRASameCarrierCallReportsLossTotalBeforeDrain(t *testing.T) {
	sink := &lossSink{}
	c, src, ctx := newSameCarrierComposer(t, sink)

	cs, ce := tetraSameCarrierCall("cc:same-carrier:1", 7, 4)
	src.lost.Add(9) // before the call: must not count
	c.handleStart(ctx, cs)
	src.lost.Add(5)
	c.handleEnd(ce)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.events) != 2 || sink.events[0] != "loss" || sink.events[1] != "drain" {
		t.Fatalf("signals = %v, want [loss drain]", sink.events)
	}
	if sink.lost != 5 || sink.callID != 7 {
		t.Fatalf("loss reported = %d for call %d, want 5 for call 7", sink.lost, sink.callID)
	}
}
