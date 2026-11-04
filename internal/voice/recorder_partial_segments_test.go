package voice

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func newPartialRecorder(t *testing.T, obs RecorderObserver) (*Recorder, *events.Subscription) {
	t.Helper()
	DefaultRegistry.Register("loud-voc-partial", func() (Vocoder, error) { return loudVocoder{}, nil })
	bus := events.NewBus(32)
	t.Cleanup(bus.Close)
	sub := bus.Subscribe()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             t.TempDir(),
		SampleRate:         8000,
		WriteCallJSON:      true,
		Observer:           obs,
		VocoderForProtocol: map[string]string{"test-drain": "loud-voc-partial"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r.EnableDrainCoordination()
	return r, sub
}

func writeFrames(t *testing.T, r *Recorder, serial string, callID uint64, n int) {
	t.Helper()
	frame := make([]byte, 11) // loudVocoder.FrameSize()
	for i := 0; i < n; i++ {
		if err := r.WriteRawFrameForCall(serial, callID, frame, 0); err != nil {
			t.Fatal(err)
		}
	}
}

func nextComplete(t *testing.T, sub *events.Subscription) trunking.CallComplete {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				return ev.Payload.(trunking.CallComplete)
			}
		case <-deadline:
			t.Fatal("no call.complete")
		}
	}
}

// TestRecorderMarksEachSegmentWithItsOwnIQLoss pins the #1288 rule for
// per-transmission grouping: every file of a multi-over call says whether ITS
// audio has gaps. Before, the call's whole loss was reported once at the end,
// so an earlier over that lost IQ was announced (and uploaded, and
// transcribed) as complete, and the final over was charged with loss it never
// had.
func TestRecorderMarksEachSegmentWithItsOwnIQLoss(t *testing.T) {
	obs := &partialObs{}
	r, sub := newPartialRecorder(t, obs)
	cs := drainCallStart()
	cs.Grant.CallID = 7
	var lost atomic.Uint64
	r.handleStart(cs)
	r.TrackIQLoss(cs.DeviceSerial, 7, lost.Load)

	writeFrames(t, r, cs.DeviceSerial, 7, 5)
	lost.Store(2) // the first over lost two chunks
	r.handleSegment(trunking.CallSegment{DeviceSerial: cs.DeviceSerial, At: cs.StartedAt.Add(500 * time.Millisecond)})
	first := nextComplete(t, sub)

	writeFrames(t, r, cs.DeviceSerial, 7, 5) // the second over: clean
	r.handleSegment(trunking.CallSegment{DeviceSerial: cs.DeviceSerial, At: cs.StartedAt.Add(800 * time.Millisecond)})
	second := nextComplete(t, sub)

	writeFrames(t, r, cs.DeviceSerial, 7, 5)
	lost.Store(5) // the third over lost three more
	r.handleEnd(drainCallEnd(cs))
	r.NotifyDrainCompleteForCall(cs.DeviceSerial, 7)
	third := nextComplete(t, sub)

	got := []uint64{first.IQLostChunks, second.IQLostChunks, third.IQLostChunks}
	if got[0] != 2 || got[1] != 0 || got[2] != 3 {
		t.Fatalf("per-file IQLostChunks = %v (segments %d/%d/%d), want [2 0 3]",
			got, first.Segment, second.Segment, third.Segment)
	}
	if n := obs.partial.Load(); n != 2 {
		t.Fatalf("RecordingPartial counted %d files, want 2", n)
	}
	r.mu.Lock()
	left := len(r.lossTrack)
	r.mu.Unlock()
	if left != 0 {
		t.Fatalf("%d loss trackers left after the call ended", left)
	}
}

// TestRecorderLossTrackerFollowsItsCall: a re-key reuses the serial; the old
// call's late CallEnd must neither read nor drop the new call's tracker, and
// a tracker registered for another call id never marks this one.
func TestRecorderLossTrackerFollowsItsCall(t *testing.T) {
	r, sub := newPartialRecorder(t, nil)
	cs := drainCallStart()
	cs.Grant.CallID = 7
	var other atomic.Uint64
	other.Store(9)
	r.handleStart(cs)
	r.TrackIQLoss(cs.DeviceSerial, 99, other.Load) // another call on this serial
	r.TrackIQLoss("OTHER-SERIAL", 7, other.Load)
	writeFrames(t, r, cs.DeviceSerial, 7, 5)
	r.handleEnd(drainCallEnd(cs))
	r.NotifyDrainCompleteForCall(cs.DeviceSerial, 7)
	if cc := nextComplete(t, sub); cc.Partial() {
		t.Fatalf("another call's loss marked this recording: %d", cc.IQLostChunks)
	}
	r.mu.Lock()
	_, kept99 := r.lossTrack[iqLossKey{cs.DeviceSerial, 99}]
	_, keptOther := r.lossTrack[iqLossKey{"OTHER-SERIAL", 7}]
	r.mu.Unlock()
	if !kept99 || !keptOther {
		t.Fatalf("call 7's end dropped other calls' trackers (99 kept %v, other serial kept %v)", kept99, keptOther)
	}
}
