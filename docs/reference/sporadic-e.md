---
slug: sporadic-e
title: Sporadic E (Es)
entry_type: term
category: propagation
description: Sporadic E is a VHF propagation mode where dense, patchy ionization in the ionosphere's E layer briefly reflects signals over long distances, opening the low VHF bands.
keywords: sporadic E, Es, E-skip, E layer, VHF opening, ionization patch, six meters, band opening, ionospheric propagation
aka: [sporadic E, Es, E-skip]
autolink: true
infobox:
  - { label: Type, value: Intermittent VHF skip mode }
  - { label: Mechanism, value: Dense patches in the E layer }
  - { label: Best band, value: Low VHF (~30–150 MHz) }
see_also: [ionospheric-propagation, sky-wave, radio-propagation, broadcast-fm, meteor-scatter]
cite_urls:
  - https://en.wikipedia.org/wiki/Sporadic_E_propagation
  - https://en.wikipedia.org/wiki/E_layer
---

**Sporadic E** (abbreviated **Es** or "E-skip") is an intermittent propagation mode in
which small, intensely ionised patches in the ionosphere's E layer reflect
[VHF](/reference/frequency-bands/) [radio waves](/reference/radio-wave/) back to earth,
opening long-distance paths on bands that are normally
[line-of-sight](/reference/radio-horizon/) only.[^wiki] Unlike ordinary
[ionospheric propagation](/reference/ionospheric-propagation/), which fades gradually
with the sun, Es appears suddenly, drifts, and vanishes — often within minutes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A transmitter launching a VHF ray up to a small dense ionization patch in the E layer that reflects it down to a distant receiver, while a normal ray passes through where no patch exists." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="135" x2="440" y2="135" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="20" y1="45" x2="440" y2="45" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="6 4"/><text x="20" y="38" font-size="9" fill="currentColor">E layer (~100 km)</text>
  <ellipse cx="230" cy="45" rx="34" ry="8" fill="currentColor" fill-opacity="0.35"/><text x="230" y="30" text-anchor="middle" font-size="8" fill="currentColor">Es patch</text>
  <line x1="55" y1="133" x2="55" y2="116" stroke="currentColor" stroke-width="2"/><text x="55" y="148" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <line x1="405" y1="133" x2="405" y2="116" stroke="currentColor" stroke-width="2"/><text x="405" y="148" text-anchor="middle" font-size="8" fill="currentColor">RX</text>
  <path d="M55 116 L228 50 L405 116" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#sear)"/>
  <path d="M120 118 L300 52" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.4" stroke-dasharray="3 3"/><text x="300" y="52" font-size="7" fill="currentColor" fill-opacity="0.6">no patch → escapes</text>
  <defs><marker id="sear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A transient, sharply ionised cloud at E-layer height reflects VHF signals over long distances; away from the patch the same frequency passes straight through.</figcaption>
</figure>

## How it works

Around 90–120 km up, thin clouds of unusually dense ionisation form and drift through
the E region. When the electron density in a patch is high enough, it can reflect
frequencies far above the normal E-layer limit — routinely into the low VHF range and,
in strong events, past 100 MHz. Because the reflecting patch is small and localised,
the path it supports is narrow and shifts as the cloud moves, so a station that is
booming in one minute can drop out the next.

The physics of the patches is still not fully settled, but the leading explanation is
**wind shear**: layers of the neutral atmosphere moving at different speeds concentrate
long-lived metallic ions (from meteor ablation) into thin, dense sheets. Key features:

- **Seasonality.** Es peaks sharply in late spring and summer, with a smaller winter
  peak, and is far less tied to the solar cycle than F-layer skip.
- **Frequency ceiling.** The higher the patch's density, the higher the frequency it
  reflects; openings climb from low VHF upward as an event intensifies.
- **Short single hops.** A typical Es hop covers roughly 800–2,200 km; occasional
  multi-hop paths reach much farther.

## In practice

Sporadic E is famous among VHF operators as the source of summer "band openings" on the
6-metre amateur band, and it is why [FM broadcast](/reference/broadcast-fm/) listeners
and analog-TV DXers once logged stations a thousand kilometres away for a few minutes at
a time. Its close cousin [meteor scatter](/reference/meteor-scatter/) shares the same
ion supply but works on individual trails rather than settled layers.

## Relevance to SDR

Because Es lands in the low VHF spectrum, it is directly relevant to wideband SDR
monitoring: during an event an [RTL-SDR](/reference/rtl-sdr/) or
[Airspy](/reference/airspy/) can capture distant [FM](/reference/broadcast-fm/),
aircraft-band, or low-VHF land-mobile signals from far outside the normal footprint. For
a trunking scanner such as **GopherTrunk**, Es is an occasional interference source
rather than a target — a distant co-channel transmitter briefly ducted in by an E-cloud
can disrupt decoding of a local system. GopherTrunk itself models none of this; it
decodes whatever reaches the front end, and Es simply explains the rare appearance of
faraway VHF signals on an otherwise line-of-sight band.

## Sources

[^wiki]: [Sporadic E propagation](https://en.wikipedia.org/wiki/Sporadic_E_propagation) — Wikipedia, on transient dense E-layer ionisation, wind-shear formation, seasonality, and VHF reflection.
