---
slug: root-raised-cosine-filter
title: Root-raised-cosine filter
entry_type: algorithm
category: sdr-dsp
description: A root-raised-cosine (RRC) filter is a pulse-shaping filter used at both transmitter and receiver to limit bandwidth while minimising intersymbol interference.
keywords: root raised cosine, RRC, pulse shaping, intersymbol interference, matched filter, roll-off
aka: [root-raised-cosine filter, RRC filter]
autolink: true
infobox:
  - { label: Type, value: Pulse-shaping filter }
  - { label: Goal, value: Limit bandwidth, minimise ISI }
  - { label: Used, value: TX and RX (matched pair) }
see_also: [matched-filter, digital-filter, symbol-rate, eye-diagram]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Root-raised-cosine filter (Wikipedia)", url: https://en.wikipedia.org/wiki/Root-raised-cosine_filter }
---

A **root-raised-cosine** (**RRC**) filter is a pulse-shaping
[filter](/reference/digital-filter/) applied at both transmitter and receiver. Split
across the link, the two halves combine into a raised-cosine response that limits
bandwidth while minimising **intersymbol interference**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A root-raised-cosine pulse shape: a central peak with small symmetric ripples that cross zero at adjacent symbol instants." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 90 Q 70 96 110 90 Q 150 80 170 90 Q 200 105 230 35 Q 260 105 290 90 Q 310 80 350 90 Q 390 96 440 90" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g fill="currentColor" fill-opacity="0.5"><circle cx="110" cy="90" r="2"/><circle cx="170" cy="90" r="2"/><circle cx="290" cy="90" r="2"/><circle cx="350" cy="90" r="2"/></g>
  <text x="230" y="120" text-anchor="middle" font-size="9" fill="currentColor">zero at neighbouring symbol times → no inter-symbol interference</text>
</svg>
<figcaption>Root-raised-cosine shaping limits bandwidth while keeping symbols from smearing into their neighbours.</figcaption>
</figure>

## How it works

The roll-off factor trades [bandwidth](/reference/bandwidth/) against pulse compactness.
The receiver's RRC also acts as a [matched filter](/reference/matched-filter/),
maximising SNR at the sampling instant — visible as a clean
[eye diagram](/reference/eye-diagram/).

## Relevance to SDR

Applying the correct RRC is part of demodulating digital signals that use it, sharpening
[symbol](/reference/symbol-rate/) decisions.
