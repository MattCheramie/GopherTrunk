---
slug: fred-gardner
title: Floyd M. Gardner
entry_type: person
category: people
description: Floyd M. Gardner was an engineer and author whose work on phase-locked loops and the Gardner timing-error detector shaped modern digital symbol-timing recovery.
keywords: Floyd Gardner, Gardner timing recovery, phaselock techniques, symbol timing, PLL
aka: ["Floyd Gardner", "Floyd M. Gardner", "Fred Gardner"]
autolink: true
see_also: [gardner-timing-recovery, clock-recovery, mueller-muller-timing-recovery]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Floyd_M._Gardner
---

**Floyd M. Gardner** was an engineer and author whose work on phase-locked loops — and
the **[Gardner timing-error detector](/reference/gardner-timing-recovery/)** — underpins
modern digital **symbol-timing recovery**.[^wiki] His book *Phaselock Techniques* is a standard
reference.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 120" role="img" aria-label="A symbol waveform sampled twice per symbol at the midpoint and peak, the basis of the Gardner detector." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 85 C 90 85 90 35 150 35 C 210 35 210 85 270 85 C 330 85 330 35 390 35" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="90" cy="60" r="3"/><circle cx="150" cy="35" r="3"/><circle cx="210" cy="60" r="3"/><circle cx="270" cy="85" r="3"/></g>
  <text x="220" y="110" text-anchor="middle" font-size="8.5" fill="currentColor">two samples per symbol (midpoint + peak) estimate timing</text>
</svg>
<figcaption>Gardner's detector uses a midpoint and a peak sample per symbol to drive timing recovery.</figcaption>
</figure>

## Contribution

The Gardner detector is prized because it works without carrier phase lock, making it a
common choice in SDR demodulators (it features in several GopherTrunk decoders).

## Sources

[^wiki]: [Floyd M. Gardner](https://en.wikipedia.org/wiki/Floyd_M._Gardner) — Wikipedia, for biography and his work on phase-locked loops and timing recovery.
