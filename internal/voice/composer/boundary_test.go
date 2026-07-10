package composer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func (e *fakeEngine) endedCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.ended)
}

// mkBoundaryComposer builds a Composer wired to a fake engine + a real
// bus, with the given grouping mode and hangtime, for exercising the
// shared boundaryTracker directly (no IQ / chain).
func mkBoundaryComposer(t *testing.T, split bool, hangtime time.Duration) (*Composer, *events.Bus, *fakeEngine) {
	t.Helper()
	bus := events.NewBus(8)
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:                  bus,
		Devices:              &fakeDevices{},
		Sink:                 &recordingSink{},
		Engine:               eng,
		IQSampleRate:         2_400_000,
		PCMSampleRate:        8000,
		VoiceHangtime:        hangtime,
		SplitPerTransmission: split,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, bus, eng
}

// TestBoundaryTalkgroupGating: audio for the granted talkgroup is
// written, an unknown-TG frame (LDU2) inherits the last decision, and a
// sustained foreign talkgroup is dropped and ends the call.
func TestBoundaryTalkgroupGating(t *testing.T) {
	c, _, eng := mkBoundaryComposer(t, true, time.Second)
	bt := c.newBoundaryTracker("VOICE-1", 42, nil)

	if !bt.onVoice(42) {
		t.Error("matching talkgroup should be written")
	}
	if !bt.onVoice(0) {
		t.Error("tg=0 (LDU2) should inherit the matching decision")
	}
	if bt.onVoice(99) {
		t.Error("foreign talkgroup should not be written")
	}
	if bt.onVoice(99) {
		t.Error("foreign talkgroup should not be written")
	}
	if eng.endedCount() == 0 {
		t.Error("a sustained foreign talkgroup should end the call")
	}
}

// TestBoundaryGatingDisabledWithoutGrantTG: grantTG 0 (conventional /
// no in-band TG) writes everything and never ends on talkgroup.
func TestBoundaryGatingDisabledWithoutGrantTG(t *testing.T) {
	c, _, eng := mkBoundaryComposer(t, true, time.Second)
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)
	for _, tg := range []uint32{1, 2, 3, 0, 7} {
		if !bt.onVoice(tg) {
			t.Errorf("grantTG=0 should write all audio (tg=%d)", tg)
		}
	}
	if eng.endedCount() != 0 {
		t.Error("gating disabled should never end the call")
	}
}

// TestBoundaryPatchedTalkgroupMatches: a patched member talkgroup counts
// as a match.
func TestBoundaryPatchedTalkgroupMatches(t *testing.T) {
	c, _, eng := mkBoundaryComposer(t, true, time.Second)
	bt := c.newBoundaryTracker("VOICE-1", 42, []uint32{77})
	if !bt.onVoice(77) {
		t.Error("patched member talkgroup should be written")
	}
	if eng.endedCount() != 0 {
		t.Error("patched member must not end the call")
	}
}

// TestBoundaryMeasuresSignalDbFS: the tracker measures the mean received
// channel power of the IQ it observes and stamps it onto the engine (via
// UpdateSignal) at end-of-call, before EndCall. Constant-magnitude IQ of
// |0.1|² = 0.01 mean power ⇒ 10·log10(0.01) = −20 dBFS.
func TestBoundaryMeasuresSignalDbFS(t *testing.T) {
	c, _, eng := mkBoundaryComposer(t, false, 150*time.Millisecond)
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bt.run(ctx)

	chunk := make([]complex64, 2048)
	for i := range chunk {
		chunk[i] = complex(0.1, 0)
	}
	bt.observe(chunk)
	bt.onVoice(0) // one voice frame, then silence → hangtime ends the call

	waitFor(t, time.Second, func() bool {
		_, ok := eng.signal("VOICE-1")
		return ok
	})
	got, ok := eng.signal("VOICE-1")
	if !ok {
		t.Fatal("UpdateSignal was never called at end-of-call")
	}
	if got >= 0 {
		t.Errorf("SignalDbFS = %v, want negative (below digital full scale)", got)
	}
	if math.Abs(got+20) > 1 {
		t.Errorf("SignalDbFS = %v dBFS, want ~-20 (mean|iq|² = 0.01)", got)
	}
}

