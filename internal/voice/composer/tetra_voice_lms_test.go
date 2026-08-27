package composer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComposerTETRAVoiceChainWithTrafficLMS is the no-harm wiring pin for
// tetra_traffic_lms: a grant carrying TETRATrafficLMS starts a chain whose
// receiver feeds the extractor's StashSymbols alongside StashSoft (the
// GT_TETRA_LMS harness wiring, now production-reachable), and the chain
// still processes IQ and tears down exactly like the default chain — the
// DSP win itself is pinned by tetra's traffic_lms_test.go; this guards the
// composer plumbing (symbol/dibit alignment through the real receiver, no
// panic, clean lifecycle).
func TestComposerTETRAVoiceChainWithTrafficLMS(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.35
		slots = 24
	)
	dibits := buildTETRATrafficDibits(slots)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sps * 18000),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		VoiceHangtime: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "TETRASite", Protocol: "tetra",
				GroupID: 1234, FrequencyHz: 390_000_000, Timeslot: 1,
				TETRATrafficLMS: true,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 10*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// Same teardown contract as the default chain: the filler bursts carry
	// no valid TCH/S speech, so the no-voice startup timeout ends the call.
	waitFor(t, 15*time.Second, func() bool { return eng.endReason("VOICE-1") == trunking.EndReasonTimeout })
}
