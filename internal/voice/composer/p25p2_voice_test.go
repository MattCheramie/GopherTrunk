package composer

import (
	"bytes"
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25p2VoicePayload builds a deterministic, unique VoiceFrameBytes-long
// AMBE+2 frame. The 7 padding bits (low bits of the last byte) are
// cleared so an encode→extract round-trip is bit-exact.
func p25p2VoicePayload(seed int) []byte {
	out := make([]byte, p25p2.VoiceFrameBytes)
	x := uint32(seed)*2654435761 + 12345
	for i := range out {
		x = x*1664525 + 1013904223
		out[i] = byte(x >> 24)
	}
	out[p25p2.VoiceFrameBytes-1] &= 0x80
	return out
}

// buildP25P2VoiceStream assembles a dibit stream of n all-voice (4V)
// superframes preceded by a clock-settling lead-in. want holds every
// AMBE+2 payload it carries, in transmission order.
func buildP25P2VoiceStream(n int) (dibits []uint8, want [][]byte) {
	dibits = make([]uint8, 600)
	for i := range dibits {
		dibits[i] = uint8(i % 4) // never matches the outbound sync
	}
	frame := 0
	for s := 0; s < n; s++ {
		var subs [p25p2.SubframesPerSuperframe][]uint8
		for i := range subs {
			payloads := make([][]byte, p25p2.Voice4VFrameCount)
			for j := range payloads {
				p := p25p2VoicePayload(frame)
				frame++
				payloads[j] = p
				want = append(want, p)
			}
			subs[i] = p25p2.EncodeVoiceSubframe(p25p2.SlotTypeVoice4V, uint8(i), payloads)
		}
		dibits = append(dibits, p25p2.EncodeSuperframe(subs)...)
	}
	return dibits, want
}

// bestAlignmentMatches returns the largest number of bit-exact frame
// matches between got and want over any stream alignment — the metric
// for "did the live IQ → voice chain recover the modulated payloads".
func bestAlignmentMatches(got, want [][]byte) int {
	best := 0
	for g := -len(want); g < len(got); g++ {
		n := 0
		for i := range got {
			w := i - g
			if w < 0 || w >= len(want) {
				continue
			}
			if bytes.Equal(got[i], want[w]) {
				n++
			}
		}
		if n > best {
			best = n
		}
	}
	return best
}

// TestComposerP25Phase2VoiceChainExtractsRawFrames drives the full
// composer Phase 2 voice path — modulated H-DQPSK IQ → Phase 2 receiver
// → superframe decoder → ISCH/SlotType routing → AMBE+2 FEC → recorder
// .raw sidecar — and confirms the recovered frames round-trip to the
// modulated payloads.
func TestComposerP25Phase2VoiceChainExtractsRawFrames(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 8
		span        = 8
		alpha       = 0.20
		superframes = 8
	)
	framesPerSuperframe := p25p2.SubframesPerSuperframe * p25p2.Voice4VFrameCount

	dibits, want := buildP25P2VoiceStream(superframes)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/8)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
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
				System: "P25P2Site", Protocol: "p25-phase2",
				GroupID: 42, FrequencyHz: 851_062_500,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= framesPerSuperframe
	})

	got := sink.rawFrames("VOICE-1")
	for _, f := range got {
		if len(f) != p25p2.VoiceFrameBytes {
			t.Fatalf("raw frame length = %d, want %d", len(f), p25p2.VoiceFrameBytes)
		}
	}

	// At least one full superframe's worth of AMBE+2 frames must
	// round-trip to its modulated payload, proving the receiver →
	// superframe decoder → ISCH routing → voice-FEC chain is wired
	// correctly end-to-end through live IQ.
	matches := bestAlignmentMatches(got, want)
	if matches < framesPerSuperframe {
		t.Errorf("only %d/%d frames round-tripped; want at least one superframe (%d)",
			matches, len(got), framesPerSuperframe)
	}

	// The chain keeps the call alive via Engine.Touch.
	waitFor(t, time.Second, func() bool { return eng.touched.Load() > 0 })
}
