---
slug: quadrature-amplitude-modulation
title: Quadrature amplitude modulation (QAM)
entry_type: technology
category: modulation
description: QAM combines amplitude and phase modulation to pack many bits per symbol; higher-order QAM carries more data but needs a higher SNR to decode.
keywords: QAM, quadrature amplitude modulation, 16-QAM, 64-QAM, constellation, bits per symbol
aka: [quadrature amplitude modulation, QAM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Phase and amplitude }
  - { label: Used by, value: Wi-Fi, cable, LTE, broadcast }
see_also: [phase-shift-keying, frequency-shift-keying, constellation-diagram, signal-to-noise-ratio, iq-data]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Quadrature amplitude modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation }
---

**Quadrature amplitude modulation** (**QAM**) varies **both** the
[phase](/reference/phase/) and [amplitude](/reference/amplitude/) of a carrier, packing
many states into the [IQ](/reference/iq-data/) plane — 16-QAM (4 bits/symbol), 64-QAM,
and higher.

<figure class="figure" markdown="0">
<svg viewBox="0 0 240 240" role="img" aria-label="A 16-QAM constellation: a four-by-four grid of points on the IQ plane." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="120" x2="220" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="120" y1="20" x2="120" y2="220" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="212" y="134" font-size="10" fill="currentColor">I</text><text x="106" y="30" font-size="10" fill="currentColor">Q</text>
  <g fill="currentColor">
    <circle cx="60" cy="60" r="4"/><circle cx="100" cy="60" r="4"/><circle cx="140" cy="60" r="4"/><circle cx="180" cy="60" r="4"/>
    <circle cx="60" cy="100" r="4"/><circle cx="100" cy="100" r="4"/><circle cx="140" cy="100" r="4"/><circle cx="180" cy="100" r="4"/>
    <circle cx="60" cy="140" r="4"/><circle cx="100" cy="140" r="4"/><circle cx="140" cy="140" r="4"/><circle cx="180" cy="140" r="4"/>
    <circle cx="60" cy="180" r="4"/><circle cx="100" cy="180" r="4"/><circle cx="140" cy="180" r="4"/><circle cx="180" cy="180" r="4"/>
  </g>
</svg>
<figcaption>QAM varies both phase and amplitude; 16-QAM packs 4 bits per symbol but needs higher SNR to keep the points apart.</figcaption>
</figure>

## How it works

More bits per symbol means more data in the same [bandwidth](/reference/bandwidth/), but
the states sit closer together, so QAM needs a higher
[SNR](/reference/signal-to-noise-ratio/) to tell them apart.

## Relevance to SDR

QAM appears in Wi-Fi, cable, and LTE rather than scanner voice traffic, but the same
[constellation](/reference/constellation-diagram/) idea applies to reading its quality.
