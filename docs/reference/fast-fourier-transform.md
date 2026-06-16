---
slug: fast-fourier-transform
title: Fast Fourier transform (FFT)
entry_type: algorithm
category: algorithms
description: The fast Fourier transform is an efficient algorithm for computing the discrete Fourier transform, making real-time spectrum and waterfall displays practical.
keywords: FFT, fast Fourier transform, DFT, spectrum, waterfall, bins, Cooley-Tukey
aka: [fast Fourier transform, FFT]
autolink: true
infobox:
  - { label: Type, value: Algorithm }
  - { label: Computes, value: Discrete Fourier transform efficiently }
  - { label: Output, value: Spectrum (bins) }
see_also: [fourier-transform, bandwidth, sample-rate, iq-data]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/fft-and-waterfall/ }
external:
  - { title: "Fast Fourier transform (Wikipedia)", url: https://en.wikipedia.org/wiki/Fast_Fourier_transform }
---

The **fast Fourier transform** (**FFT**) is an efficient algorithm for computing the
discrete [Fourier transform](/reference/fourier-transform/), reducing the work enough to
run many times a second in real time.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A block of time samples feeding an FFT block that outputs a row of frequency bins." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="30" cy="40" r="2.5"/><circle cx="30" cy="55" r="2.5"/><circle cx="30" cy="70" r="2.5"/><circle cx="30" cy="85" r="2.5"/></g>
  <text x="30" y="105" text-anchor="middle" font-size="8" fill="currentColor">samples</text>
  <rect x="70" y="35" width="80" height="60" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="110" y="69" text-anchor="middle" font-size="11" fill="currentColor">FFT</text>
  <line x1="150" y1="65" x2="190" y2="65" stroke="currentColor" marker-end="url(#fftar)"/>
  <line x1="200" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="3"><line x1="220" y1="100" x2="220" y2="80"/><line x1="250" y1="100" x2="250" y2="55"/><line x1="280" y1="100" x2="280" y2="40"/><line x1="310" y1="100" x2="310" y2="70"/><line x1="340" y1="100" x2="340" y2="85"/><line x1="370" y1="100" x2="370" y2="60"/><line x1="400" y1="100" x2="400" y2="90"/></g>
  <text x="320" y="118" text-anchor="middle" font-size="8" fill="currentColor">frequency bins</text>
  <defs><marker id="fftar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The FFT computes the spectrum efficiently, splitting the band into equal frequency bins — the basis of the waterfall.</figcaption>
</figure>

## How it works

It splits the captured [bandwidth](/reference/bandwidth/) into a number of **bins** (the
FFT size); resolution ≈ [sample rate](/reference/sample-rate/) ÷ FFT size. More bins give
finer resolution but slower updates and more CPU.

## Relevance to SDR

The FFT drives the spectrum and waterfall displays used to find signals and spot a steady
[control channel](/reference/control-channel/).
