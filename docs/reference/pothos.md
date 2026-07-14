---
slug: pothos
title: Pothos
entry_type: technology
category: sdr-frameworks
description: "Pothos is a graph-based data-flow framework for building signal-processing applications from interconnected blocks, an alternative runtime to GNU Radio."
keywords: Pothos, Pothos SDR, data-flow framework, block scheduler, Pothos Flow, GUI, signal processing graph, GNU Radio alternative, Pothosware
aka: [Pothos, Pothos SDR, Pothos framework]
autolink: true
infobox:
  - { label: Type, value: Data-flow / block-scheduling framework }
  - { label: Idea, value: Wire processing blocks into a streaming graph }
  - { label: Examples, value: Pothos Flow GUI, gr-pothos, SoapySDR I/O }
see_also: [gnuradio, flowgraph, soapysdr, signal-processing-block, software-defined-radio, gnuradio-companion]
cite_urls:
  - https://github.com/pothosware/PothosCore/wiki
  - https://en.wikipedia.org/wiki/GNU_Radio
---

**Pothos** is a graph-based data-flow framework for building signal-processing and
software-defined-radio applications by wiring reusable **blocks** into a streaming graph,
positioned as an alternative runtime to [GNU Radio](/reference/gnuradio/).[^poth] Like GNU
Radio it lets you compose a system from small units — a source, filters, a demodulator, a
sink — but it uses its own scheduler, block API, and design tool, and it is the project that
also produced [SoapySDR](/reference/soapysdr/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A Pothos data-flow graph: a source block feeds a filter block, which feeds a demodulator block, which feeds a sink block, connected left to right by streaming ports." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="potar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="69">SDR source</text>
    <rect x="140" y="50" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="180" y="69">FIR filter</text>
    <rect x="260" y="50" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="300" y="69">demod</text>
    <rect x="380" y="50" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="415" y="69">sink</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="100" y1="65" x2="140" y2="65" marker-end="url(#potar)"/>
    <line x1="220" y1="65" x2="260" y2="65" marker-end="url(#potar)"/>
    <line x1="340" y1="65" x2="380" y2="65" marker-end="url(#potar)"/>
  </g>
  <text x="235" y="105" font-size="8" fill="currentColor" text-anchor="middle">a scheduler streams sample buffers along each connected port</text>
</svg>
<figcaption>A Pothos application is a graph of processing blocks connected by streaming ports; the runtime scheduler moves sample buffers from block to block.</figcaption>
</figure>

## How it works

The core abstraction is the **block** — an object with typed input and output ports plus a
`work()` method that consumes samples on its inputs and produces samples on its outputs. A
[flowgraph](/reference/flowgraph/) is built by connecting one block's output port to
another's input port, and a runtime **scheduler** (part of PothosCore) decides when each
block runs, hands it buffers, and propagates back-pressure so a slow downstream block throttles
the whole chain. Buffers are passed by reference and pooled, and blocks that need it can attach
metadata "labels" to positions in the stream — closely analogous to GNU Radio's
[stream tags](/reference/signal-processing-block/) — for events like a retune or a burst
boundary.

Two design choices distinguish Pothos from GNU Radio. First, blocks can be written in C++ or
Python and are described by a JSON block-registry entry, which the tooling reads to auto-generate
GUI widgets and parameter fields. Second, the graph can be **distributed**: because blocks
communicate through a serialization layer, a flowgraph can span multiple processes or even
multiple machines, with the scheduler stitching remote blocks into the same logical graph.

## Variants

Pothos is a small ecosystem of packages rather than a single binary. **PothosCore** is the
runtime and block API; **Pothos Flow** is the graphical designer where you drag blocks onto a
canvas and connect them, comparable in role to
[GNU Radio Companion](/reference/gnuradio-companion/); toolkits add DSP primitives, an SDR
source/sink built on [SoapySDR](/reference/soapysdr/), plotting widgets, and network transport.
A `gr-pothos` bridge lets existing GNU Radio blocks run inside a Pothos graph, easing migration
between the two worlds. On Windows the "Pothos SDR" installer bundles the framework together
with a large collection of SDR tools and drivers.

## Relevance to SDR

Pothos matters as a demonstration that GNU Radio's block-and-flowgraph model is a general
architecture, not a single implementation — the same "wire blocks into a streaming graph"
idea can be built with a different scheduler, a different block API, and first-class
distribution across machines. Its lasting practical contribution to the wider ecosystem is
[SoapySDR](/reference/soapysdr/), which came out of the Pothos project and is now the
vendor-neutral hardware layer that many unrelated applications depend on.

GopherTrunk is not built on Pothos or on any general data-flow framework; it is a purpose-built
pure-Go trunking decoder with a fixed [software-defined-radio](/reference/software-defined-radio/)
pipeline rather than a user-editable graph. But the underlying pattern is the same one GT
implements by hand: sources, filters, a channelizer, and demodulators connected by buffered
streams, with back-pressure and overrun handling between stages. Understanding the block/scheduler
model clarifies what a fixed-pipeline decoder like GT trades away (graph flexibility) in exchange
for a simpler, dependency-free, statically compiled runtime.

## Sources

[^poth]: [PothosCore wiki](https://github.com/pothosware/PothosCore/wiki) — Pothosware, documentation of the Pothos block API, scheduler, Pothos Flow designer, and the framework's relationship to SoapySDR and GNU Radio.
