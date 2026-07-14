---
slug: flowgraph
title: Flowgraph
entry_type: concept
category: sdr-programming
description: "A flowgraph is a directed graph of DSP blocks connected by sample streams; it is the core programming model of GNU Radio and most block-based SDR software."
keywords: flowgraph, flow graph, GNU Radio, DSP blocks, directed graph, dataflow, signal processing pipeline, block-based SDR, stream connections, top block
aka: [flow graph, flowgraph]
autolink: true
infobox:
  - { label: Type, value: Dataflow programming model }
  - { label: Idea, value: DSP blocks wired into a directed graph }
  - { label: Used in, value: "GNU Radio, SDRangel, Pothos" }
see_also: [signal-processing-block, block-scheduler, gnuradio, gnuradio-companion, hierarchical-block, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/GNU_Radio
  - https://wiki.gnuradio.org/index.php/Guided_Tutorial_GNU_Radio_in_Python
---

**A flowgraph is a directed graph of digital-signal-processing blocks connected by
sample streams**, where each [block](/reference/signal-processing-block/) consumes samples
on its inputs, does one job, and produces samples on its outputs.[^grc] It is the central
programming model of [GNU Radio](/reference/gnuradio/) and most block-based SDR toolkits:
instead of writing one long loop, you describe *what* processing happens and *how the data
flows* between the parts, and a runtime moves the samples. A flowgraph reads like the block
diagram an RF engineer would draw on a whiteboard — source, filter, demodulator, sink —
turned directly into an executable program.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A flowgraph as a directed graph: a source block feeds a filter, which feeds a demodulator, which splits to a decoder sink and a file sink, with arrows showing sample streams flowing downstream." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="10" y="55" width="66" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="43" y="73">source</text>
    <rect x="118" y="55" width="66" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="151" y="73">filter</text>
    <rect x="226" y="55" width="66" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="259" y="73">demod</text>
    <rect x="334" y="18" width="76" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="372" y="36">decode sink</text>
    <rect x="334" y="92" width="76" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="372" y="110">file sink</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="76" y1="70" x2="117" y2="70" marker-end="url(#fgar)"/>
    <line x1="184" y1="70" x2="225" y2="70" marker-end="url(#fgar)"/>
    <path d="M292 70 L313 70 L313 33 L333 33" marker-end="url(#fgar)"/>
    <path d="M292 70 L313 70 L313 107 L333 107" marker-end="url(#fgar)"/>
  </g>
</svg>
<figcaption>A flowgraph: blocks are nodes, sample streams are directed edges, and data flows from sources through processing to sinks.</figcaption>
</figure>

## How it works

A flowgraph is defined by its nodes and edges. The nodes are self-contained
[signal-processing blocks](/reference/signal-processing-block/); the edges are typed
connections that carry a stream of one item type — complex [IQ](/reference/iq-data/) samples,
real floats, bytes, or packed bits. The graph is usually built once, then handed to a
runtime that "runs" it until stopped.

- **Sources** have only outputs — a radio device, a file, a signal generator. They inject
  samples into the graph.
- **Processing blocks** have both inputs and outputs — filters, resamplers, demodulators,
  decoders. Each transforms its input stream into an output stream.
- **Sinks** have only inputs — an audio device, a file writer, a network socket, a display.
  They consume the final result.

Because the connections are explicit, the runtime knows the full topology before anything
runs. It can allocate a buffer for every edge, decide the order to invoke blocks, and
propagate rate changes (a decimating filter that outputs one sample for every eight it reads)
through the whole graph automatically. The engineer never writes the loop that shuttles data
between stages — that is the [scheduler's](/reference/block-scheduler/) job.

## In practice

Flowgraphs are typically constructed in one of two ways. In code, you instantiate each block
object and call a `connect(a, b)`-style method for every edge, giving you a directed graph in
memory (GNU Radio's Python and C++ APIs work this way, with a "top block" as the container).
Graphically, tools like [GNU Radio Companion](/reference/gnuradio-companion/) let you drag
blocks onto a canvas and draw the wires, then generate the equivalent code. Either way the
result is the same object: a graph the runtime can start, stop, lock for live edits, and
reconnect.

A subgraph can be packaged as a single reusable node — a
[hierarchical block](/reference/hierarchical-block/) — so a complex receiver stays readable
at the top level while its internals live in named sub-flowgraphs. This is how large designs
avoid becoming an unreadable tangle of hundreds of primitive blocks.

## Relevance to SDR

The flowgraph model dominates SDR software because a radio *is* naturally a pipeline: samples
flow one direction, each stage does one transformation, and the stages compose. GNU Radio,
SDRangel, Pothos, and REDHAWK all expose some form of it, and it is the mental model behind
nearly every "block diagram to running radio" workflow.

When you write SDR software, choosing the flowgraph model buys you separation of concerns
(each block is testable in isolation), reconfigurability (swap a demodulator without touching
the rest), and automatic scheduling and buffering. The cost is indirection: control flow is
implicit in the topology rather than written out, which can make step-through debugging and
tight cross-block coupling harder.

[GopherTrunk](/reference/software-defined-radio/) is a pure-Go SDR application and does **not**
use GNU Radio or a general flowgraph runtime — its P25/DMR/TETRA decode chains are hand-wired
Go pipelines wired with channels and goroutines rather than a generic block scheduler. But the
underlying shape is the same directed dataflow: a device or replay source feeds a
down-converter, which feeds a demodulator, which feeds symbol and frame recovery, which feeds
the decoders. Understanding the flowgraph abstraction is the fastest way to reason about where
each processing stage sits and how samples move between them, whether or not a formal flowgraph
runtime is in the picture.

## Sources

[^grc]: [GNU Radio in Python (guided tutorial)](https://wiki.gnuradio.org/index.php/Guided_Tutorial_GNU_Radio_in_Python) — GNU Radio Wiki, on building a radio as a connected graph of blocks with a top block, sources, processing, and sinks.
