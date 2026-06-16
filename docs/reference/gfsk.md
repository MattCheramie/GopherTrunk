---
slug: gfsk
title: GFSK
entry_type: technology
category: modulation
description: GFSK (Gaussian frequency-shift keying) is FSK whose transitions are smoothed by a Gaussian filter to narrow the spectrum — used by AIS, Bluetooth, and IoT radios.
keywords: GFSK, Gaussian frequency-shift keying, AIS modulation, pulse shaping, narrowband FSK
aka: [GFSK, Gaussian FSK]
autolink: true
see_also: [frequency-shift-keying, gmsk, pulse-shaping, ais]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Frequency-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency-shift_keying#Gaussian_frequency-shift_keying }
---

**GFSK** (**Gaussian frequency-shift keying**) is [frequency-shift keying](/reference/frequency-shift-keying/)
in which the data is passed through a **Gaussian filter** before it shifts the carrier,
rounding off the otherwise abrupt frequency steps. This smoothing narrows the
transmitted spectrum, so GFSK fits more signals into less bandwidth.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A sharp two-level FSK transition compared with a smoothly rounded Gaussian-filtered transition." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 80 H100 V40 H170 V80 H240" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5"/>
  <text x="135" y="104" text-anchor="middle" font-size="8.5" fill="currentColor">plain FSK (sharp)</text>
  <path d="M260 80 C 300 80 300 40 330 40 C 360 40 360 80 400 80 C 415 80 420 78 430 78" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="350" y="104" text-anchor="middle" font-size="8.5" fill="currentColor">GFSK (rounded)</text>
</svg>
<figcaption>A Gaussian filter rounds the frequency transitions, narrowing the spectrum at a small cost in detectability.</figcaption>
</figure>

## Overview

GFSK is closely related to [GMSK](/reference/gmsk/) (a special case with a particular
modulation index). It is widely used where spectral efficiency matters at low cost —
[AIS](/reference/ais/) marine transponders, Bluetooth, and many ISM-band radios.

## Relevance

For SDR decoding, GFSK is handled like other FSK with a matched filter tuned to the
Gaussian pulse shape, followed by [symbol-timing recovery](/reference/clock-recovery/).
