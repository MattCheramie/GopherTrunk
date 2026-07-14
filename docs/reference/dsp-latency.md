---
slug: dsp-latency
title: DSP latency
entry_type: concept
category: sdr-programming
description: "DSP latency is the delay between a sample entering a processing chain and the corresponding output emerging, set mainly by buffer and block sizes."
keywords: DSP latency, buffer latency, block latency, throughput vs latency, pipeline delay, group delay, real-time DSP, block size, buffering, end-to-end delay
aka: [processing latency, pipeline latency, buffer latency]
autolink: true
infobox:
  - { label: Type, value: Timing property of a DSP chain }
  - { label: Idea, value: Delay from input sample to output sample }
  - { label: Dominated by, value: Block/buffer size and stage count }
see_also: [real-time-dsp, sample-buffer, ring-buffer, block-scheduler, overruns-underruns]
cite_urls:
  - https://en.wikipedia.org/wiki/Latency_(audio)
  - https://en.wikipedia.org/wiki/Group_delay_and_phase_delay
---

**DSP latency** is the elapsed time between a sample arriving at the input of a
digital-signal-processing chain and the corresponding processed sample leaving the
output.[^wiki] In a [software-defined radio](/reference/software-defined-radio/) it is the
lag between antenna-borne energy being captured and a decoded symbol, audio sample, or
detection appearing downstream. It is distinct from *throughput*: a chain can sustain a very
high sample rate yet still hold each sample for a long time before releasing it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Samples flow through three processing stages, each adding a fixed block delay, so the total latency is the sum of every stage's buffer while throughput stays constant." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="20" font-size="9" fill="currentColor">input samples</text>
  <g fill="currentColor"><circle cx="24" cy="40" r="3"/><circle cx="44" cy="40" r="3"/><circle cx="64" cy="40" r="3"/><circle cx="84" cy="40" r="3"/></g>
  <rect x="110" y="26" width="70" height="28" fill="none" stroke="currentColor"/><text x="145" y="44" font-size="8" fill="currentColor" text-anchor="middle">filter</text>
  <rect x="200" y="26" width="70" height="28" fill="none" stroke="currentColor"/><text x="235" y="44" font-size="8" fill="currentColor" text-anchor="middle">demod</text>
  <rect x="290" y="26" width="70" height="28" fill="none" stroke="currentColor"/><text x="325" y="44" font-size="8" fill="currentColor" text-anchor="middle">decode</text>
  <line x1="92" y1="40" x2="108" y2="40" stroke="currentColor" marker-end="url(#dlar)"/>
  <line x1="182" y1="40" x2="198" y2="40" stroke="currentColor" marker-end="url(#dlar)"/>
  <line x1="272" y1="40" x2="288" y2="40" stroke="currentColor" marker-end="url(#dlar)"/>
  <line x1="362" y1="40" x2="392" y2="40" stroke="currentColor" marker-end="url(#dlar)"/>
  <text x="410" y="43" font-size="8" fill="currentColor">out</text>
  <line x1="110" y1="80" x2="360" y2="80" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="110" y1="76" x2="110" y2="84" stroke="currentColor"/><line x1="360" y1="76" x2="360" y2="84" stroke="currentColor"/>
  <text x="235" y="98" font-size="8.5" fill="currentColor" text-anchor="middle">total latency = sum of each stage's buffer delay</text>
  <text x="235" y="120" font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.7">throughput is unchanged — it is samples/second, not delay</text>
  <defs><marker id="dlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Latency accumulates additively across pipeline stages, each of which must fill a buffer before it can emit; throughput, the sustained sample rate, is a separate quantity.</figcaption>
</figure>

## How it works

Latency in a DSP chain comes from three sources that add together.

- **Block / buffer size.** Software rarely processes one sample at a time; it works on blocks
  (frames) held in a [sample buffer](/reference/sample-buffer/). A stage that consumes 4096
  samples per call cannot emit anything until that block is full, so it imposes at least one
  block-time of delay — 4096 samples at 48 kHz is about 85 ms. Larger blocks amortise
  per-call overhead and vectorise better (higher throughput) but push latency up. This is the
  central *throughput-versus-latency* trade.
- **Algorithmic / group delay.** Filters and framing introduce their own delay. A
  linear-phase FIR of length *N* delays the signal by (N−1)/2 samples; a symbol-framer must
  collect a whole frame before it can be validated. This is intrinsic to the maths, not the
  implementation.
- **Queueing.** Between stages, [ring buffers](/reference/ring-buffer/) and thread hand-offs
  hold samples waiting for the next worker. Under load these queues fill, and standing queue
  depth becomes latency; if they overflow you get [overruns](/reference/overruns-underruns/)
  instead.

Total end-to-end latency is roughly the sum of every stage's block delay plus its group
delay plus mean queue occupancy. Halving latency usually means smaller blocks, fewer stages,
or shorter filters — each of which costs throughput headroom or filter quality.

## In practice

The engineering question is always *how much latency is acceptable*, because reducing it is
never free. Interactive or closed-loop applications — a transceiver's TX/RX turnaround, a
feedback control loop, live voice — need latency in the low tens of milliseconds and pay for
it with small blocks and tight scheduling. Batch or monitoring applications — spectrum
logging, offline replay, a scanner that only has to keep up — can use large blocks, deep
queues, and long filters, trading hundreds of milliseconds of delay for efficiency and
robustness against scheduling jitter. Choosing block size is therefore one of the first
architectural decisions in an SDR program, and it ripples through buffer sizing, thread
count, and how much slack the [block scheduler](/reference/block-scheduler/) has before it
underflows.

## Relevance to SDR

Every SDR receiver is a latency budget. Samples cross the USB or network link in bursts,
sit in a driver buffer, get down-converted and filtered, then framed and decoded — each step
a contribution. A trunking scanner like **GopherTrunk** is latency-*tolerant* but
throughput-*critical*: what matters is that the [real-time DSP](/reference/real-time-dsp/)
chain keeps pace with the control channel so no bursts are dropped, while an extra hundred
milliseconds before a talkgroup's audio starts is imperceptible to a listener. So GopherTrunk
favours block sizes large enough to process the control-channel and voice streams efficiently
and to absorb host scheduling jitter, accepting the modest added delay. A latency-sensitive
application — a repeater controller, a phase-locked ranging system, a two-way voice link —
would make the opposite call, shrinking blocks to trim the delay even at the cost of CPU
efficiency. The point applies to any SDR software: latency and throughput are separate axes,
and the block size that sets them is a design choice, not a default to leave unexamined.

## Sources

[^wiki]: [Latency (audio)](https://en.wikipedia.org/wiki/Latency_(audio)) — Wikipedia, on buffering delay in real-time audio/DSP pipelines and the role of block size.
