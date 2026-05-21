// Package baseband adds wideband IQ recording and offline replay to
// the SDR layer. A RecordingDevice tees a live device's IQ stream to a
// WAV file; a FileDriver mounts those recordings (and SDRtrunk's, which
// use the same layout) back into the pool as virtual tuners.
//
// The on-disk format matches SDRtrunk's baseband recordings: a
// canonical RIFF/WAVE file with two 16-bit signed PCM channels — the
// in-phase sample in channel 1, the quadrature sample in channel 2 —
// at the IQ sample rate.
package baseband

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	iqWavHeaderSize    = 44
	iqWavChannels      = 2
	iqWavBitsPerSample = 16
	iqWavBlockAlign    = iqWavChannels * iqWavBitsPerSample / 8 // 4 bytes/frame
)

// IQWriter streams complex64 IQ samples to a two-channel 16-bit WAV.
// The RIFF/data length fields are patched on Close so a daemon crash
// still leaves a readable (if short) recording behind.
type IQWriter struct {
	f            *os.File
	sampleRate   uint32
	bytesWritten uint32
	closed       bool
}

// NewIQWriter creates (or truncates) path and writes the WAV header.
func NewIQWriter(path string, sampleRate uint32) (*IQWriter, error) {
	if sampleRate == 0 {
		return nil, errors.New("baseband: IQ WAV sample rate must be > 0")
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := &IQWriter{f: f, sampleRate: sampleRate}
	if err := w.writeHeader(); err != nil {
		f.Close()
		return nil, err
	}
	return w, nil
}

// Write appends a block of IQ samples. Each complex64 is clamped to
// [-1, 1] and scaled to a signed 16-bit pair (I, Q).
func (w *IQWriter) Write(samples []complex64) error {
	if w.closed {
		return errors.New("baseband: IQ writer is closed")
	}
	if len(samples) == 0 {
		return nil
	}
	buf := make([]byte, iqWavBlockAlign*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[4*i:], uint16(floatToI16(real(s))))
		binary.LittleEndian.PutUint16(buf[4*i+2:], uint16(floatToI16(imag(s))))
	}
	n, err := w.f.Write(buf)
	w.bytesWritten += uint32(n)
	return err
}

// BytesWritten reports the IQ payload bytes written so far.
func (w *IQWriter) BytesWritten() uint32 { return w.bytesWritten }

// Close patches the length fields and closes the file.
func (w *IQWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.patchHeader(); err != nil {
		w.f.Close()
		return err
	}
	return w.f.Close()
}

func (w *IQWriter) writeHeader() error {
	h := make([]byte, iqWavHeaderSize)
	copy(h[0:4], "RIFF")
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16)
	binary.LittleEndian.PutUint16(h[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(h[22:24], iqWavChannels)
	binary.LittleEndian.PutUint32(h[24:28], w.sampleRate)
	binary.LittleEndian.PutUint32(h[28:32], w.sampleRate*iqWavBlockAlign)
	binary.LittleEndian.PutUint16(h[32:34], iqWavBlockAlign)
	binary.LittleEndian.PutUint16(h[34:36], iqWavBitsPerSample)
	copy(h[36:40], "data")
	_, err := w.f.Write(h)
	return err
}

func (w *IQWriter) patchHeader() error {
	if _, err := w.f.Seek(4, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.f, binary.LittleEndian, uint32(36+w.bytesWritten)); err != nil {
		return err
	}
	if _, err := w.f.Seek(40, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.f, binary.LittleEndian, w.bytesWritten); err != nil {
		return err
	}
	_, _ = w.f.Seek(0, io.SeekEnd)
	return nil
}

// IQWavInfo describes a baseband WAV without loading its samples.
type IQWavInfo struct {
	SampleRate uint32
	Channels   uint16
	Samples    int // IQ-sample frames in the data chunk
}

// ReadIQWavInfo parses just the header of a baseband WAV.
func ReadIQWavInfo(path string) (IQWavInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return IQWavInfo{}, err
	}
	defer f.Close()
	_, info, err := parseIQWavHeader(f)
	return info, err
}

// parseIQWavHeader reads the RIFF chunks up to (and including) the
// "data" chunk header, leaving f positioned at the first data byte. It
// returns the data-chunk byte length and the format info.
func parseIQWavHeader(f *os.File) (dataBytes uint32, info IQWavInfo, err error) {
	hdr := make([]byte, 12)
	if _, err = io.ReadFull(f, hdr); err != nil {
		return 0, info, fmt.Errorf("baseband: read RIFF header: %w", err)
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return 0, info, errors.New("baseband: not a RIFF/WAVE file")
	}
	var (
		bits   uint16
		gotFmt bool
	)
	chunkHdr := make([]byte, 8)
	for {
		if _, err = io.ReadFull(f, chunkHdr); err != nil {
			return 0, info, errors.New("baseband: WAV ended before a data chunk")
		}
		id := string(chunkHdr[0:4])
		size := binary.LittleEndian.Uint32(chunkHdr[4:8])
		switch id {
		case "fmt ":
			fmtBuf := make([]byte, size)
			if _, err = io.ReadFull(f, fmtBuf); err != nil {
				return 0, info, fmt.Errorf("baseband: read fmt chunk: %w", err)
			}
			if len(fmtBuf) < 16 {
				return 0, info, errors.New("baseband: short fmt chunk")
			}
			info.Channels = binary.LittleEndian.Uint16(fmtBuf[2:4])
			info.SampleRate = binary.LittleEndian.Uint32(fmtBuf[4:8])
			bits = binary.LittleEndian.Uint16(fmtBuf[14:16])
			gotFmt = true
		case "data":
			if !gotFmt {
				return 0, info, errors.New("baseband: data chunk before fmt chunk")
			}
			if info.Channels != iqWavChannels {
				return 0, info, fmt.Errorf("baseband: WAV has %d channels, IQ recordings need 2", info.Channels)
			}
			if bits != iqWavBitsPerSample {
				return 0, info, fmt.Errorf("baseband: WAV is %d-bit, IQ recordings need 16-bit", bits)
			}
			info.Samples = int(size / iqWavBlockAlign)
			return size, info, nil
		default:
			skip := int64(size)
			if size%2 == 1 {
				skip++
			}
			if _, err = f.Seek(skip, io.SeekCurrent); err != nil {
				return 0, info, fmt.Errorf("baseband: skip %q chunk: %w", id, err)
			}
		}
	}
}

// floatToI16 clamps a normalised sample to [-1, 1] and scales it to a
// signed 16-bit value.
func floatToI16(v float32) int16 {
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	return int16(v * 32767)
}
