---
slug: joseph-fourier
title: Joseph Fourier
entry_type: person
category: people
description: Joseph Fourier (1768–1830) was a French mathematician who showed signals decompose into sinusoids — the basis of the Fourier transform and spectrum analysis.
keywords: Joseph Fourier, Fourier series, Fourier transform, frequency analysis, mathematics
aka: [Joseph Fourier, Fourier]
autolink: true
infobox:
  - { label: Lived, value: "1768–1830" }
  - { label: Field, value: Mathematics / physics }
  - { label: Known for, value: Fourier series and analysis }
see_also: [fourier-transform, fast-fourier-transform, frequency]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/fft-and-waterfall/ }
external:
  - { title: "Joseph Fourier (Wikipedia)", url: https://en.wikipedia.org/wiki/Joseph_Fourier }
---

**Joseph Fourier** (1768–1830) was a French mathematician and physicist who showed that
functions can be represented as sums of sinusoids — the insight behind the
[Fourier transform](/reference/fourier-transform/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A complex waveform shown as the sum of several simple sine waves." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 40 q20 -20 40 0 t40 0 t40 0 t40 0 t40 0 t40 0" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6"/>
  <path d="M20 65 q10 -12 20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6"/>
  <text x="300" y="50" font-size="14" fill="currentColor">= </text>
  <path d="M330 95 q10 -22 20 0 q10 8 20 0 q10 -16 20 0 q10 14 20 0 q10 -10 20 0" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="120" y="110" text-anchor="middle" font-size="9" fill="currentColor">simple sines</text><text x="380" y="110" text-anchor="middle" font-size="9" fill="currentColor">sum</text>
</svg>
<figcaption>Fourier showed any signal can be expressed as a sum of sinusoids — the mathematics behind the FFT and spectra.</figcaption>
</figure>

## Life and work

Studying heat conduction, Fourier introduced what became Fourier series and analysis,
decomposing complex signals into simple frequency components.

## Contribution

His mathematics is the bridge between the time and frequency domains — the very operation
an [FFT](/reference/fast-fourier-transform/) performs.

## Legacy

Every spectrum display and waterfall in an SDR is, at heart, Fourier's idea made digital.
