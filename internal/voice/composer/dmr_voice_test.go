package composer

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestPackAMBEFrame(t *testing.T) {
	bits := make([]byte, dmrvoice.AMBEFrameBits)
	for i := range bits {
		if i%3 == 0 {
			bits[i] = 1
		}
	}
	got := packAMBEFrame(bits)
	if len(got) != 9 {
		t.Fatalf("packed length = %d, want 9", len(got))
	}
	for i := 0; i < dmrvoice.AMBEFrameBits; i++ {
		bit := (got[i>>3] >> uint(7-(i&7))) & 1
		want := byte(0)
		if i%3 == 0 {
			want = 1
		}
		if bit != want {
			t.Errorf("bit %d = %d, want %d", i, bit, want)
		}
	}
}

// mkAMBEFrame builds a deterministic, unique 72-bit AMBE frame from
// seed (one bit per byte) via a small LCG so every frame in a test
// stream is distinct.
func mkAMBEFrame(seed int) []byte {
	f := make([]byte, dmrvoice.AMBEFrameBits)
	x := uint32(seed)*2654435761 + 1
	for i := range f {
		x = x*1664525 + 1013904223
		f[i] = byte(x >> 31)
	}
	return f
}

// voiceBurstDibits assembles a 132-dibit voice burst from three 72-bit
// AMBE frames and a 24-dibit sync / embedded-signalling field.
func voiceBurstDibits(frames [][]byte, sync [24]uint8) []uint8 {
	var bits []byte
	for _, f := range frames {
		bits = append(bits, f...)
	}
	toD := func(b []byte) []uint8 {
		d := make([]uint8, len(b)/2)
		for i := range d {
			d[i] = b[2*i]<<1 | b[2*i+1]
		}
		return d
	}
	out := make([]uint8, 0, dmr.BurstDibits)
	out = append(out, toD(bits[:108])...)
	out = append(out, sync[:]...)
	out = append(out, toD(bits[108:])...)
	return out
}

// buildVoiceStream assembles a dibit stream of n DMR voice superframes
// preceded by a lead-in (transitions so the receiver's clock recovery
// settles), and returns the 18*n AMBE frames it carries.
func buildVoiceStream(n int) (dibits []uint8, frames [][]byte) {
	dibits = make([]uint8, 240)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	for f := 0; f < n*dmrvoice.FramesPerSuperframe; f++ {
		frames = append(frames, mkAMBEFrame(f))
	}
	for s := 0; s < n; s++ {
		for b := 0; b < dmrvoice.BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := s*dmrvoice.FramesPerSuperframe + b*dmrvoice.FramesPerBurst
			dibits = append(dibits, voiceBurstDibits(frames[base:base+dmrvoice.FramesPerBurst], sync)...)
		}
	}
	return dibits, frames
}

// TestComposerDMRVoiceChainExtractsRawFrames drives the full composer
// DMR voice path — modulated IQ → DMR receiver → voice superframe
// decoder → packed AMBE frames → recorder .raw sidecar — and confirms
// a decoded superframe matches the modulated input exactly.
func TestComposerDMRVoiceChainExtractsRawFrames(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 10
		span        = 8
		alpha       = 0.20
		deviation   = 1944.0
		superframes = 12
	)
	dibits, frames := buildVoiceStream(superframes)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRate, deviation)

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
				System: "DMRSite", Protocol: "dmr-tier3",
				GroupID: 7, FrequencyHz: 460_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	// Wait for the chain to start (StreamIQ called) before feeding IQ.
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 4*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= dmrvoice.FramesPerSuperframe
	})

	got := sink.rawFrames("VOICE-1")
	if len(got) == 0 || len(got)%dmrvoice.FramesPerSuperframe != 0 {
		t.Fatalf("got %d raw frames, want a non-zero multiple of %d",
			len(got), dmrvoice.FramesPerSuperframe)
	}
	for _, f := range got {
		if len(f) != 9 {
			t.Fatalf("raw frame length = %d, want 9 (72 bits packed)", len(f))
		}
	}

	// At least one decoded superframe must match an input superframe
	// exactly — proving the chain extracts the right bits in order.
	if !matchesAnySuperframe(got, frames) {
		t.Errorf("no decoded superframe matched an input superframe exactly")
	}

	// The chain keeps the call alive via Engine.Touch; the ticker may
	// not have fired yet when the frames landed, so wait for it.
	waitFor(t, time.Second, func() bool { return eng.touched.Load() > 0 })
}

func matchesAnySuperframe(got [][]byte, frames [][]byte) bool {
	const sf = dmrvoice.FramesPerSuperframe
	for in := 0; in+sf <= len(frames); in += sf {
		for g := 0; g+sf <= len(got); g += sf {
			ok := true
			for k := 0; k < sf; k++ {
				if !bytes.Equal(got[g+k], packAMBEFrame(frames[in+k])) {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}
	return false
}
