// Package spectrum produces frame-rate-limited windowed FFT magnitude
// frames from a stream of IQ chunks. It is the producer side of the
// live spectrum / waterfall feature — a Frame goes onto an SSE/WS wire
// at a configurable refresh rate (10 Hz by default), low enough that
// the producer's CPU cost is negligible relative to control-channel
// decode and the wire bandwidth stays reasonable for LAN clients.
//
// Design notes:
//
//   - One FFT per Frame, not per IQ chunk. The producer consumes
//     chunks continuously but only invokes an FFT once per
//     1/FrameRate interval. Intermediate chunks are dropped — for a
//     2.048 MS/s control SDR running 10 fps, ~204k samples elapse
//     between frames; processing them all would burn CPU for no
//     visual benefit.
//
//   - Power normalized to dBFS so the wire representation is human-
//     readable and small (float32 per bin). Subscribers receive
//     freshly-allocated slices to avoid coupling consumers.
//
//   - The producer doesn't own the IQ source — it consumes from a
//     caller-provided <-chan []complex64 (typically an
//     iqtap.Subscriber.C). Lifecycle is "run until ctx cancels OR the
//     input channel closes."
package spectrum

import (
	"context"
	"errors"
	"math"
	"math/cmplx"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// Frame is one spectrum snapshot. Bins are dBFS magnitudes, length
// equal to the configured FFT size. Bin order is FFT-shifted so
// Bins[0] corresponds to (CenterHz - SampleRate/2) and
// Bins[len-1] corresponds to (CenterHz + SampleRate/2 - SampleRate/N).
type Frame struct {
	Timestamp  time.Time
	CenterHz   uint32
	SampleRate uint32
	Bins       []float32
}

// Options configure a Producer.
type Options struct {
	// FFTSize is the number of complex samples per FFT. Must be a
	// power of two; 4096 is a good default for a 2.048 MS/s control
	// stream (~500 Hz/bin resolution).
	FFTSize int

	// FrameRate caps the output rate in frames per second. 0 picks
	// 10 fps. Higher rates burn CPU and wire bandwidth without
	// visible benefit beyond ~15 fps.
	FrameRate float64

	// CenterFreqHz and SampleRateHz are stamped on every Frame so
	// downstream clients can map bins to absolute frequencies
	// without consulting a side channel. Callers should update them
	// when the device retunes (Producer.SetCenter / SetSampleRate).
	CenterFreqHz uint32
	SampleRateHz uint32
}

// Producer consumes IQ chunks and emits Frames at a capped rate.
// One Producer per (device, subscriber) pairing — the bulk of the
// CPU cost is the FFT, so two Producers off the same source pay
// double. Callers that want multiple subscribers should keep a
// single Producer and fan its output channel.
type Producer struct {
	size   int
	period time.Duration

	plan   fft.Plan
	win    []float64
	winSum float64 // Σ(window²) — used for power normalization

	bufIQ  []complex128 // reusable FFT input buffer (windowed)
	bufOut []complex128 // reusable FFT output buffer

	centerHz atomic.Uint32
	rateHz   atomic.Uint32

	dropped atomic.Uint64
}

// New builds a Producer with the given options. Returns an error if
// the FFT size isn't a positive power of two.
func New(opts Options) (*Producer, error) {
	size := opts.FFTSize
	if size <= 0 {
		size = 4096
	}
	if size&(size-1) != 0 {
		return nil, errors.New("spectrum: FFTSize must be a power of two")
	}
	rate := opts.FrameRate
	if rate <= 0 {
		rate = 10
	}
	p := &Producer{
		size:   size,
		period: time.Duration(float64(time.Second) / rate),
		plan:   fft.New(size),
		win:    window.Hann(size),
		bufIQ:  make([]complex128, size),
		bufOut: make([]complex128, size),
	}
	for _, w := range p.win {
		p.winSum += w * w
	}
	p.centerHz.Store(opts.CenterFreqHz)
	p.rateHz.Store(opts.SampleRateHz)
	return p, nil
}

// SetCenter updates the centre frequency stamped on subsequent
// Frames. Call after the underlying device retunes.
func (p *Producer) SetCenter(hz uint32) { p.centerHz.Store(hz) }

// SetSampleRate updates the sample rate stamped on subsequent
// Frames. Call after the underlying device changes rates.
func (p *Producer) SetSampleRate(hz uint32) { p.rateHz.Store(hz) }

// FFTSize returns the configured FFT size (bins per Frame).
func (p *Producer) FFTSize() int { return p.size }

// Dropped returns the cumulative number of IQ chunks consumed but
// not processed (i.e. arrived during a rate-limited interval).
// Non-zero is expected and healthy at typical SDR rates.
func (p *Producer) Dropped() uint64 { return p.dropped.Load() }

// Run consumes from in and emits Frames onto out until ctx cancels
// or in closes. Frames are sent non-blocking; a slow consumer
// silently drops them (the rate-cap means losses are at most
// FrameRate per second).
//
// Run owns the goroutine; callers spawn it and read frames on out.
func (p *Producer) Run(ctx context.Context, in <-chan []complex64, out chan<- Frame) error {
	if in == nil {
		return errors.New("spectrum: input channel is nil")
	}
	if out == nil {
		return errors.New("spectrum: output channel is nil")
	}

	// Accumulator: build up FFTSize samples from streaming chunks
	// before processing one Frame. Chunks rarely align with FFTSize,
	// so we copy-fill.
	pending := make([]complex64, 0, p.size)
	nextTick := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			// Rate-limit: drop the chunk if we're not due for a
			// frame yet. Cheap — no FFT, no copy.
			if time.Now().Before(nextTick) {
				p.dropped.Add(1)
				continue
			}
			// Accumulate.
			needed := p.size - len(pending)
			if len(chunk) < needed {
				pending = append(pending, chunk...)
				continue
			}
			pending = append(pending, chunk[:needed]...)
			// Emit a frame.
			frame := p.frame(pending)
			select {
			case out <- frame:
			default:
				// Subscriber is full; drop and keep the
				// producer healthy.
			}
			pending = pending[:0]
			nextTick = time.Now().Add(p.period)
		}
	}
}

