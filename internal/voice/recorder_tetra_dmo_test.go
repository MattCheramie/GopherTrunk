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

// TestRecorderTETRADMOForcesRawSidecar: a "tetra-dmo" call carries the same
// post-FEC 137-bit TCH/S speech frames as "tetra", so it must get the same
// always-on .raw sidecar even with the recorder's global WriteRaw flag off.
// Failing-first: tetraVoiceProtocol used to match only "tetra", so DMO calls
// recorded a WAV with no speech-frame sidecar for out-of-band tools.
func TestRecorderTETRADMOForcesRawSidecar(t *testing.T) {
	r, bus, dir := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "DMO", Protocol: "tetra-dmo", GroupID: 0, SourceID: 0,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if r.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// An 18-byte buffer is the composer's packed 137-bit TCH/S frame shape;
	// the sidecar write is verbatim and independent of vocoder decode.
	frame := make([]byte, 18)
	for i := range frame {
		frame[i] = byte(i)
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

	rawPath := filepath.Join(dir, "DMO", "0", "20260505_010203_0.raw")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("expected .raw sidecar at %s: %v", rawPath, err)
	}
	if len(data) != len(frame) {
		t.Errorf("raw size = %d, want %d", len(data), len(frame))
	}
}
