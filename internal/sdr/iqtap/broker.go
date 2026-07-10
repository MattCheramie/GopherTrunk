// Package iqtap fans an SDR's IQ stream out to additional observers
// without disturbing the primary consumer's StreamIQ contract.
//
// GopherTrunk's per-dongle StreamIQ is single-consumer: one goroutine
// owns the bulk-IN URB reaper (or the file replay timer) and a second
// caller would conflict at the USB / driver layer. Several trunking-
// adjacent features added on top of the existing scanner — live
// spectrum, paging decoders, an rtl_tcp server, signal-domain
// diagnostics — want a copy of the same IQ stream the primary is
// already reading.
//
// A Broker wraps the device's StreamIQ. The primary consumer (the
// control-channel decoder or the conventional scanner) keeps its
// existing API: one StreamIQ call returns one channel, exactly as
// today. Secondary observers register with Subscribe and receive
// chunk copies on a persistent channel that stays alive across
// multiple StreamIQ sessions (e.g. across USB-disconnect retries).
//
// Slow subscribers don't back-pressure the primary: per-subscriber
// channels are bounded and delivery is non-blocking — drops are
// counted, not blocking. The primary path stays zero-copy; secondaries
// receive freshly-allocated slices so they can hold or mutate without
// affecting the primary or each other.
//
// SDR replacement (pool.Reacquire after a USB disconnect) is handled
// via SetInner: the daemon calls SetInner on the broker after
// Reacquire so the next StreamIQ session reads from the fresh handle.
// Active sessions on the old handle drain naturally when the old
// channel closes.
package iqtap

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// DefaultSubBuffer is the per-subscriber channel depth (in IQ chunks)
// used when New is called with subBuffer == 0. At a typical 2.048 MS/s
// rate and ~16 KiB chunks (~4 ms), 16 chunks ≈ 60 ms of buffering —
// enough to absorb a typical SSE/WS frame's worth of jitter without
// dropping, but small enough that a wedged consumer falls behind
// fast enough to be visible in the drop counter.
const DefaultSubBuffer = 16

// primaryHandoffDepth is the buffer depth (in IQ chunks) of the broker's
// primary handoff channel returned by StreamIQ. A small buffer lets the
// fan-out goroutine stay ahead of a momentarily-busy primary consumer (the
// per-chunk fanout copy plus a brief decode hiccup) without stalling its
// drain of the inner device's bounded reaper buffer — which would otherwise
// force the driver to drop whole IQ chunks mid-frame (issue #402). It stays
// small so the inner driver's buffer remains the dominant latency bound and
// a wedged consumer still falls behind fast enough to surface in the drop
// counter.
const primaryHandoffDepth = 2

// Broker wraps an sdr.Device, exposing the same Device interface
// while internally fanning each IQ chunk out to any registered
// Subscribers. The primary consumer that calls StreamIQ sees no
// behavioural change beyond a single extra goroutine hop.
type Broker struct {
	log *slog.Logger

	innerMu sync.RWMutex
	inner   sdr.Device

	subsMu      sync.Mutex
	subs        map[*Subscriber]struct{}
	subBuffer   int
	subsDropped atomic.Uint64

	streamActive atomic.Bool

	// centerHz / rateHz record the most-recent successful
	// SetCenterFreq / SetSampleRate values. Used by the spectrum
	// publisher to stamp frames with the correct frequency context
	// without forcing every sdr.Device implementation to grow a
	// CenterFreq() getter. Setters that fail leave these untouched.
	centerHz atomic.Uint32
	rateHz   atomic.Uint32
}

// Subscriber is a secondary observer's handle. C delivers chunk
// copies while a primary stream is active. The channel stays open
// across multiple StreamIQ sessions — chunks simply pause when no
// primary stream is running and resume on the next StreamIQ call.
//
// Close to unregister. Idempotent.
type Subscriber struct {
	C       <-chan []complex64
	ch      chan []complex64
	dropped atomic.Uint64
	b       *Broker
	closed  atomic.Bool

	// sendMu serialises a non-blocking send (trySend) against Close so a
	// fanout send can never race the close(ch): Close holds sendMu while
	// it closes the channel, and trySend re-checks closed under sendMu
	// before sending. The atomic closed flag remains the cheap, lock-free
	// pre-check used by fanout and Stats.
	sendMu sync.Mutex
}

// Stats reports broker-level counters. Useful for /metrics and for
// diagnosing slow secondary consumers.
type Stats struct {
	Subscribers  int
	DroppedTotal uint64
	Streaming    bool
}

