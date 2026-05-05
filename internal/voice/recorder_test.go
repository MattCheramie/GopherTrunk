package voice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func mkRecorder(t *testing.T, writeRaw bool) (*Recorder, *events.Bus, string) {
	t.Helper()
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:        bus,
		OutDir:     dir,
		SampleRate: 8000,
		WriteRaw:   writeRaw,
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, bus, dir
}

func TestRecorderWritesPerCallWav(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System:   "TestSystem",
			Protocol: "p25",
			GroupID:  1234,
			SourceID: 56789,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP"},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 12, 30, 45, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	// Wait for the recorder to open the session.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := r.WritePCM("VOICE-1", make([]int16, 1600)); err != nil {
		t.Fatal(err)
	}

	end := trunking.CallEnd{
		Grant:        cs.Grant,
		Talkgroup:    cs.Talkgroup,
		DeviceSerial: "VOICE-1",
		StartedAt:    cs.StartedAt,
		EndedAt:      cs.StartedAt.Add(2 * time.Second),
		Reason:       trunking.EndReasonNormal,
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: end})

	// Wait for session to drain.
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	want := filepath.Join(dir, "TestSystem", "FIRE-DISP", "20260505T123045Z_src56789.wav")
	st, err := os.Stat(want)
	if err != nil {
		t.Fatalf("expected wav at %s: %v", want, err)
	}
	if st.Size() < int64(wavHeaderSize+1600*2) {
		t.Errorf("wav size = %d, want at least %d", st.Size(), wavHeaderSize+1600*2)
	}
}

func TestRecorderWritePCMDropsWithoutSession(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	if err := r.WritePCM("UNKNOWN", []int16{1, 2, 3}); err != nil {
		t.Errorf("WritePCM with no session should drop silently, got %v", err)
	}
}

func TestRecorderRawFrameSidecar(t *testing.T) {
	r, bus, dir := mkRecorder(t, true)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", GroupID: 99, SourceID: 7},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	frame := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42}
	if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})

	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	rawPath := filepath.Join(dir, "S", "99", "20260505T000000Z_src7.raw")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 2*len(frame) {
		t.Errorf("raw size = %d, want %d", len(data), 2*len(frame))
	}
}

func TestRecorderClosesOpenSessions(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{System: "S", GroupID: 1}, DeviceSerial: "X",
			StartedAt: time.Now().UTC(),
		},
	})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("X") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := r.SessionCount(); got != 0 {
		t.Errorf("sessions not drained: %d", got)
	}
}

func TestSanitize(t *testing.T) {
	// Dots are intentionally preserved (talkgroup names like "Site 1.2"
	// are common); slashes / spaces / shell metacharacters are mapped to
	// underscores. Empty input stays empty.
	cases := map[string]string{
		"FIRE-DISP":          "FIRE-DISP",
		"Fire / EMS":         "Fire___EMS",
		"  spaces  ":         "spaces",
		"path/../traversal":  "path_.._traversal",
		"":                   "",
	}
	for in, want := range cases {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
