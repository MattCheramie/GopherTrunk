package baseband

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// flacIQBlockSize matches siglab's IQ container block size so both FLAC IQ
// writers produce identically-framed streams.
const flacIQBlockSize = 4096

// FLACIQEncoder losslessly encodes 16-bit I/Q pairs into a two-channel FLAC
// stream (I in the left channel, Q in the right) — the FLAC twin of the
// two-channel 16-bit RIFF/WAVE layout IQWriter emits, typically 30–50%
// smaller thanks to linear prediction on the continuous baseband waveform.
//
// It is the shared encode core for every FLAC IQ writer in the tree: the
// file-owning FLACIQWriter below and siglab's IQContainer both feed it int16
// pairs, so the two paths cannot drift apart. Callers do their own
// float→int16 conversion (the packages differ deliberately: baseband scales
// ×32767, siglab ×32768) and must call Finalize once — it flushes the last
// partial block and patches the STREAMINFO (sample count + MD5) via a seek,
// but never closes the underlying writer.
type FLACIQEncoder struct {
	enc   *flac.Encoder
	left  []int32
	right []int32
}

// NewFLACIQEncoder starts a two-channel 16-bit FLAC stream on ws at the given
// IQ sample rate. ws must be positioned at offset 0 of a fresh stream.
func NewFLACIQEncoder(ws io.WriteSeeker, sampleRate uint32) (*FLACIQEncoder, error) {
	if sampleRate == 0 {
		return nil, errors.New("baseband: FLAC IQ sample rate must be > 0")
	}
	info := &meta.StreamInfo{
		SampleRate:    sampleRate,
		NChannels:     2,
		BitsPerSample: 16,
	}
	enc, err := flac.NewEncoder(nopCloserWriteSeeker{ws}, info)
	if err != nil {
		return nil, fmt.Errorf("baseband: init flac encoder: %w", err)
	}
	// Analyse each block for the best fixed/LPC predictor so the stream is
	// actually compressed rather than stored verbatim.
	enc.EnablePredictionAnalysis(true)
	return &FLACIQEncoder{
		enc:   enc,
		left:  make([]int32, 0, flacIQBlockSize),
		right: make([]int32, 0, flacIQBlockSize),
	}, nil
}

// WriteI16 appends one I/Q sample pair, flushing a FLAC frame whenever a full
// block has accumulated.
func (e *FLACIQEncoder) WriteI16(i, q int16) error {
	e.left = append(e.left, int32(i))
	e.right = append(e.right, int32(q))
	if len(e.left) >= flacIQBlockSize {
		return e.flushBlock()
	}
	return nil
}

// Finalize flushes the pending partial block and closes the FLAC stream,
// patching the STREAMINFO via a seek on the underlying writer. It does NOT
// close the writer — the caller owns its lifetime.
func (e *FLACIQEncoder) Finalize() error {
	if len(e.left) > 0 {
		if err := e.flushBlock(); err != nil {
			return err
		}
	}
	if err := e.enc.Close(); err != nil {
		return fmt.Errorf("baseband: finalize flac: %w", err)
	}
	return nil
}

// flushBlock encodes the pending L/R samples as one FLAC frame and resets the
// block buffers. Subframes start verbatim; the encoder's prediction analysis
// upgrades them to fixed/LPC predictors.
func (e *FLACIQEncoder) flushBlock() error {
	n := len(e.left)
	if n == 0 {
		return nil
	}
	hdr := frame.Header{
		HasFixedBlockSize: true,
		BlockSize:         uint16(n),
		SampleRate:        e.enc.Info.SampleRate,
		Channels:          frame.ChannelsLR,
		BitsPerSample:     16,
	}
	// Copy the pending samples: WriteFrame decorrelates in place and reverts,
	// but the slices are reused for the next block, so hand it fresh backing.
	left := make([]int32, n)
	right := make([]int32, n)
	copy(left, e.left)
	copy(right, e.right)
	fr := &frame.Frame{
		Header: hdr,
		Subframes: []*frame.Subframe{
			{SubHeader: frame.SubHeader{Pred: frame.PredVerbatim}, Samples: left, NSamples: n},
			{SubHeader: frame.SubHeader{Pred: frame.PredVerbatim}, Samples: right, NSamples: n},
		},
	}
	if err := e.enc.WriteFrame(fr); err != nil {
		return fmt.Errorf("baseband: encode flac frame: %w", err)
	}
	e.left = e.left[:0]
	e.right = e.right[:0]
	return nil
}

// nopCloserWriteSeeker hides an underlying Close from the FLAC encoder so
// enc.Close patches the STREAMINFO without closing the caller's file.
type nopCloserWriteSeeker struct{ ws io.WriteSeeker }

func (w nopCloserWriteSeeker) Write(p []byte) (int, error) { return w.ws.Write(p) }
func (w nopCloserWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	return w.ws.Seek(offset, whence)
}

