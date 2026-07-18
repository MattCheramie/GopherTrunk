package siglab

import (
	"io"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// CaptureWriter streams IQ to an io.Writer one chunk at a time in a headerless
// on-disk format (u8, f32, cs16). It is the streaming counterpart of
// EncodeCapture: instead of encoding a whole capture into a fresh []byte (and
// forcing a second full copy in RAM), it encodes each chunk into a single
// reused scratch buffer and writes it straight out, so peak memory is one chunk
// regardless of how long the capture runs. The bytes it emits are identical to
// EncodeCapture of the concatenated input, so a streamed file round-trips
// through the same decoders.
//
// FormatWAV is emitted as the same headerless 16-bit body EncodeCapture already
// produces for it (a real RIFF/WAVE file needs a rate-aware header written via
// baseband.IQWriter) — matched here so the two paths stay byte-for-byte equal.
type CaptureWriter struct {
	w       io.Writer
	bpp     int // bytes per IQ sample in this format
	format  SampleFormat
	scratch []byte
	samples int64
}

// NewCaptureWriter returns a CaptureWriter that encodes to format and writes to
// w. w is typically a bufio.Writer wrapping the staged capture file.
func NewCaptureWriter(w io.Writer, format SampleFormat) *CaptureWriter {
	_, bpp := format.Decoder()
	return &CaptureWriter{w: w, bpp: bpp, format: format}
}

// Write encodes one chunk of IQ and writes it to the underlying writer. An
// empty chunk is a no-op (a decimating source can legitimately yield zero
// output samples for a small input block).
func (c *CaptureWriter) Write(iq []complex64) error {
	if len(iq) == 0 {
		return nil
	}
	n := len(iq) * c.bpp
	if cap(c.scratch) < n {
		c.scratch = make([]byte, n)
	}
	buf := c.scratch[:n]
	encodeCaptureInto(buf, iq, c.format)
	if _, err := c.w.Write(buf); err != nil {
		return err
	}
	c.samples += int64(len(iq))
	return nil
}

// Samples returns the number of IQ samples written so far.
func (c *CaptureWriter) Samples() int64 { return c.samples }

// encodeCaptureInto writes iq into dst using the same per-format layout as
// EncodeCapture. dst must hold at least len(iq)*bytesPerSample(format) bytes.
func encodeCaptureInto(dst []byte, iq []complex64, format SampleFormat) {
	switch format {
	case FormatF32:
		encodeF32Into(dst, iq)
	case FormatS16, FormatWAV:
		encodeSW16Into(dst, iq)
	default:
		encodeU8Into(dst, iq)
	}
}

// StreamDownconverter carves a narrowband channel out of a live wideband
// capture chunk by chunk, retaining the down-converter's NCO + polyphase-FIR
// state across calls. This is the streaming counterpart of the one-shot
// Downconvert: because filter history is carried between chunks, the
// concatenated per-chunk output is bit-equivalent to downconverting the whole
// capture at once — but without ever holding the full wideband stream in RAM.
//
// Construct one per capture (a fresh capture, or a retune, needs a new one — do
// not reuse across channels, since that is what Reset guards against in the
// live pipeline).
type StreamDownconverter struct {
	ddc *ccdecoder.Downconverter
	dst []complex64
}

// NewStreamDownconverter builds a streaming DDC that shifts the channel at
// offsetHz (offset from the wideband centre) to DC and decimates inRateHz to
// ~targetHz. OutRateHz reports the exact achieved rate.
func NewStreamDownconverter(inRateHz, offsetHz, targetHz float64) *StreamDownconverter {
	return &StreamDownconverter{ddc: ccdecoder.NewDownconverterWithOffset(inRateHz, targetHz, offsetHz)}
}

// Process decimates one wideband chunk to the narrowband rate, reusing an
// internal buffer. The returned slice is valid only until the next Process
// call, so callers must consume (encode/write) it before processing the next
// chunk — which the streaming capture path does.
func (d *StreamDownconverter) Process(chunk []complex64) []complex64 {
	out := d.ddc.Process(d.dst, chunk)
	// Retain the resampler's (possibly grown) output buffer to reuse next call —
	// the standard ddc reuse idiom. The caller encodes/writes out before the
	// next Process, so overwriting it next call is safe.
	d.dst = out
	return out
}

// OutRateHz returns the achieved narrowband output rate.
func (d *StreamDownconverter) OutRateHz() float64 { return d.ddc.OutRateHz() }
