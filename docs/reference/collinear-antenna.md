---
slug: collinear-antenna
title: Collinear antenna
entry_type: term
category: antennas
description: A collinear antenna stacks several in-phase vertical elements along one line to squash the pattern toward the horizon, giving omnidirectional gain over a plain vertical.
keywords: collinear antenna, colinear, stacked dipole, phased vertical, omnidirectional gain, coco antenna, high-gain vertical, base station antenna
aka: [collinear array, colinear antenna, stacked dipole]
autolink: true
infobox:
  - { label: Type, value: Stacked in-phase vertical array }
  - { label: Elements, value: Multiple λ/2 sections, co-phased }
  - { label: Pattern, value: Omnidirectional, low-angle }
see_also: [monopole-antenna, ground-plane-antenna, antenna-gain, radiation-pattern, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Collinear_antenna_array
  - https://en.wikipedia.org/wiki/Antenna_gain
---

A **collinear antenna** is an array of several radiating elements arranged end-to-end
along a single vertical line and driven **in phase**, so their fields add toward the
horizon.[^wiki] Unlike a [Yagi](/reference/yagi-uda-antenna/), it stays
**omnidirectional** in azimuth; the [gain](/reference/antenna-gain/) comes purely from
compressing the vertical [radiation pattern](/reference/radiation-pattern/) into a
flatter, disc-like lobe. It is the standard high-gain base-station vertical for VHF/UHF.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A vertical collinear antenna of stacked half-wave elements separated by phasing sections, with its radiation pattern squashed into a thin low-angle disc compared with a fatter single-element pattern." xmlns="http://www.w3.org/2000/svg">
  <line x1="130" y1="30" x2="130" y2="70" stroke="currentColor" stroke-width="3"/>
  <path d="M130 70 q 12 8 0 16" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="130" y1="86" x2="130" y2="126" stroke="currentColor" stroke-width="3"/>
  <path d="M130 126 q 12 8 0 16" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="130" y1="142" x2="130" y2="182" stroke="currentColor" stroke-width="3"/>
  <text x="140" y="55" font-size="9" fill="currentColor">λ/2</text>
  <text x="146" y="82" font-size="8" fill="currentColor">phasing</text>
  <text x="140" y="112" font-size="9" fill="currentColor">λ/2</text>
  <text x="140" y="168" font-size="9" fill="currentColor">λ/2</text>
  <ellipse cx="320" cy="106" rx="90" ry="14" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="90" font-size="9" fill="currentColor">collinear: thin, low-angle</text>
  <ellipse cx="320" cy="150" rx="45" ry="30" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
  <text x="285" y="195" font-size="8" fill="currentColor">single element (fatter)</text>
</svg>
<figcaption>Stacking co-phased half-wave sections squashes the vertical pattern into a thin disc aimed at the horizon, adding gain while staying omnidirectional.</figcaption>
</figure>

## How it works

A single [dipole](/reference/dipole-antenna/) or [monopole](/reference/monopole-antenna/)
radiates in a fat doughnut — much of its energy goes at steep angles into the sky and the
ground, wasted for terrestrial work. A collinear stacks two or more half-wave elements
vertically and feeds them so their currents are **in phase**. In the horizontal plane the
fields from all sections add directly; at angles above and below the horizon they arrive
with path differences that make them partly cancel. The net effect is **array gain**:
the same total power is squeezed into a thinner, lower-angle lobe, raising signal strength
toward distant stations near the horizon.

The engineering problem is keeping the elements in phase. Simply butting half-wave rods
together would put adjacent sections out of phase, because current reverses every half
wavelength. Collinears solve this with **phasing sections** between the radiating
elements: quarter-wave stubs, folded loops, or coaxial phase-reversal sections (as in the
popular "coco" coaxial-collinear) that flip the phase back so every radiating segment adds
in step. The more sections stacked, the higher the gain — typically 3 dBd for a two-element
collinear up toward 6–9 dBd for tall commercial verticals — but the vertical
[beamwidth](/reference/beamwidth/) narrows accordingly.

That narrow vertical lobe is the main caveat. A high-gain collinear aimed flat at the
horizon can *underperform* for very local, high-angle signals — a nearby site up a hill
may sit in a pattern null. Height and a clear horizon are what let the low-angle gain pay
off.

## Relevance to SDR

For a fixed SDR scanning station a collinear is the go-to when you want **more range
without giving up omnidirectional coverage**. Because it hears all directions at once, it
suits scanning a region full of [trunking sites](/reference/trunking-site/), and its extra
low-angle gain pulls in weak distant signals that a plain quarter-wave
[ground-plane](/reference/ground-plane-antenna/) would miss. Its vertical
[polarization](/reference/polarization/) matches land-mobile P25, DMR, and NXDN traffic.

GopherTrunk is a receive-only decoder and benefits from the improved signal-to-noise a
collinear provides, with no configuration needed. The trade-offs versus other antennas
are the usual ones: a [discone](/reference/discone-antenna/) covers far more bandwidth at
lower gain, and a [Yagi](/reference/yagi-uda-antenna/) gives more gain but only in one
direction. A collinear occupies the middle ground — band-limited, omnidirectional, and
higher-gain than a single vertical.

## Sources

[^wiki]: [Collinear antenna array](https://en.wikipedia.org/wiki/Collinear_antenna_array) — Wikipedia, for the in-phase stacked-element principle, phasing sections, and horizon-directed gain.