// New wraps inner. subBuffer is the per-subscriber channel depth in
// IQ chunks; pass 0 to use DefaultSubBuffer.
func New(inner sdr.Device, subBuffer int, log *slog.Logger) *Broker {
	if log == nil {
		log = slog.Default()
	}
	if subBuffer <= 0 {
		subBuffer = DefaultSubBuffer
	}
	return &Broker{
		log:       log,
		inner:     inner,
		subs:      make(map[*Subscriber]struct{}),
		subBuffer: subBuffer,
	}
}

// Inner returns the currently-wrapped device. Useful for tests and for
// callers that need to bypass the broker (e.g. the watchdog).
func (b *Broker) Inner() sdr.Device {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	return b.inner
}

// SetInner swaps the wrapped device. The daemon calls this after
// pool.Reacquire replaces the underlying SDR handle so the next
// StreamIQ call reads from the fresh device. An in-flight StreamIQ
// session on the old handle is untouched — it drains naturally when
// the old handle's channel closes.
func (b *Broker) SetInner(inner sdr.Device) {
	b.innerMu.Lock()
	b.inner = inner
	b.innerMu.Unlock()
}

// sdr.Device pass-throughs. All forward to the currently-wrapped
// inner. Concurrent SetInner is safe — each call grabs a fresh read
// lock on inner.

func (b *Broker) Info() sdr.Info {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	return b.inner.Info()
}

// FreqRange forwards the wrapped device's tuning range when it exposes
// one (sdr.FreqRanger); otherwise it reports (0,0) so callers fall back to
// an explicit band.
func (b *Broker) FreqRange() (minHz, maxHz uint32) {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	if fr, ok := b.inner.(sdr.FreqRanger); ok {
		return fr.FreqRange()
	}
	return 0, 0
}

func (b *Broker) SetCenterFreq(hz uint32) error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	if err := b.inner.SetCenterFreq(hz); err != nil {
		return err
	}
	b.centerHz.Store(hz)
	return nil
}

func (b *Broker) SetSampleRate(hz uint32) error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	if err := b.inner.SetSampleRate(hz); err != nil {
		return err
	}
	b.rateHz.Store(hz)
	return nil
}

// CenterHz returns the most-recent value successfully programmed via
// SetCenterFreq. Zero before the first SetCenterFreq call. Used by
// the spectrum publisher to stamp frames with frequency context.
func (b *Broker) CenterHz() uint32 { return b.centerHz.Load() }

// SampleRateHz returns the most-recent value successfully programmed
// via SetSampleRate. Zero before the first call.
func (b *Broker) SampleRateHz() uint32 { return b.rateHz.Load() }

// Seed populates the centerHz / rateHz caches without calling into the
// inner device. The daemon uses it to inject the bootstrap sample rate
// the pool already programmed on the raw device before the broker was
// wrapped around it, so spectrum frames stamp the correct rate from
// the very first frame. Zero arguments leave the corresponding field
// unchanged so callers can seed one field at a time.
func (b *Broker) Seed(centerHz, rateHz uint32) {
	if centerHz != 0 {
		b.centerHz.Store(centerHz)
	}
	if rateHz != 0 {
		b.rateHz.Store(rateHz)
	}
}

func (b *Broker) SetGain(tenthDB int) error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	return b.inner.SetGain(tenthDB)
}

func (b *Broker) SetPPM(ppm int) error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	return b.inner.SetPPM(ppm)
}

func (b *Broker) SetBiasTee(enable bool) error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	return b.inner.SetBiasTee(enable)
}

func (b *Broker) Close() error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	return b.inner.Close()
}

// StreamDeadCause forwards the wrapped device's stream-death cause when it
// exposes the optional interface{ StreamDeadCause() error } extension
// (airspy / rtlsdr-purego), otherwise nil. The wideband engine streams the
// dongle through the broker, so without this pass-through the concrete USB
// error that killed the reaper would never reach the daemon's "IQ stream died"
// log — the same gap the broker had for ActualSampleRate. It reads the
// currently-wrapped inner: the engine queries it on stream-close before the
// daemon swaps in a reacquired handle, so it observes the dead device's cause.
func (b *Broker) StreamDeadCause() error {
	b.innerMu.RLock()
	defer b.innerMu.RUnlock()
	if c, ok := b.inner.(interface{ StreamDeadCause() error }); ok {
		return c.StreamDeadCause()
	}
	return nil
}

