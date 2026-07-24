package composer

import (
	"context"
	"encoding/binary"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// sameCarrierReplaySource is a same-carrier IQ tap for the demux replay harness:
// every serial backed by it shares one CarrierKey (so they share one demux), and
// StreamIQ replays a fixed 144 kHz IQ buffer once. It reports the pipeline rate so
// the demux front end runs at decim==1 like the live CCVoiceSource.
type sameCarrierReplaySource struct {
	iq   []complex64
	rate uint32
}

func (s *sameCarrierReplaySource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		const chunk = 4096
		for i := 0; i < len(s.iq); i += chunk {
			e := i + chunk
			if e > len(s.iq) {
				e = len(s.iq)
			}
			select {
			case <-ctx.Done():
				return
			case ch <- s.iq[i:e]:
			}
		}
		<-ctx.Done()
	}()
	return ch, nil
}

func (s *sameCarrierReplaySource) SampleRateHz() uint32     { return s.rate }
func (s *sameCarrierReplaySource) SampleRateExactHz() float64 { return float64(s.rate) }
func (s *sameCarrierReplaySource) CarrierKey() string       { return "replay-carrier" }

// TestTETRASameCarrierNoLeakReplay drives the composer's shared per-carrier slot
// demux with a real same-carrier cs16 capture and asserts a call granted an EMPTY
// timeslot records NO speech — i.e. the busy slot's audio does not leak into it.
// Under the old per-call extractors the empty-slot call's fresh extractor would
// accept every slot's speech until it anchored (~1 s), so this test fails on the
// pre-fix behaviour and passes with the demux.
//
//	GT_TETRA_IQ=<cs16> GT_TETRA_IQ_RATE=2500000 GT_TETRA_BUSY_SLOT=2 GT_TETRA_EMPTY_SLOT=3 \
//	  go test ./internal/voice/composer -run TestTETRASameCarrierNoLeakReplay -v
func TestTETRASameCarrierNoLeakReplay(t *testing.T) {
	path := os.Getenv("GT_TETRA_IQ")
	if path == "" {
		t.Skip("set GT_TETRA_IQ (cs16) + GT_TETRA_IQ_RATE to run the same-carrier no-leak replay")
	}
	inRate := 2500000.0
	if v := os.Getenv("GT_TETRA_IQ_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			inRate = f
		}
	}
	busySlot := uint8(envInt("GT_TETRA_BUSY_SLOT", 2))
	emptySlot := uint8(envInt("GT_TETRA_EMPTY_SLOT", 3))
	// The cell's extended colour code (descramble seed). Overridable per capture;
	// defaults to the reporter's cell, which decodes the 24-Jul same-carrier grabs.
	colourExt := uint32(envInt("GT_TETRA_COLOUR", 262144876))

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	full := make([]complex64, len(raw)/4)
	for i := range full {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		full[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	// Channelise to the 144 kHz pipeline rate the live CCVoiceSource delivers.
	ddc := ccdecoder.NewDownconverter(inRate, 144000)
	var iq144 []complex64
	const chunk = 65536
	var scratch []complex64
	for i := 0; i < len(full); i += chunk {
		e := i + chunk
		if e > len(full) {
			e = len(full)
		}
		scratch = ddc.Process(scratch[:0], full[i:e])
		iq144 = append(iq144, scratch...)
	}

	src := &sameCarrierReplaySource{iq: iq144, rate: uint32(ddc.OutRateHz() + 0.5)}
	bus := events.NewBus(64)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"cc:sc:busy": src, "cc:sc:empty": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  144000,
		PCMSampleRate: 8000,
		TouchInterval: 50 * time.Millisecond,
		VoiceHangtime: 30 * time.Second, // keep both calls open for the whole replay
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	grant := func(serial string, slot uint8) {
		bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "TETRASite", Protocol: "tetra", GroupID: uint32(slot) * 1000,
				FrequencyHz: 467913000, Timeslot: slot, TETRAColourExt: colourExt,
			},
			DeviceSerial: serial,
		}})
	}
	// Register both calls BEFORE the IQ replays so slot ownership is set. The busy
	// slot's grant creates the demux (and its single StreamIQ subscription).
	grant("cc:sc:busy", busySlot)
	grant("cc:sc:empty", emptySlot)

	// Wait until both chains are registered (owners added), then let the replay run
	// to completion by waiting for the busy slot to accumulate speech.
	waitFor(t, 5*time.Second, func() bool { return len(c.ActiveChains()) == 2 })
	waitFor(t, 30*time.Second, func() bool { return len(sink.rawFrames("cc:sc:busy")) > 0 })
	// Give the whole buffer time to drain through the single demux goroutine.
	time.Sleep(500 * time.Millisecond)

	busy := len(sink.rawFrames("cc:sc:busy"))
	empty := len(sink.rawFrames("cc:sc:empty"))
	t.Logf("busy slot %d frames=%d ; empty slot %d frames=%d", busySlot, busy, emptySlot, empty)
	if busy == 0 {
		t.Fatalf("busy slot %d recorded no speech — capture/slot mismatch (set GT_TETRA_BUSY_SLOT)", busySlot)
	}
	if empty != 0 {
		t.Errorf("empty slot %d recorded %d speech frames — busy slot leaked into it", emptySlot, empty)
	}
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
