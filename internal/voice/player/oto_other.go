// Backend factory wiring for non-Linux platforms. Linux runs through
// alsa_linux.go which talks to libasound2.so.2 directly via purego
// (no cgo, no build-time headers required). macOS / Windows / others
// keep using github.com/ebitengine/oto/v3 — oto routes to CoreAudio
// on Darwin and WASAPI on Windows, both via purego, so neither
// platform needs a system audio dev-headers package at build time
// either.

//go:build !linux

package player

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// otoCtx is the process-global oto context. oto.NewContext can only
// be called once per process, so we cache the first context and
// re-use it across daemon restarts (e.g. inside tests that boot the
// daemon repeatedly).
var (
	otoOnce  sync.Once
	otoCtx   *oto.Context
	otoErr   error
	otoRateA uint32 // the sample rate the cached context was opened at
)

func init() {
	defaultBackendFactory = func(cfg Config) (Backend, error) {
		ctx, err := acquireOtoContext(cfg.SampleRate)
		if err != nil {
			return nil, err
		}
		// Reader feeds the oto.Player on demand. We hand it the
		// channel side so the audio goroutine in player.go can
		// push samples without blocking.
		r := newPCMReader()
		p := ctx.NewPlayer(r)
		p.SetBufferSize(otoBufferBytes(cfg))
		p.Play()
		return &otoBackend{
			ctx:    ctx,
			player: p,
			reader: r,
		}, nil
	}
}

// acquireOtoContext returns the cached oto context, creating it on
// the first call. If a context already exists with a different
// sample rate the new request is rejected — oto only supports one
// rate per process.
func acquireOtoContext(rate uint32) (*oto.Context, error) {
	otoOnce.Do(func() {
		opts := &oto.NewContextOptions{
			SampleRate:   int(rate),
			ChannelCount: 1,
			Format:       oto.FormatSignedInt16LE,
		}
		ctx, ready, err := oto.NewContext(opts)
		if err != nil {
			otoErr = err
			return
		}
		<-ready
		otoCtx = ctx
		otoRateA = rate
	})
	if otoErr != nil {
		return nil, fmt.Errorf("oto: %w", otoErr)
	}
	if otoCtx == nil {
		return nil, errors.New("oto: context not initialised")
	}
	if otoRateA != rate {
		return nil, fmt.Errorf("oto: context already opened at %d Hz; %d Hz requested", otoRateA, rate)
	}
	return otoCtx, nil
}

func otoBufferBytes(cfg Config) int {
	ms := cfg.BufferMs
	if ms <= 0 {
		ms = 80
	}
	rate := int(cfg.SampleRate)
	if rate <= 0 {
		rate = 8000
	}
	// 2 bytes per int16, mono.
	return rate * ms / 1000 * 2
}

// otoBackend implements Backend on top of oto.Player.
type otoBackend struct {
	ctx    *oto.Context
	player *oto.Player
	reader *pcmReader

	closeOnce sync.Once
	closeErr  error
}

func (b *otoBackend) Write(samples []int16) error {
	return b.reader.write(samples)
}

func (b *otoBackend) Close() error {
	b.closeOnce.Do(func() {
		b.reader.close()
		b.closeErr = b.player.Close()
	})
	return b.closeErr
}

// pcmReader is the io.Reader oto pulls from. PCM is queued from the
// caller via write() and dequeued from the audio thread via Read().
// A bounded ring drops the oldest samples when the producer
// outpaces the consumer — this only happens in pathological cases
// because the producer is itself queue-bounded in player.go.
type pcmReader struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
	// maxBytes caps the ring at ~1s of audio at 8kHz/mono/int16.
	// Big enough to absorb a stalled output device for a moment,
	// small enough to bound memory.
	maxBytes int
}

func newPCMReader() *pcmReader {
	r := &pcmReader{maxBytes: 16000 * 2}
	r.cond = sync.NewCond(&r.mu)
	return r
}

func (r *pcmReader) write(samples []int16) error {
	if len(samples) == 0 {
		return nil
	}
	bs := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(bs[i*2:], uint16(s))
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return io.ErrClosedPipe
	}
	r.buf = append(r.buf, bs...)
	if over := len(r.buf) - r.maxBytes; over > 0 {
		// drop oldest
		r.buf = r.buf[over:]
	}
	r.cond.Signal()
	r.mu.Unlock()
	return nil
}

func (r *pcmReader) close() {
	r.mu.Lock()
	r.closed = true
	r.cond.Broadcast()
	r.mu.Unlock()
}

// Read drains queued samples. If the queue is empty it waits briefly
// for samples to arrive (so oto isn't spun in a tight loop), and
// returns silence when the wait elapses. Returning silence rather
// than blocking forever keeps the underlying audio device happy
// during inter-call gaps.
func (r *pcmReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.mu.Lock()
	if !r.closed && len(r.buf) == 0 {
		waitUntil := time.Now().Add(20 * time.Millisecond)
		for !r.closed && len(r.buf) == 0 {
			waitDur := time.Until(waitUntil)
			if waitDur <= 0 {
				break
			}
			waitWithTimeout(r.cond, waitDur)
		}
	}
	if r.closed && len(r.buf) == 0 {
		r.mu.Unlock()
		return 0, io.EOF
	}
	if len(r.buf) == 0 {
		r.mu.Unlock()
		// silence: zero-fill p so oto outputs silence between calls.
		for i := range p {
			p[i] = 0
		}
		return len(p), nil
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	r.mu.Unlock()
	return n, nil
}

// waitWithTimeout is a cond.Wait with a timeout. The std-lib
// sync.Cond has no native timeout so we spawn a one-shot goroutine
// that wakes the cond if the duration elapses first. Cheap because
// audio gaps between calls are seconds, not milliseconds.
func waitWithTimeout(c *sync.Cond, d time.Duration) {
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-time.After(d):
			c.L.Lock()
			c.Broadcast()
			c.L.Unlock()
		case <-stop:
		}
	}()
	c.Wait()
}
