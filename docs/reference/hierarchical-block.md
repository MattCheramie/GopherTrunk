---
slug: hierarchical-block
title: Hierarchical block
entry_type: concept
category: sdr-programming
description: "A hierarchical block is a sub-flowgraph packaged to look like a single block, letting a group of DSP blocks be reused and nested to keep large flowgraphs readable."
keywords: hierarchical block, hier block, GNU Radio, sub-flowgraph, gr hier_block2, composition, nested flowgraph, reusable block, abstraction, encapsulation
aka: [hier block, hierarchical block]
autolink: true
infobox:
  - { label: Type, value: Composite block }
  - { label: Idea, value: A sub-flowgraph exposed as one block }
  - { label: Used in, value: "GNU Radio, GRC, OOT modules" }
see_also: [flowgraph, signal-processing-block, out-of-tree-module, gnuradio-companion, abstraction]
cite_urls:
  - https://wiki.gnuradio.org/index.php/Hierarchical_Blocks
  - https://en.wikipedia.org/wiki/GNU_Radio
---

**A hierarchical block is a sub-[flowgraph](/reference/flowgraph/) packaged so that it looks
and behaves like a single [block](/reference/signal-processing-block/).**[^hier] Internally it
is several blocks wired together; externally it exposes just input and output ports, so the rest
of a design can drop it in as one node and never see its innards. Hierarchical blocks are how
block-based SDR scales: they let you name a chunk of processing ("FM receiver," "P25 demod
front end"), reuse it, and nest it, so a large radio stays a readable handful of high-level
boxes instead of a wall of hundreds of primitive blocks.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A hierarchical block drawn as a large box with one input and one output port; inside it three smaller blocks are wired in series, showing that the composite exposes simple ports while hiding its internal flowgraph." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="hbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="70" y="35" width="320" height="100" rx="8" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="230" y="28" font-size="9" fill="currentColor" text-anchor="middle">hierarchical block</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="95" y="70" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="125" y="88">filter</text>
    <rect x="200" y="70" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="88">demod</text>
    <rect x="305" y="70" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="335" y="88">decode</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="20" y1="85" x2="94" y2="85" marker-end="url(#hbar)"/>
    <line x1="155" y1="85" x2="199" y2="85" marker-end="url(#hbar)"/>
    <line x1="260" y1="85" x2="304" y2="85" marker-end="url(#hbar)"/>
    <line x1="365" y1="85" x2="440" y2="85" marker-end="url(#hbar)"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="45" y="76">in</text><text x="415" y="76">out</text></g>
</svg>
<figcaption>A hierarchical block exposes plain input/output ports while its internal sub-flowgraph — here filter, demod, decode — is hidden behind that interface.</figcaption>
</figure>

## How it works

A hierarchical block defines its own input and output signatures, then wires internal blocks
between special *self* ports that stand for the block's own boundary. Connecting an internal
block to the hier block's input port routes the composite's incoming samples into it; connecting
to the output port routes results back out. Once defined, the hier block is a first-class node:
you connect it into a parent [flowgraph](/reference/flowgraph/) exactly like any primitive
block, and the [scheduler](/reference/block-scheduler/) flattens the hierarchy — it schedules
the internal blocks directly, so nesting is a source-level convenience with no runtime penalty
from the boxing itself.

The interface is the point. By exposing only ports and a few construction parameters (a center
frequency, a decimation factor), the hier block hides its internal wiring the way a function
hides its body. This is ordinary [abstraction](/reference/abstraction/): callers depend on the
signature, not the implementation, so you can rework the internals — swap the demodulator, add
an AGC — without touching anything that uses the block.

## In practice

Hierarchical blocks are created two ways that produce the same thing. In
[GNU Radio Companion](/reference/gnuradio-companion/) you build a flowgraph whose source and
sink are "pad" blocks; GRC generates a reusable hier block from it that then appears in your
block library. In code you subclass the hier-block base type, declare the io signatures in the
constructor, and connect the internals. Either way the result can be published in an
[out-of-tree module](/reference/out-of-tree-module/) and shared like any other block.

Good hier blocks are cohesive — one clear job, a small parameter list, clean port types — for
the same reasons good functions are. A hier block that reaches into global state or needs a
dozen tightly-coupled parameters is a sign the boundary was drawn in the wrong place.

## Relevance to SDR

Composition is what keeps real receivers maintainable. A production decoder is dozens of
primitive stages; without a way to group them, the top-level flowgraph becomes unreadable and
nothing is reusable. Hier blocks give SDR the same modularity that functions and modules give
general software — build once, name it, reuse it, nest it — and they make it practical to ship
libraries of ready-made radio front ends.

[GopherTrunk](/reference/software-defined-radio/) is pure Go and does not use GNU Radio's
`hier_block2`, but it applies the identical principle through Go packages and types: the
down-conversion front end, the C4FM demodulator, and each protocol decoder are self-contained
units with narrow interfaces that a higher layer composes into a full scanner, hiding their
internals behind a handful of exported methods. Whether the mechanism is a GNU Radio hier block
or a Go package, the goal is the same — express a complex radio as a small number of
well-named, reusable pieces rather than one flat mass of DSP.

## Sources

[^hier]: [Hierarchical Blocks](https://wiki.gnuradio.org/index.php/Hierarchical_Blocks) — GNU Radio Wiki, on composing sub-flowgraphs into reusable blocks with their own io signatures and pad ports.
