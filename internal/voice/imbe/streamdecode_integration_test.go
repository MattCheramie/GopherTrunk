package imbe

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// memWriteSeeker mirrors the shape used by voice's wav_test.go so
// imbe-package tests can exercise voice.DecodeStream without
// touching the filesystem. The voice package's memWriteSeeker is
// test-internal so we redefine here.
type memWriteSeeker struct {
	buf []byte
	pos int64
}

func (m *memWriteSeeker) Write(p []byte) (int, error) {
	end := m.pos + int64(len(p))
	if end > int64(len(m.buf)) {
		grown := make([]byte, end)
		copy(grown, m.buf)
		m.buf = grown
	}
	copy(m.buf[m.pos:], p)
	m.pos = end
	return len(p), nil
}

func (m *memWriteSeeker) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		m.pos = off
	case io.SeekCurrent:
		m.pos += off
	case io.SeekEnd:
		m.pos = int64(len(m.buf)) + off
	}
	return m.pos, nil
}

const wavHeaderSize = 44

// TestDecodeStreamThroughImbe: voice.DecodeStream with name "imbe"
// pulls the pure-Go IMBE decoder out of the registry and produces
// a playable WAV from a stream of synthetic IMBE frames. Pins the
// out-of-band ".raw → .wav" pipeline operators use to decode
// captured frames after a session.
func TestDecodeStreamThroughImbe(t *testing.T) {
	const n = 5
	frame := goodVoiceFrame() // valid non-idle voice frame
	in := bytes.NewReader(bytes.Repeat(frame, n))
	var sink memWriteSeeker
	frames, err := voice.DecodeStream(in, VocoderName, &sink)
	if err != nil {
		t.Fatalf("voice.DecodeStream: %v", err)
	}
	if frames != n {
		t.Errorf("frames = %d, want %d", frames, n)
	}
	// One IMBE frame at 8 kHz / 20 ms = 160 samples × 2 bytes per
	// sample. Plus the 44-byte WAV header.
	wantBytes := wavHeaderSize + n*160*2
	if len(sink.buf) != wantBytes {
		t.Errorf("wrote %d bytes, want %d", len(sink.buf), wantBytes)
	}
	// Confirm the WAV's data-chunk length matches.
	dataSize := binary.LittleEndian.Uint32(sink.buf[40:44])
	if dataSize != uint32(n*160*2) {
		t.Errorf("data chunk size = %d, want %d", dataSize, n*160*2)
	}
	// Confirm at least some samples are non-zero (synthesis ran).
	var nonZero int
	for i := wavHeaderSize; i+1 < len(sink.buf); i += 2 {
		if int16(binary.LittleEndian.Uint16(sink.buf[i:i+2])) != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("all samples zero; IMBE synthesis didn't run through DecodeStream")
	}
}

// TestDecodeStreamSilenceFrameThroughImbe: a single IMBE silence-
// window frame (b₀=216) decodes to all-zero PCM through the
// stream API. Mirrors the per-decoder TestDecodeReturnsSilenceForB0SilenceFrame.
func TestDecodeStreamSilenceFrameThroughImbe(t *testing.T) {
	frame := make([]byte, FrameBytes)
	frame[0] = 0xD8 // b₀ = 216 (silence window)
	var sink memWriteSeeker
	frames, err := voice.DecodeStream(bytes.NewReader(frame), VocoderName, &sink)
	if err != nil {
		t.Fatalf("voice.DecodeStream: %v", err)
	}
	if frames != 1 {
		t.Errorf("frames = %d, want 1", frames)
	}
	for i := wavHeaderSize; i+1 < len(sink.buf); i += 2 {
		if s := int16(binary.LittleEndian.Uint16(sink.buf[i : i+2])); s != 0 {
			t.Fatalf("sample[%d] = %d, want 0 (silence frame)",
				(i-wavHeaderSize)/2, s)
		}
	}
}
