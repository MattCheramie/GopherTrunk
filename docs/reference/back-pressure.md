---
slug: back-pressure
title: Back-pressure
entry_type: concept
category: sdr-programming
description: Back-pressure is flow control in which a slow consumer signals upstream to slow down, keeping a streaming DSP pipeline's buffers bounded instead of overflowing when one stage falls behind.
keywords: back-pressure, backpressure, flow control, bounded buffer, blocking queue, credit, demand, streaming, pipeline, producer consumer, throttle
aka: [back-pressure, backpressure, flow control]
autolink: true
infobox:
  - { label: Type, value: Flow-control mechanism }
  - { label: Idea, value: Slow consumer throttles the producer }
  - { label: Prevents, value: Unbounded buffering, sample drops }
see_also: [block-scheduler, sample-buffer, throttle-block, ring-buffer, overruns-underruns, real-time-dsp]
cite_urls:
  - https://en.wikipedia.org/wiki/Back_pressure
  - https://www.reactive-streams.org/
---

**Back-pressure** is a flow-control strategy in which a stage that is falling behind pushes a signal *upstream* so its producer slows down, rather than letting an unbounded backlog pile up between them.[^wiki] In a streaming DSP pipeline it is what keeps every buffer bounded: when the demodulator can only process so many samples per second, back-pressure ensures the source is throttled to that rate instead of the buffer overflowing into an [overrun](/reference/overruns-underruns/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Three pipeline blocks — source, filter, and a slow demod — connected by bounded buffers, with a dashed back-pressure signal running upstream from the slow block to throttle the source." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="12" fill="currentColor">
    <rect x="10" y="40" width="90" height="40" fill="none" stroke="currentColor"/><text x="30" y="64">source</text>
    <rect x="150" y="40" width="90" height="40" fill="none" stroke="currentColor"/><text x="176" y="64">filter</text>
    <rect x="290" y="40" width="110" height="40" fill="none" stroke="currentColor"/><text x="300" y="58">demod</text><text x="300" y="72" font-size="9">(slow)</text>
    <line x1="100" y1="60" x2="148" y2="60" stroke="currentColor" marker-end="url(#bpar)"/>
    <line x1="240" y1="60" x2="288" y2="60" stroke="currentColor" marker-end="url(#bpar)"/>
    <path d="M345 82 C345 120, 55 120, 55 82" fill="none" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.7" marker-end="url(#bpar)"/>
    <text x="150" y="132" font-size="11" fill="currentColor">back-pressure: "slow down"</text>
  </g>
</svg>
<figcaption>Demand flows upstream: the slow demod's inability to accept more data propagates back to throttle the source, so no buffer between them grows without bound.</figcaption>
</figure>

## How it works

The mechanism is a **bounded** buffer between each pair of stages. When the buffer fills, the producer can no longer deposit into it and must wait — that wait *is* the back-pressure. There are two common shapes:

- **Blocking (implicit).** The producer calls a push that simply blocks when the [ring buffer](/reference/ring-buffer/) is full, or writes to a bounded channel that parks the sender until a slot frees. No explicit message travels upstream; the buffer's fullness is the signal, and the producer's own thread stalls until the consumer catches up.
- **Demand-based (explicit).** The consumer announces how much it can accept — a credit or request count — and the producer sends no more than that. This is the model formalized by Reactive Streams, where a subscriber requests *n* items and the publisher is contractually forbidden from exceeding outstanding demand.[^rs]

Either way, the steady-state effect is the same: the whole chain runs at the rate of its slowest stage, and memory use stays bounded because nobody is allowed to get arbitrarily far ahead.

## In practice

Back-pressure is the right tool when the source *can* be slowed — a file, a socket you can stop reading, or a synthetic generator. It cannot save you when the source is a free-running ADC: a live radio's clock does not accept "slow down," so an over-full pipeline there must drop samples (overrun) instead. This is the key distinction an SDR author has to keep straight. Back-pressure also interacts with [latency](/reference/dsp-latency/): a deep buffer with back-pressure trades responsiveness for smoothness, while a shallow one throttles sooner but reacts faster. A pipeline that mixes real-time and non-real-time sources typically applies back-pressure everywhere except the live front end, and monitors that front end for drops.

## Relevance to SDR

Block-based frameworks build back-pressure into their [scheduler](/reference/block-scheduler/): a downstream block that stops consuming leaves its input buffer full, the upstream block's output buffer then fills, and the [scheduler](/reference/block-scheduler/) simply does not run a block that has no room to write — demand propagates backward for free. When there is no hardware to set the pace, a deliberate [throttle](/reference/throttle-block/) block is inserted so the flowgraph self-limits to real time instead of spinning as fast as the CPU allows.

**GopherTrunk** relies on this pattern when the source is throttleable. Its `.cfile` replay path reads from a file, so a slow decoder naturally back-pressures the reader — reads just proceed at the decode's pace, which is why replay is deterministic and easy to test. Against a live radio, GT cannot back-pressure the ADC, so it instead sizes its buffers and counts drops, letting the [overrun](/reference/overruns-underruns/) counter reveal when a consumer stage has fallen behind. Go's bounded channels give the same blocking-producer semantics on the paths where GT does control the source.

## Sources

[^wiki]: [Back pressure](https://en.wikipedia.org/wiki/Back_pressure) — Wikipedia, on back-pressure as a flow-control concept in pipelines and information systems.
[^rs]: [Reactive Streams](https://www.reactive-streams.org/) — Reactive Streams specification, on demand-based (request-*n*) back-pressure between publishers and subscribers.
