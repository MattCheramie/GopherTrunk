---
slug: clock-recovery
title: Clock recovery
entry_type: term
category: sdr-dsp
description: Clock recovery determines a digital signal's symbol timing from the signal itself, so the receiver samples each symbol at its centre where the eye is widest.
keywords: clock recovery, symbol timing, timing recovery, symbol synchronization, Gardner, Mueller-Muller
aka: [clock recovery, symbol timing]
autolink: true
infobox:
  - { label: Type, value: Timing-synchronisation stage }
  - { label: Recovers, value: Symbol timing from the signal }
  - { label: Algorithms, value: Gardner, Mueller–Müller }
see_also: [gardner-timing-recovery, mueller-muller-timing-recovery, symbol-rate, eye-diagram, demodulation]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
external:
  - { title: "Clock recovery (Wikipedia)", url: https://en.wikipedia.org/wiki/Clock_recovery }
---

**Clock recovery** determines a digital signal's [symbol](/reference/symbol-rate/) timing
from the signal itself, since the transmitter's clock is not shared. It lets the receiver
sample each symbol at its **centre**, where the [eye](/reference/eye-diagram/) is widest.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 140" role="img" aria-label="An eye diagram with a dashed line at the centre showing where clock recovery aims the sampling instant." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.1" stroke-opacity="0.85">
    <path d="M40 30 C120 30 120 110 200 110 C280 110 280 30 360 30"/>
    <path d="M40 110 C120 110 120 30 200 30 C280 30 280 110 360 110"/>
  </g>
  <line x1="200" y1="20" x2="200" y2="120" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="200" y="136" text-anchor="middle" font-size="10" fill="currentColor">recovered clock samples at the eye centre</text>
</svg>
<figcaption>Clock recovery finds the symbol rhythm from the signal itself so each symbol is sampled at its centre.</figcaption>
</figure>

## How it works

A timing-recovery loop watches where transitions fall and nudges the sampling instant to
stay centred, tracking small clock drift. Common algorithms are
[Gardner](/reference/gardner-timing-recovery/) and
[Mueller–Müller](/reference/mueller-muller-timing-recovery/).

## Relevance to SDR

Loss of symbol lock — from low SNR or [multipath](/reference/multipath-propagation/) —
closes the eye and breaks the decode, a key thing the scopes reveal.
