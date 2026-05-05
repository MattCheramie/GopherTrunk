package voice

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// memWriteSeeker is an in-memory implementation of io.WriteSeeker for tests.
type memWriteSeeker struct {
	buf []byte
	pos int64
}

func (m *memWriteSeeker) Write(p []byte) (int, error) {
	end := int(m.pos) + len(p)
	if end > len(m.buf) {
		m.buf = append(m.buf, make([]byte, end-len(m.buf))...)
	}
	n := copy(m.buf[m.pos:], p)
	m.pos += int64(n)
	return n, nil
}

func (m *memWriteSeeker) Seek(off int64, whence int) (int64, error) {
	var p int64
	switch whence {
	case io.SeekStart:
		p = off
	case io.SeekCurrent:
		p = m.pos + off
	case io.SeekEnd:
		p = int64(len(m.buf)) + off
	default:
		return 0, errors.New("bad whence")
	}
	if p < 0 {
		return 0, errors.New("negative position")
	}
	m.pos = p
	return p, nil
}

func TestWavHeaderShape(t *testing.T) {
	m := &memWriteSeeker{}
	w, err := NewWavWriter(m, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteSamples([]int16{1, -1, 2, -2}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m.buf[0:4], []byte("RIFF")) {
		t.Errorf("missing RIFF at 0..4")
	}
	if !bytes.Equal(m.buf[8:12], []byte("WAVE")) {
		t.Errorf("missing WAVE at 8..12")
	}
	if !bytes.Equal(m.buf[12:16], []byte("fmt ")) {
		t.Errorf("missing fmt at 12..16")
	}
	if !bytes.Equal(m.buf[36:40], []byte("data")) {
		t.Errorf("missing data at 36..40")
	}
	dataSize := binary.LittleEndian.Uint32(m.buf[40:44])
	if dataSize != 8 { // 4 samples * 2 bytes
		t.Errorf("data size = %d, want 8", dataSize)
	}
	riffSize := binary.LittleEndian.Uint32(m.buf[4:8])
	if riffSize != 36+8 {
		t.Errorf("RIFF size = %d, want 44", riffSize)
	}
	rate := binary.LittleEndian.Uint32(m.buf[24:28])
	if rate != 8000 {
		t.Errorf("sample rate = %d", rate)
	}
}

func TestWavSamplesAreLittleEndian(t *testing.T) {
	m := &memWriteSeeker{}
	w, _ := NewWavWriter(m, 8000)
	w.WriteSamples([]int16{0x4242, -0x4242})
	w.Close()
	got := m.buf[44:48]
	want := []byte{0x42, 0x42, 0xBE, 0xBD}
	if !bytes.Equal(got, want) {
		t.Errorf("samples = % X, want % X", got, want)
	}
}

func TestWavFileEndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.wav")
	w, err := NewWavFile(path, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteSamples(make([]int16, 1600)); err != nil { // 200 ms
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != int64(wavHeaderSize+1600*2) {
		t.Errorf("file size = %d, want %d", st.Size(), wavHeaderSize+1600*2)
	}
}

func TestWavCloseTwice(t *testing.T) {
	m := &memWriteSeeker{}
	w, _ := NewWavWriter(m, 8000)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("second Close should be a no-op, got %v", err)
	}
}

func TestWavRejectsZeroSampleRate(t *testing.T) {
	if _, err := NewWavWriter(&memWriteSeeker{}, 0); err == nil {
		t.Error("expected error for zero sample rate")
	}
}
