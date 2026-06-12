package composer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// errRecordingSink implements both rawFrameSink and errAwareRawSink. It
// records, per raw frame, whether the chain delivered it via the error-aware
// path (correctedBits >= 0) or the plain fallback (sentinel -1).
type errRecordingSink struct {
	mu     sync.Mutex
	frames [][]byte
	errs   []int
}

func (r *errRecordingSink) WritePCM(serial string, samples []int16) error { return nil }

func (r *errRecordingSink) WriteRawFrame(serial string, frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = append(r.frames, append([]byte(nil), frame...))
	r.errs = append(r.errs, -1) // sentinel: plain (non-error-aware) path taken
	return nil
}

func (r *errRecordingSink) WriteRawFrameWithErrors(serial string, frame []byte, correctedBits int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = append(r.frames, append([]byte(nil), frame...))
	r.errs = append(r.errs, correctedBits)
	return nil
}

func (r *errRecordingSink) snapshot() (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	plain := false
	for _, e := range r.errs {
		if e < 0 {
			plain = true
		}
	}
	return len(r.frames), plain
}

// TestComposerDMRVoiceForwardsCorrectedBits: when the sink implements
// errAwareRawSink, the DMR voice chain must route every FEC-decoded frame
// through WriteRawFrameWithErrors (carrying the Golay corrected-bit count to
// the AMBE+2 decoder's adaptive smoothing), never the plain WriteRawFrame
// fallback. This is the plumbing that drives the §6.x audio-quality parity
// for DMR.
func TestComposerDMRVoiceForwardsCorrectedBits(t *testing.T) {
	const (
		sampleRate  = 48_000.0
		sps         = 10
		span        = 8
		alpha       = 0.20
		deviation   = 1944.0
		superframes = 12
	)
	dibits, _ := buildVoiceStream(t, superframes)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &errRecordingSink{}
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

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 4*time.Second, func() bool {
		n, _ := sink.snapshot()
		return n >= dmrvoice.FramesPerSuperframe
	})

	n, plain := sink.snapshot()
	if n == 0 {
		t.Fatal("got no raw frames")
	}
	if plain {
		t.Error("at least one frame took the plain WriteRawFrame path; corrected-bit count was dropped")
	}
}
