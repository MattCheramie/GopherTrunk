---
slug: cfile-format
title: cfile format
entry_type: term
category: sdr-data-streaming
description: "A cfile is GNU Radio's raw IQ recording format: interleaved little-endian 32-bit float I,Q samples with no header, the format GopherTrunk replays."
keywords: cfile, cfile format, GNU Radio cfile, complex float32, cf32, fc32, IQ recording, file sink, interleaved float IQ, gophertrunk replay
aka: [cfile, .cfile, complex float file]
autolink: true
infobox:
  - { label: Type, value: Raw IQ recording (headerless) }
  - { label: Sample, value: "2× little-endian float32 (I,Q) = 8 bytes" }
  - { label: Origin, value: GNU Radio file sink }
see_also: [iq-file-format, cs16-format, sample-format, file-source-sink, sigmf, interleaved-iq, iq-recording-playback]
cite_urls:
  - https://wiki.gnuradio.org/index.php/File_Sink
  - https://en.wikipedia.org/wiki/GNU_Radio
---

A **cfile** is GNU Radio's raw [IQ](/reference/iq-data/) recording format: a headerless stream of
[interleaved](/reference/interleaved-iq/) little-endian **32-bit float** I and Q values, eight bytes
per complex sample.[^sink] The `c` stands for *complex* and the format is nothing more than what a
GNU Radio **File Sink** writes when fed a `complex` stream — which is exactly why it became the
de-facto lossless capture format, and the format **GopherTrunk** replays via `-format f32`.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A cfile stores each complex sample as two little-endian float32 values, I then Q, eight bytes per sample, with no header." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="90" height="30" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.2"/>
    <text x="75" y="60">I₀ float32</text><text x="75" y="70" font-size="7">4 bytes LE</text>
    <rect x="120" y="44" width="90" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="165" y="60">Q₀ float32</text><text x="165" y="70" font-size="7">4 bytes LE</text>
    <rect x="210" y="44" width="90" height="30" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.2"/>
    <text x="255" y="60">I₁ float32</text>
    <rect x="300" y="44" width="90" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="345" y="60">Q₁ float32</text>
    <text x="415" y="62">· · ·</text>
    <path d="M30 84 h180" stroke="currentColor" stroke-width="1" marker-end="url(#cfar)"/><text x="120" y="98" font-size="8">8 bytes = one sample</text>
  </g>
  <defs><marker id="cfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A cfile is pairs of little-endian float32 values, I then Q, eight bytes per complex sample, with no header.</figcaption>
</figure>

## How it works

The layout is deliberately trivial. Each I and each Q is one IEEE-754 single-precision float in
little-endian byte order; they alternate I, Q, I, Q for the length of the recording. There is no
magic number, no header, no embedded [sample rate](/reference/sample-rate/) — the file is pure
payload. That means, as with any raw [IQ file](/reference/iq-file-format/), you must supply the
sample rate and centre frequency separately: GNU Radio flowgraphs and every replay tool take them
as parameters. Amplitudes are conventionally normalised to roughly ±1.0, the range GNU Radio
sources emit, though nothing enforces it.

The float32 choice trades size for fidelity. At eight bytes per sample a cfile is four times larger
than an 8-bit `cu8` capture, but it preserves the full [dynamic range](/reference/dynamic-range/) of
the samples and needs no scaling before DSP, since the values are already the floating-point form
the math operates on. This makes cfiles the natural intermediate format for analysis and testing,
where disk is cheap and losing bits is not acceptable.

## Relevance to SDR

The cfile is the workhorse of GNU Radio experimentation: drop a
[File Sink](/reference/file-source-sink/) after a source to record, drop a File Source to replay,
and any flowgraph becomes an offline experiment. Tools across the ecosystem — Inspectrum, gr-based
decoders, and analysis scripts in NumPy (`numpy.fromfile(..., dtype=numpy.complex64)`) — read the
same bytes, because `complex64` in NumPy is precisely two interleaved float32 values. The
[SigMF](/reference/sigmf/) standard's `cf32_le` datatype describes exactly this layout, so a cfile
is a SigMF data file waiting for a sidecar.

GopherTrunk consumes cfiles directly. Its offline engine's `-format f32` decoder (also spelled
`float32` or `cfile`) reads interleaved little-endian float32 into the same production receiver and
control-channel pipelines the live daemon runs, so a lock on a replayed cfile implies a lock on air,
and a failure makes the capture a reproducible fixture. This is the format the project's DSP and
replay guidance refers to when it talks about replaying `.cfile` captures: a control-channel
recording saved as a cfile can be re-run bit-for-bit to reproduce a decode problem on any developer's
machine, which is the backbone of GopherTrunk's [record-and-replay](/reference/iq-recording-playback/)
testing.

## In practice

Because the extension is a convention, not a marker, cfiles appear as `.cfile`, `.cf32`, `.fc32`, or
plain `.iq` — always verify the dtype, not the name. A quick sanity check: file size in bytes divided
by eight is the sample count, and dividing that by the sample rate gives the capture duration; if the
duration comes out sensible, the file really is float32 IQ.

## Sources

[^sink]: [File Sink](https://wiki.gnuradio.org/index.php/File_Sink) — GNU Radio wiki, documenting the File Sink block that writes raw interleaved samples (complex → interleaved float32, the cfile) with no header.
