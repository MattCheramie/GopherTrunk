package voice

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// syncBuffer is a goroutine-safe log sink for the recorder's async Run loop.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestRecorderShortSpanDiagnostic pins the "recording shorter than call span"
// diagnostic: when a digital call decodes far fewer frames than its wall-clock
// span, the log line reports the frame count and the audio-yield percentage so
// the sparse-decode symptom (the TETRA short-recording report) is diagnosable
// from one line. 3 null-vocoder frames = 480 samples = 0.06 s of audio over a
// 5 s call → ~1.2% yield.
func TestRecorderShortSpanDiagnostic(t *testing.T) {
	logs := &syncBuffer{}
	handler := slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:                bus,
		OutDir:             dir,
		SampleRate:         8000,
		VocoderForProtocol: map[string]string{"test-null": "null"},
		Log:                slog.New(handler),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: "test-null", GroupID: 1, SourceID: 2},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitUntil(t, func() bool { return r.HasSession("VOICE-1") })

	// 3 frames of audio (0.06 s) …
	for i := 0; i < 3; i++ {
		if err := r.WriteRawFrame("VOICE-1", make([]byte, 11)); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
	}
	// … over a 5 s call span → the diagnostic must fire.
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(5 * time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})
	waitUntil(t, func() bool { return !r.HasSession("VOICE-1") })

	out := logs.String()
	line := findLine(out, "recording shorter than call span")
	if line == "" {
		t.Fatalf("diagnostic not logged; log was:\n%s", out)
	}
	for _, want := range []string{"frames=3", "audio_seconds=0.1", "wall_seconds=5", "audio_pct=1.2", "vocoder=null"} {
		if !strings.Contains(line, want) {
			t.Errorf("diagnostic line missing %q\nline: %s", want, line)
		}
	}
}

func waitUntil(t *testing.T, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func findLine(out, substr string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, substr) {
			return l
		}
	}
	return ""
}
