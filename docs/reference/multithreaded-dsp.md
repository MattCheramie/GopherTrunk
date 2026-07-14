---
slug: multithreaded-dsp
title: Multithreaded DSP
entry_type: concept
category: sdr-programming
description: "Multithreaded DSP spreads signal-processing work across CPU cores using per-block and pipeline parallelism, the main way SDR software scales to high sample rates."
keywords: multithreaded DSP, parallel DSP, pipeline parallelism, data parallelism, per-block threads, worker pool, concurrency, thread pool, SDR performance, multicore
aka: [multi-threaded DSP, parallel DSP, concurrent DSP]
autolink: true
infobox:
  - { label: Type, value: DSP parallelization strategy }
  - { label: Idea, value: Spread blocks/stages across CPU cores }
  - { label: Forms, value: "Pipeline and data parallelism" }
see_also: [block-scheduler, real-time-dsp, concurrency, goroutines, vectorization-simd]
cite_urls:
  - https://en.wikipedia.org/wiki/Data_parallelism
  - https://en.wikipedia.org/wiki/Pipeline_(computing)
---

**Multithreaded DSP** is the use of multiple CPU threads to process a signal stream in
parallel, so that a chain running faster than one core can sustain is split across several
cores.[^dp] As sample rates and channel counts climb, a single-threaded
[SDR](/reference/software-defined-radio/) pipeline eventually saturates one core; multithreading
is how software radio scales onto the many cores of a modern processor while still meeting the
[real-time](/reference/real-time-dsp/) deadline of consuming samples as fast as the radio
produces them.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="Pipeline parallelism runs different processing stages on different threads simultaneously, while data parallelism runs the same stage on several blocks across a worker pool." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="16" font-size="9" fill="currentColor">pipeline parallelism: stages on separate threads</text>
  <rect x="20" y="26" width="56" height="22" fill="none" stroke="currentColor"/><text x="48" y="40" font-size="7.5" fill="currentColor" text-anchor="middle">filter T1</text>
  <line x1="78" y1="37" x2="102" y2="37" stroke="currentColor" marker-end="url(#mtar)"/>
  <rect x="104" y="26" width="56" height="22" fill="none" stroke="currentColor"/><text x="132" y="40" font-size="7.5" fill="currentColor" text-anchor="middle">demod T2</text>
  <line x1="162" y1="37" x2="186" y2="37" stroke="currentColor" marker-end="url(#mtar)"/>
  <rect x="188" y="26" width="56" height="22" fill="none" stroke="currentColor"/><text x="216" y="40" font-size="7.5" fill="currentColor" text-anchor="middle">decode T3</text>
  <text x="20" y="86" font-size="9" fill="currentColor">data parallelism: same stage, many blocks, worker pool</text>
  <rect x="20" y="96" width="46" height="20" fill="none" stroke="currentColor"/><text x="43" y="109" font-size="7" fill="currentColor" text-anchor="middle">split</text>
  <line x1="68" y1="106" x2="92" y2="96" stroke="currentColor" marker-end="url(#mtar)"/>
  <line x1="68" y1="106" x2="92" y2="116" stroke="currentColor" marker-end="url(#mtar)"/>
  <line x1="68" y1="106" x2="92" y2="136" stroke="currentColor" marker-end="url(#mtar)"/>
  <rect x="94" y="88" width="70" height="16" fill="none" stroke="currentColor"/><text x="129" y="100" font-size="7" fill="currentColor" text-anchor="middle">worker A</text>
  <rect x="94" y="110" width="70" height="16" fill="none" stroke="currentColor"/><text x="129" y="122" font-size="7" fill="currentColor" text-anchor="middle">worker B</text>
  <rect x="94" y="132" width="70" height="16" fill="none" stroke="currentColor"/><text x="129" y="144" font-size="7" fill="currentColor" text-anchor="middle">worker C</text>
  <line x1="166" y1="96" x2="190" y2="112" stroke="currentColor" marker-end="url(#mtar)"/>
  <line x1="166" y1="118" x2="190" y2="116" stroke="currentColor" marker-end="url(#mtar)"/>
  <line x1="166" y1="140" x2="190" y2="120" stroke="currentColor" marker-end="url(#mtar)"/>
  <rect x="192" y="104" width="46" height="20" fill="none" stroke="currentColor"/><text x="215" y="117" font-size="7" fill="currentColor" text-anchor="middle">merge</text>
  <defs><marker id="mtar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Two ways to parallelize: run consecutive stages on different threads so they overlap (pipeline), or run one stage on many blocks at once across a pool of workers (data-parallel), then reorder the outputs.</figcaption>
