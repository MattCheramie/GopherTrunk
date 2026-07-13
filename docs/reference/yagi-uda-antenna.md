---
slug: yagi-uda-antenna
title: Yagi-Uda antenna (Yagi)
entry_type: term
category: antennas
description: A Yagi-Uda antenna is a directional array of a driven element plus a reflector and directors that focus gain and front-to-back ratio in one direction.
keywords: yagi antenna, yagi-uda, beam antenna, directional antenna, parasitic array, reflector, director, driven element, antenna gain, boom
aka: [yagi, yagi-uda, beam antenna]
autolink: true
infobox:
  - { label: Type, value: Parasitic directional array }
  - { label: Elements, value: Reflector + driven + directors }
  - { label: Pattern, value: Unidirectional, high gain }
see_also: [antenna-gain, radiation-pattern, front-to-back-ratio, log-periodic-antenna, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Yagi%E2%80%93Uda_antenna
  - https://en.wikipedia.org/wiki/Parasitic_element_(electrical_networks)
---

A **Yagi-Uda antenna** (usually just "Yagi") is a directional
[antenna](/reference/antenna/) made of one driven [dipole](/reference/dipole-antenna/)
plus several **parasitic elements** — a longer reflector behind it and one or more
shorter directors in front — all mounted on a common boom.[^wiki] The parasitic elements
are not fed; they re-radiate coupled energy with the right phase to reinforce the signal
in one direction and cancel it in others, producing high
[gain](/reference/antenna-gain/) and a strong
[front-to-back ratio](/reference/front-to-back-ratio/). It is the classic "TV aerial"
shape and the standard high-gain beam for point-to-point VHF/UHF work.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A Yagi on a horizontal boom: a long reflector at the rear, a driven element with a feedpoint, and several shorter directors toward the front, with an arrow showing the main beam direction." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="yagiar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="60" y1="80" x2="400" y2="80" stroke="currentColor" stroke-width="2"/>
  <line x1="80" y1="30" x2="80" y2="130" stroke="currentColor" stroke-width="3"/>
  <text x="60" y="150" font-size="9" fill="currentColor">reflector</text>
  <line x1="150" y1="40" x2="150" y2="120" stroke="currentColor" stroke-width="3"/>
  <circle cx="150" cy="80" r="3" fill="currentColor"/>
  <text x="128" y="150" font-size="9" fill="currentColor">driven (fed)</text>
  <line x1="220" y1="48" x2="220" y2="112" stroke="currentColor" stroke-width="3"/>
  <line x1="280" y1="52" x2="280" y2="108" stroke="currentColor" stroke-width="3"/>
  <line x1="340" y1="55" x2="340" y2="105" stroke="currentColor" stroke-width="3"/>
  <text x="250" y="150" font-size="9" fill="currentColor">directors</text>
  <line x1="405" y1="80" x2="445" y2="80" stroke="currentColor" stroke-width="2" marker-end="url(#yagiar)"/>
  <text x="405" y="72" font-size="9" fill="currentColor">main beam</text>
</svg>
<figcaption>A Yagi's reflector and progressively shorter directors steer the driven element's pattern into a narrow forward beam.</figcaption>
</figure>

## How it works

Only the driven element connects to the feedline; the rest are **parasitic**. When the
driven dipole radiates, it induces currents in the nearby reflector and directors, and
those elements re-radiate. Their length and spacing set the phase of the re-radiated
field. A **reflector** is cut slightly longer than resonance so it looks inductive and
its re-radiation adds in the forward direction while cancelling to the rear. Each
**director** is cut slightly shorter, looking capacitive, and pulls the wavefront
forward. Stack several directors and the reinforcement compounds, narrowing the beam.

The payoffs are directivity and rejection. Forward [gain](/reference/antenna-gain/) rises
with boom length — roughly the number of elements — reaching perhaps 6 dBd for a
three-element Yagi and well over 12 dBd for long designs. The
[radiation pattern](/reference/radiation-pattern/) becomes a narrow forward lobe with a
suppressed rear, quantified by the [front-to-back ratio](/reference/front-to-back-ratio/),
often 15–25 dB. That rejection is as valuable as the gain: it lets a receiver ignore
interference and multipath arriving from other directions.

The trade-off is [bandwidth](/reference/bandwidth/) and complexity. A high-gain Yagi is
tuned for a narrow band; pushing it wideband costs gain. The driven element's impedance
is also pulled well below a dipole's 73 Ω by the close parasitics, so most Yagis use a
folded dipole or a matching device (gamma, hairpin) to reach 50 Ω.

## Relevance to SDR

For an SDR trunking receiver a Yagi is the tool for **weak or distant single sites**.
Pointed at one [trunking site](/reference/trunking-site/), it adds forward gain and,
crucially, uses its [front-to-back ratio](/reference/front-to-back-ratio/) and narrow
beam to reject co-channel signals from other sites and reflections — cleaning up the
constellation before the demodulator ever sees it.

The cost is that a Yagi hears well in only one direction, so it suits fixed monitoring of
a known site, not scanning a whole region. GopherTrunk is a receive-only decoder and does
no beam steering; the directivity is purely mechanical, set by where you aim the boom.
Where omnidirectional coverage matters, a [ground-plane](/reference/ground-plane-antenna/)
or [collinear](/reference/collinear-antenna/) vertical is the better match, and a
[log-periodic](/reference/log-periodic-antenna/) trades some Yagi gain for far wider
bandwidth.

## Sources

[^wiki]: [Yagi–Uda antenna](https://en.wikipedia.org/wiki/Yagi%E2%80%93Uda_antenna) — Wikipedia, for the driven/reflector/director structure, parasitic phasing, and typical gain and front-to-back figures.
