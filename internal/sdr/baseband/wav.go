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
	"math"
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

// IQWavEncoding is the sample encoding of a baseband WAV's data chunk.
// GopherTrunk and SDRtrunk write 16-bit PCM; SDR# ("SDRSharp") baseband
// recordings are 32-bit IEEE float (or 8-bit PCM), and every reader in the
// tree used to assume 16-bit PCM without looking at the fmt chunk, decoding a
// float capture as garbage int16 pairs (issue #1184: the FleetSync captures
// read as 2x the samples at -2.5 dBFS of noise). The fmt chunk decides.
type IQWavEncoding uint8

const (
	// IQWavPCM16 is two-channel 16-bit signed PCM (the canonical layout).
	IQWavPCM16 IQWavEncoding = iota
	// IQWavFloat32 is two-channel 32-bit IEEE float (SDR#, GNU Radio wav sinks).
	IQWavFloat32
	// IQWavPCM8 is two-channel 8-bit unsigned PCM (SDR# 8-bit recordings).
	IQWavPCM8
)

// String renders the encoding for logs and errors.
func (e IQWavEncoding) String() string {
	switch e {
	case IQWavFloat32:
		return "float32"
	case IQWavPCM8:
		return "pcm8"
	default:
		return "pcm16"
	}
}

// BlockAlign is the byte length of one IQ frame (both channels) in this encoding.
func (e IQWavEncoding) BlockAlign() int {
	switch e {
	case IQWavFloat32:
		return 8
	case IQWavPCM8:
		return 2
	default:
		return iqWavBlockAlign
	}
}

// DecodeIQ converts a data-chunk byte run in this encoding into complex64
// IQ (I = channel 1, Q = channel 2), normalised to [-1, 1] for the PCM
// encodings and passed through for float.
func (e IQWavEncoding) DecodeIQ(buf []byte) []complex64 {
	n := len(buf) / e.BlockAlign()
	out := make([]complex64, n)
	switch e {
	case IQWavFloat32:
		for i := 0; i < n; i++ {
			iv := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i:]))
			qv := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i+4:]))
			out[i] = complex(iv, qv)
		}
	case IQWavPCM8:
		for i := 0; i < n; i++ {
			out[i] = complex((float32(buf[2*i])-127.5)/127.5, (float32(buf[2*i+1])-127.5)/127.5)
		}
	default:
		for i := 0; i < n; i++ {
			iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
			qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
			out[i] = complex(float32(iv)/32768, float32(qv)/32768)
		}
	}
	return out
}

// IQWavInfo describes a baseband WAV without loading its samples.
type IQWavInfo struct {
	SampleRate uint32
	Channels   uint16
	Samples    int // IQ-sample frames in the data chunk
	// Encoding is the data chunk's sample encoding (fmt chunk: format tag +
	// bits per sample); BlockAlign is the byte length of one IQ frame in it.
	Encoding   IQWavEncoding
	BlockAlign int
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

// WAVE format tags (RIFF/WAVE fmt chunk).
const (
	wavFormatPCM        = 0x0001
	wavFormatIEEEFloat  = 0x0003
	wavFormatExtensible = 0xFFFE
)

// applyWavFmt fills info from a fmt chunk body: channels, rate, and the
// encoding resolved from the format tag and bits per sample. A
// WAVE_FORMAT_EXTENSIBLE chunk carries the real tag in the first two bytes
// of its SubFormat GUID (offset 24), which is how SDR# and sox label 32-bit
// float files.
func applyWavFmt(info *IQWavInfo, fmtBuf []byte) error {
	if len(fmtBuf) < 16 {
		return errors.New("baseband: short fmt chunk")
	}
	tag := binary.LittleEndian.Uint16(fmtBuf[0:2])
	info.Channels = binary.LittleEndian.Uint16(fmtBuf[2:4])
	info.SampleRate = binary.LittleEndian.Uint32(fmtBuf[4:8])
	bits := binary.LittleEndian.Uint16(fmtBuf[14:16])
	if tag == wavFormatExtensible {
		if len(fmtBuf) < 26 {
			return errors.New("baseband: short WAVE_FORMAT_EXTENSIBLE fmt chunk")
		}
		tag = binary.LittleEndian.Uint16(fmtBuf[24:26])
	}
	switch {
	case tag == wavFormatPCM && bits == 16:
		info.Encoding = IQWavPCM16
	case tag == wavFormatPCM && bits == 8:
		info.Encoding = IQWavPCM8
	case tag == wavFormatIEEEFloat && bits == 32:
		info.Encoding = IQWavFloat32
	default:
		return fmt.Errorf("baseband: WAV format tag 0x%04X at %d bits is not a supported IQ encoding (16-bit PCM, 8-bit PCM or 32-bit float)", tag, bits)
	}
	info.BlockAlign = info.Encoding.BlockAlign()
	return nil
}

// validateIQWav is the data-chunk gate both header parsers share.
func validateIQWav(info IQWavInfo, gotFmt bool) error {
	if !gotFmt {
		return errors.New("baseband: data chunk before fmt chunk")
	}
	if info.Channels != iqWavChannels {
		return fmt.Errorf("baseband: WAV has %d channels, IQ recordings need 2", info.Channels)
	}
	return nil
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
	gotFmt := false
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
			if err = applyWavFmt(&info, fmtBuf); err != nil {
				return 0, info, err
			}
			gotFmt = true
		case "data":
			if err = validateIQWav(info, gotFmt); err != nil {
				return 0, info, err
			}
			info.Samples = int(size) / info.BlockAlign
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

// ReadIQWavStreamHeader consumes the RIFF/WAVE header from a forward-only
// reader, leaving r positioned at the first IQ data byte, and returns the
// format info. Unlike parseIQWavHeader it never seeks — unknown chunks are
// discarded by reading past them — so it works on a pipe, an HTTP body, or
// any non-seekable capture source. info.Samples is left 0 (the data length
// is not needed when the payload is streamed to EOF).
func ReadIQWavStreamHeader(r io.Reader) (IQWavInfo, error) {
	var info IQWavInfo
	hdr := make([]byte, 12)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return info, fmt.Errorf("baseband: read RIFF header: %w", err)
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return info, errors.New("baseband: not a RIFF/WAVE file")
	}
	gotFmt := false
	chunkHdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, chunkHdr); err != nil {
			return info, errors.New("baseband: WAV ended before a data chunk")
		}
		id := string(chunkHdr[0:4])
		size := binary.LittleEndian.Uint32(chunkHdr[4:8])
		switch id {
		case "fmt ":
			fmtBuf := make([]byte, size)
			if _, err := io.ReadFull(r, fmtBuf); err != nil {
				return info, fmt.Errorf("baseband: read fmt chunk: %w", err)
			}
			if err := applyWavFmt(&info, fmtBuf); err != nil {
				return info, err
			}
			gotFmt = true
		case "data":
			if err := validateIQWav(info, gotFmt); err != nil {
				return info, err
			}
			return info, nil
		default:
			skip := int64(size)
			if size%2 == 1 {
				skip++
			}
			if _, err := io.CopyN(io.Discard, r, skip); err != nil {
				return info, fmt.Errorf("baseband: skip %q chunk: %w", id, err)
			}
		}
	}
}

// DecodeIQ16 converts one interleaved 16-bit I/Q PCM block into complex64,
// matching the normalisation IQWriter uses. Exported so offline replay can
// decode baseband WAV payloads through the same math the driver uses.
func DecodeIQ16(buf []byte) []complex64 { return IQWavPCM16.DecodeIQ(buf) }

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
