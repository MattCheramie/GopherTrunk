package voice

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// WavWriter writes a 16-bit PCM mono WAV file. Length fields in the RIFF
// and data chunks are patched in Close() so that a daemon crash leaves a
// readable (if length-zero) file behind rather than something most media
// players reject.
//
// Construction takes any io.WriteSeeker (so tests can use bytes.Buffer
// wrapped in an in-memory seeker). The dedicated NewFile helper opens a
// regular file on disk.
type WavWriter struct {
	w            io.WriteSeeker
	closed       bool
	sampleRate   uint32
	bytesWritten uint32 // payload bytes (excluding header)
	closeFn      func() error
}

const (
	wavHeaderSize    = 44
	wavBitsPerSample = 16
	wavChannels      = 1
)

// NewWavWriter wraps an io.WriteSeeker and emits the WAV header. The
// sample-rate parameter is the PCM rate in Hz (8000 is typical for
// digital-radio voice).
func NewWavWriter(w io.WriteSeeker, sampleRate uint32) (*WavWriter, error) {
	if sampleRate == 0 {
		return nil, errors.New("voice: WAV sample rate must be > 0")
	}
	wr := &WavWriter{w: w, sampleRate: sampleRate}
	if err := wr.writeHeader(); err != nil {
		return nil, err
	}
	return wr, nil
}

// NewWavFile opens path for write (creating or truncating) and returns a
// WavWriter that closes the file on Close().
func NewWavFile(path string, sampleRate uint32) (*WavWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	wr, err := NewWavWriter(f, sampleRate)
	if err != nil {
		f.Close()
		return nil, err
	}
	wr.closeFn = f.Close
	return wr, nil
}

// WriteSamples appends 16-bit PCM samples (little-endian).
func (w *WavWriter) WriteSamples(samples []int16) error {
	if w.closed {
		return errors.New("voice: WAV writer is closed")
	}
	buf := make([]byte, 2*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(s))
	}
	n, err := w.w.Write(buf)
	w.bytesWritten += uint32(n)
	return err
}

// Close patches the length fields and closes the underlying file (if the
// writer owns one).
func (w *WavWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.patchHeader(); err != nil {
		// Don't suppress the close error; chain.
		if w.closeFn != nil {
			_ = w.closeFn()
		}
		return err
	}
	if w.closeFn != nil {
		return w.closeFn()
	}
	return nil
}

func (w *WavWriter) writeHeader() error {
	// Fields are patched in Close once the data length is known. We write
	// zero placeholders for now.
	header := make([]byte, wavHeaderSize)
	copy(header[0:4], "RIFF")
	// 4..8 file-size-minus-8 (patched)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(header[22:24], wavChannels)
	binary.LittleEndian.PutUint32(header[24:28], w.sampleRate)
	byteRate := w.sampleRate * wavChannels * wavBitsPerSample / 8
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	blockAlign := uint16(wavChannels * wavBitsPerSample / 8)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], wavBitsPerSample)
	copy(header[36:40], "data")
	// 40..44 data chunk size (patched)
	_, err := w.w.Write(header)
	return err
}

func (w *WavWriter) patchHeader() error {
	// RIFF size = total file - 8 = 36 + bytesWritten
	if _, err := w.w.Seek(4, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(36+w.bytesWritten)); err != nil {
		return err
	}
	if _, err := w.w.Seek(40, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, w.bytesWritten); err != nil {
		return err
	}
	// Best-effort seek back to end so subsequent writes (if any) append.
	_, _ = w.w.Seek(0, io.SeekEnd)
	return nil
}
