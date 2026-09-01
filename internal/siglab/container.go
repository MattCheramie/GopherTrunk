package siglab

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/mewkiz/flac"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// IQContainer streams complex64 IQ to an *os.File in a chosen on-disk container:
// a headerless body (u8/f32/cs16), a RIFF/WAVE file (wav), or a FLAC file
// (flac). wav and flac both wrap the EXACT cs16 16-bit I/Q body — I in the left
// channel, Q in the right, normalised ×32768 (floatToI16) — so a wav dump
// replays byte-for-byte identically to the cs16 equivalent (siglab strips the
// 44-byte RIFF header on read) and a flac dump losslessly decodes back to the
// same int16 samples. FLAC uses linear prediction on the continuous baseband
// waveform for a typical 30–50% size reduction with no loss.
//
// Container-only formats (wav/flac) need a header written up front and a
// finalize step (patch the RIFF length fields / finalize the FLAC stream);
// Finalize does that but does NOT close the file — the caller owns the *os.File
// lifetime (it created it and closes it after Finalize). The headerless formats
// are append-only, so Finalize just flushes the buffer.
type IQContainer struct {
	format SampleFormat
	f      *os.File

	// Headerless (u8/f32/cs16) and the wav body both go through a buffered
	// CaptureWriter; for wav the body is forced to the sw16 encoding so it is
	// byte-identical to a cs16 dump.
	bw      *bufio.Writer
	cw      *CaptureWriter
	dataLen uint32 // wav: body bytes written (for the RIFF length patch)

	// FLAC: the shared two-channel encode core (also behind the baseband
	// recorder's FLACIQWriter), fed siglab-normalised int16 pairs.
	enc *baseband.FLACIQEncoder

	samples int64
}

const (
	iqWavHeaderSize = 44
	iqWavBlockAlign = 4 // 2 channels × 16-bit
)

// NewIQContainer creates a container writer over f for the given format.
// sampleRateHz is required for the wav/flac headers (ignored for headerless
// formats). f must be freshly created/truncated and positioned at offset 0.
func NewIQContainer(f *os.File, format SampleFormat, sampleRateHz int) (*IQContainer, error) {
	c := &IQContainer{format: format, f: f}
	switch format {
	case FormatWAV:
		if sampleRateHz <= 0 {
			return nil, fmt.Errorf("siglab: wav container needs a positive sample rate, got %d", sampleRateHz)
		}
		c.bw = bufio.NewWriterSize(f, 1<<16)
		if err := writeIQWavHeader(c.bw, uint32(sampleRateHz)); err != nil {
			return nil, fmt.Errorf("siglab: write wav header: %w", err)
		}
		// Force the 16-bit body encoding so a wav dump is a cs16 body + header.
		c.cw = NewCaptureWriter(c.bw, FormatS16)
	case FormatFLAC:
		if sampleRateHz <= 0 {
			return nil, fmt.Errorf("siglab: flac container needs a positive sample rate, got %d", sampleRateHz)
		}
		// The shared encoder patches the STREAMINFO via Seek on Finalize but
		// never closes f — uniform with the other formats (the caller owns f).
		enc, err := baseband.NewFLACIQEncoder(f, uint32(sampleRateHz))
		if err != nil {
			return nil, fmt.Errorf("siglab: init flac encoder: %w", err)
		}
		c.enc = enc
	default:
		c.bw = bufio.NewWriterSize(f, 1<<16)
		c.cw = NewCaptureWriter(c.bw, format)
	}
	return c, nil
}

// Write appends one chunk of IQ. An empty chunk is a no-op.
func (c *IQContainer) Write(iq []complex64) error {
	if len(iq) == 0 {
		return nil
	}
	switch c.format {
	case FormatFLAC:
		for _, s := range iq {
			// siglab's own float→int16 normalisation (×32768), so a flac dump
			// losslessly matches the cs16 body — the baseband recorder feeds
			// the same core its ×32767 samples instead.
			if err := c.enc.WriteI16(floatToI16(real(s)), floatToI16(imag(s))); err != nil {
				return err
			}
		}
	default: // headerless and wav both stream through the CaptureWriter
		if err := c.cw.Write(iq); err != nil {
			return err
		}
		if c.format == FormatWAV {
			c.dataLen += uint32(len(iq) * iqWavBlockAlign)
		}
	}
	c.samples += int64(len(iq))
	return nil
}

// Samples returns the number of IQ samples written so far.
func (c *IQContainer) Samples() int64 { return c.samples }

// Finalize completes the container (flushing buffers, patching the RIFF length
// fields or finalizing the FLAC stream). It does NOT close the underlying file.
// Safe to call once; the caller closes the file afterward.
func (c *IQContainer) Finalize() error {
	switch c.format {
	case FormatFLAC:
		// Finalize flushes the pending block and patches the STREAMINFO
		// (sample count + MD5) via a seek on f; it does not close f itself.
		return c.enc.Finalize()
	case FormatWAV:
		if err := c.bw.Flush(); err != nil {
			return err
		}
		return patchIQWavHeader(c.f, c.dataLen)
	default:
		return c.bw.Flush()
	}
}

