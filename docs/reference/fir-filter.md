---
slug: fir-filter
title: FIR filter
entry_type: algorithm
category: sdr-dsp
description: A finite impulse response (FIR) filter sums weighted recent input samples — the workhorse SDR digital filter, always stable with exactly linear phase.
keywords: FIR filter, finite impulse response, tapped delay line, linear phase, digital filter, convolution
aka: [FIR, "finite impulse response filter"]
autolink: true
see_also: [digital-filter, iir-filter, decimation, matched-filter, root-raised-cosine-filter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Finite impulse response (Wikipedia)", url: https://en.wikipedia.org/wiki/Finite_impulse_response }
---

A **FIR** (**finite impulse response**) filter produces each output sample as a
**weighted sum of the most recent input samples** — a tapped delay line multiplied by a
set of coefficients (taps). It is the most common [digital filter](/reference/digital-filter/)
in SDR because it is always stable and can have exactly linear phase.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A tapped delay line of samples, each multiplied by a coefficient and summed to form the output." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none"><rect x="40" y="30" width="40" height="26"/><rect x="100" y="30" width="40" height="26"/><rect x="160" y="30" width="40" height="26"/><rect x="220" y="30" width="40" height="26"/></g>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="60" y="48">z⁻¹</text><text x="120" y="48">z⁻¹</text><text x="180" y="48">z⁻¹</text><text x="240" y="48">z⁻¹</text></g>
  <g stroke="currentColor" stroke-width="1"><line x1="60" y1="56" x2="60" y2="85"/><line x1="120" y1="56" x2="120" y2="85"/><line x1="180" y1="56" x2="180" y2="85"/><line x1="240" y1="56" x2="240" y2="85"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="60" y="98">×a₀</text><text x="120" y="98">×a₁</text><text x="180" y="98">×a₂</text><text x="240" y="98">×a₃</text></g>
  <circle cx="300" cy="92" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="300" y="96" text-anchor="middle" font-size="11" fill="currentColor">Σ</text>
  <g stroke="currentColor" stroke-width="1"><line x1="60" y1="100" x2="289" y2="92"/><line x1="120" y1="100" x2="289" y2="92"/><line x1="180" y1="100" x2="289" y2="92"/><line x1="240" y1="100" x2="289" y2="92"/></g>
  <line x1="312" y1="92" x2="360" y2="92" stroke="currentColor" marker-end="url(#firar)"/><text x="385" y="96" font-size="9" fill="currentColor">out</text>
  <defs><marker id="firar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A FIR filter delays the input, scales each tap by a coefficient, and sums them — a direct convolution.</figcaption>
</figure>

## Overview

FIR filters are used for channel selection, [pulse shaping](/reference/pulse-shaping/),
and as the anti-alias filter before [decimation](/reference/decimation/). Their
coefficients directly define the [frequency response](/reference/digital-filter/).