// StreamIQ opens a stream against the currently-wrapped inner, starts
// a fan-out goroutine that copies each chunk to every active
// Subscriber, and returns the primary's channel. Multiple sequential
// calls are supported — each opens a fresh inner stream and a fresh
// primary channel, matching the contract every real driver
// implementation honours.
//
// The primary's channel has a small buffer (primaryHandoffDepth) so the
// fan-out goroutine isn't stalled by a momentarily-busy primary consumer;
// sustained back-pressure from a slow primary still propagates upstream
// (i.e. the underlying driver's USB reaper buffer absorbs jitter; if it
// fills, the driver drops). Secondaries are decoupled via their own bounded
// channels and never block the primary.
func (b *Broker) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	b.innerMu.RLock()
	inner := b.inner
	b.innerMu.RUnlock()
	if inner == nil {
		return nil, errors.New("iqtap: broker has no inner device")
	}
	in, err := inner.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	b.streamActive.Store(true)

	out := make(chan []complex64, primaryHandoffDepth)
	go func() {
		defer close(out)
		defer b.streamActive.Store(false)
		// A panic in fanout would otherwise crash the daemon silently
		// (issue #492); recover into a logged close so the primary
		// consumer sees a clean stream end instead.
		defer gtlog.Recover(b.log, "iqtap-fanout", nil)
		for chunk := range in {
			b.fanout(chunk)
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// fanout copies the chunk to every subscriber's channel. Non-blocking
// per subscriber — drops are counted but never block the primary.
func (b *Broker) fanout(chunk []complex64) {
	b.subsMu.Lock()
	if len(b.subs) == 0 {
		b.subsMu.Unlock()
		return
	}
	// Snapshot the subscriber set so we can drop subsMu before allocating
	// + sending. The closed-vs-send race is handled per subscriber by
	// trySend / Close sharing s.sendMu — not by subsMu — so a concurrent
	// Subscribe / Close here is safe.
	subs := make([]*Subscriber, 0, len(b.subs))
	for s := range b.subs {
		subs = append(subs, s)
	}
	b.subsMu.Unlock()

	for _, s := range subs {
		// Cheap lock-free early-out: skip the allocation for a
		// subscriber that's already closed. trySend re-checks under
		// s.sendMu for correctness.
		if s.closed.Load() {
			continue
		}
		cp := make([]complex64, len(chunk))
		copy(cp, chunk)
		s.trySend(cp)
	}
}

// trySend delivers a pre-copied chunk to the subscriber's channel without
// blocking. s.sendMu serialises the non-blocking send against Close: Close
// holds the same mutex while it closes the channel, so re-checking closed
// under the lock here guarantees the send can never land on a closed
// channel. Drops (full buffer) are counted, never blocking.
func (s *Subscriber) trySend(cp []complex64) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.closed.Load() {
		return
	}
	select {
	case s.ch <- cp:
	default:
		s.dropped.Add(1)
		s.b.subsDropped.Add(1)
	}
}

// Subscribe registers a new secondary observer. Caller must Close the
// returned Subscriber when done — leaked subscribers keep the broker
// fanning frames into a buffer nobody reads, eventually saturating
// the drop counter (visible but wasteful).
func (b *Broker) Subscribe() *Subscriber {
	ch := make(chan []complex64, b.subBuffer)
	s := &Subscriber{C: ch, ch: ch, b: b}
	b.subsMu.Lock()
	b.subs[s] = struct{}{}
	b.subsMu.Unlock()
	return s
}

// Stats reports broker-level counters.
func (b *Broker) Stats() Stats {
	b.subsMu.Lock()
	count := len(b.subs)
	b.subsMu.Unlock()
	return Stats{
		Subscribers:  count,
		DroppedTotal: b.subsDropped.Load(),
		Streaming:    b.streamActive.Load(),
	}
}

// Close removes the subscriber and closes its channel. Idempotent.
// After Close, reads from Subscriber.C return the zero value with
// ok==false.
func (s *Subscriber) Close() {
	if s.closed.Swap(true) {
		return
	}
	// Close the channel under sendMu so it can't interleave a concurrent
	// trySend. closed is already true (set above), so any trySend that
	// acquires sendMu after this returns without sending.
	s.sendMu.Lock()
	close(s.ch)
	s.sendMu.Unlock()

	s.b.subsMu.Lock()
	delete(s.b.subs, s)
	s.b.subsMu.Unlock()
}

// Dropped returns the cumulative chunks dropped on this subscriber.
// A non-zero value indicates the consumer is falling behind the
// broker's primary delivery rate.
func (s *Subscriber) Dropped() uint64 { return s.dropped.Load() }

// Ensure Broker satisfies sdr.Device.
var _ sdr.Device = (*Broker)(nil)
