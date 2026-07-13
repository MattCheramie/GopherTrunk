---
slug: discrete-fourier-transform
title: Discrete Fourier transform (DFT)
entry_type: algorithm
category: algorithms
description: The discrete Fourier transform (DFT) converts N sampled signal points into N frequency bins, giving SDRs the spectrum used for waterfalls and channel detection.
keywords: discrete Fourier transform, DFT, frequency bins, spectral resolution, DFT vs FFT, spectrum analysis, SDR waterfall, bin spacing, twiddle factors
aka: [DFT]
autolink: true
infobox:
  - { label: Type, value: Spectral transform }
  - { label: Maps, value: N samples to N frequency bins }
  - { label: Complexity, value: O(N^2) direct, O(N log N) via FFT }
see_also: [fourier-transform, fast-fourier-transform, window-function, welch-method, goertzel-algorithm, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/Discrete_Fourier_transform
  - https://ieeexplore.ieee.org/document/1162034
---

The **discrete Fourier transform (DFT)** is the finite, sampled form of the
[Fourier transform](/reference/fourier-transform/): it takes a block of *N* evenly
spaced samples and returns *N* complex numbers, each describing the amplitude and
[phase](/reference/phase/) of one frequency component present in that block.[^wiki] Where
the continuous transform integrates over all time and yields a continuous spectrum, the
DFT sums over a finite record and produces a **discrete** spectrum — one value per bin —
which is exactly what a computer can hold and what a software-defined radio needs to see
its band.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A block of time samples on the left transforms into a set of discrete frequency bins on the right, each bin a vertical bar of differing height." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="70" y="16">N time samples</text>
    <g stroke="currentColor" stroke-width="1.2">
      <line x1="20" y1="110" x2="130" y2="110"/>
      <line x1="30" y1="110" x2="30" y2="70"/><line x1="45" y1="110" x2="45" y2="55"/>
      <line x1="60" y1="110" x2="60" y2="85"/><line x1="75" y1="110" x2="75" y2="60"/>
      <line x1="90" y1="110" x2="90" y2="95"/><line x1="105" y1="110" x2="105" y2="72"/>
      <line x1="120" y1="110" x2="120" y2="88"/>
    </g>
    <text x="230" y="70">DFT</text>
    <line x1="150" y1="90" x2="300" y2="90" stroke="currentColor" stroke-width="1.3" marker-end="url(#dftar)"/>
    <text x="390" y="16">N frequency bins</text>
    <g stroke="currentColor" stroke-width="3">
      <line x1="320" y1="110" x2="320" y2="100"/><line x1="335" y1="110" x2="335" y2="62"/>
      <line x1="350" y1="110" x2="350" y2="80"/><line x1="365" y1="110" x2="365" y2="104"/>
      <line x1="380" y1="110" x2="380" y2="96"/><line x1="395" y1="110" x2="395" y2="70"/>
      <line x1="410" y1="110" x2="410" y2="102"/><line x1="425" y1="110" x2="425" y2="88"/>
    </g>
    <line x1="315" y1="110" x2="432" y2="110" stroke="currentColor" stroke-width="1"/>
    <text x="373" y="128">frequency &#8594;</text>
  </g>
  <defs><marker id="dftar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The DFT maps a finite record of N samples into N discrete frequency bins — the spectrum a software radio plots as a waterfall.</figcaption>
</figure>

## How it works

Given complex samples `x[0] … x[N-1]`, the DFT computes each output bin as a sum:

`X[k] = Σ x[n] · e^(-j2πkn/N)` for `n = 0 … N-1`.

Each output `X[k]` is the correlation of the input against a complex sinusoid ("twiddle
factor") that completes exactly *k* whole cycles across the block. If the signal contains
energy at that frequency, the products line up and the sum is large; if not, they cancel.
The result `X[k]` is complex — its magnitude gives the component's amplitude and its angle
gives the phase.

- **Bin spacing.** For a block of *N* samples taken at [sample rate](/reference/sample-rate/)
  `fs`, bin *k* sits at frequency `k · fs / N`. The resolution — the gap between adjacent
  bins — is therefore `fs / N` hertz. More samples (a longer record) means finer resolution
  but a longer observation time; this is the fundamental time–frequency trade.
- **Bin range and aliasing.** The *N* bins span 0 to `fs`, with the upper half representing
  negative frequencies for real input. Anything above the [Nyquist](/reference/nyquist-theorem/)
  limit folds back down, so [aliasing](/reference/aliasing/) sets what the transform can
  honestly show.
- **Windowing.** A raw block has hard edges, and the DFT assumes the block repeats forever.
  The discontinuity smears energy across bins (spectral leakage), so a
  [window function](/reference/window-function/) is normally applied first.

## Relation to the FFT

The direct sum above costs `O(N^2)` operations — one full inner product per bin. The
[fast Fourier transform](/reference/fast-fourier-transform/) is not a different transform;
it is a family of algorithms that compute the *same* DFT in `O(N log N)` by recursively
reusing shared twiddle-factor products. For the block sizes used in a real waterfall
(1024, 4096, 65536 points), the FFT is thousands of times faster, which is why practical
SDR software never runs the naive sum — it always calls an FFT. When only a handful of
specific bins are needed rather than the whole spectrum, the
[Goertzel algorithm](/reference/goertzel-algorithm/) evaluates individual DFT terms even
more cheaply than a full FFT.

## Relevance to SDR

The DFT is the workhorse of spectral display and detection. Every SDR waterfall, panadapter,
and [spectrum plot](/reference/hardware-spectrum/) is a stream of DFTs (via FFT) of the
incoming [I/Q data](/reference/iq-data/), one block after another, stacked over time.
[Energy detection](/reference/energy-detection/) — deciding whether a channel is busy —
compares the magnitude of the relevant bins against a threshold. Averaged DFTs form the
periodograms behind [Welch's method](/reference/welch-method/) for power-spectral-density
estimation, and DFT-based fast convolution underlies efficient filtering and
[channelization](/reference/channelizer/).

GopherTrunk uses FFT-based spectral processing to visualise wideband captures and to help
locate control channels within a monitored band; the transform it computes there is, by
definition, the DFT. The decode chain proper leans more on time-domain filtering and
symbol recovery, but the DFT remains the lens through which the operator and the scanner
first see the spectrum.

## Sources

[^wiki]: [Discrete Fourier transform](https://en.wikipedia.org/wiki/Discrete_Fourier_transform) — Wikipedia, on the definition, bin structure, and relationship to the continuous transform and the FFT.
