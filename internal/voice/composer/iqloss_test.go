package composer

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// lossySource is a fakeSource whose stream reports IQ loss
// (iqLossSource), as the wideband channel and virtual-tuner sources do.
type lossySource struct {
	*fakeSource
	lost atomic.Uint64
}

func (s *lossySource) StreamIQWithLoss(ctx context.Context) (<-chan []complex64, func() uint64, error) {
	ch, err := s.StreamIQ(ctx)
	return ch, s.lost.Load, err
}

// lossSink records the order of the loss and drain signals.
type lossSink struct {
	recordingSink
	mu     sync.Mutex
	events []string
	lost   uint64
	callID uint64
}

func (s *lossSink) EnableDrainCoordination()   {}
func (s *lossSink) NotifyDrainComplete(string) {}
func (s *lossSink) NotifyDrainCompleteForCall(string, uint64) {
	s.mu.Lock()
	s.events = append(s.events, "drain")
	s.mu.Unlock()
}
func (s *lossSink) NotifyIQLoss(_ string, callID uint64, n uint64) {
	s.mu.Lock()
	s.events = append(s.events, "loss")
	s.lost, s.callID = n, callID
	s.mu.Unlock()
}

func newLossComposer(t *testing.T, src IQSource, serial string) (*Composer, *lossSink) {
	t.Helper()
	bus := events.NewBus(16)
	t.Cleanup(bus.Close)
	sink := &lossSink{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{serial: src}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, sink
}

func fmCall(serial string) (trunking.CallStart, trunking.CallEnd) {
	g := trunking.Grant{System: "scanner", Protocol: "fm-conv", GroupID: 1, CallID: 42}
	now := time.Now()
	return trunking.CallStart{Grant: g, DeviceSerial: serial, StartedAt: now},
		trunking.CallEnd{Grant: g, DeviceSerial: serial, StartedAt: now, EndedAt: now.Add(time.Second), Reason: trunking.EndReasonNormal}
}

// TestComposerReportsCallIQLossBeforeDrain pins the #1288 hand-off: the IQ a
// call's stream lost reaches the recorder, named with the call's id, BEFORE
// the drain signal that lets the recorder finalize (afterwards it would miss
// the call.complete and the sidecar).
func TestComposerReportsCallIQLossBeforeDrain(t *testing.T) {
	src := &lossySource{fakeSource: newFakeSource()}
	c, sink := newLossComposer(t, src, "WB-CH-1")
	cs, ce := fmCall("WB-CH-1")
	c.handleStart(context.Background(), cs)
	src.lost.Store(6) // the tap dropped six chunks during the call
	c.handleEnd(ce)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.events) != 2 || sink.events[0] != "loss" || sink.events[1] != "drain" {
		t.Fatalf("signals = %v, want [loss drain]", sink.events)
	}
	if sink.lost != 6 || sink.callID != 42 {
		t.Fatalf("loss reported = %d for call %d, want 6 for call 42", sink.lost, sink.callID)
	}
}

// TestComposerReportsNoLossForCleanCall: nothing lost ⇒ only the drain.
func TestComposerReportsNoLossForCleanCall(t *testing.T) {
	src := &lossySource{fakeSource: newFakeSource()}
	c, sink := newLossComposer(t, src, "WB-CH-1")
	cs, ce := fmCall("WB-CH-1")
	c.handleStart(context.Background(), cs)
	c.handleEnd(ce)
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.events) != 1 || sink.events[0] != "drain" {
		t.Fatalf("signals = %v, want [drain]", sink.events)
	}
}

// TestComposerCountsPhysicalSDRDropsForPlainSource: a source without the
// loss capability (a physical voice SDR) is covered by the driver-drop
// counter of its own serial, counted only from the call's start.
func TestComposerCountsPhysicalSDRDropsForPlainSource(t *testing.T) {
	const serial = "PHYS-VOICE-1288"
	info := sdr.Info{Driver: "test", Serial: serial}
	sdr.NotifyIQDrop(info) // before the call: must not count
	c, sink := newLossComposer(t, newFakeSource(), serial)
	cs, ce := fmCall(serial)
	c.handleStart(context.Background(), cs)
	sdr.NotifyIQDrop(info)
	sdr.NotifyIQDrop(info)
	c.handleEnd(ce)
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if sink.lost != 2 {
		t.Fatalf("loss reported = %d, want 2 (drops during the call only)", sink.lost)
	}
}
