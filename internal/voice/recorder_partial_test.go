package voice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// partialObs is a RecorderObserver that also counts partial recordings.
type partialObs struct{ partial atomic.Int32 }

func (*partialObs) RecordingCompleted(time.Duration) {}
func (*partialObs) RecordingDiscarded(string)        {}
func (*partialObs) RecordingError(string)            {}
func (*partialObs) RecordingTruncated()              {}
func (o *partialObs) RecordingPartial()              { o.partial.Add(1) }

// recordOneCall runs one drain-coordinated call of a few frames through a
// recorder, calling during() between the CallEnd and the drain signal (where
// the composer reports IQ loss), and returns the call.complete and the
// sidecar's iq_lost_chunks.
func recordOneCall(t *testing.T, obs RecorderObserver, during func(r *Recorder, cs trunking.CallStart)) (trunking.CallComplete, uint64) {
	t.Helper()
	DefaultRegistry.Register("loud-voc-partial", func() (Vocoder, error) { return loudVocoder{}, nil })
	bus := events.NewBus(16)
	t.Cleanup(bus.Close)
	sub := bus.Subscribe()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteCallJSON:      true,
		Observer:           obs,
		VocoderForProtocol: map[string]string{"test-drain": "loud-voc-partial"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r.EnableDrainCoordination()
	cs := drainCallStart()
	cs.Grant.CallID = 7
	r.handleStart(cs)
	frame := make([]byte, 11) // loudVocoder.FrameSize()
	for i := 0; i < 5; i++ {
		if err := r.WriteRawFrameForCall(cs.DeviceSerial, 7, frame, 0); err != nil {
			t.Fatal(err)
		}
	}
	r.handleEnd(drainCallEnd(cs))
	during(r, cs)
	r.NotifyDrainCompleteForCall(cs.DeviceSerial, 7)

	var cc trunking.CallComplete
	deadline := time.After(2 * time.Second)
	for cc.AudioPath == "" {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				cc = ev.Payload.(trunking.CallComplete)
			}
		case <-deadline:
			t.Fatal("no call.complete")
		}
	}
	b, err := os.ReadFile(strings.TrimSuffix(cc.AudioPath, filepath.Ext(cc.AudioPath)) + ".json")
	if err != nil {
		t.Fatalf("sidecar: %v", err)
	}
	var side map[string]any
	if err := json.Unmarshal(b, &side); err != nil {
		t.Fatal(err)
	}
	var lost uint64
	if v, ok := side["iq_lost_chunks"]; ok {
		lost = uint64(v.(float64))
	}
	return cc, lost
}

// TestRecorderMarksRecordingPartialOnIQLoss pins issue #1288's "no completed
// recording is misrepresented as lossless": a call whose IQ stream lost
// chunks is announced with IQLostChunks on its call.complete and in its
// sidecar, and is counted as partial.
func TestRecorderMarksRecordingPartialOnIQLoss(t *testing.T) {
	obs := &partialObs{}
	cc, sideLost := recordOneCall(t, obs, func(r *Recorder, cs trunking.CallStart) {
		r.NotifyIQLoss(cs.DeviceSerial, 7, 3)
		r.NotifyIQLoss(cs.DeviceSerial, 7, 2) // counts accumulate
	})
	if cc.IQLostChunks != 5 || !cc.Partial() {
		t.Fatalf("call.complete IQLostChunks = %d (partial %v), want 5", cc.IQLostChunks, cc.Partial())
	}
	if sideLost != 5 {
		t.Fatalf("sidecar iq_lost_chunks = %d, want 5", sideLost)
	}
	if n := obs.partial.Load(); n != 1 {
		t.Fatalf("RecordingPartial called %d times, want 1", n)
	}
}

// TestRecorderCompleteCallIsNotPartial: no loss reported ⇒ nothing marked,
// and the sidecar omits the field.
func TestRecorderCompleteCallIsNotPartial(t *testing.T) {
	obs := &partialObs{}
	cc, sideLost := recordOneCall(t, obs, func(*Recorder, trunking.CallStart) {})
	if cc.Partial() || sideLost != 0 || obs.partial.Load() != 0 {
		t.Fatalf("clean call marked partial: cc=%d sidecar=%d observer=%d", cc.IQLostChunks, sideLost, obs.partial.Load())
	}
}

// TestRecorderIgnoresIQLossForAnotherCall: a loss count naming a different
// call on the serial (a re-keyed serial's previous call) must not mark this
// recording.
func TestRecorderIgnoresIQLossForAnotherCall(t *testing.T) {
	cc, sideLost := recordOneCall(t, nil, func(r *Recorder, cs trunking.CallStart) {
		r.NotifyIQLoss(cs.DeviceSerial, 99, 4)
		r.NotifyIQLoss("OTHER-SERIAL", 7, 4)
	})
	if cc.Partial() || sideLost != 0 {
		t.Fatalf("another call's loss marked this recording: cc=%d sidecar=%d", cc.IQLostChunks, sideLost)
	}
}
