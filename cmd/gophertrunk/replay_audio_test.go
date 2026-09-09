package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a goroutine-safe bytes.Buffer: the voice rig writes PCM from
// its own goroutines while the test inspects progress.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

// TestPCMStreamWriterEncodesS16LE pins the -audio-out wire format: raw signed
// 16-bit little-endian mono, sample order preserved, no framing.
func TestPCMStreamWriterEncodesS16LE(t *testing.T) {
	var buf bytes.Buffer
	w := newPCMStreamWriter(&buf)
	if err := w.WritePCM("any-serial", []int16{0x0102, -2, 0, 0x7FFF, -0x8000}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	want := []byte{
		0x02, 0x01, // 0x0102
		0xFE, 0xFF, // -2
		0x00, 0x00, // 0
		0xFF, 0x7F, // 32767
		0x00, 0x80, // -32768
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("encoded bytes = % X, want % X", buf.Bytes(), want)
	}
	// A second write appends — the stream is continuous, not per-call framed.
	if err := w.WritePCM("other-serial", []int16{1}); err != nil {
		t.Fatalf("WritePCM #2: %v", err)
	}
	if got := buf.Len(); got != len(want)+2 {
		t.Fatalf("stream length after second write = %d, want %d", got, len(want)+2)
	}
}

// countingErrWriter fails every write and counts attempts.
type countingErrWriter struct{ calls int }

func (e *countingErrWriter) Write(p []byte) (int, error) {
	e.calls++
	return 0, errors.New("downstream pipe gone")
}

// TestPCMStreamWriterErrorIsSticky pins the torn-pipe behaviour: after the
// first write error the sink stops touching the writer (the decode must not
// hammer a dead pipe) and keeps reporting the error.
func TestPCMStreamWriterErrorIsSticky(t *testing.T) {
	ew := &countingErrWriter{}
	w := newPCMStreamWriter(ew)
	if err := w.WritePCM("s", []int16{1, 2}); err == nil {
		t.Fatal("expected the first write to surface the writer error")
	}
	if err := w.WritePCM("s", []int16{3}); err == nil {
		t.Fatal("expected the sticky error on the second write")
	}
	if ew.calls != 1 {
		t.Fatalf("underlying writer called %d times, want 1 (sticky error must stop writes)", ew.calls)
	}
	if w.Err() == nil {
		t.Fatal("Err() should report the retained write error")
	}
}

// TestSetupReplayVoiceStreamOnly pins the -audio-out-without--record-voice
// shape: an empty outDir builds a decode-only rig (no recordings directory is
// required or created) wired to the live PCM stream.
func TestSetupReplayVoiceStreamOnly(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	var buf syncBuffer
	rig, err := setupReplayVoice("", 48000, 100*time.Millisecond, newPCMStreamWriter(&buf), log)
	if err != nil {
		t.Fatalf("setupReplayVoice with empty outDir (stream-only): %v", err)
	}
	rig.finalize()
}
