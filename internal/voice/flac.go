package voice

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// flacVoiceBlockSize is the FLAC frame block size for voice recordings —
// ~half a second at 8 kHz, the upstream encoder's conventional default.
const flacVoiceBlockSize = 4096

// FlacWriter writes 16-bit mono PCM to a lossless FLAC stream — the FLAC
// twin of WavWriter with the same surface (WriteSamples / DataBytes / Close),
// so the recorder can treat the two interchangeably. DataBytes reports the
// UNCOMPRESSED PCM payload bytes (2 per sample), keeping the recorder's
// duration math and empty-call checks format-independent; the on-disk file is
// smaller. Close flushes the last partial block and finalizes the stream
// (patching the STREAMINFO sample count + MD5 via a seek), so a daemon crash
// before Close leaves a header with a zero sample count — still parseable by
// players that stream frames, mirroring WavWriter's crash posture.
type FlacWriter struct {
	enc          *flac.Encoder
	closed       bool
	sampleRate   uint32
	bytesWritten uint32 // PCM payload bytes (2 × samples), pre-compression
	buf          []int32
	closeFn      func() error
}

// NewFlacWriter wraps an io.WriteSeeker and starts the FLAC stream.
func NewFlacWriter(w io.WriteSeeker, sampleRate uint32) (*FlacWriter, error) {
	if sampleRate == 0 {
		return nil, errors.New("voice: FLAC sample rate must be > 0")
	}
	info := &meta.StreamInfo{
		SampleRate:    sampleRate,
		NChannels:     1,
		BitsPerSample: 16,
	}
	enc, err := flac.NewEncoder(voiceWriteSeekerNoCloser{w}, info)
	if err != nil {
		return nil, fmt.Errorf("voice: init flac encoder: %w", err)
	}
	// Analyse each block for the best fixed/LPC predictor so the stream is
	// actually compressed rather than stored verbatim.
	enc.EnablePredictionAnalysis(true)
	return &FlacWriter{
		enc:        enc,
		sampleRate: sampleRate,
		buf:        make([]int32, 0, flacVoiceBlockSize),
	}, nil
}

// NewFlacFile opens path for write (creating or truncating) and returns a
// FlacWriter that closes the file on Close().
func NewFlacFile(path string, sampleRate uint32) (*FlacWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w, err := NewFlacWriter(f, sampleRate)
	if err != nil {
		f.Close()
		return nil, err
	}
	w.closeFn = f.Close
	return w, nil
}

// WriteSamples appends 16-bit PCM samples.
func (w *FlacWriter) WriteSamples(samples []int16) error {
	if w.closed {
		return errors.New("voice: FLAC writer is closed")
	}
	for _, s := range samples {
		w.buf = append(w.buf, int32(s))
		if len(w.buf) >= flacVoiceBlockSize {
			if err := w.flushBlock(); err != nil {
				return err
			}
		}
	}
	w.bytesWritten += uint32(2 * len(samples))
	return nil
}

// DataBytes returns the uncompressed PCM payload bytes written so far —
// WavWriter-compatible, so callers' duration/empty math is unchanged. Stays
// readable after Close.
func (w *FlacWriter) DataBytes() uint32 { return w.bytesWritten }

// Close flushes the pending block, finalizes the FLAC stream, and closes the
// underlying file (if the writer owns one).
func (w *FlacWriter) Close() error {
	// Defensive nil-receiver no-op, mirroring WavWriter.Close.
	if w == nil {
		return nil
	}
	if w.closed {
		return nil
	}
	w.closed = true
	err := w.flushBlock()
	if cerr := w.enc.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("voice: finalize flac: %w", cerr)
	}
	if w.closeFn != nil {
		if cerr := w.closeFn(); err == nil {
			err = cerr
		}
	}
	return err
}

