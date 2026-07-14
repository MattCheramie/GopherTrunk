---
slug: file-source-sink
title: File source & sink
entry_type: concept
category: sdr-app-building
description: "A file source reads recorded IQ samples into a flowgraph and a file sink writes samples back out, letting an SDR pipeline run against stored data instead of a live radio."
keywords: file source, file sink, IQ file reader, IQ file writer, flowgraph source block, sample recording, replay, gnuradio file source, file sink block, offline processing
aka: ["file source block", "file sink block", "IQ file reader/writer"]
autolink: true
infobox:
  - { label: Type, value: "Flowgraph source/sink block" }
  - { label: Source, value: "Reads samples from a file into the graph" }
  - { label: Sink, value: "Writes samples from the graph to a file" }
see_also: [cfile-format, iq-file-format, throttle-block, simulation-driven-sdr, iq-recording-playback]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://wiki.gnuradio.org/index.php/File_Source
---

A **file source** is a flowgraph block that reads samples from a file and pushes
them into a signal-processing pipeline; a **file sink** does the reverse, writing
the samples flowing past it out to disk.[^gr] Together they let an SDR
[flowgraph](/reference/flowgraph/) run entirely on stored data — the source stands
in for a radio's receive path, and the sink stands in for a radio's transmit path
or simply captures an intermediate stream for later inspection.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A file source reads an IQ file and feeds a DSP block, whose output can go to a file sink for storage or to further processing." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fssar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="48" width="86" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="63" y="62" font-size="8">file source</text><text x="63" y="74" font-size="7">read IQ</text>
    <line x1="106" y1="65" x2="150" y2="65" stroke="currentColor" stroke-width="1.2" marker-end="url(#fssar)"/>
    <rect x="152" y="48" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="195" y="62" font-size="8">DSP block</text><text x="195" y="74" font-size="7">filter/demod</text>
    <line x1="238" y1="65" x2="282" y2="65" stroke="currentColor" stroke-width="1.2" marker-end="url(#fssar)"/>
    <rect x="284" y="48" width="86" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="327" y="62" font-size="8">file sink</text><text x="327" y="74" font-size="7">write out</text>
    <line x1="370" y1="65" x2="410" y2="65" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3" marker-end="url(#fssar)"/>
    <text x="428" y="68" font-size="7">disk</text>
  </g>
</svg>
<figcaption>Source and sink blocks bracket a flowgraph, swapping a radio for a file at either end.</figcaption>
</figure>

## How it works

Inside a streaming flowgraph, every block has the same contract: pull a buffer of
samples from upstream, do work, push a buffer downstream. A file source has no
upstream — it simply reads the next chunk of bytes from an open file, interprets
them according to a declared [sample format](/reference/sample-format/) (e.g.
interleaved 32-bit float I/Q, or 8-bit unsigned like raw RTL-SDR), and emits them
as complex samples. A file sink has no downstream — it serializes each incoming
buffer to disk in the chosen format.

Two details make or break correct use:

- **Format agreement.** The reader must interpret bytes exactly as the writer laid
  them down: the same sample type, I-before-Q ordering, and endianness. A
  [cfile](/reference/cfile-format/) (interleaved float32) and a
  [SigMF](/reference/iq-file-format/) recording (data plus a metadata sidecar
  naming rate and format) are the two common conventions; guessing wrong turns a
  clean signal into noise.
- **Rate.** A raw IQ file carries no timestamps, so the source has no idea how fast
  to emit samples — it just reads as fast as it can. That is exactly right for an
  offline decode (run flat out), but wrong for anything that must behave as if in
  real time, which is what a [throttle block](/reference/throttle-block/) is for.

## In practice

For **offline batch processing** — running a decoder over a capture to see what it
extracts — you want the file source to run at full speed, with no throttle, so the
job finishes in a fraction of real time. For **feeding a UI or a real-time-paced
demo**, you insert a [throttle](/reference/throttle-block/) after the source so the
graph consumes samples at the recording's true rate rather than saturating a CPU
core and rendering a waterfall that scrolls past instantly.

File sinks are the recording half of the loop: tap one anywhere in a graph to
capture baseband for a bug report, or the demodulated output to compare against a
[golden vector](/reference/golden-test-vectors/). Recording a short segment once
and replaying it forever is the foundation of
[simulation-driven development](/reference/simulation-driven-sdr/) and
[hardware-free testing](/reference/testing-dsp-without-hardware/).

A common gotcha is the **end of file**: a source can either stop (right for a
finite test) or loop back to the start (handy for a continuous demo), and the two
behaviors give very different test semantics — a looped file re-introduces a
discontinuity at each wrap that timing-recovery loops must re-acquire.

## Relevance to SDR

Source and sink blocks are a defining convenience of software radio: because the
device is just another block, replacing it with a file changes nothing downstream.
GNU Radio ships explicit *File Source* and *File Sink* blocks, and essentially
every SDR framework offers equivalents; the [SigMF](/reference/iq-file-format/)
standard exists precisely so a recording made by one tool can be replayed by
another without ambiguity.

**GopherTrunk** embodies the source side directly. Its `replay` path opens a
recorded [cfile](/reference/cfile-format/) and streams the samples through the
identical downconvert-and-decode chain the live scanner drives from a dongle — the
file is the source, and the decoder cannot tell the difference. This is what makes
GT's control-channel decoding reproducible from committed captures and lets field
bugs be reproduced from a reporter's raw recording. GT is a receiver, so its
emphasis is squarely on the source (reading captures) rather than a transmit-style
sink, though writing intermediate baseband to a file for offline analysis fits the
same model.

## Sources

[^gr]: [File Source block](https://wiki.gnuradio.org/index.php/File_Source) — GNU Radio Wiki, on reading raw sample files into a flowgraph and the format/rate considerations involved.
[^sdr]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, on the block-stream architecture that lets a file stand in for a radio device.
