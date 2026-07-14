---
slug: ring-buffer
title: Ring buffer
entry_type: concept
category: sdr-programming
description: A ring buffer is a fixed-size circular queue that decouples a producer from a consumer, the standard structure for carrying a live IQ sample stream between an SDR driver and the DSP that reads it.
keywords: ring buffer, circular buffer, circular queue, FIFO, lock-free, SPSC, single producer single consumer, IQ streaming, sample buffer, wrap-around
aka: [ring buffer, circular buffer, circular queue]
autolink: true
infobox:
  - { label: Type, value: Fixed-size circular queue }
  - { label: Idea, value: "Read/write indices wrap modulo N" }
  - { label: Used in, value: SDR driver-to-DSP handoff }
see_also: [sample-buffer, zero-copy-dsp, overruns-underruns, real-time-dsp, back-pressure]
cite_urls:
  - https://en.wikipedia.org/wiki/Circular_buffer
  - https://rigtorp.se/ringbuffer/
---

**A ring buffer** (circular buffer) is a fixed-size array used as a queue, where a write index and a read index each advance and wrap back to the start when they reach the end, so the storage is reused endlessly without allocation.[^wiki] In software-defined radio it is the standard structure for handing a live [IQ sample](/reference/iq-data/) stream from the code that fills it to the code that drains it, giving each side a stable place to work while the other runs at its own pace.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A circle of eight buffer slots with a write pointer ahead of a read pointer, both moving clockwise and wrapping around, showing the producer filling slots the consumer has not yet read." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g transform="translate(150 100)">
    <circle cx="0" cy="0" r="70" fill="none" stroke="currentColor" stroke-opacity="0.35"/>
    <g stroke="currentColor" stroke-opacity="0.5">
      <line x1="0" y1="-70" x2="0" y2="-52"/><line x1="49" y1="-49" x2="37" y2="-37"/>
      <line x1="70" y1="0" x2="52" y2="0"/><line x1="49" y1="49" x2="37" y2="37"/>
      <line x1="0" y1="70" x2="0" y2="52"/><line x1="-49" y1="49" x2="-37" y2="37"/>
      <line x1="-70" y1="0" x2="-52" y2="0"/><line x1="-49" y1="-49" x2="-37" y2="-37"/>
    </g>
    <g fill="currentColor" fill-opacity="0.25"><path d="M0 0 L0 -70 A70 70 0 0 1 70 0 Z"/><path d="M0 0 L70 0 A70 70 0 0 1 49 49 Z"/></g>
    <path d="M20 -78 A78 78 0 0 1 78 20" fill="none" stroke="currentColor" marker-end="url(#rbar)"/>
    <text x="8" y="-84" font-size="11" fill="currentColor">write</text>
    <text x="-92" y="4" font-size="11" fill="currentColor">read</text>
    <line x1="-70" y1="0" x2="-84" y2="0" stroke="currentColor"/>
  </g>
  <g transform="translate(300 60)" font-size="12" fill="currentColor">
    <rect x="0" y="0" width="16" height="16" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
    <text x="24" y="13">filled, awaiting read</text>
    <rect x="0" y="30" width="16" height="16" fill="none" stroke="currentColor"/>
    <text x="24" y="43">free, available to write</text>
  </g>
</svg>
<figcaption>The producer's write index chases the consumer's read index around a fixed ring; the gap between them is the buffered backlog.</figcaption>
</figure>

## How it works

A ring buffer is a plain array of length *N* plus two positions. The **producer** writes a sample at the write index and advances it; the **consumer** reads at the read index and advances it. Both indices are taken modulo *N*, so when either reaches the end it wraps to slot 0 and keeps going — the memory is a logical circle. The buffer is empty when the two indices are equal and full when the writer has caught up to one slot behind the reader, so a capacity-*N* array usually stores *N−1* items to keep those two states distinguishable.

The reason this structure dominates streaming is that it needs **no allocation and no copying of the backlog** in steady state. The array is allocated once; every push and pop is a couple of index arithmetic operations. Because the producer only ever touches the write index and the consumer only the read index, the common **single-producer / single-consumer (SPSC)** case can be made *lock-free*: with the indices published through atomic loads and stores (and the right memory ordering) each side sees the other's progress without a mutex, so the DSP thread never blocks waiting for a lock held by the USB thread.[^rigtorp]

## In practice

- **Sizing.** Capacity is a latency-versus-safety trade. A larger ring rides out longer scheduling stalls before it overflows, but every buffered sample is added [latency](/reference/dsp-latency/). Radios are typically sized to hold tens of milliseconds of IQ.
- **Contiguity.** DSP kernels like to process a run of samples in one call, but a run near the end of the array wraps. Implementations either hand out the two pieces separately, or use a mirrored/virtual-memory mapping so the wrap looks contiguous — a common enabler of [zero-copy](/reference/zero-copy-dsp/) reads.
- **False sharing.** For lock-free rings the read and write indices are padded onto separate cache lines so the two threads' atomic updates don't ping-pong the same line.

## Relevance to SDR

An SDR front end delivers samples on the schedule of the USB or network transport — in bursty chunks, from a callback or a driver thread you do not control. The DSP downstream wants a smooth, arbitrary-sized supply. A ring buffer sits between them as the classic [sample buffer](/reference/sample-buffer/): the driver's delivery thread is the producer, the demodulator is the consumer, and the ring absorbs the timing mismatch. When the consumer falls behind and the writer laps the reader, that is an [overrun](/reference/overruns-underruns/); when the consumer outruns a slow producer, an underrun. Sizing and monitoring the ring is therefore how a real-time SDR keeps up with its sample rate.

**GopherTrunk** is a pure-Go SDR application and leans on exactly this pattern. Its device drivers stream IQ into buffers that the scanner and decoders drain independently, and the drivers expose a dropped-sample counter so a live overrun is visible rather than silent — the airspy path, for example, tracks discarded IQ chunks explicitly. Go's channels are a higher-level bounded queue with similar decoupling semantics, but for the tight, high-rate sample path a direct ring over a slice avoids per-item overhead. The [back-pressure](/reference/back-pressure/) a full ring applies is what ultimately protects the pipeline from unbounded memory growth.

## Sources

[^wiki]: [Circular buffer](https://en.wikipedia.org/wiki/Circular_buffer) — Wikipedia, on the circular-buffer data structure, wrap-around indexing, and full/empty conditions.
[^rigtorp]: [Correctly implementing a lock-free SPSC queue](https://rigtorp.se/ringbuffer/) — Erik Rigtorp, on a cache-friendly, lock-free single-producer/single-consumer ring buffer.
