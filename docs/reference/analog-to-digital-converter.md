---
slug: analog-to-digital-converter
title: Analog-to-digital converter (ADC)
entry_type: term
category: sdr-dsp
description: An analog-to-digital converter samples a continuous signal into discrete numbers; in an SDR its sample rate sets capture bandwidth and its full scale sets the clipping ceiling.
keywords: ADC, analog to digital converter, sampling, quantization, bit depth, ENOB, clipping, dBFS
aka: [analog-to-digital converter, ADC]
autolink: true
infobox:
  - { label: Type, value: Sampling device }
  - { label: Sets, value: Capture bandwidth (rate), clipping (full scale) }
  - { label: Overflow, value: Clipping at 0 dBFS }
see_also: [sample-rate, nyquist-theorem, quantization, dither, enob, oversampling, dbfs, automatic-gain-control, software-defined-radio]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Analog-to-digital_converter
  - https://en.wikipedia.org/wiki/Effective_number_of_bits
---

An **analog-to-digital converter** (**ADC**) measures a continuous signal many times per
second, turning it into a stream of numbers.[^wiki] In an SDR it produces the
[IQ](/reference/iq-data/) samples that software works on, and its two headline specs — how
fast it samples and how finely it resolves amplitude — bound almost everything downstream.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A smooth analog wave overlaid with a stair-step series of sampled values taken at regular intervals." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 70 C 90 15, 160 15, 230 70 S 370 125, 440 70" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <polyline points="20,70 50,52 80,33 110,28 140,40 170,58 200,70 230,70 260,82 290,98 320,103 350,95 380,80 410,70 440,62" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g fill="currentColor"><circle cx="50" cy="52" r="2.5"/><circle cx="110" cy="28" r="2.5"/><circle cx="170" cy="58" r="2.5"/><circle cx="290" cy="98" r="2.5"/><circle cx="350" cy="95" r="2.5"/></g>
  <text x="20" y="118" font-size="9" fill="currentColor">regular samples turn the continuous wave into numbers</text>
</svg>
<figcaption>The ADC measures the signal at a fixed rate, converting the continuous waveform into digital samples.</figcaption>
</figure>

## How it works

Conversion has two axes. Along **time**, the ADC samples at a fixed
[sample rate](/reference/sample-rate/), which sets how much
[bandwidth](/reference/bandwidth/) can be captured per the
[Nyquist theorem](/reference/nyquist-theorem/) and requires an anti-alias filter to keep
out-of-band energy from [folding](/reference/aliasing/) into the band. Along **amplitude**,
each sample is rounded to the nearest of a finite set of levels — this is
[quantization](/reference/quantization/), and the tiny rounding error it introduces is
quantization noise that sets a hard noise floor. An *N*-bit converter offers 2ᴺ levels; each
extra bit adds about 6 dB of ideal [dynamic range](/reference/dynamic-range/), so an 8-bit
RTL-SDR resolves roughly 48 dB while a 12-bit or 14-bit radio resolves far more.

The converter's input range defines **full scale**. Drive it past that ceiling and the
signal **clips**, producing harmonics and [intermodulation](/reference/intermodulation/)
that no software can undo — the peak is measured in [dBFS](/reference/dbfs/), with 0 dBFS
being the ceiling. Real ADCs never reach the ideal bit count; their true resolution is the
[ENOB](/reference/enob/) (effective number of bits), which folds in thermal noise,
distortion, and clock [jitter](/reference/phase-noise/).[^enob]

## Variants

- **Successive-approximation (SAR)** — a binary search per sample; common at moderate
  speeds.
- **Pipeline** — staged conversion for the tens-to-hundreds of MSa/s that direct-sampling
  SDRs need.
- **Delta-sigma (ΔΣ)** — heavily oversamples a coarse quantiser and shapes the noise out of
  band, trading speed for resolution; the basis of many HF/audio front-ends.
- **Flash** — a bank of comparators for the fastest, lowest-resolution conversion.

## In practice

Two techniques squeeze more out of a given converter. [Oversampling](/reference/oversampling/)
runs the ADC faster than Nyquist strictly requires, spreading the fixed quantization noise
over a wider band so that after filtering and [decimation](/reference/decimation/) less of
it lands in the channel — each 4× of oversampling buys about one extra effective bit.
Adding a small amount of [dither](/reference/dither/) — random noise — before quantization
decorrelates the rounding error, trading a slightly higher noise floor for the removal of
ugly quantization *tones*.

## Relevance to SDR

Setting [gain](/reference/automatic-gain-control/) so strong signals stay just below the
ADC's ceiling, without burying weak ones in the quantization and thermal noise floor, is
central to clean reception — the essence of gain staging. GopherTrunk assumes the incoming
IQ has been digitised sanely; a capture that clipped in the ADC (peaks at 0 dBFS) is
unrecoverable regardless of DSP.

## Sources

[^wiki]: [Analog-to-digital converter](https://en.wikipedia.org/wiki/Analog-to-digital_converter) — Wikipedia, on sampling a continuous signal into discrete digital values.
[^enob]: [Effective number of bits](https://en.wikipedia.org/wiki/Effective_number_of_bits) — Wikipedia, on the real-world resolution of an ADC after noise and distortion.