// flacSW16Reader lazily decodes a FLAC stream into the headerless interleaved
// 16-bit I/Q body (I=left channel, Q=right), so a flac dump feeds the same
// downstream decoder path as cs16. It emits little-endian int16 pairs.
type flacSW16Reader struct {
	stream *flac.Stream
	buf    []byte // decoded-but-unread body bytes
}

// newFLACSW16Reader parses the FLAC signature + STREAMINFO from r and returns a
// reader that yields the sw16 body plus the stream's sample rate. The stream
// must be two-channel 16-bit (as written by IQContainer).
func newFLACSW16Reader(r io.Reader) (io.Reader, uint32, error) {
	stream, err := flac.New(r)
	if err != nil {
		return nil, 0, fmt.Errorf("siglab: open flac stream: %w", err)
	}
	if stream.Info == nil {
		return nil, 0, fmt.Errorf("siglab: flac stream has no STREAMINFO")
	}
	if stream.Info.NChannels != 2 {
		return nil, 0, fmt.Errorf("siglab: flac IQ stream has %d channels, want 2 (I/Q)", stream.Info.NChannels)
	}
	if stream.Info.BitsPerSample != 16 {
		return nil, 0, fmt.Errorf("siglab: flac IQ stream is %d-bit, want 16", stream.Info.BitsPerSample)
	}
	return &flacSW16Reader{stream: stream}, stream.Info.SampleRate, nil
}

func (fr *flacSW16Reader) Read(p []byte) (int, error) {
	for len(fr.buf) == 0 {
		f, err := fr.stream.ParseNext()
		if err != nil {
			return 0, err // includes io.EOF
		}
		if len(f.Subframes) != 2 {
			return 0, fmt.Errorf("siglab: flac frame has %d subframes, want 2", len(f.Subframes))
		}
		l, q := f.Subframes[0].Samples, f.Subframes[1].Samples
		n := len(l)
		body := make([]byte, n*iqWavBlockAlign)
		for i := 0; i < n; i++ {
			binary.LittleEndian.PutUint16(body[4*i:], uint16(int16(l[i])))
			binary.LittleEndian.PutUint16(body[4*i+2:], uint16(int16(q[i])))
		}
		fr.buf = body
	}
	n := copy(p, fr.buf)
	fr.buf = fr.buf[n:]
	return n, nil
}

// DecodeContainerFile reads a wav or flac IQ capture (as written by
// IQContainer) back to complex64 samples plus the container's sample rate,
// sniffing the container from the file content. Small-file convenience for
// tests and tools; the replay engine streams instead.
func DecodeContainerFile(path string) ([]complex64, uint32, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	var (
		body []byte
		rate uint32
	)
	switch {
	case len(b) >= 4 && string(b[0:4]) == "fLaC":
		fr, r, err := newFLACSW16Reader(bytes.NewReader(b))
		if err != nil {
			return nil, 0, err
		}
		body, err = io.ReadAll(fr)
		if err != nil && err != io.EOF {
			return nil, 0, err
		}
		rate = r
	case len(b) >= iqWavHeaderSize && string(b[0:4]) == "RIFF":
		rate = binary.LittleEndian.Uint32(b[24:28])
		body = b[iqWavHeaderSize:]
	default:
		return nil, 0, fmt.Errorf("siglab: %s is neither a RIFF/WAVE nor a FLAC capture", path)
	}
	out := make([]complex64, len(body)/iqWavBlockAlign)
	decodeSW16(body[:len(out)*iqWavBlockAlign], out)
	return out, rate, nil
}
// header (the length fields are patched on finalize). Same layout as
// baseband.IQWriter, kept here so a wav dump's body is siglab's cs16 body.
func writeIQWavHeader(w io.Writer, sampleRate uint32) error {
	h := make([]byte, iqWavHeaderSize)
	copy(h[0:4], "RIFF")
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16)
	binary.LittleEndian.PutUint16(h[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(h[22:24], 2) // channels
	binary.LittleEndian.PutUint32(h[24:28], sampleRate)
	binary.LittleEndian.PutUint32(h[28:32], sampleRate*iqWavBlockAlign)
	binary.LittleEndian.PutUint16(h[32:34], iqWavBlockAlign)
	binary.LittleEndian.PutUint16(h[34:36], 16) // bits per sample
	copy(h[36:40], "data")
	_, err := w.Write(h)
	return err
}

// patchIQWavHeader seeks f and writes the RIFF chunk size and data chunk size
// now that the payload length (dataLen) is known, then repositions at EOF.
func patchIQWavHeader(f *os.File, dataLen uint32) error {
	if _, err := f.Seek(4, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(36+dataLen)); err != nil {
		return err
	}
	if _, err := f.Seek(40, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, dataLen); err != nil {
		return err
	}
	_, err := f.Seek(0, io.SeekEnd)
	return err
}
