---
slug: sample-buffer
title: Sample buffer
entry_type: concept
category: sdr-programming
description: "A sample buffer is the producer/consumer memory region between two DSP blocks that decouples their rates, enabling flow control, back-pressure, and overrun handling."
keywords: sample buffer, DSP buffer, producer consumer, inter-block buffer, ring buffer, circular buffer, back-pressure, overrun, underrun, buffer sizing, flow control
aka: [DSP buffer, inter-block buffer, sample buffer]
autolink: true
infobox:
  - { label: Type, value: Producer/consumer memory }
  - { label: Job, value: Decouple block rates + flow-control }
  - { label: Common form, value: Ring buffer }
see_also: [ring-buffer, back-pressure, overruns-underruns, block-scheduler, dsp-latency, zero-copy-dsp]
cite_urls:
  - https://en.wikipedia.org/wiki/Circular_buffer
  - https://wiki.gnuradio.org/index.php/BlocksCodingGuide
---

**A sample buffer is the shared memory region between two DSP [blocks](/reference/signal-processing-block/)
that one block writes samples into and the next reads out of.**[^circ] It is the edge of a
[flowgraph](/reference/flowgraph/) made concrete: every connection between blocks is backed by a
buffer, and that buffer is what lets a producer and a consumer run at slightly different speeds
without either losing data. Buffers are where the practical realities of streaming SDR live —
flow control, [back-pressure](/reference/back-pressure/), latency, and the
[overruns and underruns](/reference/overruns-underruns/) that show up when the rates diverge too
far.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 155" role="img" aria-label="A producer block writing samples into a buffer at a write pointer while a consumer block reads them out at a read pointer; the gap between the pointers is the fill level, and the buffer decouples the two blocks' rates." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sbfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="15" y="55" width="66" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="48" y="75">producer</text>
    <rect x="379" y="55" width="66" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="412" y="75">consumer</text>
  </g>
  <rect x="120" y="58" width="220" height="28" rx="3" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <g fill="currentColor"><rect x="120" y="58" width="120" height="28" opacity="0.18"/></g>
  <g stroke="currentColor" stroke-width="1"><line x1="120" y1="52" x2="120" y2="92"/><line x1="240" y1="52" x2="240" y2="92"/></g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle"><text x="120" y="46">write ptr</text><text x="240" y="46">read ptr</text><text x="230" y="106">fill level = written − read</text></g>
  <line x1="81" y1="72" x2="119" y2="72" stroke="currentColor" stroke-width="1.2" marker-end="url(#sbfar)"/>
  <line x1="340" y1="72" x2="378" y2="72" stroke="currentColor" stroke-width="1.2" marker-end="url(#sbfar)"/>
</svg>
<figcaption>A buffer decouples producer and consumer: the write pointer leads, the read pointer trails, and the gap between them is how much data is queued.</figcaption>
</figure>

## How it works

A sample buffer is a fixed-size region with a write pointer (where the producer appends) and a
read pointer (where the consumer takes from). The producer may only write up to the free space;
the consumer may only read up to what has been written. The difference between the pointers is
the *fill level* — the amount of queued, not-yet-consumed data.

Its purpose is *decoupling*. Real DSP stages do not run in perfect lockstep: a block runs in
bursts as the [scheduler](/reference/block-scheduler/) gives it CPU, a hardware source delivers
samples in USB transfers, a consumer occasionally stalls on I/O. The buffer absorbs that jitter
so short-term speed differences do not stall or starve the neighbours. Two failure modes bound
it:

- **Overrun** — the producer wants to write but the buffer is full because the consumer is too
  slow. Something must give: block the producer (back-pressure) or drop samples.
- **Underrun** — the consumer wants to read but the buffer is empty because the producer is too
  slow. The consumer stalls or emits a gap.

Which behaviour is chosen distinguishes the two regimes. Between software blocks, a full buffer
usually *blocks* the producer — [back-pressure](/reference/back-pressure/) — so no data is lost;
throughput just settles to the slowest stage. At a real-time hardware boundary the source cannot
be paused, so a full buffer means dropped samples: an
[overrun](/reference/overruns-underruns/).

## In practice

Most inter-block buffers are [ring buffers](/reference/ring-buffer/) — circular arrays where the
pointers wrap around, giving continuous streaming without ever copying data back to the front.
GNU Radio goes further with a double-mapped ("vmcircbuf") trick: the same physical pages are
mapped twice, back to back, so a wrapped read still looks contiguous and blocks can process
across the wrap with no special-casing — a form of [zero-copy](/reference/zero-copy-dsp/)
streaming.

Buffer **size** is the central tuning knob and a direct throughput-versus-latency trade. Larger
buffers tolerate more jitter and amortize per-call overhead (higher throughput) but hold more
samples in flight, raising [latency](/reference/dsp-latency/). Smaller buffers cut latency but
schedule more often and are less forgiving of a momentary stall. Real-time receive from live
hardware forces the issue: the buffer must be large enough that the consumer never falls
irrecoverably behind, or samples are lost at the antenna.

## Relevance to SDR

Buffers are unavoidable in streaming SDR — any two stages running at their own pace need one
between them — so nearly every SDR performance question routes through buffer behaviour. "Why am
I getting overruns" is a buffer question; "why is my latency high" is often a buffer-size
question; "why does one slow block wedge the whole chain" is back-pressure through a buffer.

[GopherTrunk](/reference/software-defined-radio/) is a real pure-Go SDR application and depends
on exactly these mechanics: buffered Go channels and slices carry IQ and symbols between its
decode stages, the Go runtime provides back-pressure when a channel fills, and the radio-source
boundary is where overruns manifest if the decode chain can't keep up — the same failure the ring
buffer and overrun concepts describe. GopherTrunk's decode chain is also
[rate-invariant](/reference/software-defined-radio/) at its core: it normalizes to a fixed
per-protocol channel rate, so buffer sizing is governed by that steady output rate rather than by
whatever the capture rate happened to be. Understanding sample buffers is thus directly load-
bearing for reasoning about GopherTrunk's real-time behaviour, not just GNU Radio's.

## Sources

[^circ]: [Circular buffer](https://en.wikipedia.org/wiki/Circular_buffer) — Wikipedia, on the producer/consumer ring-buffer structure with wrapping read and write pointers used to decouple DSP stages.
