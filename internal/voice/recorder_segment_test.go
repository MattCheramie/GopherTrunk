package voice

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRecorderFrequencyInFilename confirms the RF voice-channel
// frequency is tagged into the recording filename.
func TestRecorderFrequencyInFilename(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "S", Protocol: "fm", GroupID: 5, SourceID: 9,
			FrequencyHz: 451_287_500,
		},
		DeviceSerial: "V1",
		StartedAt:    time.Date(2026, 6, 8, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "V1", true)
	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "V1", StartedAt: cs.StartedAt,
		EndedAt: cs.StartedAt.Add(time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "V1", false)

	want := filepath.Join(dir, "S", "5", "20260608T010203Z_freq451287500_src9.wav")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected freq-tagged wav at %s: %v", want, err)
	}
}

// TestRecorderSegmentRollsToNewFile confirms a KindCallSegment finalizes
// the current recording (publishing CallComplete) and the next write
// opens a fresh file — one file per over — without ending the call.
func TestRecorderSegmentRollsToNewFile(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	sub := bus.Subscribe()
	defer sub.Close()

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "S", Protocol: "fm", GroupID: 5, SourceID: 9,
			FrequencyHz: 451_000_000,
		},
		DeviceSerial: "V1",
		StartedAt:    time.Date(2026, 6, 8, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "V1", true)

	// Over 1.
	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	// End of over 1 → roll.
	bus.Publish(events.Event{Kind: events.KindCallSegment, Payload: trunking.CallSegment{
		DeviceSerial: "V1", At: cs.StartedAt.Add(2 * time.Second),
	}})
	// Give the recorder a moment to park the dormant session, then over 2.
	time.Sleep(50 * time.Millisecond)
	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "V1", StartedAt: cs.StartedAt,
		EndedAt: cs.StartedAt.Add(5 * time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "V1", false)

	// Two distinct WAVs (one per over) must exist.
	entries, err := os.ReadDir(filepath.Join(dir, "S", "5"))
	if err != nil {
		t.Fatal(err)
	}
	wavs := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wav") {
			wavs++
		}
	}
	if wavs != 2 {
		t.Errorf("got %d wav files, want 2 (one per transmission)", wavs)
	}

	// The first over must have produced a CallComplete (so it streams).
	completes := 0
	deadline := time.After(300 * time.Millisecond)
loop:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallComplete {
				completes++
			}
		case <-deadline:
			break loop
		}
	}
	if completes < 2 {
		t.Errorf("got %d CallComplete events, want >= 2 (one per over)", completes)
	}
}

func waitSession(t *testing.T, r *Recorder, serial string, want bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if r.HasSession(serial) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for HasSession(%q) == %v", serial, want)
}
