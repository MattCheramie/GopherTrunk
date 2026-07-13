---
slug: gnuradio
title: GNU Radio
entry_type: technology
category: sdr-software
description: "GNU Radio is an open-source signal-processing framework that builds radios as flowgraphs of interconnected DSP blocks, from a GUI or Python."
keywords: GNU Radio, GNU Radio Companion, GRC, flowgraph, DSP framework, signal processing blocks, software radio, SDR toolkit, gr-blocks
aka: [GNU Radio, GNU Radio Companion, GRC]
autolink: true
infobox:
  - { label: Type, value: Signal-processing framework }
  - { label: Idea, value: Radios as DSP block flowgraphs }
  - { label: Examples, value: "GRC, gr-osmosdr, out-of-tree modules" }
see_also: [software-defined-radio, soapysdr, gqrx, digital-down-converter, fir-filter, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/GNU_Radio
  - https://www.gnuradio.org/
---

**GNU Radio** is a free, open-source [software-defined radio](/reference/software-defined-radio/)
framework that lets you build a receiver or transmitter as a **flowgraph** — a directed
graph of signal-processing blocks that pass streams of samples to one another.[^proj] Rather
than writing a monolithic program, an engineer wires together sources, filters, demodulators,
and sinks; the framework schedules the blocks and moves [IQ](/reference/iq-data/) samples
between them. It is the de-facto prototyping toolkit for experimental and research radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A GNU Radio flowgraph: a source block feeds a low-pass filter, then a demodulator, then an audio sink, connected by arrows carrying sample streams." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="grar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="8" y="46" width="78" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="47" y="63">SDR source</text>
    <rect x="126" y="46" width="78" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="165" y="60">low-pass</text><text x="165" y="69">filter</text>
    <rect x="244" y="46" width="78" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="283" y="63">demod</text>
    <rect x="362" y="46" width="78" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="401" y="63">audio sink</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="86" y1="60" x2="124" y2="60" marker-end="url(#grar)"/>
    <line x1="204" y1="60" x2="242" y2="60" marker-end="url(#grar)"/>
    <line x1="322" y1="60" x2="360" y2="60" marker-end="url(#grar)"/>
  </g>
  <text x="230" y="20" text-anchor="middle" font-size="9" fill="currentColor">flowgraph: streams of IQ samples flow left to right</text>
</svg>
<figcaption>In GNU Radio a radio is a flowgraph — DSP blocks connected by sample streams, from an SDR source through filters and a demodulator to an output sink.</figcaption>
</figure>

## How it works

At the core is a **runtime scheduler** that executes blocks concurrently and manages the
buffers between them. Each block declares how many input samples it consumes and how many
output samples it produces; the scheduler calls each block's work function whenever enough
data is available, moving samples through shared circular buffers with as little copying as
possible. Blocks fall into a few families:

- **Sources and sinks** — hardware front ends (via `gr-osmosdr` or [SoapySDR](/reference/soapysdr/)),
  files, UDP sockets, or the audio card.
- **Streaming DSP** — [FIR filters](/reference/fir-filter/), a
  [digital down-converter](/reference/digital-down-converter/), resamplers, FFTs, and the
  arithmetic primitives to combine streams.
- **Modulators and demodulators** — for AM, FM, PSK, FSK, OFDM, and many digital modes.
- **Sync and framing** — timing recovery, carrier recovery, correlators, and packet
  deframers.

Most users assemble these visually in **GNU Radio Companion (GRC)**, a drag-and-drop editor
that generates Python. Because the generated program is ordinary Python calling into C++
blocks, users can drop into code for anything the GUI does not cover, and can package their
own **out-of-tree (OOT) modules** to extend the block library. In addition to streaming
samples, blocks pass **tagged stream** metadata and asynchronous **message** ports, which
carry control information and decoded packets alongside the sample flow.

The design goal is reuse: a timing-recovery block written once works in any flowgraph, so
building a new decoder is largely a matter of connecting proven pieces and writing only the
protocol-specific glue.

## Relevance to SDR

GNU Radio is the reference environment for SDR experimentation and teaching. It underpins
countless research projects and specialized decoders: satellite telemetry receivers,
amateur-radio digital modes, cellular and IoT sniffers, and the DSP back ends of desktop
receivers. Applications such as [Gqrx](/reference/gqrx/) are built directly on GNU Radio's
block library, and the framework interoperates with almost any front end through
`gr-osmosdr` and [SoapySDR](/reference/soapysdr/). Its wide-ranging block set makes it the
usual place a new modulation scheme is first demodulated before anyone writes a dedicated tool.

**GopherTrunk** does not embed or depend on GNU Radio. GopherTrunk is a self-contained,
pure-Go decoder that implements its own DSP chain — channelization, timing and carrier
recovery, symbol slicing, and framing — directly in Go, so it ships as a single static
binary with no GNU Radio runtime. The two occupy overlapping territory (both turn IQ samples
into decoded symbols), but they are independent codebases: GNU Radio is a general framework
you assemble, while GopherTrunk is a purpose-built trunking scanner. GNU Radio remains a
useful companion for GopherTrunk work as a bench tool — for capturing IQ, visualizing a
spectrum, or prototyping a demodulator before porting the idea into Go.

## Sources

[^proj]: [GNU Radio](https://www.gnuradio.org/) — the official project site and documentation, describing the flowgraph model, the block scheduler, GNU Radio Companion, and out-of-tree modules.