</figure>

## How it works

There are two fundamental ways to split DSP work across threads, and real systems combine them.

- **Pipeline parallelism.** Assign each processing stage — filter, down-convert, demodulate,
  decode — to its own thread. While the decode thread works on block *n*, the demod thread is
  already on block *n+1* and the filter on *n+2*, so all cores stay busy on different parts of
  the stream at once. Throughput rises to roughly the *slowest* stage's rate, and stages
  communicate through queues ([ring buffers](/reference/ring-buffer/)). The limits are the
  bottleneck stage and the cost of hand-offs.
- **Data parallelism.** Take one expensive stage and run it on several independent blocks
  concurrently across a worker pool — block 0 to worker A, block 1 to worker B, and so on.
  This scales past the pipeline's slowest-stage limit for work that has no cross-block state,
  but it requires *reordering*: outputs finish out of sequence and must be reassembled into the
  original stream order.

Both forms live or die on synchronization. Threads must exchange data without data races,
which means lock-free queues or careful mutexes, and every lock or cache-line bounce between
cores costs time. State is the enemy of parallelism: a stage that carries filter memory or a
running phase accumulator cannot simply be sharded across blocks, so the natural parallel
boundaries fall between independent channels or at stateless stages. A
[block scheduler](/reference/block-scheduler/) decides which stage runs on which thread when,
balancing load and keeping the pipeline from stalling.

## In practice

The cleanest parallelism in an SDR receiver is usually *per channel*: once a wideband capture
is split into several narrowband channels, each channel's decode chain is independent and maps
to its own thread with no shared state — a natural fit for a scanner watching many channels.
Within a single channel, pipeline parallelism across the stages is the common structure, with
data-parallel workers reserved for genuinely stateless heavy lifting (large FFTs, batched
correlation). Multithreading is orthogonal to [SIMD vectorization](/reference/vectorization-simd/):
threads scale across cores while SIMD scales within a core's inner loop, and a well-tuned DSP
program uses both. It is also a source of subtle bugs — races, deadlocks, and priority
inversion — which is why disciplined [concurrency](/reference/concurrency/) primitives matter
as much as raw core count.

## Relevance to SDR

Every serious SDR framework is multithreaded: GNU Radio gives each block its own thread and a
scheduler moves data between them; channelized scanners run one decoder thread per channel.
**GopherTrunk** is written in [Go](/reference/go-language/), whose
[goroutines](/reference/goroutines/) and channels make this style natural — lightweight
concurrent workers connected by typed queues rather than manual OS threads and locks. GopherTrunk
uses that model to run its capture, down-conversion, and per-channel decode concurrently, so
the control-channel decoder and voice decoders progress in parallel and the runtime spreads
them across available cores. This is one of the places the language genuinely helps an SDR
application: the concurrency model that Go was built around maps directly onto the
pipeline-and-per-channel parallelism that real-time software radio needs. The honest caveat is
that concurrency is not free performance — it only helps when the work is genuinely parallel and
the synchronization overhead stays below the savings.

## Sources

[^dp]: [Data parallelism](https://en.wikipedia.org/wiki/Data_parallelism) — Wikipedia, on splitting identical work across processing units, the basis of data-parallel DSP.
