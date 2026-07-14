---
slug: sample-format
title: Sample format (CS8/CS16/CF32)
entry_type: term
category: sdr-programming
description: A sample format is the numeric encoding of each IQ sample — signed 8- or 16-bit integers or 32-bit floats — that an SDR delivers, setting its dynamic range and the conversion a program must apply.
keywords: sample format, CS8, CS16, CF32, complex int8, complex int16, complex float32, sc16, cf32, sample encoding, IQ format, scaling, normalization, bit depth
aka: [CS8, CS16, CF32, SC16, complex int8, complex int16, complex float32]
autolink: true
infobox:
  - { label: Type, value: Numeric encoding of an IQ sample }
  - { label: Common forms, value: CS8 (int8), CS16 (int16), CF32 (float32) }
  - { label: Sets, value: Dynamic range and bytes/sample }
see_also: [interleaved-iq, iq-data, analog-to-digital-converter, numerical-precision-dsp, dynamic-range, sample-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Audio_bit_depth
  - https://pysdr.org/content/iq_files.html
---

**A sample format** is the numeric type used to store each component of an [IQ sample](/reference/iq-data/): most commonly signed 8-bit integers (**CS8**), signed 16-bit integers (**CS16**), or 32-bit IEEE floats (**CF32**), where the leading *C* denotes complex (an I and a Q value per sample).[^pysdr] The format an SDR emits fixes both the [dynamic range](/reference/dynamic-range/) available to the signal and the number of bytes each sample costs, and it dictates the scaling a program must perform before the DSP can treat the stream as ordinary numbers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Three horizontal bars showing bytes per complex sample and range for CS8 at two bytes, CS16 at four bytes, and CF32 at eight bytes, all mapping to the same normalized minus-one-to-plus-one float range for DSP." xmlns="http://www.w3.org/2000/svg">
  <g font-size="12" fill="currentColor">
    <text x="0" y="24">CS8</text><rect x="70" y="12" width="60" height="18" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="140" y="25" font-size="10">2 B/sample · −128…127</text>
    <text x="0" y="64">CS16</text><rect x="70" y="52" width="120" height="18" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="200" y="65" font-size="10">4 B/sample · −32768…32767</text>
    <text x="0" y="104">CF32</text><rect x="70" y="92" width="240" height="18" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="320" y="105" font-size="10">8 B/sample · float</text>
    <line x1="70" y1="128" x2="380" y2="128" stroke="currentColor" stroke-opacity="0.4"/>
    <text x="70" y="150" font-size="10">all normalize to float −1.0 … +1.0 for DSP</text>
  </g>
</svg>
<figcaption>Wider formats cost more bytes per sample but carry more dynamic range; whatever the wire format, DSP typically works on floats scaled to about −1…+1.</figcaption>
</figure>

## How it works

The [ADC](/reference/analog-to-digital-converter/) in an SDR produces integers of some native width — 8 bits for an RTL-SDR, 12–16 bits for higher-end radios. The sample format is how those integers are packed for transport:

- **CS8** — one signed byte for I and one for Q, range −128…127. Half the bytes of CS16, so it halves USB and disk bandwidth, at the cost of coarse [quantization](/reference/quantization/). Some 8-bit radios also use *unsigned* bytes centred at 127, which the host must re-centre.
- **CS16** — 16-bit signed integers, the workhorse for radios with 12–16-bit ADCs. Often only the low 12 bits are meaningful, the value left- or right-justified within the 16-bit field.
- **CF32** — 32-bit float per component. It wastes bandwidth on a native-integer source but is the natural currency of DSP: no re-scaling needed, no overflow to police, and it is the standard format for IQ files (the `.cfile`).

Converting between them is mechanical but must be done right. Integer-to-float means dividing by the full-scale value (e.g. 32768.0 for CS16) so the signal lands in roughly −1.0…+1.0; the reverse means multiplying and then **clamping** so a hot sample cannot wrap around the integer range into a spurious value. Getting the scale factor or the signed/unsigned convention wrong shows up immediately as a squashed constellation, a DC bias, or clipping.

## Relevance to SDR

Sample format is one of the first decisions a receive pipeline confronts, because it trades three things at once: **bandwidth** (CS8 is half of CS16 over USB and on disk), **dynamic range** (more bits push the [noise floor](/reference/noise-floor/) of the quantizer down, widening the gap between the weakest recoverable signal and a strong nearby one), and **[numerical precision](/reference/numerical-precision-dsp/)** downstream. High-throughput or storage-bound applications favour the narrowest integer format that still meets the link's dynamic-range budget, then convert to float only inside the DSP.

**GopherTrunk** meets this head-on as a pure-Go application spanning several radios. Its RTL-SDR path takes native 8-bit samples, its airspy path unpacks 12-bit ADC data delivered as 16-bit words, and its offline replay reads and writes complex float32 `.cfile` captures. In every case GT converts the wire format to floats scaled near −1…+1 at the front of the chain, so the rest of the demodulator is written once against a single representation regardless of which format the hardware produced. This normalization is also why a capture recorded in one format can be replayed through the same decode path as a live stream in another.

## In practice

Beware endianness and justification: two radios may both claim "16-bit IQ" yet differ in byte order or in whether the 12 valid bits sit in the high or low end of the word. And a persistent DC offset baked into the samples is a property of the capture, not the format — no amount of re-scaling removes it, which is why front-end effects must be reproduced from the raw [interleaved IQ](/reference/interleaved-iq/) rather than blamed on the conversion.

## Sources

[^pysdr]: [IQ files and SigMF](https://pysdr.org/content/iq_files.html) — PySDR, on storing IQ as int8/int16/float32, scaling, and the complex-file convention.
