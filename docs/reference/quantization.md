---
slug: quantization
title: Quantization
entry_type: term
category: sdr-dsp
description: "Quantization rounds each sampled value to the nearest of a finite set of levels; the rounding error appears as quantization noise, and every extra bit buys about 6 dB."
keywords: quantization, quantization noise, quantization error, bit depth, resolution, 6 dB per bit, SQNR, ADC levels, dBFS
aka: [quantisation, amplitude quantization]
autolink: true
infobox:
  - { label: Type, value: Amplitude discretisation }
  - { label: Rule of thumb, value: "~6.02 dB SNR per bit" }
  - { label: Cost, value: Quantization noise }
see_also: [analog-to-digital-converter, dither, enob, dbfs, dynamic-range, digital-to-analog-converter]
cite_urls:
  - https://en.wikipedia.org/wiki/Quantization_(signal_processing)
  - https://en.wikipedia.org/wiki/Signal-to-quantization-noise_ratio
---

**Quantization** is the step in digitising a signal where each sampled amplitude is
rounded to the nearest value in a finite set of levels.[^wiki] Where
[sampling](/reference/analog-to-digital-converter/) discretises a signal in *time*,
quantization discretises it in *amplitude*: a converter with *N* bits has 2<sup>N</sup>
levels, and every real measurement is snapped to whichever level is closest. The small
rounding error left behind is **quantization noise**, and it sets the noise floor of an
ideal converter.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A smooth analog ramp overlaid with a staircase of discrete quantization levels, and below it the sawtooth-shaped rounding error that is the quantization noise." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.25"><line x1="30" y1="30" x2="300" y2="30"/><line x1="30" y1="50" x2="300" y2="50"/><line x1="30" y1="70" x2="300" y2="70"/><line x1="30" y1="90" x2="300" y2="90"/></g>
  <line x1="30" y1="100" x2="300" y2="20" stroke="currentColor" stroke-opacity="0.6"/>
  <polyline points="30,90 60,90 60,70 130,70 130,50 200,50 200,30 300,30" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="34" y="118" font-size="8" fill="currentColor">staircase = quantized samples</text>
  <line x1="330" y1="70" x2="445" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M330 80 L360 60 L360 80 L390 60 L390 80 L420 60 L420 80" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="330" y="118" font-size="8" fill="currentColor">error = quantization noise</text>
  <text x="330" y="48" font-size="8" fill="currentColor">±½ LSB</text>
</svg>
<figcaption>Rounding the analog value to the nearest level produces a staircase; the difference between the two is a sawtooth error, the quantization noise.</figcaption>
</figure>

## How it works

Divide the converter's full-scale range into 2<sup>N</sup> equal steps. The width of one
step is the **least significant bit** (LSB). Rounding to the nearest level means the error
on any sample lies between −½ and +½ LSB. For a busy signal that exercises many levels,
that error behaves like additive random noise, uniformly distributed over one LSB, with
power equal to Δ²/12 (Δ being the step size). This gives the standard result for an ideal
uniform quantizer:

> SQNR ≈ 6.02·*N* + 1.76 dB

where SQNR is the signal-to-quantization-noise ratio for a full-scale sine wave. The
practical takeaway is the **6 dB-per-bit** rule: each additional bit halves the step size,
cutting quantization noise power by a factor of four and adding roughly 6 dB of
[dynamic range](/reference/dynamic-range/). An 8-bit converter reaches about 50 dB, a
12-bit about 74 dB, a 16-bit about 98 dB — before real-world imperfections erode it.

## In practice

The clean 6-dB formula assumes the error is random and uncorrelated with the signal. For
low-level or slowly varying signals that only cross a few levels, that assumption breaks:
the error becomes a **correlated** distortion, producing spurious tones instead of smooth
noise. [Dither](/reference/dither/) — a tiny amount of added noise before quantization —
decorrelates the error, trading those ugly tones for a slightly higher but benign noise
floor. Real converters also fall short of the ideal: differential and integral
nonlinearity, aperture jitter, and thermal noise mean the *effective* resolution,
captured by the [ENOB](/reference/enob/) figure, is always below the nominal bit count.

Quantization is measured relative to full scale, so a signal's headroom is expressed in
[dBFS](/reference/dbfs/); the same discretisation happens in reverse in a
[digital-to-analog converter](/reference/digital-to-analog-converter/).

## Relevance to SDR

An SDR's bit depth sets how far a weak signal can sit below a strong one before the weak
one disappears into quantization noise — the core of receiver
[dynamic range](/reference/dynamic-range/). It is why an 8-bit RTL-SDR struggles when a
strong nearby transmitter shares the passband, while a 12- or 16-bit receiver has room to
spare, and why front-end gain must be staged to fill the ADC without clipping. In a
wideband trunking capture, quantization noise across the whole span competes with every
channel at once, so bit depth directly affects how weak a control channel can be and still
decode.

GopherTrunk consumes already-quantized IQ samples, so it inherits whatever quantization
noise the capture device imposed; its DSP cannot recover detail lost below the LSB, which
is why capture gain staging matters upstream of the decoder.

## Sources

[^wiki]: [Quantization (signal processing)](https://en.wikipedia.org/wiki/Quantization_(signal_processing)) — Wikipedia, on rounding to discrete levels and the resulting quantization error.
