---
slug: gnuradio-companion
title: GNU Radio Companion (GRC)
entry_type: technology
category: sdr-frameworks
description: "GNU Radio Companion is the visual flowgraph editor for GNU Radio: engineers wire DSP blocks together on a canvas and GRC generates the runnable Python program."
keywords: GNU Radio Companion, GRC, flowgraph editor, visual programming, GNU Radio GUI, block canvas, generated Python, .grc file, hier block
aka: [GNU Radio Companion, GRC]
autolink: true
infobox:
  - { label: Type, value: Visual flowgraph editor }
  - { label: Idea, value: Wire DSP blocks, generate Python }
  - { label: Part of, value: GNU Radio }
see_also: [gnuradio, flowgraph, signal-processing-block, gr-osmosdr, python-language, software-defined-radio]
cite_urls:
  - https://wiki.gnuradio.org/index.php/GNURadioCompanion
  - https://www.gnuradio.org/
---

**GNU Radio Companion** (**GRC**) is the graphical **flowgraph** editor bundled with
[GNU Radio](/reference/gnuradio/): a drag-and-drop canvas on which an engineer places
[signal-processing blocks](/reference/signal-processing-block/), connects their inputs and
outputs, and sets each block's parameters, after which GRC generates a complete, runnable
Python program.[^grc] It turns building a [software-defined radio](/reference/software-defined-radio/)
from a coding exercise into a wiring exercise, which is why it is the entry point most people
use to learn GNU Radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The GNU Radio Companion canvas showing a source, throttle, filter, and sink blocks wired left to right, with an arrow indicating the canvas is compiled into a generated Python top_block file." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="grcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="8" y="10" width="300" height="86" rx="6" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <text x="158" y="24" text-anchor="middle" font-size="8" fill="currentColor">GRC canvas (.grc)</text>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="18" y="52" width="58" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="47" y="68">source</text>
    <rect x="98" y="52" width="58" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="127" y="68">throttle</text>
    <rect x="178" y="52" width="58" height="26" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="207" y="68">filter</text>
    <rect x="258" y="52" width="42" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="279" y="68">sink</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="76" y1="65" x2="96" y2="65" marker-end="url(#grcar)"/>
    <line x1="156" y1="65" x2="176" y2="65" marker-end="url(#grcar)"/>
    <line x1="236" y1="65" x2="256" y2="65" marker-end="url(#grcar)"/>
    <line x1="308" y1="53" x2="356" y2="53" marker-end="url(#grcar)"/>
  </g>
  <rect x="358" y="40" width="94" height="52" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="405" y="60" text-anchor="middle" font-size="7.5" fill="currentColor">generated</text>
  <text x="405" y="71" text-anchor="middle" font-size="7.5" fill="currentColor">top_block.py</text>
  <text x="335" y="35" text-anchor="middle" font-size="7.5" fill="currentColor">generate</text>
  <text x="230" y="118" text-anchor="middle" font-size="9" fill="currentColor">wire blocks visually → GRC writes Python you can run or edit</text>
</svg>
<figcaption>GRC compiles a visual flowgraph into a Python top-block program: the canvas is the design, and the generated code is what actually runs.</figcaption>
</figure>

## How it works

GRC reads block definitions from GNU Radio's installed component libraries and lists them in
a searchable tree. Dragging a block onto the canvas creates an instance whose parameters —
sample rate, filter taps, gain, frequency — are edited in a properties dialog. Ports are typed
and colored by sample format (complex, float, byte, message), and GRC refuses to connect
mismatched ports, catching a whole class of errors before the program ever runs. Variables and
GUI controls (sliders, entry boxes, `QT` range widgets) can be dropped in too, so a parameter
like center frequency becomes a live knob.

The document itself is a `.grc` file — a YAML description of blocks, parameters, and
connections. When you press **Generate**, GRC's code generator (`grcc`) walks that graph and
emits a Python module: a `top_block` class that instantiates each block, calls
`self.connect(...)` for every wire, and exposes the GUI. Running it simply executes the
[GNU Radio](/reference/gnuradio/) runtime with that [flowgraph](/reference/flowgraph/). Two
consequences follow:

- **The GUI is not a black box.** The generated Python is ordinary, readable code you can open,
  extend, or embed in a larger program — GRC is a starting point, not a cage.
- **Custom logic still fits.** An **Embedded Python Block** lets you write a small block inline
  on the canvas, and anything the block tree lacks can be added as an out-of-tree module and it
  appears alongside the built-ins.

Reusable sub-designs are packaged as **hierarchical blocks**: a flowgraph exported as a hier
block shows up as a single block in other flowgraphs, so complex receivers stay legible.

## Relevance to SDR

GRC is where most SDR practitioners first assemble a working radio, and it remains the fastest
way to prototype one. Tuning a dongle, filtering a channel, demodulating FM or a digital mode,
and piping audio or bits to a sink is a few minutes of wiring rather than an afternoon of code.
Because it sources from [gr-osmosdr](/reference/gr-osmosdr/) and SoapySDR, the same flowgraph
drives an RTL-SDR, a HackRF, or a USRP. Teaching material, conference demos, and the first cut
of many decoders all begin as `.grc` files, and researchers routinely export a proven flowgraph
to Python and then harden it into a standalone tool.

**GopherTrunk** does not use GRC and produces no flowgraphs — it is a purpose-built, pure-Go
trunking scanner with its own hand-written DSP chain, shipped as a single static binary with no
GNU Radio runtime. GRC is nonetheless a natural bench companion for GopherTrunk work: it is an
excellent scratchpad for inspecting a signal, capturing IQ to a file, or prototyping a
demodulator idea visually before that idea is reimplemented in Go for GopherTrunk's decode path.

## Sources

[^grc]: [GNU Radio Companion](https://wiki.gnuradio.org/index.php/GNURadioCompanion) — GNU Radio project wiki, documenting the flowgraph canvas, block/port typing, the `.grc` document, Python code generation, embedded blocks, and hierarchical blocks.
