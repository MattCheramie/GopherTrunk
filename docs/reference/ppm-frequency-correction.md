---
slug: ppm-frequency-correction
title: PPM frequency correction
entry_type: term
category: sdr-dsp
description: PPM frequency correction compensates an SDR's reference-oscillator error, measured in parts per million, so signals appear at their true frequency and digital modes lock.
keywords: PPM, parts per million, frequency error, oscillator drift, calibration, rotating constellation
aka: [PPM correction, PPM]
autolink: true
infobox:
  - { label: Type, value: Calibration parameter }
  - { label: Corrects, value: Reference-oscillator error }
  - { label: Symptom if wrong, value: Rotating constellation, no lock }
see_also: [local-oscillator, frequency, costas-loop, constellation-diagram]
related_lessons:
  - { title: "Calibration & troubleshooting", url: /learn/rf-sdr/calibration-troubleshooting/ }
external:
  - { title: "Frequency error (Wikipedia)", url: https://en.wikipedia.org/wiki/Clock_drift }
---

**PPM frequency correction** compensates for the small error in an SDR's reference
oscillator, measured in **parts per million**, so signals land on their **true
frequency**. At UHF a 30 PPM error is several kilohertz — more than a channel's width.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A signal sitting off the channel centre due to oscillator error, and the same signal centred after PPM correction." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="220" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="125" y1="30" x2="125" y2="78" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <path d="M150 70 L160 38 L170 70 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="125" y="95" text-anchor="middle" font-size="9" fill="currentColor">off-centre (uncorrected)</text>
  <line x1="245" y1="55" x2="285" y2="55" stroke="currentColor" marker-end="url(#ppar)"/><text x="265" y="48" font-size="8" fill="currentColor" text-anchor="middle">+PPM</text>
  <line x1="300" y1="70" x2="450" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="375" y1="30" x2="375" y2="78" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <path d="M365 70 L375 38 L385 70 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="375" y="95" text-anchor="middle" font-size="9" fill="currentColor">centred (corrected)</text>
  <defs><marker id="ppar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>PPM correction compensates the dongle's oscillator error so signals land on their true frequency.</figcaption>
</figure>

## How it works

Setting the right PPM shifts the [local oscillator](/reference/local-oscillator/) so a
known reference sits exactly where it should. The error drifts a little with temperature,
so warm-up matters.

## Relevance to SDR

A wrong PPM produces the classic *rotating
[constellation](/reference/constellation-diagram/)* that won't lock — fixed by
calibration, not a better antenna.
