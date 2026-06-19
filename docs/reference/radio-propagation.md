---
slug: radio-propagation
title: Radio propagation
entry_type: term
category: antennas-propagation
description: Radio propagation is the behaviour of radio waves as they travel from transmitter to receiver — line-of-sight, reflection, diffraction, and atmospheric effects.
keywords: radio propagation, line of sight, reflection, diffraction, fading, coverage
aka: [radio propagation, propagation]
autolink: true
infobox:
  - { label: Type, value: Wave-travel behaviour }
  - { label: VHF/UHF, value: Mostly line-of-sight }
  - { label: HF, value: Ionospheric skip possible }
see_also: [multipath-propagation, radio-horizon, ionospheric-propagation, path-loss, frequency-bands]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_propagation
---

**Radio propagation** describes how [radio waves](/reference/radio-wave/) travel from
transmitter to receiver, including line-of-sight travel, reflection, diffraction, and
atmospheric effects.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A curved earth with a tall transmitter tower and a receiver, a straight line-of-sight path, and an obstacle blocking a lower path." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 140 Q230 95 450 140" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <line x1="80" y1="122" x2="80" y2="55" stroke="currentColor" stroke-width="2"/><text x="62" y="48" font-size="9" fill="currentColor">TX</text>
  <line x1="380" y1="120" x2="380" y2="88" stroke="currentColor" stroke-width="2"/><text x="368" y="80" font-size="9" fill="currentColor">RX</text>
  <line x1="80" y1="55" x2="380" y2="88" stroke="currentColor" stroke-width="1.4" stroke-dasharray="5 3"/><text x="200" y="58" font-size="9" fill="currentColor">line of sight</text>
  <rect x="225" y="100" width="14" height="22" fill="currentColor" fill-opacity="0.3"/>
</svg>
<figcaption>At VHF/UHF, propagation is line-of-sight; height and a clear path matter more than raw distance.</figcaption>
</figure>

## How it works

At VHF and UHF, propagation is essentially line-of-sight, bounded by the
[radio horizon](/reference/radio-horizon/); reflections cause
[multipath](/reference/multipath-propagation/). At HF, the
[ionosphere](/reference/ionospheric-propagation/) can refract signals over great
distances. Loss along the way is [path loss](/reference/path-loss/).

## Relevance to SDR

Understanding propagation explains why antenna height and a clear path often matter
more than the radio, and why a distant hilltop system can beat a closer obstructed one.

## Sources

[^wiki]: [Radio propagation](https://en.wikipedia.org/wiki/Radio_propagation) — Wikipedia, on line-of-sight, reflection, diffraction, and atmospheric propagation.
