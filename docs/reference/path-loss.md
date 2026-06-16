---
slug: path-loss
title: Path loss
entry_type: term
category: rf-fundamentals
description: Path loss is the attenuation a radio signal suffers travelling from transmitter to receiver, dominated by the spreading of energy over distance and obstacles in the way.
keywords: path loss, free space loss, propagation loss, distance, dB
aka: [path loss]
autolink: true
infobox:
  - { label: Type, value: Propagation attenuation }
  - { label: Unit, value: Decibels (dB) }
  - { label: Grows with, value: Distance and frequency }
see_also: [attenuation, radio-propagation, decibel, signal-to-noise-ratio]
related_lessons:
  - { title: "How signals travel", url: /learn/propagation/ }
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "Path loss (Wikipedia)", url: https://en.wikipedia.org/wiki/Path_loss }
---

**Path loss** is the [attenuation](/reference/attenuation/) a signal experiences
travelling from transmitter to receiver. It is dominated by the spreading of energy
over distance, plus losses from terrain, buildings, and foliage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A curve of received power falling steeply with distance, illustrating path loss in decibels." xmlns="http://www.w3.org/2000/svg">
  <line x1="50" y1="20" x2="50" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="50" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M55 28 C 120 70, 200 100, 435 115" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="20" y="70" font-size="10" fill="currentColor" transform="rotate(-90 20 70)">power (dB)</text>
  <text x="240" y="140" text-anchor="middle" font-size="10" fill="currentColor">distance →</text>
</svg>
<figcaption>Path loss grows with distance (and frequency); it can exceed 100 dB over a few kilometres.</figcaption>
</figure>

## How it works

In free space, power falls with the square of distance, and loss rises with
[frequency](/reference/frequency/); real environments add much more. Path loss can
total 100+ dB over a few kilometres, which is why link budgets are done in
[decibels](/reference/decibel/).

## Relevance to SDR

Path loss explains why a distant or obstructed system arrives near the
[noise floor](/reference/noise-floor/), and why antenna height and a clear path
([propagation](/reference/radio-propagation/)) matter so much.
