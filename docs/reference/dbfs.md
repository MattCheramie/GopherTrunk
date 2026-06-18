---
slug: dbfs
title: dBFS
entry_type: term
category: rf-fundamentals
description: dBFS is level expressed in decibels relative to digital full scale, used inside an SDR; 0 dBFS is the ADC's maximum, and exceeding it causes clipping.
keywords: dBFS, decibels full scale, ADC, clipping, headroom, digital level
aka: [dBFS]
autolink: true
infobox:
  - { label: Type, value: Digital level unit }
  - { label: Reference, value: ADC full scale (0 dBFS max) }
  - { label: Risk at 0 dBFS, value: Clipping / distortion }
see_also: [decibel, dbm, analog-to-digital-converter, automatic-gain-control]
related_lessons:
  - { title: "Gain, AGC & avoiding overload", url: /learn/rf-sdr/gain-and-agc/ }
external:
  - { title: "dBFS (Wikipedia)", url: https://en.wikipedia.org/wiki/DBFS }
---

**dBFS** (decibels relative to **full scale**) measures level inside the digital domain
of an SDR. Here **0 dBFS is the ceiling** — the largest value the
[ADC](/reference/analog-to-digital-converter/) can represent — and real samples sit
below it as negative numbers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 150" role="img" aria-label="A vertical scale with 0 dBFS at the top as the ADC ceiling, a signal sitting below it with headroom, and the noise near the bottom." xmlns="http://www.w3.org/2000/svg">
  <rect x="120" y="20" width="60" height="110" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="120" y1="30" x2="180" y2="30" stroke="currentColor" stroke-width="2"/>
  <text x="190" y="33" font-size="10" fill="currentColor">0 dBFS (clip)</text>
  <rect x="120" y="55" width="60" height="55" fill="currentColor" fill-opacity="0.2"/>
  <text x="190" y="58" font-size="10" fill="currentColor">headroom</text>
  <text x="190" y="85" font-size="10" fill="currentColor">signal</text>
  <text x="190" y="125" font-size="10" fill="currentColor">noise</text>
</svg>
<figcaption>dBFS is the digital scale inside the SDR; 0 dBFS is the ADC's ceiling, so real samples sit below it.</figcaption>
</figure>

## How it works

If a signal reaches 0 dBFS the converter **clips**, flattening peaks and spraying
distortion across the spectrum. Leaving headroom below 0 dBFS keeps the signal clean.

## Relevance to SDR

[Gain](/reference/automatic-gain-control/) should be set so the strongest signal peaks
comfortably under 0 dBFS. dBm describes the world outside the radio;
[dBFS](/reference/dbfs/) describes the world inside it.
