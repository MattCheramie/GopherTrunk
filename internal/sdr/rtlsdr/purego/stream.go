package purego

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// Async-bulk geometry copied verbatim from the CGO driver
// (internal/sdr/rtlsdr/rtlsdr_cgo.go:42-45). 32 buffers × 16 KiB ≈
// 6 ms of headroom at 2.4 MS/s. Changing these requires a
// matching update on the C side until PR-09 deletes that file.
const (
	asyncBufCount = 32
	asyncBufLen   = 16 * 1024

	// streamChanDepth is the consumer-channel depth. The CGO driver
	// (rtlsdr_cgo.go:190) used 8; the pure-Go backend deliberately runs
	// deeper — 32 chunks ≈ 110 ms at 2.4 MS/s — to absorb GC/scheduler
	// jitter on the consumer (e.g. the ccdecoder resample+decode loop)
	// without dropping live IQ. See issue #489: a CC SDR was shedding
	// 25–48 % of chunks under a momentarily-busy consumer. Drop-on-overrun
	// semantics in [Device.deliver] still rely on this being non-zero;
	// deeper only adds slack, it is not a substitute for the per-chunk
	// allocation removal (the ring in [Device.ring]) that landed with it.
	streamChanDepth = 32

	// streamBufferLatency is how much IQ the consumer channel is sized
	// to hold at the stream's sample rate (issue #1256), with
	// streamChanDepth as the floor: 400 ms is ~118 full chunks at
	// 2.4 MS/s. A phone's scheduler can pause the consumer for longer
	// than the old fixed 32 chunks (~110 ms at 2.4 MS/s) covered. Same
	// value as soapyremote's streamBufferLatency and iqtap's
	// SubBufferLatency (#1242: size buffers by time, not by chunk count).
	streamBufferLatency = 400 * time.Millisecond

	// bulkInEndpoint is the RTL-SDR bulk-IN endpoint address. Hardware
	// invariant — every RTL2832U exposes the IQ stream here.
	bulkInEndpoint = 0x81
)

// streamDepthFor sizes the consumer channel to hold streamBufferLatency
// of IQ at rateHz in full asyncBufLen chunks, never fewer than
// streamChanDepth (also the answer when the rate is unknown).
func streamDepthFor(rateHz uint32) int {
	perChunk := float64(asyncBufLen / 2)
	depth := int(math.Ceil(streamBufferLatency.Seconds() * float64(rateHz) / perChunk))
	return max(depth, streamChanDepth)
}

