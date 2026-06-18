---
slug: pulse-shaping
title: Pulse shaping
entry_type: term
category: modulation
description: Pulse shaping filters each transmitted symbol so the signal occupies less bandwidth and avoids inter-symbol interference — commonly with a root-raised-cosine filter.
keywords: pulse shaping, root-raised-cosine, inter-symbol interference, spectral splatter, Nyquist filter
aka: ["pulse shaping"]
autolink: true
see_also: [root-raised-cosine-filter, bandwidth, symbol-rate, matched-filter]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
external:
  - { title: "Pulse shaping (Wikipedia)", url: https://en.wikipedia.org/wiki/Pulse_shaping }
---

**Pulse shaping** is filtering each transmitted symbol's pulse so the signal occupies
**less bandwidth** and successive symbols don't smear into one another
(inter-symbol interference). Sharp rectangular pulses spray energy into adjacent
channels; a shaped pulse keeps it contained.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A rectangular pulse with a wide spectrum versus a shaped pulse with a narrow spectrum." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 90 V50 H80 V90" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5"/>
  <text x="55" y="106" text-anchor="middle" font-size="8" fill="currentColor">rectangular</text>
  <path d="M150 90 Q 185 90 195 55 Q 205 35 215 55 Q 225 90 260 90" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="205" y="106" text-anchor="middle" font-size="8" fill="currentColor">shaped</text>
  <line x1="300" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M300 88 L350 88 L360 78 L370 88 L420 88" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <path d="M300 88 C 340 88 340 50 360 30 C 380 50 380 88 420 88" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.5" stroke-dasharray="3 2"/>
  <text x="360" y="106" text-anchor="middle" font-size="8" fill="currentColor">narrow vs wide spectrum</text>
</svg>
<figcaption>Shaping each symbol pulse (often with a root-raised-cosine filter) contains the signal's spectrum.</figcaption>
</figure>

## Overview

The most common choice is the [root-raised-cosine filter](/reference/root-raised-cosine-filter/),
split between transmitter and receiver so that together they form a Nyquist filter with
zero inter-symbol interference at the sampling instants.
