package phase1_test

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// TestLDUEndToEndIntoRecorder is the load-bearing integration
// test for the IQ→vocoder-frame→audio pipeline. We build a
// synthetic 1728-bit LDU containing 9 valid IMBE voice
// subframes, hand it to ExtractVoiceFrames, feed the resulting
// frames into a recorder that's been configured with
// Protocol="p25" (auto-instantiates the pure-Go imbe vocoder),
// and confirm the WAV ends up with the expected payload size +
// non-zero samples.
//
// The pipeline under test:
//
//	[synth 9 IMBE info bits]
//	  → imbe.EncodeChannel + Scramble → 144 ch bits per subframe
//	  → place at lduVoiceOffsets in LDU payload
//	  → phase1.InjectStatusSymbols → 1728-bit on-air LDU
//	  ↓
//	  → phase1.ExtractVoiceFrames → [9] 11-byte recorder frames
//	  → recorder.WriteRawFrame for each
//	  ↓
//	  → voice/imbe.Decoder.Decode (auto-instantiated per
//	    Protocol="p25" via voice.DefaultVocoderForProtocol)
//	  → WAV samples written, header patched on call end.
//
// Lives in an _test package so it can import imbe, voice, and
// phase1 simultaneously without creating the import cycles the
// internal imbe / voice packages avoid.
func TestLDUEndToEndIntoRecorder(t *testing.T) {
	// Build 9 synthetic IMBE info-bit patterns, one per voice
	// subframe.
	var infos [phase1.LDUVoiceSubframeCount][]byte
	for i := 0; i < phase1.LDUVoiceSubframeCount; i++ {
		bits := make([]byte, imbe.InfoBits)
		for k := range bits {
			bits[k] = byte((i*11 + k*3) % 2)
		}
		infos[i] = bits
	}

	payload := make([]byte, phase1.LDUPayloadBits)
	for i := 0; i < phase1.LDUVoiceSubframeCount; i++ {
		encoded, err := imbe.EncodeChannel(infos[i])
		if err != nil {
			t.Fatalf("EncodeChannel u_%d: %v", i, err)
		}
		scrambled, err := imbe.Scramble(encoded)
		if err != nil {
			t.Fatalf("Scramble u_%d: %v", i, err)
		}
		// LDU voice slot offset comes from the package's
		// internal table; we use the public sequence-builder
		// approach below to avoid exposing the array.
		off := lduVoiceOffsetForTest(i)
		copy(payload[off:off+phase1.LDUVoiceSubframeBits], scrambled)
	}
	var status [phase1.LDUStatusSymbolCount]uint8
	ldu, err := phase1.InjectStatusSymbols(payload, status)
	if err != nil {
		t.Fatalf("InjectStatusSymbols: %v", err)
	}

	frames, errs, err := phase1.ExtractVoiceFrames(ldu)
	if err != nil {
		t.Fatalf("ExtractVoiceFrames: %v", err)
	}
	if errs != 0 {
		t.Errorf("ExtractVoiceFrames errs = %d, want 0 on clean LDU", errs)
	}

	// Wire a recorder that maps "p25" → "imbe" (the default mapping).
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
			System: "S", Protocol: "p25", GroupID: 7, SourceID: 42,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
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

	// Push all 9 extracted frames through the recorder.
	for i, frame := range frames {
		if err := rec.WriteRawFrame("VOICE-1", frame); err != nil {
			t.Fatalf("WriteRawFrame u_%d: %v", i, err)
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

	wavPath := filepath.Join(dir, "S", "7", "20260505T000000Z_src42.wav")
	wavBytes, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatalf("read wav: %v", err)
	}
	const wavHeaderSize = 44
	// 9 frames × 160 samples × 2 bytes per sample.
	wantData := uint32(phase1.LDUVoiceSubframeCount * 160 * 2)
	dataSize := binary.LittleEndian.Uint32(wavBytes[40:44])
	if dataSize != wantData {
		t.Errorf("WAV data chunk size = %d, want %d", dataSize, wantData)
	}
	if uint32(len(wavBytes)) != wavHeaderSize+wantData {
		t.Errorf("wav file size = %d, want %d",
			len(wavBytes), wavHeaderSize+wantData)
	}
	// Confirm at least one sample is non-zero — synthesis ran end-to-end.
	var nonZero bool
	for i := wavHeaderSize; i+1 < len(wavBytes); i += 2 {
		if int16(binary.LittleEndian.Uint16(wavBytes[i:i+2])) != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("WAV has only silence samples; LDU → IMBE → WAV path didn't fire")
	}
}

// lduVoiceOffsetForTest mirrors the package's internal
// lduVoiceOffsets table — needed because we're in an _test
// package and the internal table is unexported. Keep in sync
// with ldu.go's lduVoiceOffsets.
func lduVoiceOffsetForTest(i int) int {
	return []int{112, 256, 440, 624, 808, 992, 1176, 1360, 1520}[i]
}