// frame runs one FFT over the accumulated samples, returning a fresh
// Frame with FFT-shifted dBFS bins.
func (p *Producer) frame(samples []complex64) Frame {
	bins := make([]float32, p.size)
	PowerDB(p.plan, p.win, p.winSum, samples, p.bufIQ, p.bufOut, bins)
	return Frame{
		Timestamp:  time.Now(),
		CenterHz:   p.centerHz.Load(),
		SampleRate: p.rateHz.Load(),
		Bins:       bins,
	}
}

// PowerDB computes the FFT-shifted dBFS power spectrum of in into dst.
// It is the single source of truth for the spectrum math, shared by the
// streaming Producer and one-shot callers (the Mixer plot's per-window
// FFTs).
//
// All of in, dst, win, bufIn and bufOut must have the same length n — a
// power of two — and plan must be fft.New(n). bufIn/bufOut are caller-
// owned FFT scratch (reused across calls to stay allocation-free); their
// contents are overwritten. winSum is Σ(win²), precomputed once.
//
// Bins are FFT-shifted so dst[0] is the lowest frequency (-Fs/2) and
// dst[n/2] is DC, matching Frame's documented layout. Normalization is
// power = |X|² / (n · Σ(w²)), so a unit-amplitude tone reads ~0 dBFS at
// its bin.
func PowerDB(plan fft.Plan, win []float64, winSum float64, in []complex64, bufIn, bufOut []complex128, dst []float32) {
	n := len(win)
	// Window + convert complex64 → complex128.
	for i := 0; i < n; i++ {
		s := in[i]
		w := win[i]
		bufIn[i] = complex(float64(real(s))*w, float64(imag(s))*w)
	}
	out := plan.Forward(bufOut, bufIn)

	// Magnitude (dBFS), FFT-shifted so DC is in the middle.
	norm := 1.0 / (float64(n) * winSum)
	half := n / 2
	for i := 0; i < n; i++ {
		// FFT-shift: bin i in the output maps to index (i + half) mod N
		// so DC (i=0 in unshifted) lands at index `half`.
		dstIdx := (i + half) % n
		mag := cmplx.Abs(out[i])
		power := mag * mag * norm
		// Clamp before log to avoid -Inf.
		if power < 1e-30 {
			power = 1e-30
		}
		dst[dstIdx] = float32(10 * math.Log10(power))
	}
}
