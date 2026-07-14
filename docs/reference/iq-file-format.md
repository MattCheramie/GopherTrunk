---
slug: iq-file-format
title: IQ file format
entry_type: term
category: sdr-data-streaming
description: "An IQ file format is a convention for storing SDR baseband recordings as raw interleaved I,Q samples, defined by a sample data type and a sample rate."
keywords: IQ file format, IQ recording, interleaved IQ, cfile, cu8, cs16, cf32, complex float, baseband recording, SDR capture file, dtype
aka: [IQ file, IQ recording format, complex sample file]
autolink: true
infobox:
  - { label: Type, value: Raw baseband recording layout }
  - { label: Layout, value: "Interleaved I, Q, I, Q …" }
  - { label: Defined by, value: Sample data type + sample rate }
see_also: [interleaved-iq, sample-format, cfile-format, sigmf, wav-iq-recording, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/In-phase_and_quadrature_components
  - https://pysdr.org/content/iq_files.html
---

An **IQ file format** is a convention for writing a stream of [IQ data](/reference/iq-data/) to
disk as raw [interleaved](/reference/interleaved-iq/) I, Q, I, Q… samples, with the meaning of the
bytes fixed by two parameters: the per-component **data type** and the **sample rate**.[^pysdr]
Most SDR capture files carry no header at all — they are just the samples — so those two numbers
must travel out-of-band, in the filename, in documentation, or in a metadata sidecar.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An IQ file is a flat sequence of interleaved I and Q samples; the data type sets the bytes per value and the sample rate sets the time between pairs." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <g stroke="currentColor" stroke-width="1.2">
      <rect x="30" y="42" width="46" height="26" fill="currentColor" fill-opacity="0.16"/>
      <rect x="76" y="42" width="46" height="26" fill="none"/>
      <rect x="122" y="42" width="46" height="26" fill="currentColor" fill-opacity="0.16"/>
      <rect x="168" y="42" width="46" height="26" fill="none"/>
      <rect x="214" y="42" width="46" height="26" fill="currentColor" fill-opacity="0.16"/>
      <rect x="260" y="42" width="46" height="26" fill="none"/>
    </g>
    <text x="53" y="59">I₀</text><text x="99" y="59">Q₀</text><text x="145" y="59">I₁</text><text x="191" y="59">Q₁</text><text x="237" y="59">I₂</text><text x="283" y="59">Q₂</text>
    <text x="330" y="59">· · ·</text>
    <text x="99" y="90" font-size="8">one complex sample = 2 values</text>
    <text x="360" y="35" font-size="8">dtype → bytes/value</text>
    <text x="360" y="90" font-size="8">rate → pairs/second</text>
  </g>
</svg>
<figcaption>An IQ file is a flat run of interleaved I,Q samples; the data type sets bytes-per-value and the sample rate sets time between pairs.</figcaption>
</figure>

## How it works

To read an IQ file you need to know its **data type** — how each I or Q value is encoded — and its
**sample rate**. The common encodings, and their usual filename tags, are:

- **cu8 / u8** — 8-bit unsigned, one byte per component, DC at 127.5. This is what `rtl_sdr`
  writes and the smallest on disk; a 2.4 MS/s capture is about 4.8 MB/s.
- **cs16 / ci16** — 16-bit signed integer, the native output of many Airspy/SDRplay/USRP paths.
- **cf32 / fc32** — 32-bit IEEE float per component, the GNU Radio [cfile](/reference/cfile-format/);
  bulky (8 bytes per complex sample) but lossless and trivially processed.

The word *complex* and the leading `c` signal that the values are interleaved I and Q rather than a
single real channel. Byte order is almost always little-endian on the platforms SDR runs on. Since
the raw stream has no self-description, a mislabelled dtype is the classic failure mode: read a
cf32 file as cu8 and you get noise, because the parser slices the bytes on the wrong boundaries.

## Relevance to SDR

Raw interleaved IQ files are the universal interchange currency for offline SDR work — capturing a
signal once and replaying it through many decoders, building regression fixtures, and assembling
training sets. Because the format is so minimal, higher-level conventions grew on top of it:
[SigMF](/reference/sigmf/) standardises a JSON sidecar carrying exactly the dtype and rate an IQ
file omits, and the [WAV IQ recording](/reference/wav-iq-recording/) borrows the RIFF header so the
sample rate at least rides along in the file.

GopherTrunk's replay and analysis subcommands read these files directly. The `-format` flag selects
the decoder — `u8` for rtl_sdr 8-bit, `f32` for a GNU Radio [cfile](/reference/cfile-format/), and
`wav` for 16-bit PCM — while `-sample-rate` and `-tune-hz` supply the rate and any residual offset
the bare bytes cannot carry. A capture recorded off a control channel becomes a reproducible test
case: if GopherTrunk fails to lock on a live signal, saving the IQ file turns that failure into
something a developer can replay bit-for-bit on another machine, which is the whole point of a
[sample format](/reference/sample-format/) everyone agrees on.

## In practice

Filenames often encode the parameters, e.g. `capture_450M00_2400000_cu8.iq` — 450.00 MHz centre,
2.4 MS/s, 8-bit unsigned. Adopting that habit, or a SigMF sidecar, is what separates a reusable
capture from an unreadable one six months later. When in doubt about a mystery file, the sample
count divided by the file size in bytes reveals the bytes-per-sample and therefore the likely dtype.

## Sources

[^pysdr]: [IQ Files and SigMF](https://pysdr.org/content/iq_files.html) — PySDR textbook chapter, on storing IQ recordings, the cf32/cs16/cu8 data types, and why metadata must accompany the raw samples.
