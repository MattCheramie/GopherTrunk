package purego

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

	// bulkInEndpoint is the RTL-SDR bulk-IN endpoint address. Hardware
	// invariant — every RTL2832U exposes the IQ stream here.
	bulkInEndpoint = 0x81
)

// StreamIQ resets the USB FIFO, kicks off a bulk-IN ring on the
// transport, and returns a buffered channel of complex64 samples.
// The channel closes when ctx cancels or [Device.Close] is called.
//
// Behaviour invariants — kept identical to rtlsdr_cgo.go:181-217 so
// the new driver is a drop-in replacement:
//
//   - Returns an error if a stream is already active on this Device.
//   - 8-deep buffered channel; senders use non-blocking select with
//     a default branch so slow consumers result in dropped buffers
//     rather than back-pressure on the kernel.
//   - The reaper goroutine inside [usb.Transport.StartBulkIn]
//     dispatches one onPacket call per completed URB.
//   - On ctx cancellation, [Device.cancelStream] runs StopBulkIn and
//     closes the channel.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.streamMu.Lock()
	defer d.streamMu.Unlock()
	if d.closed.Load() {
		return nil, ErrClosed
	}
	if d.out != nil {
		return nil, errors.New("rtlsdr: stream already active")
	}
	if err := d.demod.ResetBuffer(); err != nil {
		return nil, fmt.Errorf("rtlsdr: reset buffer: %w", err)
	}
	out := make(chan []complex64, streamChanDepth)
	d.out = out
	d.stopOnce = sync.Once{}

	// Provision the reuse ring for this stream (issue #489). Sized
	// streamChanDepth+2: at most streamChanDepth buffers can sit queued
	// on out plus one in the consumer, so cycling through depth+2 slots
	// guarantees the producer never overwrites a buffer still in flight.
	// Tied to the stream lifetime — a fresh ring per stream means a slow
	// consumer draining the *previous* (closed) channel can never alias a
	// buffer the new stream is writing.
	ring := make([][]complex64, streamChanDepth+2)
	for i := range ring {
		ring[i] = make([]complex64, asyncBufLen/2)
	}
	d.ring = ring
	d.ringIdx = 0

	if err := d.transport.StartBulkIn(bulkInEndpoint, asyncBufCount, asyncBufLen, d.deliver, d.onStreamDead); err != nil {
		d.out = nil
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
// issue #345.
func (d *Device) onStreamDead(error) {
	d.cancelStream()
}

// deliver receives one bulk-IN buffer from the transport reaper,
// converts the u8 IQ pairs into complex64, and pushes the result
// onto the consumer channel. Drop-on-overrun if the consumer is
// behind. Bit-identical conversion math to rtlsdr_cgo.go:225-240
// so downstream DSP sees the same bit pattern from either backend.
func (d *Device) deliver(buf []byte) {
	// The whole critical section runs under streamMu. On Linux/Windows a
	// single reaper goroutine drives deliver, so the lock is uncontended;
	// on Darwin the per-slot reapers (usb_darwin.go) call deliver
	// concurrently, and the lock is what makes the shared reuse ring safe.
	// The conversion is cheap (a u8→complex64 LUT) — the expensive resample
	// lives in the consumer — so serializing it costs effectively nothing
	// against the USB ring's ~110 ms of buffering headroom.
	d.streamMu.Lock()
	out := d.out
	if out == nil {
		d.streamMu.Unlock()
		return
	}
	samples := d.fillBufferLocked(buf)
	select {
	case out <- samples:
		// Advance the ring only on a successful enqueue: a dropped chunk
		// reuses the same buffer next time, so a buffer's slot is never
		// recycled until that chunk has actually been handed to the
		// consumer.
		if len(d.ring) > 0 {
			d.ringIdx = (d.ringIdx + 1) % len(d.ring)
		}
		d.streamMu.Unlock()
	default:
		d.streamMu.Unlock()
		// Consumer can't keep up. Drop the whole chunk and record it:
		// this was silent before issue #402, which is why live IQ loss
		// could not be distinguished from RF problems. A non-zero
		// iq_underruns_total / drop count during decode points at a
		// live-path overrun rather than the air. The dropObserver hook
		// drives the operator-facing Prometheus counter + a
		// rate-limited warning — see internal/metrics and
		// sdr.SetIQDropObserver. NotifyIQDrop runs outside the lock so the
		// observer callback can't stall the reaper.
		d.dropped.Add(1)
		sdr.NotifyIQDrop(d.info)
	}
}

// fillBufferLocked converts one bulk-IN buffer into complex64 samples,
// reusing the current ring slot to avoid a ~64 KiB allocation per chunk
// (issue #489: that churn drove GC pauses that pushed the CC consumer
// over its real-time budget and shed live IQ). Falls back to a fresh
// allocation when no ring is provisioned (direct deliver in tests) or for
// the rare oversized packet that would not fit a ring slot. Caller must
// hold streamMu.
func (d *Device) fillBufferLocked(buf []byte) []complex64 {
	n := len(buf) / 2
	if len(d.ring) == 0 {
		return convertU8IQ(buf)
	}
	dst := d.ring[d.ringIdx]
	if cap(dst) < n {
		// Packet larger than a ring slot — don't disturb the ring,
		// just allocate this one. Should not happen for asyncBufLen URBs.
		return convertU8IQ(buf)
	}
	dst = dst[:n]
	convertU8IQInto(dst, buf)
	return dst
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
// no-ring fallback in [Device.fillBufferLocked]).
//
// Pinned by [TestConvertU8IQ_BitIdenticalWithCGO] so any drift from
// the CGO driver shows up immediately.
func convertU8IQ(buf []byte) []complex64 {
	out := make([]complex64, len(buf)/2)
	convertU8IQInto(out, buf)
	return out
}
