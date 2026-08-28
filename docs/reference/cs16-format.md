---
slug: cs16-format
title: cs16 format
entry_type: term
category: sdr-data-streaming
description: "cs16 is a raw IQ recording and streaming format: headerless interleaved little-endian 16-bit signed I,Q pairs — half the size of float32 cfiles, and the native sample format of many SDR front ends."
keywords: cs16, sc16, int16 IQ, interleaved IQ, raw IQ capture, complex short, SDR recording format, SoapySDR CS16, gophertrunk replay, 16-bit samples
aka: [cs16, sc16, complex int16, ci16]
autolink: true
infobox:
  - { label: Type, value: Raw IQ recording (headerless) }
  - { label: Sample, value: "2× little-endian int16 (I,Q) = 4 bytes" }
  - { label: Full scale, value: "±32767 → 0 dBFS" }
  - { label: Metadata, value: "None — rate and frequency travel separately" }
see_also: [cfile-format, iq-data, iq-file-format, sigmf, wav-iq-recording, dbfs]
cite_urls:
  - https://en.wikipedia.org/wiki/Raw_image_format
  - https://github.com/pothosware/SoapySDR/wiki
---

**cs16** ("complex signed 16-bit", also written **sc16**) is a raw
[IQ](/reference/iq-data/) format: an interleaved stream of little-endian 16-bit signed
integers, `I₀ Q₀ I₁ Q₁ …`, with no header, no metadata, and nothing else in the
file.[^soapy] One complex sample costs 4 bytes — half the size of the float32
[cfile](/reference/cfile-format/) — and because 12–16-bit
[ADCs](/reference/analog-to-digital-converter/) produce integer samples natively, cs16 is
often a bit-exact record of what the hardware delivered rather than a converted copy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A byte stream divided into repeating four-byte groups, each holding a little-endian 16-bit I value then a 16-bit Q value, with no header before the first sample." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="24" y="40" width="52" height="28"/><rect x="76" y="40" width="52" height="28"/>
    <rect x="132" y="40" width="52" height="28"/><rect x="184" y="40" width="52" height="28"/>
    <rect x="240" y="40" width="52" height="28"/><rect x="292" y="40" width="52" height="28"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="50" y="58">I₀</text><text x="102" y="58">Q₀</text><text x="158" y="58">I₁</text><text x="210" y="58">Q₁</text><text x="266" y="58">I₂</text><text x="318" y="58">Q₂</text>
  </g>
  <text x="352" y="58" font-size="10" fill="currentColor">…</text>
  <text x="24" y="32" font-size="8" fill="currentColor">byte 0 — no header</text>
  <text x="184" y="86" font-size="8.5" fill="currentColor" text-anchor="middle">each box = one little-endian int16 (2 bytes); one complex sample = 4 bytes</text>
</svg>
<figcaption>File offset zero is already sample data: everything you need to know about the capture — rate, centre frequency, scaling — has to travel alongside the file.</figcaption>
</figure>

## Conventions

- **Scaling.** Full scale is ±32767, which maps to 0 [dBFS](/reference/dbfs/); consumers
  typically divide by 32768 to get ±1.0 floats. A capture's peak level in dBFS is
  meaningful and portable — one reason integer captures are convenient evidence when
  overload or gain staging is in question.
- **Endianness and order.** Little-endian, I before Q, is the near-universal convention
  (matching SoapySDR's `CS16` stream format and USRP `sc16` wire format), but nothing in
  the file enforces it — a byte-swapped or IQ-swapped read produces plausible-looking
  garbage, so trust but verify with a spectrum sanity check.
- **No metadata.** Sample rate and centre frequency must be carried in a sidecar file, the
  filename, or a note. [SigMF](/reference/sigmf/) exists precisely to formalise this; a
  bare `.cs16`/`.raw` file depends on discipline.

Against its neighbours: a [cfile](/reference/cfile-format/) (float32) spends twice the
bytes to preserve processing headroom beyond the ADC's range — worthwhile for *processed*
signals, moot for raw captures of a 12-bit front end. 8-bit formats
(rtl_sdr's unsigned `u8`) halve the size again at the cost of dynamic range. A 2-channel
16-bit [WAV](/reference/wav-iq-recording/) is cs16 plus a 44-byte header that documents the
sample rate — a big usability win the raw format trades away for simplicity.

## Relevance to SDR

cs16 is GopherTrunk's workhorse capture format: `gophertrunk replay -in capture.raw
-format cs16 -sample-rate …` replays one, the repository's regression fixtures
(`testdata/*.cs16`) and operators' field captures use it, and the pre-combine
`diversity_capture` tap writes one headerless cs16 file per receiver branch plus a JSON
sidecar carrying the metadata the format itself omits. The half-size-of-float economy is
the reason: a 30-second two-branch capture at 250 kS/s is ~120 MB in float32 and ~60 MB in
cs16, with nothing lost — the samples were 16-bit integers on the wire anyway. The habits
that make headerless captures useful later are the same ones this format demands: record
the sample rate, centre frequency, gain, and device alongside every file, at capture time,
every time.

## Sources

[^soapy]: [SoapySDR wiki](https://github.com/pothosware/SoapySDR/wiki) — Pothosware, on the CS16 complex-int16 stream format and its role as a native SDR sample type.