// StreamIQ resets the USB FIFO, kicks off a bulk-IN ring on the
// transport, and returns a buffered channel of complex64 samples.
// The channel closes when ctx cancels or [Device.Close] is called.
//
// Behaviour invariants — kept identical to rtlsdr_cgo.go:181-217 so
// the new driver is a drop-in replacement:
//
//   - Returns an error if a stream is already active on this Device.
//   - Buffered channel (see streamDepthFor); senders use non-blocking select with
//     a default branch so slow consumers result in dropped buffers
//     rather than back-pressure on the kernel.
//   - The reaper goroutine inside [usb.Transport.StartBulkIn]
//     dispatches one onPacket call per completed URB.
//   - On ctx cancellation, [Device.cancelStream] runs StopBulkIn and
//     closes the channel.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.streamMu.Lock()
	defer d.streamMu.Unlock()
	for {
		if d.closed.Load() {
			return nil, ErrClosed
		}
		if d.out == nil {
			break
		}
		// A stream is still registered. If its owner already cancelled
		// it, the teardown goroutine is on its way (StopBulkIn can take
		// a while to reap the URBs): wait for it rather than failing, so
		// cancel-then-reopen (a scanner advancing to the next channel) is
		// not a race (issue #1261, the airspy family's #686). A stream
		// whose ctx is still live is a genuine second consumer — refuse.
		if d.streamCtx == nil || d.streamCtx.Err() == nil || d.streamDone == nil {
			return nil, errors.New("rtlsdr: stream already active")
		}
		done := d.streamDone
		d.streamMu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			d.streamMu.Lock()
			return nil, ctx.Err()
		}
		d.streamMu.Lock()
	}
	// hwMu from here to StartBulkIn: a runtime USB-reset recovery
	// (recover.go) must never run between the buffer reset and the bulk
	// ring starting.
	d.hwMu.Lock()
	defer d.hwMu.Unlock()
	if err := d.demod.ResetBuffer(); err != nil {
		return nil, fmt.Errorf("rtlsdr: reset buffer: %w", err)
	}
	depth := streamDepthFor(d.demod.GetSampleRate())
	out := make(chan []complex64, depth)
	d.out = out
	d.streamCtx = ctx
	d.streamDone = make(chan struct{})
	d.stopOnce = sync.Once{}
	// Drop any cause from a previous stream so it can't be misattributed to
	// this session's eventual (possibly clean) end.
	d.streamDeadErr.Store(nil)

	// Provision the reuse ring for this stream (issue #489). Sized
	// depth+2: at most depth buffers can sit queued on out plus one in
	// the consumer, so cycling through depth+2 slots guarantees the
	// producer never overwrites a buffer still in flight (the slot being
	// packed is the one not yet queued).
	// Tied to the stream lifetime — a fresh ring per stream means a slow
	// consumer draining the *previous* (closed) channel can never alias a
	// buffer the new stream is writing.
	ring := make([][]complex64, depth+2)
	for i := range ring {
		ring[i] = make([]complex64, asyncBufLen/2)
	}
	d.ring = ring
	d.ringIdx = 0
	d.ringFill = 0

	d.streaming.Store(true)
	if err := d.transport.StartBulkIn(bulkInEndpoint, asyncBufCount, asyncBufLen, d.deliver, d.onStreamDead); err != nil {
		d.streaming.Store(false)
		d.out = nil
		d.streamCtx = nil
		d.streamDone = nil
		return nil, fmt.Errorf("rtlsdr: StartBulkIn: %w", err)
	}

	// Cancel goroutine: when ctx fires, tear the stream down. Mirrors
	// rtlsdr_cgo.go:211-214.
	go func() {
		<-ctx.Done()
		d.cancelStream()
	}()

	return out, nil
}

// onStreamDead is the [usb.Transport] StartBulkIn callback fired when
// every bulk-IN URB dies of an unrecoverable USB error and the reaper
// exits without StopBulkIn being called. It runs cancelStream so the
// consumer channel closes and the IQ consumer (e.g. ccdecoder) sees a
// real EOF rather than hanging forever. Idempotent via d.stopOnce.
// The underlying error surfaces upstream as ccdecoder.ErrIQStreamClosed;
// the daemon logs it once and decides whether to retry / exit. See
// issue #345. The cause is recorded (StreamDeadCause) so that log line names
// the concrete USB error instead of a generic message.
func (d *Device) onStreamDead(cause error) {
	if cause != nil {
		d.streamDeadErr.Store(&cause)
	}
	d.cancelStream()
}

