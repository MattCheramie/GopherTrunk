---
slug: dbm
title: dBm
entry_type: term
category: rf-fundamentals
description: dBm is power expressed in decibels relative to one milliwatt, giving an absolute signal-strength figure; received radio signals are negative dBm values.
keywords: dBm, decibel milliwatt, absolute power, received signal strength
aka: [dBm]
autolink: true
infobox:
  - { label: Type, value: Absolute power unit }
  - { label: Reference, value: 1 milliwatt }
  - { label: Examples, value: "0 dBm = 1 mW; −80 dBm ≈ solid signal" }
see_also: [decibel, dbfs, noise-floor, signal-to-noise-ratio]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "dBm (Wikipedia)", url: https://en.wikipedia.org/wiki/DBm }
---

**dBm** is power expressed in [decibels](/reference/decibel/) relative to **one
milliwatt**, making it an *absolute* measure of signal strength rather than a mere
ratio. 0 dBm equals 1 mW; +30 dBm is 1 watt.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A dBm scale showing reference points from +30 dBm (1 watt) down to -120 dBm, with received signals in the negative range." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.5"/>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="80" y1="54" x2="80" y2="66" stroke="currentColor"/><text x="80" y="80">+30</text><text x="80" y="44">1 W</text>
    <line x1="170" y1="54" x2="170" y2="66" stroke="currentColor"/><text x="170" y="80">0</text><text x="170" y="44">1 mW</text>
    <line x1="280" y1="54" x2="280" y2="66" stroke="currentColor"/><text x="280" y="80">-80</text><text x="280" y="44">strong RX</text>
    <line x1="400" y1="54" x2="400" y2="66" stroke="currentColor"/><text x="400" y="80">-120 dBm</text><text x="400" y="44">in the noise</text>
  </g>
</svg>
<figcaption>dBm is absolute power referenced to 1 mW. Received signals are negative; closer to zero is stronger.</figcaption>
</figure>

## How it works

Because received radio signals are tiny fractions of a milliwatt, they are **negative**
dBm values — and the one closer to zero is stronger (−70 dBm beats −90 dBm by 100×).

## Relevance to SDR

Receiver meters report signal and [noise-floor](/reference/noise-floor/) levels in dBm;
their difference is the [SNR](/reference/signal-to-noise-ratio/) that determines whether
a signal decodes.
