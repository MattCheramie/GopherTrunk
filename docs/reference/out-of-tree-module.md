---
slug: out-of-tree-module
title: Out-of-tree module (OOT)
entry_type: concept
category: sdr-programming
description: "An out-of-tree module is a custom GNU Radio component built and installed outside the main source tree, the standard way to add your own blocks with gr_modtool."
keywords: out-of-tree module, OOT module, gr_modtool, GNU Radio blocks, custom block, gr-, CMake, block bindings, third-party blocks, extending GNU Radio
aka: [OOT module, out-of-tree module, gr-module]
autolink: true
infobox:
  - { label: Type, value: Third-party GNU Radio package }
  - { label: Tooling, value: "gr_modtool + CMake" }
  - { label: Naming, value: "gr-<name> convention" }
see_also: [signal-processing-block, gnuradio, hierarchical-block, gr-osmosdr, build-systems]
cite_urls:
  - https://wiki.gnuradio.org/index.php/OutOfTreeModules
  - https://en.wikipedia.org/wiki/GNU_Radio
---

**An out-of-tree module (OOT) is a custom [GNU Radio](/reference/gnuradio/) component built
and installed outside GNU Radio's own source tree.**[^oot] It is the sanctioned way to add your
own [blocks](/reference/signal-processing-block/) without forking or modifying GNU Radio
itself: you create a small package — conventionally named `gr-<something>` — that compiles
against the installed GNU Radio libraries and registers new blocks into the runtime and into
[GNU Radio Companion](/reference/gnuradio-companion/). Nearly every third-party GNU Radio
capability, from decoders to hardware drivers, ships as an OOT module.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A diagram showing the GNU Radio core tree on the left and a separate out-of-tree module package on the right; the OOT module links against the core and registers new blocks that appear in the shared block library." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ootar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="45" width="130" height="80" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="85" y="40" font-size="9" fill="currentColor" text-anchor="middle">GNU Radio core</text>
  <text x="85" y="90" font-size="8" fill="currentColor" text-anchor="middle">runtime + blocks</text>
  <rect x="300" y="45" width="140" height="80" rx="6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 4"/>
  <text x="370" y="40" font-size="9" fill="currentColor" text-anchor="middle">gr-mymod (OOT)</text>
  <text x="370" y="85" font-size="8" fill="currentColor" text-anchor="middle">your blocks</text>
  <text x="370" y="100" font-size="8" fill="currentColor" text-anchor="middle">+ CMake + bindings</text>
  <line x1="300" y1="85" x2="151" y2="85" stroke="currentColor" stroke-width="1.2" marker-end="url(#ootar)"/>
  <text x="225" y="78" font-size="7.5" fill="currentColor" text-anchor="middle">links against / registers into</text>
</svg>
<figcaption>An OOT module is a separate package that links against installed GNU Radio and registers new blocks into the shared runtime and block library.</figcaption>
</figure>

## How it works

An OOT module is generated and maintained with **`gr_modtool`**, GNU Radio's scaffolding
utility. `gr_modtool newmod <name>` creates the package skeleton; `gr_modtool add <block>`
generates the source, header, QA test, GRC block definition (a YAML file), and Python binding
stubs for a new block. You fill in the block's `work()`/`general_work()`, then build and install
with CMake. After installation the block is indistinguishable from a core block: it is importable
from Python, usable from C++, and shows up in the GRC palette.

The package is deliberately self-contained so it can version and distribute independently of
GNU Radio releases. It carries:

- **Block implementations** (C++ and/or Python) plus their public headers.
- **Bindings** that expose C++ blocks to Python (pybind11 in modern GNU Radio).
- **GRC block YAML** so the block appears with parameters and ports in the GUI.
- **A CMake [build](/reference/build-systems/)** that finds the installed GNU Radio and links
  against it.
- **QA / unit tests**, run against captured data with no radio hardware needed.

Because it links against GNU Radio's stable ABI rather than editing it, an OOT module rides
along with the installed framework: upgrade GNU Radio and, if the API is compatible, rebuild
the module against it.

## In practice

OOT modules are the unit of sharing in the GNU Radio ecosystem. Community packages —
[gr-osmosdr](/reference/gr-osmosdr/) for hardware access, and many protocol decoders — are OOT
modules you install alongside core. A module can bundle a mix of primitive blocks and
[hierarchical blocks](/reference/hierarchical-block/), so a project can ship both low-level DSP
and ready-assembled front ends together. Convention names the package `gr-<name>` and the
Python namespace to match, which is why installed third-party GNU Radio content is easy to spot.

## Relevance to SDR

The OOT pattern is what makes GNU Radio extensible without central gatekeeping: anyone can add a
demodulator or a device driver as a clean, versioned, testable package rather than patching the
framework. For SDR developers it is the concrete answer to "how do I add my own processing to
GNU Radio" — scaffold with `gr_modtool`, implement the block, build, install, done — and it
enforces good habits (isolated builds, bindings, and hardware-free QA tests) along the way.

[GopherTrunk](/reference/software-defined-radio/) has no GNU Radio dependency, so it neither is
nor consumes OOT modules — its blocks are Go packages compiled into a single static binary
rather than gr-modules linked against a shared GNU Radio install. The relevant contrast is
architectural: the OOT model plugs new DSP into an external framework's runtime and block
registry, whereas GopherTrunk builds its DSP directly in Go and depends on nothing external at
run time. Knowing the OOT workflow is still essential for anyone comparing GopherTrunk with the
mainstream GNU Radio decoders it competes with, and for reusing the wider ecosystem's blocks
outside of Go.

## Sources

[^oot]: [Out Of Tree Modules](https://wiki.gnuradio.org/index.php/OutOfTreeModules) — GNU Radio Wiki, on creating `gr-` modules with gr_modtool, CMake builds, bindings, and GRC integration.
