---
slug: gardner-timing-recovery
title: Gardner timing recovery
entry_type: algorithm
category: algorithms
description: Gardner timing recovery is a feedback algorithm that estimates symbol timing error from samples taken at the symbol and half-symbol points, without needing carrier phase.
keywords: Gardner timing recovery, symbol timing, timing error detector, clock recovery, non-data-aided
aka: [Gardner timing recovery, Gardner]
autolink: true
infobox:
  - { label: Type, value: Symbol-timing algorithm }
  - { label: Feature, value: Independent of carrier phase }
  - { label: Use, value: Clock recovery for digital modems }
see_also: [clock-recovery, mueller-muller-timing-recovery, symbol-rate, demodulation]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
related_reading:
  - { title: "SDR Internals, Part 7: Symbol timing & sync recovery", url: /blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Symbol_synchronization
---

**Gardner timing recovery** is a feedback algorithm that estimates
[symbol-timing](/reference/clock-recovery/) error using samples at the symbol and
half-symbol instants.[^wiki] A useful property is that it works **independently of carrier
phase**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A symbol waveform sampled twice per symbol — at the midpoint and the peak — to estimate timing error." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 90 C 90 90 90 40 150 40 C 210 40 210 90 270 90 C 330 90 330 40 390 40" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="90" cy="65" r="3"/><circle cx="150" cy="40" r="3"/><circle cx="210" cy="65" r="3"/><circle cx="270" cy="90" r="3"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="150" y="30">peak</text><text x="90" y="80">mid</text></g>
  <text x="230" y="118" text-anchor="middle" font-size="9" fill="currentColor">two samples per symbol estimate the timing error</text>
</svg>
<figcaption>Gardner timing recovery uses two samples per symbol (midpoint and peak) to track symbol timing.</figcaption>
</figure>

## How it works

Its timing-error detector drives a loop that nudges the sampling instant toward the centre
of each [symbol](/reference/symbol-rate/), where the [eye](/reference/eye-diagram/) is
widest, tracking small clock drift.

## Relevance to SDR

Gardner recovery is a common choice in SDR demodulators for locking symbol timing on PSK
and QAM signals.

## Sources

[^wiki]: [Symbol synchronization](https://en.wikipedia.org/wiki/Symbol_synchronization) — Wikipedia, for symbol-timing recovery including the Gardner timing-error detector.