// TestBoundaryHangtimeEndsCall: after voice stops for longer than the
// hangtime, the tracker ends the call.
func TestBoundaryHangtimeEndsCall(t *testing.T) {
	c, _, eng := mkBoundaryComposer(t, true, 150*time.Millisecond)
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bt.run(ctx)
	bt.onVoice(0) // one voice frame, then silence
	waitFor(t, time.Second, func() bool { return eng.endedCount() > 0 })
}

// TestBoundaryNoVoiceWithinStartupNoEnd: a tracker that has not yet seen
// voice does not end the call before the no-voice startup window elapses —
// a healthy call gets the full window to acquire sync and deliver its
// first LDU.
func TestBoundaryNoVoiceWithinStartupNoEnd(t *testing.T) {
	// hangtime 1s ⇒ noVoiceTimeout 2s, so 300ms in nothing should end yet.
	c, _, eng := mkBoundaryComposer(t, true, time.Second)
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bt.run(ctx)
	time.Sleep(300 * time.Millisecond)
	if eng.endedCount() != 0 {
		t.Error("tracker must not end the call inside the no-voice startup window")
	}
}

// TestBoundaryNoVoiceEndsCall is the regression for the "hanging silent
// call": a tracker that NEVER decodes a matching voice frame must end the
// call once the no-voice startup window elapses (reason=timeout), instead
// of holding the voice tap for the full engine call-timeout watchdog.
func TestBoundaryNoVoiceEndsCall(t *testing.T) {
	// hangtime 50ms ⇒ noVoiceTimeout 100ms; never call onVoice.
	c, _, eng := mkBoundaryComposer(t, true, 50*time.Millisecond)
	bt := c.newBoundaryTracker("VOICE-1", 42, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bt.run(ctx)
	waitFor(t, time.Second, func() bool { return eng.endedCount() > 0 })
	if got := eng.endReason("VOICE-1"); got != trunking.EndReasonTimeout {
		t.Errorf("silent-from-start call end reason = %v, want EndReasonTimeout", got)
	}
}

func segmentCount(t *testing.T, sub *events.Subscription, wait time.Duration) int {
	t.Helper()
	n := 0
	deadline := time.After(wait)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallSegment {
				if _, ok := ev.Payload.(trunking.CallSegment); ok {
					n++
				}
			}
		case <-deadline:
			return n
		}
	}
}

// TestBoundarySplitEmitsSegment: in split mode, an end-of-transmission
// after audio publishes a KindCallSegment; with no audio it does not.
func TestBoundarySplitEmitsSegment(t *testing.T) {
	c, bus, _ := mkBoundaryComposer(t, true, time.Second)
	sub := bus.Subscribe()
	defer sub.Close()
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)

	// No audio yet → no segment.
	bt.onTransmissionEnd()
	// One over of audio → terminator rolls a segment.
	bt.onVoice(0)
	bt.onTransmissionEnd()
	// Immediate second terminator (no audio since) → no extra segment.
	bt.onTransmissionEnd()

	if got := segmentCount(t, sub, 300*time.Millisecond); got != 1 {
		t.Errorf("split mode segment events = %d, want 1", got)
	}
}

// TestBoundaryConversationNoSegment: in conversation mode an
// end-of-transmission never rolls a segment.
func TestBoundaryConversationNoSegment(t *testing.T) {
	c, bus, _ := mkBoundaryComposer(t, false, time.Second)
	sub := bus.Subscribe()
	defer sub.Close()
	bt := c.newBoundaryTracker("VOICE-1", 0, nil)
	bt.onVoice(0)
	bt.onTransmissionEnd()
	if got := segmentCount(t, sub, 300*time.Millisecond); got != 0 {
		t.Errorf("conversation mode segment events = %d, want 0", got)
	}
}
