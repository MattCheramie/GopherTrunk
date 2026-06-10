package imbe

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestRecorderDecodesP25IntoWav: end-to-end sanity check of the
// live-pipeline wire-up. A CallStart with Grant.Protocol="p25"
// opens a recorder session that auto-instantiates the in-binary
// IMBE vocoder. WriteRawFrame on that session decodes each frame
// and appends the resulting PCM to the WAV. The recorder's own
// tests can't import imbe (cycle), so the integration assertion
// lives here in the imbe package.
func TestRecorderDecodesP25IntoWav(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	rec, err := voice.NewRecorder(voice.RecorderOptions{
		Bus:        bus,
		OutDir:     dir,
		SampleRate: 8000,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go rec.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System:   "S",
			Protocol: "p25", // → DefaultVocoderForProtocol → "imbe"
			GroupID:  100,
			SourceID: 200,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	// Wait for the recorder to open the session.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if rec.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !rec.HasSession("VOICE-1") {
		t.Fatal("session never opened")
	}

	// Feed N IMBE frames. Each frame is 11 bytes; goodVoiceFrame is a
	// valid non-idle voice frame (b₀ just outside the idle-tone corner)
	// so synthesis runs from the first frame instead of the onset mute.
	const n = 4
	for i := 0; i < n; i++ {
		if err := rec.WriteRawFrame("VOICE-1", goodVoiceFrame()); err != nil {
			t.Fatalf("WriteRawFrame: %v", err)
		}
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
		if !rec.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	wavPath := filepath.Join(dir, "S", "100", "20260505T120000Z_src200.wav")
	wavBytes, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatalf("read wav: %v", err)
	}

	// Each IMBE frame decodes to 160 16-bit samples; with N frames
	// the data chunk is N·160·2 bytes.
	wantData := uint32(n * 160 * 2)
	dataSize := binary.LittleEndian.Uint32(wavBytes[40:44])
	if dataSize != wantData {
		t.Errorf("WAV data chunk size = %d, want %d", dataSize, wantData)
	}
	if uint32(len(wavBytes)) != 44+wantData {
		t.Errorf("wav file size = %d, want %d", len(wavBytes), 44+wantData)
	}

	// Confirm at least one sample is non-zero — the IMBE pipeline
	// produces synthesis output for valid voice frames, and a fully
	// silent WAV would mean the vocoder didn't run.
	var nonZero bool
	for i := 44; i+1 < len(wavBytes); i += 2 {
		if int16(binary.LittleEndian.Uint16(wavBytes[i:i+2])) != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("wav has only silence samples; recorder→vocoder→wav path didn't fire")
	}
}
