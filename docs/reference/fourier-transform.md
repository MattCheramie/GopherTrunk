---
slug: fourier-transform
title: Fourier transform
entry_type: algorithm
category: algorithms
description: The Fourier transform decomposes a signal into its constituent frequencies, converting between the time and frequency domains; it is the mathematical basis of spectrum analysis.
keywords: Fourier transform, frequency domain, time domain, spectrum, Joseph Fourier
aka: [Fourier transform]
autolink: true
infobox:
  - { label: Type, value: Mathematical transform }
  - { label: Converts, value: Time domain ↔ frequency domain }
  - { label: Named for, value: Joseph Fourier }
see_also: [fast-fourier-transform, joseph-fourier, bandwidth, software-defined-radio]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/rf-sdr/fft-and-waterfall/ }
related_reading:
  - { title: "SDR Internals, Part 8: Equalization, diversity & the FFT", url: /blog/deep-dives/sdr-internals-08-equalization-diversity-fft/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Fourier_transform
---

The **Fourier transform** decomposes a signal into the frequencies that compose it,
converting between the time and frequency domains.[^wiki] It is the mathematical foundation of
spectrum analysis, named for [Joseph Fourier](/reference/joseph-fourier/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A complex time-domain waveform on the left transforms into a set of frequency peaks on the right." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q10 -22 20 0 q10 18 20 0 q10 -28 20 0 q10 24 20 0 q10 -16 20 0 q10 20 20 0 q10 -22 20 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="100" y="105" text-anchor="middle" font-size="9" fill="currentColor">time domain</text>
  <line x1="225" y1="60" x2="265" y2="60" stroke="currentColor" marker-end="url(#ftar)"/><text x="245" y="50" text-anchor="middle" font-size="8" fill="currentColor">FT</text>
  <line x1="290" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="2"><line x1="320" y1="90" x2="320" y2="55"/><line x1="360" y1="90" x2="360" y2="35"/><line x1="400" y1="90" x2="400" y2="68"/></g>
  <text x="365" y="105" text-anchor="middle" font-size="9" fill="currentColor">frequency domain</text>
  <defs><marker id="ftar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The Fourier transform expresses a signal as a sum of frequencies — converting time into spectrum.</figcaption>
</figure>

## How it works

Any signal can be expressed as a sum of sinusoids; the transform reports how much energy
exists at each frequency. Its discrete, efficient form is the
[FFT](/reference/fast-fourier-transform/).

## Relevance to SDR

Turning [IQ](/reference/iq-data/) samples into a spectrum — and thus a waterfall — is a
Fourier transform, the reason you can *see* signals on an SDR.

## Sources

[^wiki]: [Fourier transform](https://en.wikipedia.org/wiki/Fourier_transform) — Wikipedia, for the mathematical definition and time/frequency-domain background.