// FLACIQWriter is the FLAC twin of IQWriter: it streams complex64 IQ to a
// two-channel 16-bit FLAC file with the same clamp/scale (floatToI16) as the
// WAV writer, so a flac baseband recording decodes to the same int16 samples
// a wav one carries.
type FLACIQWriter struct {
	f            *os.File
	enc          *FLACIQEncoder
	bytesWritten uint32 // PCM payload bytes (pre-compression), mirroring IQWriter
	closed       bool
}

// NewFLACIQWriter creates (or truncates) path and starts the FLAC stream.
func NewFLACIQWriter(path string, sampleRate uint32) (*FLACIQWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	enc, err := NewFLACIQEncoder(f, sampleRate)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &FLACIQWriter{f: f, enc: enc}, nil
}

// Write appends a block of IQ samples, clamped and scaled exactly like
// IQWriter.Write.
func (w *FLACIQWriter) Write(samples []complex64) error {
	if w.closed {
		return errors.New("baseband: FLAC IQ writer is closed")
	}
	for _, s := range samples {
		if err := w.enc.WriteI16(floatToI16(real(s)), floatToI16(imag(s))); err != nil {
			return err
		}
	}
	w.bytesWritten += uint32(iqWavBlockAlign * len(samples))
	return nil
}

// BytesWritten reports the uncompressed IQ payload bytes written so far
// (int16 pairs, i.e. what the equivalent WAV body would hold) — the
// compressed on-disk size is smaller.
func (w *FLACIQWriter) BytesWritten() uint32 { return w.bytesWritten }

// Close finalizes the FLAC stream and closes the file.
func (w *FLACIQWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.enc.Finalize(); err != nil {
		w.f.Close()
		return err
	}
	return w.f.Close()
}

// IQRecorderWriter is the writer contract shared by the wav (IQWriter) and
// flac (FLACIQWriter) baseband recorders.
type IQRecorderWriter interface {
	Write(samples []complex64) error
	BytesWritten() uint32
	Close() error
}

// NewIQRecorderWriter opens a baseband IQ recording at path in the given
// container format: "wav" (or "") for the canonical two-channel RIFF/WAVE,
// "flac" for the lossless compressed twin.
func NewIQRecorderWriter(path string, sampleRate uint32, format string) (IQRecorderWriter, error) {
	switch format {
	case "", "wav":
		return NewIQWriter(path, sampleRate)
	case "flac":
		return NewFLACIQWriter(path, sampleRate)
	default:
		return nil, fmt.Errorf("baseband: unknown IQ recording format %q (want wav or flac)", format)
	}
}

// IQRecordingExt returns the filename extension (without dot) for a baseband
// recording format accepted by NewIQRecorderWriter.
func IQRecordingExt(format string) string {
	if format == "flac" {
		return "flac"
	}
	return "wav"
}

// isFLACRecording sniffs the 4-byte "fLaC" stream marker so the replay driver
// picks the right decoder from the file itself rather than its extension.
func isFLACRecording(path string) bool {
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

// flacIQStream opens a FLAC stream over r and validates it carries the
// two-channel 16-bit I/Q layout the baseband writers emit.
func flacIQStream(r io.Reader) (*flac.Stream, error) {
	stream, err := flac.New(r)
	if err != nil {
		return nil, fmt.Errorf("baseband: open flac recording: %w", err)
	}
	if stream.Info == nil {
		return nil, errors.New("baseband: flac recording has no STREAMINFO")
	}
	if stream.Info.NChannels != iqWavChannels {
		return nil, fmt.Errorf("baseband: flac recording has %d channels, IQ recordings need 2", stream.Info.NChannels)
	}
	if stream.Info.BitsPerSample != iqWavBitsPerSample {
		return nil, fmt.Errorf("baseband: flac recording is %d-bit, IQ recordings need 16-bit", stream.Info.BitsPerSample)
	}
	return stream, nil
}

// readIQFLACInfo parses a FLAC baseband recording's STREAMINFO into the same
// shape ReadIQWavInfo returns for WAV recordings.
func readIQFLACInfo(path string) (IQWavInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return IQWavInfo{}, err
	}
	defer f.Close()
	stream, err := flacIQStream(f)
	if err != nil {
		return IQWavInfo{}, err
	}
	return IQWavInfo{
		SampleRate: stream.Info.SampleRate,
		Channels:   uint16(stream.Info.NChannels),
		Samples:    int(stream.Info.NSamples),
	}, nil
}

// ReadIQRecordingInfo describes a baseband recording (WAV or FLAC) without
// loading its samples, sniffing the container from the file content.
func ReadIQRecordingInfo(path string) (IQWavInfo, error) {
	if isFLACRecording(path) {
		return readIQFLACInfo(path)
	}
	return ReadIQWavInfo(path)
}
