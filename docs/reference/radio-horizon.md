---
slug: radio-horizon
title: Radio horizon
entry_type: term
category: antennas-propagation
description: The radio horizon is the farthest distance a line-of-sight signal reaches before the Earth's curvature blocks it, slightly beyond the visual horizon due to atmospheric refraction.
keywords: radio horizon, line of sight, Earth curvature, antenna height, coverage range
aka: [radio horizon]
autolink: true
infobox:
  - { label: Type, value: Propagation limit }
  - { label: Extended by, value: Antenna height }
  - { label: Vs visual horizon, value: Slightly farther }
see_also: [radio-propagation, frequency-bands, antenna]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Line-of-sight_propagation
---

The **radio horizon** is the farthest point a line-of-sight signal reaches before the
Earth's curvature gets in the way.[^wiki] It lies slightly beyond the visual horizon because
the atmosphere refracts radio waves a little.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A curved earth with a tall antenna whose line of sight reaches further around the curve than a short antenna." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 140 Q230 80 450 140" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <line x1="120" y1="118" x2="120" y2="40" stroke="currentColor" stroke-width="2"/><text x="120" y="34" text-anchor="middle" font-size="8" fill="currentColor">tall</text>
  <line x1="120" y1="40" x2="370" y2="106" stroke="currentColor" stroke-width="1.3" stroke-dasharray="5 3"/>
  <circle cx="370" cy="106" r="3" fill="currentColor"/><text x="385" y="104" font-size="8" fill="currentColor">horizon</text>
  <text x="230" y="135" text-anchor="middle" font-size="9" fill="currentColor">height extends the radio horizon</text>
</svg>
<figcaption>The radio horizon is the farthest line-of-sight point before the Earth's curve blocks it; raising the antenna extends it.</figcaption>
</figure>

## How it works

Raising either antenna pushes the radio horizon outward, which is why repeaters sit on
towers and hilltops and why getting a receive antenna up high extends range at
[VHF/UHF](/reference/frequency-bands/).

## Relevance to SDR

For line-of-sight bands, antenna height is often the most effective way to reach more
distant systems.

## Sources

[^wiki]: [Line-of-sight propagation](https://en.wikipedia.org/wiki/Line-of-sight_propagation) — Wikipedia, on the radio horizon, Earth curvature, and atmospheric refraction.
