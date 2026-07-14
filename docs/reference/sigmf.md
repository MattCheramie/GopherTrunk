---
slug: sigmf
title: SigMF (Signal Metadata Format)
entry_type: technology
category: sdr-data-streaming
description: "SigMF is an open standard that pairs a raw IQ recording with a JSON sidecar describing its sample format, RF metadata, captures, and annotations."
keywords: SigMF, Signal Metadata Format, sigmf-data, sigmf-meta, IQ recording metadata, GNU Radio dataset, RF dataset format, signal annotations, DARPA SDR
aka: [SigMF, Signal Metadata Format]
autolink: true
infobox:
  - { label: Type, value: IQ recording + metadata standard }
  - { label: Files, value: ".sigmf-data (raw IQ) + .sigmf-meta (JSON)" }
  - { label: Maintainer, value: "GNU Radio / SigMF community" }
see_also: [iq-file-format, cfile-format, stream-tags, rf-machine-learning, sample-format]
cite_urls:
  - https://github.com/sigmf/SigMF
  - https://en.wikipedia.org/wiki/SigMF
---

**SigMF** (the Signal Metadata Format) is an open standard that stores a raw [IQ](/reference/iq-data/)
recording alongside a small JSON file describing everything a decoder needs to interpret it —
the [sample format](/reference/sample-format/), sample rate, centre frequency, and time-stamped
annotations.[^spec] It solves the oldest problem in SDR file exchange: a bare
[IQ file](/reference/iq-file-format/) is just a wall of numbers, and without out-of-band notes
you cannot even tell whether it is 8-bit unsigned or 32-bit float, let alone what was on the air.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A SigMF recording is two files: a .sigmf-data raw IQ file and a .sigmf-meta JSON file holding global, captures, and annotations objects." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="24" y="52" width="120" height="46" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
    <text x="84" y="72">name.sigmf-data</text><text x="84" y="86" font-size="7.5">raw interleaved IQ</text>
    <rect x="220" y="24" width="216" height="102" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <text x="328" y="20">name.sigmf-meta (JSON)</text>
    <rect x="232" y="34" width="192" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/><text x="328" y="49" font-size="7.5">global: datatype, sample_rate, hw</text>
    <rect x="232" y="62" width="192" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/><text x="328" y="77" font-size="7.5">captures[]: sample_start, frequency</text>
    <rect x="232" y="90" width="192" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/><text x="328" y="105" font-size="7.5">annotations[]: span + label</text>
    <line x1="144" y1="75" x2="219" y2="75" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#sgar)"/>
  </g>
  <defs><marker id="sgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A SigMF recording is a raw IQ dataset file plus a JSON sidecar with global, captures, and annotations objects.</figcaption>
</figure>

## How it works

A SigMF recording is a pair of files sharing a base name. The `.sigmf-data` file holds the
raw interleaved samples — no header, no framing, just the bytes — exactly like a
[GNU Radio cfile](/reference/cfile-format/). The `.sigmf-meta` file is JSON with three parts:

- **`global`** — properties that hold for the whole file: `core:datatype` (e.g. `cf32_le` for
  complex little-endian float32, `cu8` for 8-bit unsigned, `ci16_le` for 16-bit signed),
  `core:sample_rate`, hardware description, author, and SHA-512 hash of the data file.
- **`captures`** — an array of segments, each marking a `core:sample_start` index and the
  RF `core:frequency` and wall-clock time in effect from that sample onward. Retuning mid-file
  produces a new capture entry rather than a new file.
- **`annotations`** — an array of labelled regions, each spanning a sample range and optional
  frequency band, carrying a free-form `core:label` and application-specific keys. This is where
  a bursting signal, a detected emitter, or a training-set class is marked.

The datatype string is the crucial field: it names the element type, real-vs-complex, bit width,
byte order and any offset, so the reader knows precisely how to parse the accompanying data file.
Optional **namespaced extensions** (`antenna:`, `signal:`, `capture_details:`) add domain fields
without polluting the core schema, and the whole thing can be bundled into a single `.sigmf`
tarball archive for transport.

## Relevance to SDR

SigMF has become the lingua franca for *sharing* IQ captures across tools that would otherwise
disagree on format. GNU Radio, Inspectrum, IQEngine, and numerous
[machine-learning](/reference/rf-machine-learning/) pipelines read and write it, and it grew out
of a DARPA-sponsored effort to make RF datasets reproducible and self-describing. For automatic
modulation classification and RF fingerprinting, the annotations array doubles as the label file:
each burst carries its class inline, so a dataset is one directory rather than IQ plus a fragile
external CSV.

GopherTrunk does not read SigMF natively — its offline replay path (`gophertrunk replay`) takes
bare captures via a `-format` flag, understanding `u8` (rtl_sdr 8-bit), `f32` (GNU Radio
[cfile](/reference/cfile-format/) float32), and 16-bit PCM `wav`, with the sample rate and tune
offset supplied on the command line rather than read from a sidecar. Because a SigMF `.sigmf-data`
file *is* just one of those raw layouts, you can replay one by pointing GopherTrunk at the data
file and transcribing `core:datatype`, `core:sample_rate`, and `core:frequency` from the meta JSON
into the corresponding flags — the metadata SigMF standardises is exactly the set of parameters
GopherTrunk otherwise expects you to know. Adopting the SigMF sidecar convention when you record
captures for bug reports keeps them self-describing even though the replay tool reads the raw bytes
directly. SigMF's conceptual cousin inside a live flowgraph is the
[stream tag](/reference/stream-tags/), which attaches the same kind of sample-indexed metadata to
a running stream rather than a file on disk.

## Sources

[^spec]: [SigMF specification](https://github.com/sigmf/SigMF) — the official standard repository, defining the .sigmf-data / .sigmf-meta pair, the global/captures/annotations objects, and the datatype grammar.