// deliver receives one bulk-IN buffer from the transport reaper,
// converts the u8 IQ pairs into complex64, and pushes the result
// onto the consumer channel. Drop-on-overrun if the consumer is
// behind. Bit-identical conversion math to rtlsdr_cgo.go:225-240
// so downstream DSP sees the same bit pattern from either backend.
//
// Short packets are packed (issue #1256). The queue's headroom is
// depth × chunk length, and the chunk length is whatever the kernel
// completes a URB with: asyncBufLen on a desktop, but a few KiB on the
// reporter's Android phone, where the same depth held ~30 ms of IQ
// instead of ~160 ms and any consumer pause dropped data. So while the
// consumer is caught up (queue empty) a packet goes out at once, as
// before; once anything is queued, packets are appended to the current
// ring slot and the slot is queued only when full, so every queued
// chunk holds a full slot of IQ whatever size the kernel hands back.
func (d *Device) deliver(buf []byte) {
	d.packets.Add(1)
	d.packetBytes.Add(uint64(len(buf)))
	// The whole critical section runs under streamMu. On Linux/Windows a
	// single reaper goroutine drives deliver, so the lock is uncontended;
	// on Darwin the per-slot reapers (usb_darwin.go) call deliver
	// concurrently, and the lock is what makes the shared reuse ring safe.
	// The conversion is cheap (a u8→complex64 LUT) — the expensive resample
	// lives in the consumer — so serializing it costs effectively nothing.
	d.streamMu.Lock()
	out := d.out
	if out == nil {
		d.streamMu.Unlock()
		return
	}
	if len(d.ring) == 0 {
		// No ring (direct deliver in tests): one fresh chunk per packet.
		dropped := !trySend(out, convertU8IQ(buf))
		d.streamMu.Unlock()
		if dropped {
			d.noteDrop()
		}
		return
	}
	drops := 0
	for n := len(buf) / 2; n > 0; {
		slot := d.ring[d.ringIdx]
		k := min(n, len(slot)-d.ringFill)
		convertU8IQInto(slot[d.ringFill:d.ringFill+k], buf[:2*k])
		buf = buf[2*k:]
		n -= k
		d.ringFill += k
		if d.ringFill < len(slot) && len(out) > 0 {
			continue // consumer behind: keep packing this slot
		}
		if trySend(out, slot[:d.ringFill]) {
			// Advance the ring only on a successful enqueue, so a
			// slot is never recycled until its chunk has been handed
			// to the consumer.
			d.ringIdx = (d.ringIdx + 1) % len(d.ring)
		} else {
			drops++ // queue full: drop this slot's IQ and refill it
		}
		d.ringFill = 0
	}
	d.streamMu.Unlock()
	for ; drops > 0; drops-- {
		d.noteDrop()
	}
}

// trySend enqueues chunk without blocking and reports whether it went.
func trySend(out chan []complex64, chunk []complex64) bool {
	select {
	case out <- chunk:
		return true
	default:
		return false
	}
}

// noteDrop records one dropped chunk. This was silent before issue #402,
// which is why live IQ loss could not be distinguished from RF problems:
// a non-zero iq_underruns_total / drop count during decode points at a
// live-path overrun rather than the air. The dropObserver hook drives the
// operator-facing Prometheus counter + a rate-limited warning — see
// internal/metrics and sdr.SetIQDropObserver. Called outside streamMu so
// the observer callback can't stall the reaper.
func (d *Device) noteDrop() {
	d.dropped.Add(1)
	sdr.NotifyIQDrop(d.info)
}

// u8ToF32 maps each possible u8 IQ byte to its normalized float32 value,
// precomputed once so the hot conversion path is two table lookups per
// sample instead of a subtract+divide (issue #489). Each entry uses the
// exact same float32 expression as the original per-sample math
// ((b-127.5)/127.5), so the result is bit-identical — pinned by
// [TestConvertU8IQ_BitIdenticalWithCGO].
var u8ToF32 [256]float32

func init() {
	for b := 0; b < 256; b++ {
		u8ToF32[b] = (float32(b) - 127.5) / 127.5
	}
}

// convertU8IQInto translates a slice of unsigned-8-bit IQ pairs into
// normalized complex64 samples, writing into dst (which must have room
// for len(buf)/2 samples). Allocation-free — this is the form the hot
// deliver path uses against its reuse ring. The conversion is the
// bit-identical port of rtlsdr_cgo.go:225-240:
//
//   - DC bias of 127.5 (mid-range of u8) is subtracted.
//   - Result is divided by 127.5 to scale to [-1, +1).
func convertU8IQInto(dst []complex64, buf []byte) {
	n := len(buf) / 2
	for i := 0; i < n; i++ {
		dst[i] = complex(u8ToF32[buf[2*i]], u8ToF32[buf[2*i+1]])
	}
}

// convertU8IQ is the allocating convenience wrapper around
// convertU8IQInto, used where a fresh slice is wanted (tests, the
// no-ring fallback in [Device.deliver]).
//
// Pinned by [TestConvertU8IQ_BitIdenticalWithCGO] so any drift from
// the CGO driver shows up immediately.
func convertU8IQ(buf []byte) []complex64 {
	out := make([]complex64, len(buf)/2)
	convertU8IQInto(out, buf)
	return out
}