// flushBlock encodes the pending samples as one mono FLAC frame. Subframes
// start verbatim; the encoder's prediction analysis upgrades them.
func (w *FlacWriter) flushBlock() error {
	n := len(w.buf)
	if n == 0 {
		return nil
	}
	hdr := frame.Header{
		HasFixedBlockSize: true,
		BlockSize:         uint16(n),
		SampleRate:        w.sampleRate,
		Channels:          frame.ChannelsMono,
		BitsPerSample:     16,
	}
	// Copy the pending samples: WriteFrame mutates in place and the slice is
	// reused for the next block.
	samples := make([]int32, n)
	copy(samples, w.buf)
	fr := &frame.Frame{
		Header: hdr,
		Subframes: []*frame.Subframe{
			{SubHeader: frame.SubHeader{Pred: frame.PredVerbatim}, Samples: samples, NSamples: n},
		},
	}
	if err := w.enc.WriteFrame(fr); err != nil {
		return fmt.Errorf("voice: encode flac frame: %w", err)
	}
	w.buf = w.buf[:0]
	return nil
}

// voiceWriteSeekerNoCloser hides an underlying Close from the FLAC encoder so
// enc.Close patches the STREAMINFO without closing the caller's file.
type voiceWriteSeekerNoCloser struct{ ws io.WriteSeeker }

func (w voiceWriteSeekerNoCloser) Write(p []byte) (int, error) { return w.ws.Write(p) }
func (w voiceWriteSeekerNoCloser) Seek(offset int64, whence int) (int64, error) {
	return w.ws.Seek(offset, whence)
}

// ReadFLACSamples reads a 16-bit mono FLAC recording written by FlacWriter
// (or any mono 16-bit FLAC) and returns its samples and sample rate — the
// FLAC counterpart of ReadWAVSamples, with the same mono/16-bit contract.
func ReadFLACSamples(path string) (samples []int16, sampleRate uint32, err error) {
	stream, err := flac.ParseFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("voice: open flac: %w", err)
	}
	defer stream.Close()
	if stream.Info.NChannels != 1 || stream.Info.BitsPerSample != 16 {
		return nil, 0, errors.New("voice: expected 16-bit mono FLAC")
	}
	sampleRate = stream.Info.SampleRate
	for {
		fr, ferr := stream.ParseNext()
		if ferr != nil {
			break // io.EOF, or a truncated tail (keep what decoded)
		}
		for _, s := range fr.Subframes[0].Samples {
			samples = append(samples, int16(s))
		}
	}
	return samples, sampleRate, nil
}

// ReadAudioSamples reads a voice recording (WAV or FLAC, sniffed from the
// file content rather than the extension) into 16-bit mono samples plus the
// sample rate — the format-agnostic entry every downstream consumer of a
// recording (normalization, MP3 transcode) should use.
func ReadAudioSamples(path string) ([]int16, uint32, error) {
	if isFLACAudioFile(path) {
		return ReadFLACSamples(path)
	}
	return ReadWAVSamples(path)
}

// isFLACAudioFile sniffs the 4-byte "fLaC" stream marker.
func isFLACAudioFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false
	}
	return string(magic[:]) == "fLaC"
}

// AudioFormatExt returns the recording filename extension (with dot) for a
// recordings format value ("" and "wav" → ".wav", "flac" → ".flac").
func AudioFormatExt(format string) string {
	if format == "flac" {
		return ".flac"
	}
	return ".wav"
}

// NewAudioFileWriter opens a voice recording at path in the given format —
// the recorder's single dispatch point between WavWriter and FlacWriter.
func NewAudioFileWriter(path string, sampleRate uint32, format string) (AudioFileWriter, error) {
	if format == "flac" {
		return NewFlacFile(path, sampleRate)
	}
	return NewWavFile(path, sampleRate)
}

// AudioFileWriter is the per-call recording writer contract WavWriter and
// FlacWriter share.
type AudioFileWriter interface {
	WriteSamples(samples []int16) error
	DataBytes() uint32
	Close() error
}
