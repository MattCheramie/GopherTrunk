// Package mp3 provides a pure-Go MP3 encoder used to compress completed
// call audio before it is streamed to broadcast aggregators
// (Broadcastify Calls, RdioScanner, OpenMHz, Icecast).
//
// It wraps the fixed-point Shine encoder, so the daemon keeps its
// zero-CGO single-binary guarantee — no libmp3lame / libshine at build
// or runtime. Shine supports the MPEG-1, MPEG-2 and MPEG-2.5 sample-rate
// families, so 8 kHz digital-voice audio encodes directly with no
// resample stage.
package mp3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	shine "github.com/braheezy/shine-mp3/pkg/mp3"
)

// Encode compresses mono 16-bit PCM sampled at sampleRate Hz into a
// complete MP3 byte stream. The input is treated as a single channel;
// callers with stereo audio must downmix first.
func Encode(samples []int16, sampleRate int) ([]byte, error) {
	if sampleRate <= 0 {
		return nil, errors.New("mp3: sample rate must be positive")
	}
	if len(samples) == 0 {
		return nil, errors.New("mp3: no samples to encode")
	}
	enc := shine.NewEncoder(sampleRate, 1)
	var buf bytes.Buffer
	if err := enc.Write(&buf, samples); err != nil {
		return nil, fmt.Errorf("mp3: encode: %w", err)
	}
	if buf.Len() == 0 {
		return nil, errors.New("mp3: encoder produced no output")
	}
	return buf.Bytes(), nil
}

// EncodeWAVFile reads a 16-bit PCM mono WAV file from path and returns
// the equivalent MP3 byte stream plus the source sample rate.
func EncodeWAVFile(path string) ([]byte, int, error) {
	samples, rate, err := ReadWAV(path)
	if err != nil {
		return nil, 0, err
	}
	out, err := Encode(samples, rate)
	if err != nil {
		return nil, rate, err
	}
	return out, rate, nil
}

// ReadWAV parses a canonical 16-bit PCM mono WAV file and returns its
// samples and sample rate. It tolerates extra chunks between `fmt ` and
// `data` (e.g. LIST/INFO) by walking the chunk list. Only mono 16-bit
// PCM is supported — the format the recorder writes.
func ReadWAV(path string) ([]int16, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	return readWAV(f)
}

func readWAV(r io.Reader) ([]int16, int, error) {
	hdr := make([]byte, 12)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, 0, fmt.Errorf("mp3: read RIFF header: %w", err)
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return nil, 0, errors.New("mp3: not a RIFF/WAVE file")
	}
	var (
		sampleRate    int
		bitsPerSample uint16
		channels      uint16
		gotFmt        bool
	)
	chunkHdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, chunkHdr); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, 0, errors.New("mp3: WAV ended before a data chunk")
			}
			return nil, 0, fmt.Errorf("mp3: read chunk header: %w", err)
		}
		id := string(chunkHdr[0:4])
		size := binary.LittleEndian.Uint32(chunkHdr[4:8])
		switch id {
		case "fmt ":
			fmtBuf := make([]byte, size)
			if _, err := io.ReadFull(r, fmtBuf); err != nil {
				return nil, 0, fmt.Errorf("mp3: read fmt chunk: %w", err)
			}
			if len(fmtBuf) < 16 {
				return nil, 0, errors.New("mp3: short fmt chunk")
			}
			channels = binary.LittleEndian.Uint16(fmtBuf[2:4])
			sampleRate = int(binary.LittleEndian.Uint32(fmtBuf[4:8]))
			bitsPerSample = binary.LittleEndian.Uint16(fmtBuf[14:16])
			gotFmt = true
		case "data":
			if !gotFmt {
				return nil, 0, errors.New("mp3: data chunk before fmt chunk")
			}
			if channels != 1 {
				return nil, 0, fmt.Errorf("mp3: WAV has %d channels, only mono is supported", channels)
			}
			if bitsPerSample != 16 {
				return nil, 0, fmt.Errorf("mp3: WAV is %d-bit, only 16-bit PCM is supported", bitsPerSample)
			}
			data := make([]byte, size)
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, 0, fmt.Errorf("mp3: read data chunk: %w", err)
			}
			samples := make([]int16, len(data)/2)
			for i := range samples {
				samples[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
			}
			return samples, sampleRate, nil
		default:
			// Skip an unrecognised chunk. Chunks are word-aligned, so
			// an odd size carries a trailing pad byte.
			skip := int64(size)
			if size%2 == 1 {
				skip++
			}
			if _, err := io.CopyN(io.Discard, r, skip); err != nil {
				return nil, 0, fmt.Errorf("mp3: skip %q chunk: %w", id, err)
			}
		}
	}
}
