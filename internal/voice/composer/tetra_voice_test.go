package composer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildTETRATrafficDibits lays out nSlots Normal Continuous Downlink Bursts
// (255-dibit TDMA slots) into a dibit stream, each with the normal training
// sequence at its §9.4.4.3.2 intra-slot position (bit 244 → dibit 122). The
// data blocks are deterministic filler — the chain test asserts the raw
// full-slot frames flow and drive the call lifecycle, not their payload
// (the payload round-trip is TrafficExtractor's own unit test).
func buildTETRATrafficDibits(nSlots int) []uint8 {
	const (
		slot   = 255
		leadIn = 512 // receiver clock/AFC warm-up
		ntsAt  = 122 // dibit offset of the NTS within the slot
	)
	nts := tetra.NormalSyncDibits()
	stream := make([]uint8, leadIn+nSlots*slot+slot)
	for i := range stream {
		stream[i] = uint8((i*3 + 1) % 4) // non-sync filler
	}
	for s := 0; s < nSlots; s++ {
		L := leadIn + s*slot + ntsAt
		copy(stream[L:], nts)
	}
	return stream
}

// TestComposerTETRAVoiceChainWritesRawFrames drives the composer's TETRA
// traffic-follow path end to end: modulated π/4-DQPSK IQ → TETRA receiver →
// TrafficExtractor → recorder .raw sidecar, and confirms full-slot traffic
// frames flow, the engine is kept alive, and the call ends on hangtime.
func TestComposerTETRAVoiceChainWritesRawFrames(t *testing.T) {
	const (
		sps   = 8 // 18000 baud × 8 = 144 kHz intermediate rate
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
		IQSampleRate:  uint32(sps * 18000), // 144 kHz tap
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		// Generous hangtime: the no-voice teardown fires at 2×hangtime, and the
		// heavy TETRA receiver processing the whole IQ chunk must emit its first
		// burst before then — otherwise, under a saturated CI running the whole
		// suite under -race, the call would end as a no-voice Timeout instead of
		// decoding traffic. 2 s → 4 s no-voice window is ample.
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
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	// Generous timeouts throughout: this test runs in the full `-race` suite on
	// shared CI, where the TETRA receiver DSP is CPU-heavy and slow to schedule.
	waitFor(t, 10*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// Full-slot traffic frames must reach the sidecar.
	waitFor(t, 20*time.Second, func() bool { return len(sink.rawFrames("VOICE-1")) >= 5 })
	got := sink.rawFrames("VOICE-1")
	if len(got) == 0 {
		t.Fatal("no raw traffic frames written")
	}
	for i, f := range got {
		if len(f) != tetra.TrafficFrameBytes {
			t.Fatalf("frame %d is %d bytes, want %d (BKN1+BKN2 type-5)", i, len(f), tetra.TrafficFrameBytes)
		}
	}

	// The chain keeps the engine's call alive while traffic flows.
	waitFor(t, 5*time.Second, func() bool { return eng.touched.Load() > 0 })

	// After the IQ stops, the boundary tracker ends the call on hangtime
	// (VoiceHangtime after the last decoded burst).
	waitFor(t, 15*time.Second, func() bool { return eng.endReason("VOICE-1") == trunking.EndReasonNormal })
}
