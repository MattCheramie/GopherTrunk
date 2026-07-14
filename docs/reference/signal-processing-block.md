---
slug: signal-processing-block
title: Signal-processing block
entry_type: concept
category: sdr-programming
description: "A signal-processing block is a reusable DSP unit with typed input/output signatures and a work() method that the runtime calls to transform sample streams."
keywords: signal processing block, DSP block, GNU Radio block, work function, general_work, io signature, forecast, sync block, decimator, interpolator, source, sink
aka: [DSP block, GNU Radio block, signal processing block]
autolink: true
infobox:
  - { label: Type, value: Reusable DSP unit }
  - { label: Interface, value: "io signatures + work()/general_work()" }
  - { label: Used in, value: "GNU Radio flowgraphs, OOT modules" }
see_also: [flowgraph, stream-vs-message-passing, hierarchical-block, block-scheduler, out-of-tree-module, fir-filter]
cite_urls:
  - https://wiki.gnuradio.org/index.php/BlocksCodingGuide
  - https://en.wikipedia.org/wiki/GNU_Radio
---

**A signal-processing block is a self-contained DSP unit with typed input and output
signatures and a `work()` method the runtime calls to turn input samples into output
samples.**[^guide] It is the node in a [flowgraph](/reference/flowgraph/): one block does one
job — a filter, a resampler, a demodulator, a decoder — and exposes just enough interface for
a [scheduler](/reference/block-scheduler/) to hand it buffers of samples and collect the
results. Blocks are the reusable Lego bricks of block-based SDR: wire different ones together
and you get a different radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A signal-processing block shown as a box with an input port carrying N input items on the left and an output port carrying M output items on the right, and a work function inside that maps inputs to outputs." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="spbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="150" y="40" width="160" height="70" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="230" y="70" font-size="11" fill="currentColor" text-anchor="middle">block</text>
  <text x="230" y="88" font-size="8" fill="currentColor" text-anchor="middle">work(in, out)</text>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="60" y1="75" x2="149" y2="75" marker-end="url(#spbar)"/>
    <line x1="310" y1="75" x2="400" y2="75" marker-end="url(#spbar)"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="100" y="66">input stream</text>
    <text x="100" y="92">(N items)</text>
    <text x="360" y="66">output stream</text>
    <text x="360" y="92">(M items)</text>
  </g>
</svg>
<figcaption>A block declares its input and output item types (io signatures) and implements work(), which consumes input items and produces output items.</figcaption>
</figure>

## How it works

Every block declares two things up front and implements one behaviour:

- **Input signature** — how many input ports it has and the item size on each (complex 8-byte
  IQ, 4-byte float, 1-byte packed bits, a custom struct).
- **Output signature** — the same for its outputs. A source has zero inputs; a sink has zero
  outputs.
- **A work function** — the code the runtime calls with pointers to a chunk of available input
  and space for output. In GNU Radio, a *sync block* (one output item per input item)
  implements `work()`; a general block with a different or data-dependent rate implements
  `general_work()` plus a `forecast()` that tells the scheduler how many input items it needs
  to produce a given number of outputs.

The block returns how many output items it actually produced, and the runtime advances the
stream pointers accordingly. This contract — *here is your input, here is your output space,
tell me how much you consumed and produced* — is what lets the scheduler manage buffering,
[back-pressure](/reference/block-scheduler/), and rate changes without the block author having
to know anything about the blocks around it.

## In practice

Rate relationships classify the common block types:

- **Sync block** — 1:1. A per-sample gain, an FM discriminator, an AGC.
- **Decimator** — N:1. Consumes N inputs per output; the front of a channelizer.
- **Interpolator** — 1:N. Upsampling before pulse shaping.
- **General block** — arbitrary or data-dependent rate. A packet framer that emits a burst
  only when a valid frame is found uses `general_work()` and `forecast()`.

Blocks can also carry [stream tags](/reference/stream-vs-message-passing/) — metadata pinned
to a sample index — and message ports for asynchronous control, so a block is not limited to
pushing samples: it can annotate the stream or exchange out-of-band messages. Custom blocks
that ship outside the core tree live in an [out-of-tree module](/reference/out-of-tree-module/),
and a group of blocks can be bundled as a [hierarchical block](/reference/hierarchical-block/)
that itself looks like one block.

## Relevance to SDR

The block is the unit of reuse in SDR software. Because its interface is narrow and typed, a
filter written once works in any flowgraph that feeds it the right item type, and you can unit
test a block by feeding known input and asserting on the output — no radio hardware required.
This is why GNU Radio ships hundreds of stock blocks and why a large fraction of SDR
development is really "write one new block and drop it into an existing chain."

When you build SDR software, the discipline the block interface enforces — one job, explicit
input/output types, no assumptions about neighbours — is worth adopting even without a
formal block runtime. [GopherTrunk](/reference/software-defined-radio/) is written in pure Go
and does not use GNU Radio, but its decode chain is factored the same way: the down-converter,
demodulator, symbol slicer, and framer are each independent stages with clear input and output
types, which is exactly what makes them individually testable against captured
[IQ](/reference/iq-data/) files. The naming differs; the "one block, one job, typed I/O"
principle is identical.

## Sources

[^guide]: [Blocks Coding Guide](https://wiki.gnuradio.org/index.php/BlocksCodingGuide) — GNU Radio Wiki, on io signatures, sync blocks, general blocks, work()/general_work(), and forecast().
