---
slug: eye-diagram
title: Eye diagram
entry_type: term
category: modulation
description: An eye diagram overlays many symbol periods of a demodulated signal against time; a wide-open "eye" indicates good timing and noise margin for decoding.
keywords: eye diagram, eye pattern, symbol timing, noise margin, jitter, 4FSK
aka: [eye diagram, eye pattern]
autolink: true
infobox:
  - { label: Type, value: Signal-quality display }
  - { label: Axes, value: Amplitude vs. time (overlaid) }
  - { label: Open eye, value: Good timing/noise margin }
see_also: [constellation-diagram, clock-recovery, symbol-rate, c4fm]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
external:
  - { title: "Eye pattern (Wikipedia)", url: https://en.wikipedia.org/wiki/Eye_pattern }
---

An **eye diagram** overlays many short segments of a demodulated signal, each one
symbol period long, so they stack into characteristic "eye" shapes between the symbol
levels.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="An eye diagram with an open eye shape and a dashed vertical line marking the ideal sampling instant at its widest point." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" stroke-opacity="0.85">
    <path d="M40 30 C120 30 120 120 200 120 C280 120 280 30 360 30"/>
    <path d="M40 120 C120 120 120 30 200 30 C280 30 280 120 360 120"/>
    <path d="M40 30 C120 30 120 120 200 120"/><path d="M200 30 C280 30 280 120 360 120"/>
  </g>
  <line x1="200" y1="20" x2="200" y2="130" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="200" y="146" text-anchor="middle" font-size="10" fill="currentColor">sample here (eye widest)</text>
</svg>
<figcaption>An eye diagram overlays symbol periods; a wide-open eye means good timing and noise margin.</figcaption>
</figure>

## How it works

The **wider and taller the eye opening**, the more margin the decoder has to sample
each [symbol](/reference/symbol-rate/) correctly. Noise and timing
[jitter](/reference/clock-recovery/) close the eye. A 4-level signal like
[C4FM](/reference/c4fm/) shows three stacked eyes.

## Relevance to SDR

GopherTrunk's eye-diagram panel shows timing and noise margin at a glance, complementing
the [constellation](/reference/constellation-diagram/) for diagnosing a marginal signal.
