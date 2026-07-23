package acelp

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestRecorderDecodesTETRAIntoWav is the end-to-end acceptance check of the
// TETRA voice path from the on-air traffic-frame format to audible PCM. It
// exercises the exact chain the composer/recorder run for a real call, minus
// the RF/demod front end:
//
//	two 137-bit speech frames -> EncodeTCHS (TCH/S §5.5 channel coding)
//	  -> descrambled type-5 traffic frame -> TCHSpeechFrames (TCH/S decode +
//	  class-2 CRC gate) -> recorder (protocol "tetra" -> "tetra-acelp") ->
//	  ACELP vocoder -> non-silent 8 kHz WAV.
//
// A silent (or missing) WAV would mean the vocoder registration, protocol
// mapping, or decode path is broken. The recorder's own package can't import
// acelp (cycle), so this integration assertion lives here.
func TestRecorderDecodesTETRAIntoWav(t *testing.T) {
	// Build a handful of valid TCH/S traffic frames from realistic speech-frame
	// parameters (in-range quantiser indices), then TCH/S-decode them back to
	// the packed speech frames the vocoder consumes — the same bytes the
	// composer emits on a live call.
	const nTraffic = 6
	var speechFrames [][]byte
	for s := 0; s < nTraffic; s++ {
		a := prm2bits(makeFrame(s))
		b := prm2bits(makeFrame(s + 100))
		traffic := tetra.EncodeTCHS(a, b)
		frames := tetra.TCHSpeechFrames(traffic)
		if len(frames) != 2 {
			t.Fatalf("traffic %d: TCHSpeechFrames returned %d frames (CRC gate?)", s, len(frames))
		}
		speechFrames = append(speechFrames, frames...)
	}
	nFrames := len(speechFrames) // 2 per traffic frame

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
			Protocol: "tetra", // -> DefaultVocoderForProtocol -> "tetra-acelp"
			GroupID:  100,
			SourceID: 200,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if rec.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !rec.HasSession("VOICE-1") {
		t.Fatal("session never opened")
	}

	for i, f := range speechFrames {
		if err := rec.WriteRawFrame("VOICE-1", f); err != nil {
			t.Fatalf("WriteRawFrame %d: %v", i, err)
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
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !rec.HasSession("VOICE-1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Locate the produced WAV (walk the recordings dir rather than hard-coding
	// the layout).
	var wavPath string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".wav") {
			wavPath = p
		}
		return nil
	})
	if wavPath == "" {
		t.Fatal("no WAV produced by the TETRA -> ACELP recorder path")
	}
	wavBytes, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatalf("read wav: %v", err)
	}

	// Each speech frame decodes to 240 samples (30 ms). With an optional
	// end-of-call fade tail (<=10 ms = 80 samples) when the last sample is
	// non-zero, the data chunk is nFrames*240*2 (+ tail) bytes.
	voiceData := uint32(nFrames * frameLen * 2)
	const maxTailBytes = uint32(80 * 2)
	dataSize := binary.LittleEndian.Uint32(wavBytes[40:44])
	if dataSize < voiceData || dataSize > voiceData+maxTailBytes {
		t.Errorf("WAV data chunk = %d bytes, want %d..%d", dataSize, voiceData, voiceData+maxTailBytes)
	}

	// The decode must produce audible (non-silent) output.
	var nonZero bool
	for i := 44; i+1 < len(wavBytes); i += 2 {
		if int16(binary.LittleEndian.Uint16(wavBytes[i:i+2])) != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("WAV is all silence; TETRA -> TCH/S -> ACELP -> WAV path did not synthesise audio")
	}
}
