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

func TestPackBits(t *testing.T) {
	in := make([]byte, dmrvoice.AMBEFrameBits)
	for i := range in {
		if i%3 == 0 {
			in[i] = 1
		}
	}
	got := packBits(in)
	if len(got) != 9 {
		t.Fatalf("packed length = %d, want 9", len(got))
	}
	for i := range in {
		if bit := (got[i>>3] >> uint(7-(i&7))) & 1; bit != in[i] {
			t.Errorf("bit %d = %d, want %d", i, bit, in[i])
		}
	}
}

// mkInfo builds a deterministic, unique 49-bit AMBE payload from seed.
func mkInfo(seed int) []byte {
	f := make([]byte, 49)
	x := uint32(seed)*2654435761 + 1
	for i := range f {
		x = x*1664525 + 1013904223
		f[i] = byte(x >> 31)
	}
	return f
}

// voiceBurstDibits assembles a 132-dibit voice burst from three 72-bit
// on-air AMBE frames and a 24-dibit sync / embedded-signalling field.
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
// (with a clock-settling lead-in). Each of the 18*n AMBE frames it
// carries is the FEC-encoding of mkInfo(frameIndex); the returned
// slice holds those 49-bit payloads in order.
func buildVoiceStream(t *testing.T, n int) (dibits []uint8, infos [][]byte) {
	t.Helper()
	dibits = make([]uint8, 240)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	var onair [][]byte
	for f := 0; f < n*dmrvoice.FramesPerSuperframe; f++ {
		info := mkInfo(f)
		frame, err := dmrvoice.EncodeAMBEFrame(info)
		if err != nil {
			t.Fatalf("EncodeAMBEFrame: %v", err)
		}
		infos = append(infos, info)
		onair = append(onair, frame)
	}
	for s := 0; s < n; s++ {
		for b := 0; b < dmrvoice.BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := s*dmrvoice.FramesPerSuperframe + b*dmrvoice.FramesPerBurst
			dibits = append(dibits, voiceBurstDibits(onair[base:base+dmrvoice.FramesPerBurst], sync)...)
		}
	}
	return dibits, infos
}

// TestComposerDMRVoiceChainExtractsRawFrames drives the full composer
// DMR voice path — modulated IQ → DMR receiver → voice superframe
// decoder → AMBE FEC → recorder .raw sidecar — and confirms a decoded
// superframe round-trips to the modulated 49-bit payload exactly.
func TestComposerDMRVoiceChainExtractsRawFrames(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 10
		span        = 8
		alpha       = 0.20
		deviation   = 1944.0
		superframes = 12
	)
	dibits, infos := buildVoiceStream(t, superframes)
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
	if len(got) == 0 {
		t.Fatal("got no raw frames")
	}
	// The chain appends a superframe's 18 frames one WriteRawFrame call
	// at a time, so a rawFrames read can land mid-superframe. Trim any
	// partial trailing superframe before the alignment-sensitive checks.
	got = got[:len(got)-len(got)%dmrvoice.FramesPerSuperframe]
	if len(got) == 0 {
		t.Fatalf("no complete superframe among the raw frames")
	}
	for _, f := range got {
		if len(f) != 7 {
			t.Fatalf("raw frame length = %d, want 7 (49 FEC-decoded bits packed)", len(f))
		}
	}

	// At least one decoded superframe must round-trip to its modulated
	// 49-bit payload — proving deinterleave + Golay + descramble are
	// wired correctly through the live chain.
	if !matchesAnySuperframe(got, infos) {
		t.Errorf("no decoded superframe round-tripped to its modulated payload")
	}

	// The chain keeps the call alive via Engine.Touch; the ticker may
	// not have fired yet when the frames landed, so wait for it.
	waitFor(t, time.Second, func() bool { return eng.touched.Load() > 0 })
}

func matchesAnySuperframe(got [][]byte, infos [][]byte) bool {
	const sf = dmrvoice.FramesPerSuperframe
	for in := 0; in+sf <= len(infos); in += sf {
		for g := 0; g+sf <= len(got); g += sf {
			ok := true
			for k := 0; k < sf; k++ {
				if !bytes.Equal(got[g+k], packBits(infos[in+k])) {
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
