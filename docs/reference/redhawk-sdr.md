---
slug: redhawk-sdr
title: REDHAWK SDR
entry_type: technology
category: sdr-frameworks
description: "REDHAWK is an open-source, SCA-influenced component framework for building and deploying real-time software-defined-radio applications across distributed hardware."
keywords: REDHAWK, REDHAWK SDR, Software Communications Architecture, SCA, JTNC, component framework, CORBA, waveform, distributed SDR, Geon Technologies
aka: [REDHAWK, REDHAWK SDR]
autolink: true
infobox:
  - { label: Type, value: Component-based SDR framework }
  - { label: Idea, value: Deployable SCA-style components across distributed nodes }
  - { label: Examples, value: IDE, components, waveforms, domain manager }
see_also: [software-defined-radio, signal-processing-block, flowgraph, gnuradio, soapysdr, usrp-ettus]
cite_urls:
  - https://github.com/RedhawkSDR/redhawk
  - https://en.wikipedia.org/wiki/Software_Communications_Architecture
---

**REDHAWK** is an open-source software framework for developing, deploying, and managing
real-time [software-defined-radio](/reference/software-defined-radio/) applications, drawing
its architecture from the JTNC Software Communications Architecture (SCA) used in military
radio systems.[^rh] Where hobbyist frameworks emphasize a single flowgraph on one machine,
REDHAWK emphasizes distributed, managed deployment: components that can be started, stopped,
connected, and moved across a network of processing nodes at runtime.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A REDHAWK domain: a domain manager oversees two device-manager nodes, each hosting deployed components connected into a running waveform application." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rhar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="150" y="12" width="160" height="26" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="230" y="29" text-anchor="middle" font-size="9" fill="currentColor">domain manager</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="30" y="70" width="180" height="66" rx="6" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/><text x="120" y="84">node A</text>
    <rect x="45" y="94" width="60" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="75" y="110">source</text>
    <rect x="130" y="94" width="60" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="160" y="110">filter</text>
    <rect x="250" y="70" width="180" height="66" rx="6" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/><text x="340" y="84">node B</text>
    <rect x="265" y="94" width="60" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="295" y="110">demod</text>
    <rect x="350" y="94" width="60" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="380" y="110">sink</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="230" y1="38" x2="120" y2="70"/><line x1="230" y1="38" x2="340" y2="70"/>
    <line x1="105" y1="107" x2="130" y2="107" marker-end="url(#rhar)"/>
    <line x1="190" y1="107" x2="265" y2="107" marker-end="url(#rhar)"/>
    <line x1="325" y1="107" x2="350" y2="107" marker-end="url(#rhar)"/>
  </g>
</svg>
<figcaption>REDHAWK deploys a waveform as components spread across managed nodes; a domain manager coordinates the nodes while the connected components form the running signal chain.</figcaption>
</figure>

## How it works

A REDHAWK application is assembled from **components** — encapsulated
[signal-processing blocks](/reference/signal-processing-block/) with defined ports and
properties — connected into a **waveform**. At runtime a **domain manager** coordinates one
or more **device managers**, each running on a node and advertising the hardware it controls
(an SDR front end, a GPU, CPU cores). When a waveform is launched, the framework's deployment
logic places each component on a suitable node, wires the ports together, and starts them; the
whole assembly can be introspected, reconfigured, and torn down live. Data flows between
components over BulkIO streaming ports that carry sample buffers plus SRI metadata (sample rate,
center frequency, format), and control flows over a middleware transport historically based on
CORBA.

This structure is inherited from the **Software Communications Architecture**, the JTNC
standard for portable military "waveforms" that can be moved between compliant radios. REDHAWK
adopts SCA's separation of application logic from the underlying hardware and its
component/deployment model, while packaging it in a more approachable open-source form with an
Eclipse-based IDE for building components in C++, Python, or Java and designing waveforms
visually.

## In practice

REDHAWK ships an IDE (component and waveform editors, a sandbox for interactive testing, live
plotting), a core framework runtime, and a library of stock components and devices — including
front ends for common SDR hardware such as the [USRP](/reference/usrp-ettus/). Its typical home
is signals-intelligence, spectrum-monitoring, and research systems where many channels are
processed across a cluster and operators need to reconfigure the processing chain without
recompiling. That distributed, managed emphasis is what sets it apart from a single-process
[flowgraph](/reference/flowgraph/) tool like [GNU Radio](/reference/gnuradio/), though the two
address overlapping problems and are sometimes used together.

## Relevance to SDR

REDHAWK represents the "enterprise" end of the SDR software spectrum: it answers questions
that hobbyist frameworks mostly ignore — how to deploy a waveform across a rack of machines,
how to manage hardware and components as first-class runtime resources, and how to keep
application code portable across radios via an SCA-influenced contract. Studying it clarifies
that "SDR framework" spans everything from a laptop flowgraph to a distributed, managed
signal-processing platform.

GopherTrunk sits firmly at the opposite, lightweight end: it is a single self-contained pure-Go
binary with a fixed decode pipeline, no component middleware, no CORBA, and no distributed
domain manager. It solves the trunking problem by embedding the DSP directly rather than by
composing deployable SCA components. The contrast is the useful lesson — REDHAWK's flexibility
and manageability come at the cost of significant infrastructure, whereas GT trades that
flexibility for a zero-dependency, statically compiled receiver that runs anywhere Go runs.

## Sources

[^rh]: [REDHAWK repository](https://github.com/RedhawkSDR/redhawk) — the open-source REDHAWK SDR framework, documenting its component/waveform model, domain and device managers, and SCA lineage.
