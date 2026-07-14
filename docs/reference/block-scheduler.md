---
slug: block-scheduler
title: Block scheduler
entry_type: concept
category: sdr-programming
description: "A block scheduler is the runtime that drives a flowgraph, deciding when to call each block's work function and managing the buffers of samples between blocks."
keywords: block scheduler, GNU Radio scheduler, flowgraph runtime, buffer scheduling, thread per block, work function, back-pressure, dataflow scheduling, TPB
aka: [scheduler, flowgraph runtime, block scheduler]
autolink: true
infobox:
  - { label: Type, value: Dataflow runtime }
  - { label: Job, value: Drive blocks + schedule buffers }
  - { label: Used in, value: "GNU Radio, block-based SDR runtimes" }
see_also: [flowgraph, back-pressure, sample-buffer, signal-processing-block, real-time-dsp, dsp-latency]
cite_urls:
  - https://wiki.gnuradio.org/index.php/BlocksCodingGuide
  - https://en.wikipedia.org/wiki/Dataflow_programming
---

**A block scheduler is the runtime that drives a [flowgraph](/reference/flowgraph/),
deciding when to invoke each block's work function and managing the sample buffers that sit
between blocks.**[^guide] The blocks describe *what* the DSP does; the scheduler decides *when*
each block runs, *how much* data it gets, and *where* its output goes. It is the engine that
turns a static graph of [signal-processing blocks](/reference/signal-processing-block/) into a
running radio, and its design largely determines the throughput and latency of the whole
system.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A scheduler coordinating three blocks: between each pair of blocks is a buffer, and the scheduler decides when each block runs based on how full the downstream buffer is, applying back-pressure when a buffer is nearly full." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="18" y="60" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="48" y="78">block A</text>
    <rect x="200" y="60" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="78">block B</text>
    <rect x="382" y="60" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="412" y="78">block C</text>
    <rect x="100" y="66" width="78" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><text x="139" y="79">buffer</text>
    <rect x="282" y="66" width="78" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><text x="321" y="79">buffer</text>
    <rect x="150" y="120" width="160" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="230" y="135">scheduler</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="78" y1="75" x2="99" y2="75" marker-end="url(#bsar)"/>
    <line x1="178" y1="75" x2="199" y2="75" marker-end="url(#bsar)"/>
    <line x1="260" y1="75" x2="281" y2="75" marker-end="url(#bsar)"/>
    <line x1="360" y1="75" x2="381" y2="75" marker-end="url(#bsar)"/>
  </g>
  <g stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3" fill="none">
    <line x1="180" y1="120" x2="60" y2="91"/>
    <line x1="230" y1="120" x2="230" y2="91"/>
    <line x1="280" y1="120" x2="410" y2="91"/>
  </g>
</svg>
<figcaption>The scheduler watches each inter-block buffer and calls a block's work function only when it has enough input and room downstream to produce output.</figcaption>
</figure>

## How it works

The scheduler's core loop asks, for each block: does it have enough input available, and is
there enough free space in its output buffer to write results? If both are true, it calls the
block's work function with pointers to the available input and output, then advances the buffer
read and write pointers by however much the block reported consuming and producing.

Two coupled resources decide when a block may run:

- **Input availability** — a block cannot run until upstream has produced enough samples (a
  decimator that needs 8 inputs per output waits for at least 8).
- **Output space** — a block cannot run if the downstream [buffer](/reference/sample-buffer/)
  is full. This is [back-pressure](/reference/back-pressure/): a slow consumer stalls its
  producer, which naturally throttles the whole upstream chain so no buffer overflows.

Because each edge has a finite buffer, the scheduler never lets a fast block run away and
exhaust memory — the graph self-regulates to the speed of its slowest stage.

## In practice

GNU Radio's classic scheduler is **thread-per-block (TPB)**: every block gets its own OS
thread, and threads block and wake on buffer condition variables. This maps cleanly onto
multi-core CPUs — independent branches of the graph run in parallel — but incurs a context
switch and synchronization cost per scheduling decision, so very fine-grained blocks can spend
more time coordinating than computing. Alternative schedulers (GNU Radio's newer runtime work,
and other frameworks) explore single-threaded execution, thread pools, or fused blocks to
reduce that overhead.

Buffer sizing is the scheduler's main latency knob. Large buffers amortize per-call overhead
and raise throughput but add [latency](/reference/dsp-latency/), because a sample sits in the
pipeline longer before it is processed; small buffers cut latency but schedule more often. For
[real-time](/reference/real-time-dsp/) receive from live hardware, the scheduler must also keep
up on average or the source's own buffer overruns and samples are lost.

## Relevance to SDR

Whether or not you use GNU Radio, some component has to answer "which stage runs next, with how
much data" — and that is the scheduler's question. Understanding it explains the two effects
every SDR developer eventually meets: back-pressure (why a slow decoder can stall the whole
chain, or conversely why a stalled sink causes upstream overruns) and the throughput/latency
trade-off buried in buffer sizes.

[GopherTrunk](/reference/software-defined-radio/) does not embed a GNU Radio scheduler; being
pure Go, it lets the Go runtime schedule goroutines and uses buffered channels as the
inter-stage buffers, so back-pressure falls out of channel capacity rather than an explicit
DSP scheduler. The mechanism differs but the invariants are the same: a bounded buffer between
each stage, a consumer that can stall its producer, and buffer size as the throughput-versus-
latency dial. Reading a GopherTrunk decode pipeline with the block-scheduler model in mind
makes it clear why a wedged downstream stage shows up as dropped samples at the radio source.

## Sources

[^guide]: [Blocks Coding Guide](https://wiki.gnuradio.org/index.php/BlocksCodingGuide) — GNU Radio Wiki, on how the runtime calls work()/general_work(), consumes and produces items, and manages inter-block buffers.
