package voice

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

type countingObserver struct {
	mu        sync.Mutex
	completed int
	discarded map[string]int
	errors    map[string]int
	truncated int
}

func (o *countingObserver) RecordingCompleted(time.Duration) {
	o.mu.Lock()
	o.completed++
	o.mu.Unlock()
}
func (o *countingObserver) RecordingDiscarded(reason string) {
	o.mu.Lock()
	if o.discarded == nil {
		o.discarded = map[string]int{}
	}
	o.discarded[reason]++
	o.mu.Unlock()
}
func (o *countingObserver) RecordingError(stage string) {
	o.mu.Lock()
	if o.errors == nil {
		o.errors = map[string]int{}
	}
	o.errors[stage]++
	o.mu.Unlock()
}
func (o *countingObserver) RecordingTruncated() {
	o.mu.Lock()
	o.truncated++
	o.mu.Unlock()
}

// runMinDurationCase records one call of `frames` loud 20 ms frames and
// reports whether a CallComplete was published and which files remain.
func runMinDurationCase(t *testing.T, minDur time.Duration, frames int, obs RecorderObserver) (completed bool, files []string) {
	t.Helper()
	DefaultRegistry.Register("loud-fake-voc", func() (Vocoder, error) { return loudFakeVocoder{}, nil })
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		WriteRaw:           true,
		MinDuration:        minDur,
		Observer:           obs,
		VocoderForProtocol: map[string]string{"p25-fake": "loud-fake-voc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "p25-fake", GroupID: 7},
		Talkgroup:    &trunking.TalkGroup{ID: 7, AlphaTag: "PD", Record: true},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitForSession(t, r, "VOICE-1", true)
	frame := make([]byte, 11)
	for i := 0; i < frames; i++ {
		if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
			t.Fatal(err)
		}
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, Talkgroup: cs.Talkgroup, DeviceSerial: "VOICE-1",
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(5 * time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitForSession(t, r, "VOICE-1", false)
	deadline := time.After(time.Second)
loop:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				completed = true
				break loop
			}
		case <-deadline:
			break loop
		}
	}
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	return completed, files
}

// A recording whose decoded audio is shorter than recordings.min_duration_ms
// is discarded (#1288 §5): files deleted, no CallComplete — even though the
// call's wall-clock span (5 s here) is far longer, because the threshold is
// measured on the audio.
func TestRecorderDiscardsRecordingShorterThanMinDuration(t *testing.T) {
	obs := &countingObserver{}
	// 5 frames × 20 ms = 100 ms of audio against a 500 ms minimum.
	completed, files := runMinDurationCase(t, 500*time.Millisecond, 5, obs)
	if completed {
		t.Fatal("short recording published a CallComplete; want it discarded")
	}
	if len(files) != 0 {
		t.Fatalf("short recording left files behind: %v", files)
	}
	if obs.discarded["too_short"] != 1 || obs.completed != 0 {
		t.Fatalf("observer = completed %d discarded %v, want 0 / too_short:1", obs.completed, obs.discarded)
	}
}

func TestRecorderKeepsRecordingAtOrAboveMinDuration(t *testing.T) {
	obs := &countingObserver{}
	// 30 frames × 20 ms = 600 ms ≥ 500 ms.
	completed, files := runMinDurationCase(t, 500*time.Millisecond, 30, obs)
	if !completed {
		t.Fatal("recording above the minimum published no CallComplete")
	}
	if len(files) == 0 {
		t.Fatal("recording above the minimum left no files")
	}
	if obs.completed != 1 || len(obs.discarded) != 0 {
		t.Fatalf("observer = completed %d discarded %v, want 1 / none", obs.completed, obs.discarded)
	}
}

// MinDuration 0 is the legacy behaviour: a 1-frame recording is kept, and a
// nil observer is safe.
func TestRecorderMinDurationZeroKeepsEverything(t *testing.T) {
	completed, files := runMinDurationCase(t, 0, 1, nil)
	if !completed || len(files) == 0 {
		t.Fatalf("min_duration 0 dropped a 1-frame recording (completed=%v files=%v)", completed, files)
	}
}
