package hunt

import (
	"context"
	"fmt"
)

// IQSource is the tunable IQ provider the live sweeper and hunter drive. It
// abstracts over a live SDR (broker-backed, wired in the daemon) and a
// file/synthetic source (tests), so the sweep/identify/accumulate logic is the
// same offline and on-air.
type IQSource interface {
	// Tune retunes the source to centerHz. Subsequent Capture calls return IQ
	// centered there.
	Tune(centerHz uint32) error
	// Capture returns the next n complex samples at the current center, or an
	// error (including ctx cancellation). Implementations may return fewer than
	// n only at end-of-stream, in which case they return what remains.
	Capture(ctx context.Context, n int) ([]complex64, error)
	// SampleRateHz is the source's IQ sample rate.
	SampleRateHz() uint32
}

// FileIQSource is a deterministic IQSource over an in-memory IQ buffer, used by
// tests. Tune records the requested center (so captured slices can be stamped
// with an absolute frequency) but does not alter the samples — the buffer is
// assumed to already be baseband IQ for whatever carrier the test models.
// Capture serves the buffer cyclically so a sweep with many steps never starves.
type FileIQSource struct {
	iq        []complex64
	rateHz    uint32
	center    uint32
	pos       int
	tuneCalls []uint32
	perTune   map[uint32][]complex64 // optional: distinct IQ per center
	cycle     bool
}

// NewFileIQSource builds a FileIQSource over one buffer served cyclically at
// every tuned center.
func NewFileIQSource(iq []complex64, rateHz uint32) *FileIQSource {
	return &FileIQSource{iq: iq, rateHz: rateHz, cycle: true}
}

// NewMappedIQSource builds a FileIQSource that serves a distinct buffer per
// tuned center frequency (and an empty/zero stream for untuned centers), so a
// test can place a "carrier" at specific frequencies and verify the sweep finds
// exactly those. rateHz is the common sample rate.
func NewMappedIQSource(perCenter map[uint32][]complex64, rateHz uint32) *FileIQSource {
	return &FileIQSource{perTune: perCenter, rateHz: rateHz, cycle: true}
}

func (f *FileIQSource) Tune(centerHz uint32) error {
	f.center = centerHz
	f.pos = 0
	f.tuneCalls = append(f.tuneCalls, centerHz)
	return nil
}

func (f *FileIQSource) SampleRateHz() uint32 { return f.rateHz }

// TuneCalls returns the centers Tune was called with, in order (test helper).
func (f *FileIQSource) TuneCalls() []uint32 { return f.tuneCalls }

func (f *FileIQSource) Capture(ctx context.Context, n int) ([]complex64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	src := f.iq
	if f.perTune != nil {
		src = f.perTune[f.center] // nil ⇒ silence at this center
	}
	out := make([]complex64, n)
	if len(src) == 0 {
		return out, nil // zero IQ (noise-free silence)
	}
	for i := 0; i < n; i++ {
		out[i] = src[f.pos%len(src)]
		f.pos++
	}
	return out, nil
}

// captureFrameSamples is the number of IQ samples one sweep FFT consumes. It is
// max(fftSize, a small floor) so the FFT is always fully populated.
func captureFrameSamples(fftSize int) int {
	if fftSize < 256 {
		return 256
	}
	return fftSize
}

// ensurePow2 returns n when it is a positive power of two, else an error — the
// FFT requires it.
func ensurePow2(n int) error {
	if n <= 0 || n&(n-1) != 0 {
		return fmt.Errorf("hunt: FFT size %d is not a positive power of two", n)
	}
	return nil
}
