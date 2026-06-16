---
slug: mueller-muller-timing-recovery
title: Mueller–Müller timing recovery
entry_type: algorithm
category: algorithms
description: Mueller–Müller timing recovery is a decision-directed symbol-timing algorithm that needs only one sample per symbol, making it efficient for many digital demodulators.
keywords: Mueller-Muller, timing recovery, decision directed, symbol timing, clock recovery, one sample per symbol
aka: [Mueller–Müller timing recovery, Mueller-Muller]
autolink: true
infobox:
  - { label: Type, value: Symbol-timing algorithm }
  - { label: Feature, value: One sample per symbol (decision-directed) }
  - { label: Use, value: AIS, APRS, paging demodulators }
see_also: [clock-recovery, gardner-timing-recovery, symbol-rate, ais, aprs]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/clock-recovery/ }
external:
  - { title: "Symbol synchronization (Wikipedia)", url: https://en.wikipedia.org/wiki/Symbol_synchronization }
---

**Mueller–Müller timing recovery** is a decision-directed symbol-timing algorithm that
needs only **one sample per symbol**, making it computationally efficient.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A symbol waveform sampled once per symbol, with successive samples used to estimate and correct timing." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 90 C 90 90 90 40 150 40 C 210 40 210 90 270 90 C 330 90 330 40 390 40" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="150" cy="40" r="3.5"/><circle cx="270" cy="90" r="3.5"/></g>
  <line x1="150" y1="20" x2="150" y2="100" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/><line x1="270" y1="20" x2="270" y2="100" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/>
  <text x="230" y="118" text-anchor="middle" font-size="9" fill="currentColor">one sample per symbol — lower rate than Gardner</text>
</svg>
<figcaption>Mueller–Müller timing recovery needs only one sample per symbol, using successive decisions to correct timing.</figcaption>
</figure>

## How it works

It uses current and previous symbol decisions to estimate the timing error and drive a
loop that keeps sampling at the symbol centre — at the cost of needing reasonably reliable
decisions to start.

## Relevance to SDR

GopherTrunk uses Mueller–Müller recovery in decoders such as [AIS](/reference/ais/),
[APRS](/reference/aprs/), and signalling pipelines.
