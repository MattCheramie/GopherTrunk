package voice

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// rawFrameCount returns the number of fixed-size vocoder frames written to the
// single .raw sidecar under dir (frameSize bytes each).
func rawFrameCount(t *testing.T, dir string, frameSize int) int {
	t.Helper()
	var raws []string
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(p) == ".raw" {
			raws = append(raws, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(raws) != 1 {
		t.Fatalf("want exactly one .raw sidecar, found %d: %v", len(raws), raws)
	}
	fi, err := os.Stat(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size()%int64(frameSize) != 0 {
		t.Fatalf(".raw size %d not a multiple of frame size %d", fi.Size(), frameSize)
	}
	return int(fi.Size() / int64(frameSize))
}

func newDrainRecorder(t *testing.T) (*Recorder, string) {
	t.Helper()
	DefaultRegistry.Register("loud-voc-drain", func() (Vocoder, error) { return loudVocoder{}, nil })
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true,
		VocoderForProtocol: map[string]string{"test-drain": "loud-voc-drain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, dir
}

func drainCallStart() trunking.CallStart {
	return trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-drain", GroupID: 1, SourceID: 3},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC),
	}
}

func drainCallEnd(cs trunking.CallStart) trunking.CallEnd {
	return trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: cs.DeviceSerial,
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}
}

// TestRecorderDrainCoordinationKeepsTailFrames is the regression test for the
// dropped trailing-frames race: with drain coordination enabled, the recorder
// must keep the session alive after KindCallEnd until the composer signals the
// voice chain has drained, so the tail frames the chain writes during teardown
// (here, after handleEnd runs) are recorded rather than dropped onto a deleted
// session. Drives the recorder's methods directly to force the exact ordering
// deterministically (handleEnd fully processed BEFORE the tail is written).
func TestRecorderDrainCoordinationKeepsTailFrames(t *testing.T) {
	r, dir := newDrainRecorder(t)
	const frameSize = 11 // loudVocoder.FrameSize()
	frame := make([]byte, frameSize)

	r.EnableDrainCoordination()
	cs := drainCallStart()
	r.handleStart(cs)

	const bodyFrames, tailFrames = 5, 3
	for i := 0; i < bodyFrames; i++ {
		if err := r.WriteRawFrame(cs.DeviceSerial, frame); err != nil {
			t.Fatalf("WriteRawFrame body: %v", err)
		}
	}

	// Call end arrives and is fully processed. Without drain coordination the
	// old recorder deleted the session here; with it, the session must survive.
	r.handleEnd(drainCallEnd(cs))
	if !r.HasSession(cs.DeviceSerial) {
		t.Fatal("session was finalized on CallEnd; tail frames from the chain drain would be dropped (the bug)")
	}

	// The composer's chain drain writes the transmission tail AFTER CallEnd.
	for i := 0; i < tailFrames; i++ {
		if err := r.WriteRawFrame(cs.DeviceSerial, frame); err != nil {
			t.Fatalf("WriteRawFrame tail: %v", err)
		}
	}

	// Drain complete: the recorder finalizes now.
	r.NotifyDrainComplete(cs.DeviceSerial)
	if r.HasSession(cs.DeviceSerial) {
		t.Fatal("session not finalized after NotifyDrainComplete")
	}

	if got, want := rawFrameCount(t, dir, frameSize), bodyFrames+tailFrames; got != want {
		t.Fatalf("recorded %d frames, want %d (body %d + tail %d) — trailing frames dropped",
			got, want, bodyFrames, tailFrames)
	}
}

// TestRecorderDrainSignalBeforeCallEnd covers the reverse arrival order:
// NotifyDrainComplete arriving before handleEnd must still finalize exactly once
// (when the CallEnd then arrives), not leak the session or double-finalize.
func TestRecorderDrainSignalBeforeCallEnd(t *testing.T) {
	r, dir := newDrainRecorder(t)
	const frameSize = 11
	frame := make([]byte, frameSize)

	r.EnableDrainCoordination()
	cs := drainCallStart()
	r.handleStart(cs)
	for i := 0; i < 4; i++ {
		if err := r.WriteRawFrame(cs.DeviceSerial, frame); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}

	// Drain signal wins the race with the recorder's own CallEnd handling.
	r.NotifyDrainComplete(cs.DeviceSerial)
	if !r.HasSession(cs.DeviceSerial) {
		t.Fatal("session finalized on the drain signal alone, before CallEnd arrived")
	}
	r.handleEnd(drainCallEnd(cs))
	if r.HasSession(cs.DeviceSerial) {
		t.Fatal("session not finalized after both drain signal and CallEnd arrived")
	}
	if got := rawFrameCount(t, dir, frameSize); got != 4 {
		t.Fatalf("recorded %d frames, want 4", got)
	}
}

// TestRecorderDrainTimeoutFinalizes covers the safety backstop: if the composer
// never signals (e.g. the bus dropped its KindCallEnd under backpressure), the
// deferred call must still finalize when the timeout path runs, rather than
// leaking the session forever.
func TestRecorderDrainTimeoutFinalizes(t *testing.T) {
	r, dir := newDrainRecorder(t)
	const frameSize = 11
	frame := make([]byte, frameSize)

	r.EnableDrainCoordination()
	cs := drainCallStart()
	r.handleStart(cs)
	for i := 0; i < 2; i++ {
		if err := r.WriteRawFrame(cs.DeviceSerial, frame); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}
	r.handleEnd(drainCallEnd(cs)) // defers, arming the safety timer
	if !r.HasSession(cs.DeviceSerial) {
		t.Fatal("session finalized immediately instead of deferring")
	}
	// Simulate the safety timer firing (the drain signal never came).
	r.finalizeOnDrainTimeout(cs.DeviceSerial)
	if r.HasSession(cs.DeviceSerial) {
		t.Fatal("timeout backstop did not finalize the deferred call")
	}
	if got := rawFrameCount(t, dir, frameSize); got != 2 {
		t.Fatalf("recorded %d frames, want 2", got)
	}
}

// TestRecorderUncoordinatedFinalizesOnCallEnd pins that without drain
// coordination the recorder finalizes on CallEnd exactly as before (standalone
// use / non-composer callers must be unaffected).
func TestRecorderUncoordinatedFinalizesOnCallEnd(t *testing.T) {
	r, _ := newDrainRecorder(t)
	const frameSize = 11
	frame := make([]byte, frameSize)

	cs := drainCallStart()
	r.handleStart(cs)
	if err := r.WriteRawFrame(cs.DeviceSerial, frame); err != nil {
		t.Fatalf("WriteRawFrame: %v", err)
	}
	r.handleEnd(drainCallEnd(cs))
	if r.HasSession(cs.DeviceSerial) {
		t.Fatal("uncoordinated recorder should finalize immediately on CallEnd")
	}
}

// rawFrameCounts returns the frame count of every .raw sidecar under dir,
// sorted by path (one per recorded transmission).
func rawFrameCounts(t *testing.T, dir string, frameSize int) []int {
	t.Helper()
	var raws []string
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(p) == ".raw" {
			raws = append(raws, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(raws)
	out := make([]int, 0, len(raws))
	for _, p := range raws {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Size()%int64(frameSize) != 0 {
			t.Fatalf("%s: size %d not a multiple of frame size %d", p, fi.Size(), frameSize)
		}
		out = append(out, int(fi.Size()/int64(frameSize)))
	}
	return out
}

// TestRecorderRekeyDuringPendingDrainKeepsNextOver is the 17 Sep IPSC
// regression. A conventional-DMR re-key ends the previous call and grants the
// next over in the same instant, so with drain coordination the recorder can
// see the next CallStart while the previous call's CallEnd is still deferred
// waiting for the composer's drain signal. The old recorder then (1) replaced
// the previous session without finalizing it — no CallComplete, no sidecar,
// no history row — and (2) let the previous call's late drain signal finalize
// the NEW session before its first frame (silently: files open lazily), so
// every frame of the next over was dropped and its own CallEnd found nothing.
// The field log: "recorder: device already has session, replacing" followed
// by two recordings that never logged an end. Both overs must be recorded and
// completed. Fails against the old recorder (one .raw, one CallComplete).
func TestRecorderRekeyDuringPendingDrainKeepsNextOver(t *testing.T) {
	DefaultRegistry.Register("loud-voc-drain", func() (Vocoder, error) { return loudVocoder{}, nil })
	bus := events.NewBus(16)
	t.Cleanup(bus.Close)
	sub := bus.Subscribe()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true,
		VocoderForProtocol: map[string]string{"test-drain": "loud-voc-drain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r.EnableDrainCoordination()
	const frameSize = 11 // loudVocoder.FrameSize()
	frame := make([]byte, frameSize)

	// Over A: 5 frames, then its CallEnd (deferred for the drain signal).
	csA := drainCallStart()
	csA.Grant.CallID = 1
	r.handleStart(csA)
	for i := 0; i < 5; i++ {
		if err := r.WriteRawFrameForCall(csA.DeviceSerial, 1, frame, 0); err != nil {
			t.Fatal(err)
		}
	}
	r.handleEnd(drainCallEnd(csA))

	// The re-key: over B's CallStart on the same serial lands before A's chain
	// has drained.
	csB := drainCallStart()
	csB.Grant.CallID = 2
	csB.StartedAt = csA.StartedAt.Add(2 * time.Second)
	r.handleStart(csB)
	if !r.HasSession(csB.DeviceSerial) {
		t.Fatal("no session for the next over after the re-key")
	}

	// A's chain finishes draining now — after B is already open.
	r.NotifyDrainCompleteForCall(csA.DeviceSerial, 1)
	if !r.HasSession(csB.DeviceSerial) {
		t.Fatal("the previous call's drain signal finalized the next over's session (the bug)")
	}

	// Over B: 4 frames, then its own end + drain.
	for i := 0; i < 4; i++ {
		if err := r.WriteRawFrameForCall(csB.DeviceSerial, 2, frame, 0); err != nil {
			t.Fatal(err)
		}
	}
	r.handleEnd(drainCallEnd(csB))
	r.NotifyDrainCompleteForCall(csB.DeviceSerial, 2)
	if r.HasSession(csB.DeviceSerial) {
		t.Fatal("next over not finalized after its own CallEnd + drain")
	}

	if got := rawFrameCounts(t, dir, frameSize); len(got) != 2 || got[0] != 5 || got[1] != 4 {
		t.Fatalf("recorded transmissions (frames per .raw) = %v, want [5 4]: a re-key must record both overs", got)
	}
	completes := 0
	deadline := time.After(2 * time.Second)
	for completes < 2 {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				completes++
			}
		case <-deadline:
			t.Fatalf("saw %d KindCallComplete events, want 2 (the previous call was replaced without being finalized)", completes)
		}
	}
}
