---
slug: nyquist-theorem
title: Nyquist–Shannon sampling theorem
entry_type: term
category: sdr-dsp
description: The Nyquist–Shannon sampling theorem states that a signal must be sampled at least twice its bandwidth to be represented without loss; under-sampling causes aliasing.
keywords: Nyquist theorem, sampling theorem, Nyquist rate, aliasing, bandpass sampling, bandwidth, Shannon
aka: [Nyquist theorem, sampling theorem]
autolink: true
infobox:
  - { label: Type, value: Sampling principle }
  - { label: States, value: Sample ≥ 2× the bandwidth }
  - { label: Named for, value: Harry Nyquist, Claude Shannon }
see_also: [sample-rate, aliasing, bandwidth, bandpass-sampling, oversampling, harry-nyquist, claude-shannon]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem
  - https://en.wikipedia.org/wiki/Undersampling
---

The **Nyquist–Shannon sampling theorem** states that to represent a signal without loss
you must sample at least **twice its [bandwidth](/reference/bandwidth/)**.[^wiki] The
minimum rate is the *Nyquist rate*; half the sample rate is the *Nyquist frequency*, the
highest frequency a given rate can faithfully carry.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A fast wave sampled too slowly, with the sample dots tracing a slower false wave." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q12 -30 24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5"/>
  <g fill="currentColor"><circle cx="20" cy="60" r="3"/><circle cx="92" cy="48" r="3"/><circle cx="164" cy="72" r="3"/><circle cx="236" cy="48" r="3"/><circle cx="308" cy="72" r="3"/><circle cx="380" cy="48" r="3"/><circle cx="430" cy="66" r="3"/></g>
  <path d="M20 60 C 92 48, 92 48, 164 72 S 308 72, 380 48" fill="none" stroke="currentColor" stroke-width="1.8" stroke-dasharray="5 3"/>
  <text x="20" y="112" font-size="9" fill="currentColor">too few samples → a false (aliased) low-frequency wave</text>
</svg>
<figcaption>Nyquist: sample at least twice the bandwidth, or fast signals fold back as false low-frequency aliases.</figcaption>
</figure>

## How it works

The theorem's guarantee is exact and constructive: if a signal contains no energy at or
above the Nyquist frequency, its samples contain *all* the information in the original, and
the continuous waveform can be reconstructed perfectly by sinc interpolation. Nothing is
lost to sampling itself. The catch is the precondition — the signal must be band-limited.
Sample too slowly for the bandwidth actually present and information does not merely vanish;
it **corrupts** the surviving data through [aliasing](/reference/aliasing/), folding
out-of-band energy down onto real signals where it can never be separated again. That is why
every practical sampler is preceded by an anti-alias filter.

For [IQ](/reference/iq-data/) (complex) sampling the useful restatement is that usable
bandwidth ≈ [sample rate](/reference/sample-rate/), because the two quadrature channels
resolve positive and negative frequencies separately and so double the span a real-only
stream of the same rate would cover. In SDR terms, a rate of Fs lets you watch a band Fs
wide centred on the tuned frequency.

## Variants

- **Baseband sampling** — the usual case, assuming energy from 0 up to the Nyquist
  frequency.
- **[Bandpass sampling](/reference/bandpass-sampling/) (undersampling)** — the deeper form
  of the theorem: a signal of bandwidth *B* centred at a high carrier can be sampled at a
  rate just above 2*B*, deliberately aliasing it down to baseband, provided the whole band
  lands inside a single Nyquist zone.[^under] Direct-sampling and some HF receivers exploit
  this.
- **[Oversampling](/reference/oversampling/)** — sampling well above the Nyquist rate to
  relax filter requirements and spread quantization noise.

## In practice

Real filters are not brick walls, so designers leave a guard band and sample a little above
the theoretical minimum — sampling at exactly 2× a component sitting right at the edge is
fragile. The theorem also sets the floor for storage and processing: halving the bandwidth
you need to capture halves the sample rate, the file size, and the CPU load. GopherTrunk
relies on this floor to size its channel rates — 48 kHz comfortably carries a 4800-baud
C4FM channel whose occupied bandwidth is well under 24 kHz.

## Relevance to SDR

It is named for [Harry Nyquist](/reference/harry-nyquist/), who framed the sampling limit in
1928, and [Claude Shannon](/reference/claude-shannon/), who proved it rigorously in 1949; it
sets the floor on the sample rate needed to capture a given channel, and every choice of SDR
rate and decimation factor is an application of it.

## Sources

[^wiki]: [Nyquist–Shannon sampling theorem](https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem) — Wikipedia, on the minimum sampling rate for lossless representation.
[^under]: [Undersampling](https://en.wikipedia.org/wiki/Undersampling) — Wikipedia, on the bandpass form of the theorem that samples below the carrier by controlled aliasing.
