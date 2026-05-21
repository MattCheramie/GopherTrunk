package composer

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// p25p1VoiceInfo builds a deterministic, unique 88-bit IMBE info frame.
func p25p1VoiceInfo(seed int) []byte {
	bits := make([]byte, imbe.InfoBits)
	x := uint32(seed)*2654435761 + 1
	for i := range bits {
		x = x*1664525 + 1013904223
		bits[i] = byte(x >> 31)
	}
	return bits
}

// buildP25P1VoiceStream assembles a dibit stream of `ldus` P25 Phase 1
// LDU1s preceded by a clock-settling lead-in. want holds every IMBE
// frame it carries, in transmission order.
func buildP25P1VoiceStream(t *testing.T, ldus int) (dibits []uint8, want [][]byte) {
	t.Helper()
	dibits = make([]uint8, 400)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	frame := 0
	for l := 0; l < ldus; l++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for s := range voice {
			ch, err := imbe.EncodeChannel(p25p1VoiceInfo(frame))
			frame++
			if err != nil {
				t.Fatalf("EncodeChannel: %v", err)
			}
			onAir, err := imbe.Scramble(ch)
			if err != nil {
				t.Fatalf("Scramble: %v", err)
			}
			// Scramble/Descramble mutate in place — keep an independent
			// copy for the LDU so the want computation cannot corrupt it.
			voice[s] = append([]byte(nil), onAir...)
			wf, _, _ := imbe.DecodeChannelToFrame(onAir)
			want = append(want, wf)
		}
		var lces [phase1.LDULCESBlockCount][]byte
		var lsd [phase1.LDULSDBlockCount][]byte
		ldu, err := phase1.AssembleLDU(0x123, phase1.DUIDLogicalLink1, voice, lces, lsd)
		if err != nil {
			t.Fatalf("AssembleLDU: %v", err)
		}
		dibits = append(dibits, framing.BitsToDibits(ldu)...)
	}
	return dibits, want
}

// TestComposerP25Phase1VoiceChainExtractsRawFrames drives the full
// composer Phase 1 voice path — modulated C4FM IQ → Phase 1 receiver →
// LDU assembler → IMBE voice-frame extraction → recorder .raw sidecar —
// and confirms the recovered frames round-trip to the modulated ones.
func TestComposerP25Phase1VoiceChainExtractsRawFrames(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 12
	)
	framesPerLDU := phase1.LDUVoiceSubframeCount

	dibits, want := buildP25P1VoiceStream(t, ldus)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

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
				System: "P25P1Site", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= 6*framesPerLDU
	})

	got := sink.rawFrames("VOICE-1")
	for _, f := range got {
		if len(f) != imbe.FrameBytes {
			t.Fatalf("raw frame length = %d, want %d", len(f), imbe.FrameBytes)
		}
	}

	// At least one full LDU's worth of IMBE frames must round-trip to
	// the modulated frames, proving the receiver → LDU assembler →
	// voice-extraction chain is wired correctly through live IQ. (The
	// C4FM demod is not bit-exact, so a stricter all-frames check
	// belongs in the receiver's own unit tests, not this wiring test.)
	matches := bestAlignmentMatches(got, want)
	if matches < framesPerLDU {
		t.Errorf("only %d of %d captured frames round-tripped; want at least one LDU (%d)",
			matches, len(got), framesPerLDU)
	}

	waitFor(t, time.Second, func() bool { return eng.touched.Load() > 0 })
}
