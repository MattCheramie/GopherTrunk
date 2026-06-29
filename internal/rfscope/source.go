package rfscope

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// Source is a sequential producer of wideband IQ chunks at one center frequency
// and sample rate. Both an offline capture file and a bounded live SDR capture
// implement it, so the segmentation engine and every analyzer run identically on
// recorded and on-air IQ — "both modes equally" is satisfied at the core, not
// bolted on. Unlike hunt.IQSource (which retunes per sweep step), a Source
// streams a single span the scene analyzer characterizes as a whole.
type Source interface {
	// Next returns the next chunk of up to n complex samples. The returned slice
	// may be shorter than n only at end-of-stream; an empty slice with io.EOF
	// marks the end. ctx cancellation is honored.
	Next(ctx context.Context, n int) ([]complex64, error)
	// CenterHz is the source's absolute center frequency.
	CenterHz() uint32
	// SampleRateHz is the IQ sample rate.
	SampleRateHz() float64
}

// FileSource streams an on-disk IQ capture (u8/f32) in chunks, decoding through
// the same siglab.SampleFormat decoders the replay path uses so rfscope and
// replay read captures identically.
type FileSource struct {
	f        *os.File
	decode   siglab.SampleDecoder
	bytesPer int
	centerHz uint32
	rateHz   float64
	rbuf     []byte
}

// OpenFile opens an IQ capture for streaming. centerHz/rateHz describe the
// capture (the segmentation engine stamps absolute frequencies from them).
func OpenFile(path string, format siglab.SampleFormat, centerHz uint32, rateHz float64) (*FileSource, error) {
	if rateHz <= 0 {
		return nil, fmt.Errorf("rfscope: sample rate must be positive, got %g", rateHz)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("rfscope: open capture: %w", err)
	}
	decode, bytesPer := format.Decoder()
	return &FileSource{
		f:        f,
		decode:   decode,
		bytesPer: bytesPer,
		centerHz: centerHz,
		rateHz:   rateHz,
	}, nil
}

func (s *FileSource) CenterHz() uint32      { return s.centerHz }
func (s *FileSource) SampleRateHz() float64 { return s.rateHz }

// Next reads up to n samples from the file. It returns io.EOF (with any final
// partial chunk on the preceding call) once the file is exhausted.
func (s *FileSource) Next(ctx context.Context, n int) ([]complex64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, nil
	}
	want := n * s.bytesPer
	if cap(s.rbuf) < want {
		s.rbuf = make([]byte, want)
	}
	buf := s.rbuf[:want]
	got, err := io.ReadFull(s.f, buf)
	switch {
	case err == nil:
		// full read
	case err == io.EOF:
		return nil, io.EOF
	case err == io.ErrUnexpectedEOF:
		// trailing partial — decode the whole IQ pairs we did get
		got -= got % s.bytesPer
		if got == 0 {
			return nil, io.EOF
		}
	default:
		return nil, fmt.Errorf("rfscope: read capture: %w", err)
	}
	pairs := got / s.bytesPer
	out := make([]complex64, pairs)
	s.decode(buf[:got], out)
	return out, nil
}

// Close releases the underlying file.
func (s *FileSource) Close() error { return s.f.Close() }

// LiveSource adapts a hunt.IQSource (the live, broker-backed SDR provider the
// daemon wires) to a scene Source by tuning it once to a fixed center and
// streaming sequential captures.
type LiveSource struct {
	src      hunt.IQSource
	centerHz uint32
}

// NewLiveSource tunes src to centerHz and returns it as a streaming Source.
func NewLiveSource(src hunt.IQSource, centerHz uint32) (*LiveSource, error) {
	if src == nil {
		return nil, fmt.Errorf("rfscope: nil IQ source")
	}
	if err := src.Tune(centerHz); err != nil {
		return nil, fmt.Errorf("rfscope: tune %d Hz: %w", centerHz, err)
	}
	return &LiveSource{src: src, centerHz: centerHz}, nil
}

func (s *LiveSource) CenterHz() uint32      { return s.centerHz }
func (s *LiveSource) SampleRateHz() float64 { return float64(s.src.SampleRateHz()) }

func (s *LiveSource) Next(ctx context.Context, n int) ([]complex64, error) {
	return s.src.Capture(ctx, n)
}

// MemSource is a deterministic in-memory Source for tests: it serves a buffer
// once, in chunks, then returns io.EOF.
type MemSource struct {
	iq       []complex64
	centerHz uint32
	rateHz   float64
	pos      int
}

// NewMemSource builds an in-memory Source over iq.
func NewMemSource(iq []complex64, centerHz uint32, rateHz float64) *MemSource {
	return &MemSource{iq: iq, centerHz: centerHz, rateHz: rateHz}
}

func (s *MemSource) CenterHz() uint32      { return s.centerHz }
func (s *MemSource) SampleRateHz() float64 { return s.rateHz }

func (s *MemSource) Next(ctx context.Context, n int) ([]complex64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.pos >= len(s.iq) {
		return nil, io.EOF
	}
	end := s.pos + n
	if end > len(s.iq) {
		end = len(s.iq)
	}
	out := s.iq[s.pos:end]
	s.pos = end
	return out, nil
}

// Drain reads the whole source into one buffer, capped at maxSamples (0 = no
// cap). It is the convenience the offline analyze path uses to materialize a
// bounded capture; the segmentation engine can also stream Next directly for
// live use. chunk is the per-read size.
func Drain(ctx context.Context, src Source, chunk, maxSamples int) ([]complex64, error) {
	if chunk <= 0 {
		chunk = 1 << 16
	}
	var out []complex64
	for {
		if maxSamples > 0 && len(out) >= maxSamples {
			return out[:maxSamples], nil
		}
		c, err := src.Next(ctx, chunk)
		out = append(out, c...)
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		if len(c) == 0 {
			return out, nil
		}
	}
}
