---
slug: welch-method
title: Welch's method
entry_type: algorithm
category: algorithms
description: Welch's method estimates a signal's power spectral density by averaging windowed, overlapping periodograms, cutting variance for stable SDR spectrum and occupancy displays.
keywords: Welch method, power spectral density, PSD estimation, averaged periodogram, overlapping segments, variance reduction, Bartlett method, spectral estimation, Peter Welch
aka: [Welch's method, Welch periodogram, weighted overlapped segment averaging, WOSA]
autolink: true
infobox:
  - { label: Type, value: PSD estimator }
  - { label: Method, value: Averaged windowed periodograms }
  - { label: Benefit, value: Lower-variance spectrum }
see_also: [discrete-fourier-transform, fast-fourier-transform, window-function, hardware-spectrum, energy-detection]
cite_urls:
  - https://en.wikipedia.org/wiki/Welch%27s_method
  - https://ieeexplore.ieee.org/document/1161901
---

**Welch's method** estimates the **power spectral density** (PSD) of a signal by cutting it
into overlapping segments, applying a [window](/reference/window-function/) to each, taking
each segment's [FFT](/reference/fast-fourier-transform/) magnitude-squared (its
*periodogram*), and averaging those periodograms together.[^wiki] A single
[DFT](/reference/discrete-fourier-transform/) of a noisy signal is itself extremely noisy —
its variance does not shrink as you take more samples in one long transform — so Welch trades
a little frequency resolution for a much smoother, more trustworthy estimate of how power is
distributed across frequency.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A long sample stream is split into overlapping windowed segments; each is transformed to a periodogram and the periodograms are averaged into one smooth power spectral density curve." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="230" y="14">overlapping windowed segments</text>
    <rect x="30" y="24" width="150" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="110" y="44" width="150" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="190" y="64" width="150" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="270" y="84" width="150" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="90" y="120">FFT &amp; |&#183;|&#178;, then average</text>
    <line x1="230" y1="104" x2="230" y2="126" stroke="currentColor" stroke-width="1.2" marker-end="url(#wear)"/>
    <path d="M270 158 Q320 158 340 132 Q360 108 375 132 Q392 158 440 158" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <line x1="265" y1="158" x2="445" y2="158" stroke="currentColor" stroke-width="1"/>
    <text x="355" y="150">smooth PSD</text>
  </g>
  <defs><marker id="wear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Welch averages the periodograms of many overlapping, windowed segments into one low-variance estimate of power versus frequency.</figcaption>
</figure>

## How it works

Welch's method is a refinement of the earlier Bartlett method (non-overlapping,
un-windowed segments). Its steps are:

1. **Segment.** Divide the record of samples into *K* segments of length *L*, letting them
   overlap — commonly by 50%. Overlap reuses data so more independent-ish averages come from
   the same record, tightening the estimate.
2. **Window.** Multiply each segment by a window (Hann is typical) to control the spectral
   leakage that would otherwise bias the estimate. Overlap also compensates for the data the
   window taper attenuates at segment edges.
3. **Periodogram.** Take the FFT of each windowed segment, form the magnitude squared, and
   normalise by the window's power so the result is a true PSD in power-per-hertz.
4. **Average.** Average the *K* periodograms bin-by-bin. Averaging *K* roughly independent
   estimates cuts the variance by about a factor of *K*.

The core trade-off is resolution versus variance: shorter segments give more averages
(smoother curve) but fewer bins across the band (coarser resolution). Segment length,
overlap fraction, and window are the three knobs the analyst turns.

## In practice

Because the raw periodogram's variance is independent of record length, simply collecting
more data and transforming it in one giant FFT does *not* clean up the estimate — it only
adds bins. Welch is the standard fix, cheap to run in real time as a running average over
successive FFT blocks. The number of averages sets how stable the displayed
[noise floor](/reference/noise-floor/) looks; more averaging gives a steadier baseline but a
slower response to signals that come and go.

## Relevance to SDR

Welch's method is the estimator behind the smooth, stable spectrum and occupancy displays
in most SDR software. Rather than plotting each raw FFT of the incoming
[I/Q data](/reference/iq-data/) — which flickers wildly — the display averages successive
windowed periodograms so the [noise floor](/reference/noise-floor/) settles and weak carriers
stand out. Spectrum-occupancy and band-survey tools rely on the averaged PSD to decide, with
[energy detection](/reference/energy-detection/), which channels carry traffic and to build
long-term usage heat-maps. The same averaged PSD underlies measurements of channel power and
signal-to-noise ratio.

GopherTrunk's FFT-based spectral tooling uses averaging of this kind to present a legible,
low-variance view of a monitored band and to help find control channels among many carriers.
It is a display- and survey-side technique; the protocol decoders themselves work from
time-domain samples rather than the PSD.

## Sources

[^wiki]: [Welch's method](https://en.wikipedia.org/wiki/Welch%27s_method) — Wikipedia, on averaged overlapping windowed periodograms and their variance reduction for PSD estimation.
