---
title: "RF Front End, Part 9: RTL-SDR III — IQ Streaming & the GC-Churn Bug"
description: "Sustained IQ streaming from the RTL2832U in pure Go — 32 async USB buffers, a deep consumer channel, and a bit-identical U8-to-complex64 lookup table. Plus the headline bug: per-chunk allocations whose GC churn shed a quarter of the control channel's live IQ, and the zero-allocation reuse ring that fixed it."
category: deep-dives
tags: [sdr, go, rtl-sdr, streaming, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Front End"
series_part: 9
---

*Part 9 of **RF Front End**. The RTL2832U is initialized
([Part 7]({{ '/blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/' | relative_url }}))
and the tuner is locked
([Part 8]({{ '/blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/' | relative_url }})).
Now we have to keep a 2.4 MS/s flood of samples moving without dropping any — and
this is where Go's garbage collector, of all things, became the enemy.*

> **TL;DR** — This is sustained 2.4 MS/s IQ streaming off the RTL2832U: 32 async
> USB buffers, a deep consumer channel, and a bit-identical U8-to-complex64
> lookup table. The headline bug: per-chunk allocations drove GC churn that shed
> 25–48% of live IQ — fixed with a zero-allocation reuse ring. Two more bugs turn
> silent IQ loss and a silently dead stream into explicit, observable failures.

## In this post

- **The streaming geometry** — 32 async USB buffers × 16 KiB, a deep consumer
  channel, and why the numbers are what they are.
- **The U8-to-complex64 conversion** — a precomputed lookup table that's
  bit-identical to the old CGO driver.
- **The problem we hit (issue #489):** per-chunk allocations created GC churn
  that pushed the control-channel decoder over its real-time budget and shed
  25–48% of live IQ — fixed with a zero-allocation reuse ring.
- **Two more failure-signaling bugs (issues #402 and #345):** silent live IQ loss
  made invisible by a drop counter, and a silent USB-reaper death that left the
  consumer blocked forever.

## What sustained streaming does

Once the chip is tuned, the RTL2832U dumps a continuous stream of unsigned 8-bit
IQ pairs over a USB bulk-IN endpoint. At 2.4 MS/s that's 4.8 MB/s of raw bytes
that never stops. The streaming layer's job is to pull those bytes off the wire,
convert each U8 IQ pair into a `complex64`, and hand the result to the consumer
(a control-channel decoder, a trunking engine, an IQ recorder) as a channel of
`[]complex64` chunks — without ever blocking the kernel's USB reaper and without
dropping samples when the consumer hiccups.

The whole thing lives in `internal/sdr/rtlsdr/purego/stream.go`, with the device
state machine in `device.go`.

## How GopherTrunk implements it in Go

### The geometry

**The numbers are copied verbatim from the old CGO driver so the pure-Go backend is
a behavioral drop-in:**

```go
// internal/sdr/rtlsdr/purego/stream.go
const (
    asyncBufCount = 32
    asyncBufLen   = 16 * 1024

    streamChanDepth = 32
    bulkInEndpoint  = 0x81
)
```

Thirty-two async USB buffers of 16 KiB each is about **6 ms of headroom at 2.4
MS/s** — enough that a brief scheduler stall doesn't starve the bulk-IN ring. The
consumer channel is `streamChanDepth = 32` chunks deep, roughly 110 ms of slack.
The CGO driver used a depth of 8; the pure-Go backend deliberately runs deeper,
specifically to absorb GC and scheduler jitter on the consumer — and *that*
number is part of the story below.

`StreamIQ` resets the USB FIFO, provisions a reuse ring (more on that in a
moment), kicks off the bulk-IN ring on the transport, and returns the channel:

```go
// internal/sdr/rtlsdr/purego/stream.go (shape)
out := make(chan []complex64, streamChanDepth)
d.out = out
// ...provision the reuse ring...
if err := d.transport.StartBulkIn(bulkInEndpoint, asyncBufCount, asyncBufLen,
    d.deliver, d.onStreamDead); err != nil {
    d.out = nil
    return nil, fmt.Errorf("rtlsdr: StartBulkIn: %w", err)
}
go func() { <-ctx.Done(); d.cancelStream() }()
return out, nil
```

### The conversion

Each completed URB calls `deliver`, which converts the U8 IQ bytes into
`complex64`. The conversion math is the bit-identical port of the CGO driver:
subtract a DC bias of 127.5 (mid-range of a `u8`) and divide by 127.5 to scale to
[-1, +1). Done naively that's a subtract and a divide per sample, 4.8 million
times a second per dongle. Instead we precompute a 256-entry lookup table once:

```go
// internal/sdr/rtlsdr/purego/stream.go
var u8ToF32 [256]float32

func init() {
    for b := 0; b < 256; b++ {
        u8ToF32[b] = (float32(b) - 127.5) / 127.5
    }
}

func convertU8IQInto(dst []complex64, buf []byte) {
    n := len(buf) / 2
    for i := 0; i < n; i++ {
        dst[i] = complex(u8ToF32[buf[2*i]], u8ToF32[buf[2*i+1]])
    }
}
```

Each entry uses the *exact same* float32 expression as the original per-sample
math, so the result is bit-identical — pinned by
`TestConvertU8IQ_BitIdenticalWithCGO` so any drift from the CGO driver shows up
immediately. The hot path is now two table lookups per sample instead of a
subtract+divide.

## The problem we hit: GC churn shedding live IQ (issue #489)

**The symptom.** A control-channel RTL-SDR was dropping **25–48% of its IQ
chunks** under load. Not occasionally — sustained. The decoder downstream was
clearly busy, but it should have had 110 ms of channel slack to ride out a busy
moment. Something was making the consumer *systematically* unable to keep up.

**The root cause.** `convertU8IQInto`'s allocating wrapper was the culprit.
Originally, `deliver` allocated a fresh `[]complex64` for every chunk:

```go
// internal/sdr/rtlsdr/purego/stream.go
func convertU8IQ(buf []byte) []complex64 {
    out := make([]complex64, len(buf)/2)
    convertU8IQInto(out, buf)
    return out
}
```

At 16 KiB per URB that's an 8192-element `complex64` slice — ~64 KiB — allocated
fresh, hundreds of times a second, then thrown away as soon as the consumer
finished with it. That allocation rate drove the garbage collector hard, and the
GC pauses landed *on the consumer goroutine*, pushing the control-channel decoder
over its real-time budget. The decoder fell behind, the channel filled, and
`deliver`'s drop-on-overrun branch started shedding chunks — a quarter to half of
them. The deep channel didn't help because the problem wasn't a transient stall;
it was steady-state allocation pressure that the channel depth couldn't paper
over.

**The fix.** A ring of preallocated `complex64` buffers, reused across chunks.
`StreamIQ` provisions `streamChanDepth + 2` of them per stream:

```go
// internal/sdr/rtlsdr/purego/stream.go
ring := make([][]complex64, streamChanDepth+2)
for i := range ring {
    ring[i] = make([]complex64, asyncBufLen/2)
}
d.ring = ring
d.ringIdx = 0
```

The sizing is exact: at most `streamChanDepth` buffers can sit queued on the
channel, plus one in the consumer, so cycling through `depth + 2` slots
guarantees the producer never overwrites a buffer still in flight. `deliver` now
fills the current ring slot instead of allocating, and — critically — only
advances the ring index on a *successful* enqueue:

```go
// internal/sdr/rtlsdr/purego/stream.go
samples := d.fillBufferLocked(buf)
select {
case out <- samples:
    // Advance the ring only on a successful enqueue: a dropped chunk
    // reuses the same buffer next time, so a buffer's slot is never
    // recycled until that chunk has actually been handed to the consumer.
    if len(d.ring) > 0 {
        d.ringIdx = (d.ringIdx + 1) % len(d.ring)
    }
    d.streamMu.Unlock()
default:
    d.streamMu.Unlock()
    d.dropped.Add(1)
    sdr.NotifyIQDrop(d.info)
}
```

`fillBufferLocked` writes into the ring slot via the allocation-free
`convertU8IQInto`, falling back to a fresh allocation only when no ring is
provisioned (direct deliver in tests) or for a packet too large to fit a slot:

```go
// internal/sdr/rtlsdr/purego/stream.go
func (d *Device) fillBufferLocked(buf []byte) []complex64 {
    n := len(buf) / 2
    if len(d.ring) == 0 {
        return convertU8IQ(buf)
    }
    dst := d.ring[d.ringIdx]
    if cap(dst) < n {
        return convertU8IQ(buf) // oversized packet; don't disturb the ring
    }
    dst = dst[:n]
    convertU8IQInto(dst, buf)
    return dst
}
```

With the ring in place the steady state allocates **nothing** per chunk, the GC
went quiet, the consumer kept its budget, and the drop rate fell to zero. The
deeper channel depth from earlier is slack on top of this — not a substitute for
it.

## The second problem: silent live IQ loss (issue #402)

**The symptom.** When the control channel *did* drop IQ, there was no way to tell.
A decode failure looked identical whether it came from a weak signal, multipath,
or the live path silently shedding chunks. Live IQ loss was indistinguishable
from an RF problem.

**The root cause and fix.** The drop branch in `deliver` was genuinely silent — it
threw the chunk away and moved on. We added a lifetime drop counter and a
notification hook:

```go
// internal/sdr/rtlsdr/purego/device.go
dropped atomic.Uint64

func (d *Device) DroppedChunks() uint64 { return d.dropped.Load() }
```

Now the drop branch records every discard and notifies an observer outside the
lock (so the callback can't stall the reaper):

```go
// internal/sdr/rtlsdr/purego/stream.go (drop branch)
d.dropped.Add(1)
sdr.NotifyIQDrop(d.info)
```

`NotifyIQDrop` drives an operator-facing Prometheus counter
(`iq_underruns_total`) plus a rate-limited warning. A non-zero count during a
decode failure now points squarely at a live-path overrun rather than the air — a
diagnosis you simply could not make before. (The same issue-#402 work also
exposed `ActualSampleRate`, because the down-converter has to derive its symbol
clock from the rate the chip is *actually* delivering, not the requested one — but
that's a decode-path story.)

## The third problem: a silently dead stream (issue #345)

**The symptom.** When a dongle fell off the USB bus mid-stream — unplugged, or an
unrecoverable `ENODEV`/`EPROTO` — the consumer goroutine would block on the IQ
channel **forever**. No error, no EOF, just a permanent hang.

**The root cause.** The USB reaper goroutine inside `StartBulkIn` exits when every
bulk-IN URB dies of an unrecoverable error. But it exits *without* anyone calling
`StopBulkIn`, so the consumer channel was never closed. The consumer's `range`
over the channel had nothing to wake it.

**The fix.** A stream-death callback. `StartBulkIn` takes an `onStreamDead`
handler that fires exactly when the reaper exits abnormally, and it runs
`cancelStream`:

```go
// internal/sdr/rtlsdr/purego/stream.go
func (d *Device) onStreamDead(error) {
    d.cancelStream()
}
```

`cancelStream` is the same idempotent teardown that context-cancel uses — guarded
by a `sync.Once`, it stops the bulk-IN ring and **closes the consumer channel**:

```go
// internal/sdr/rtlsdr/purego/device.go
func (d *Device) cancelStream() {
    d.stopOnce.Do(func() {
        _ = d.transport.StopBulkIn()
        d.streamMu.Lock()
        if d.out != nil {
            close(d.out)
            d.out = nil
        }
        d.streamMu.Unlock()
    })
}
```

Now a dead stream surfaces to the consumer as a closed channel — a clean EOF — so
the decoder sees `ErrIQStreamClosed`, the daemon logs it once and decides whether
to retry or exit. The watchdog that does the actual reconnect is the subject of a
later post; here the point is that *failure became visible* instead of becoming a
hang.

## The design principle: zero-allocation steady state + explicit failure signaling

The streaming layer is governed by two rules that show up over and over in
real-time Go: **the hot path should allocate nothing in steady state**, and
**every failure mode should produce a signal, never silence.**

The GC-churn bug is the canonical lesson in the first rule. The allocation wasn't
wrong — it was correct, simple Go — but at 2.4 MS/s its *aggregate* cost was a GC
load heavy enough to break an unrelated real-time consumer two layers away. The
fix wasn't cleverer code; it was *no* allocation, achieved by reusing buffers
whose lifecycle is tied precisely to the channel's depth.

The drop counter and the stream-death callback are the second rule. A dropped
chunk and a dead stream were both *silent* — and silence in a real-time pipeline
is the worst possible failure, because it's indistinguishable from "working." Both
fixes turn an invisible failure into an explicit, observable one.

### How that principle shaped the Go code

- **Buffers are pooled, not allocated.** The reuse ring is sized exactly to the
  number of buffers that can be in flight (`streamChanDepth + 2`), so steady-state
  IQ delivery allocates nothing and the GC stays out of the consumer's budget.
- **The ring advances only on success.** A dropped chunk reuses its slot, so a
  buffer is never recycled until the consumer has actually taken it — backpressure
  and reuse-safety in one rule.
- **Drops are counted and reported.** `dropped atomic.Uint64` plus `NotifyIQDrop`
  turn a silent overrun into a Prometheus counter and a rate-limited warning,
  fired outside the lock so telemetry can't stall the reaper.
- **Stream death closes the channel.** `onStreamDead → cancelStream` makes an
  unrecoverable USB error surface as a clean EOF, so no consumer ever blocks
  forever. `sync.Once` makes the teardown idempotent across ctx-cancel, close, and
  reaper-death.
- **Conversion stays bit-identical.** The reuse ring changed *where* samples land,
  never *what* they are — `convertU8IQInto` is the same math as the CGO driver,
  pinned by a golden test.

## Where this goes next

That closes out the RTL-SDR trilogy: chip bring-up, tuner, and now sustained
streaming. The next family streams differently — the
[Airspy]({{ '/reference/airspy/' | relative_url }}) R2 and Mini deliver **real**
samples, not complex IQ, so
[Part 10]({{ '/blog/deep-dives/rf-front-end-10-airspy-real-to-complex/' | relative_url }})
takes apart the real-to-complex-baseband conversion that turns a real ADC stream
into the `[]complex64` the rest of the pipeline expects.

## FAQ

**Why not just use `sync.Pool` instead of a hand-rolled ring?**
`sync.Pool` doesn't give the lifetime guarantee we need: a buffer must not be
reused until the consumer has finished with it, and that's determined by the
channel depth, not by GC timing. The ring sized `streamChanDepth + 2` encodes
exactly that invariant — and a fresh ring per stream means a slow consumer
draining the *previous* (closed) channel can never alias a buffer the new stream
is writing.

**Why advance the ring index only on a successful send?**
Because a dropped chunk's buffer was never handed to anyone, so its slot is still
safe to overwrite next time. Advancing on drop would waste a slot and, worse,
could recycle a slot while its chunk is still queued. Advancing only on success
ties the ring's recycling directly to delivery.

**Does the drop-on-overrun policy lose data we care about?**
Dropping is deliberate: a slow consumer should shed live IQ rather than
back-pressure the kernel's USB reaper, which would stall every stream. The fix for
issue #489 was to make sure the consumer *isn't* slow in steady state; the
drop-and-count path is the safety valve when it momentarily is, and the counter
tells you when it fired.

## Series navigation

**Part 9 of 14** · ←
[Part 8]({{ '/blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/' | relative_url }})
· Next →
[Part 10: Airspy R2/Mini & HF+ — real samples to complex baseband]({{ '/blog/deep-dives/rf-front-end-10-airspy-real-to-complex/' | relative_url }})
