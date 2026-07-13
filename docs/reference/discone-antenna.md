---
slug: discone-antenna
title: Discone antenna
entry_type: term
category: antennas
description: A discone is an extremely wideband vertical antenna, a disc over a cone, that gives omnidirectional coverage over a decade of frequency — the classic scanner antenna.
keywords: discone antenna, disc cone antenna, wideband antenna, scanner antenna, omnidirectional vertical, decade bandwidth, biconical
aka: [discone, disc-cone antenna]
autolink: true
infobox:
  - { label: Type, value: Wideband vertical }
  - { label: Bandwidth, value: Up to ~10:1 (a decade) }
  - { label: Pattern, value: Omnidirectional, vertical }
see_also: [biconical-antenna, monopole-antenna, ground-plane-antenna, bandwidth, radiation-pattern]
cite_urls:
  - https://en.wikipedia.org/wiki/Discone_antenna
  - https://en.wikipedia.org/wiki/Biconical_antenna
---

A **discone antenna** is an ultra-wideband vertical [antenna](/reference/antenna/) formed
by a flat **disc** mounted just above the apex of a **cone**, with the feedline running to
the disc and the cone.[^wiki] The shape is a distant relative of the
[biconical antenna](/reference/biconical-antenna/) — replacing one cone with a disc — and
it delivers **omnidirectional, vertically [polarized](/reference/polarization/)** coverage
across a decade or more of frequency from a single feedpoint. That extreme
[bandwidth](/reference/bandwidth/) is why the discone is the archetypal scanner antenna.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A discone antenna: a horizontal disc at the top just above the narrow apex of a downward-flaring cone, fed at the gap between them, radiating an omnidirectional pattern toward the horizon." xmlns="http://www.w3.org/2000/svg">
  <line x1="150" y1="45" x2="310" y2="45" stroke="currentColor" stroke-width="4"/>
  <text x="315" y="49" font-size="10" fill="currentColor">disc</text>
  <circle cx="230" cy="58" r="3" fill="currentColor"/>
  <text x="240" y="62" font-size="8" fill="currentColor">feed gap</text>
  <path d="M230 60 L150 165 L310 165 Z" fill="none" stroke="currentColor" stroke-width="2.5"/>
  <line x1="230" y1="60" x2="180" y2="165" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <line x1="230" y1="60" x2="280" y2="165" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <text x="315" y="120" font-size="10" fill="currentColor">cone</text>
  <ellipse cx="230" cy="70" rx="120" ry="20" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
  <text x="70" y="70" font-size="9" fill="currentColor">omni pattern</text>
</svg>
<figcaption>A discone feeds the gap between a top disc and a flaring cone; the tapered geometry keeps its impedance and pattern usable over a very wide band.</figcaption>
</figure>

## How it works

The discone belongs to the family of **frequency-independent** antennas whose behaviour is
set by angles and tapers rather than a single resonant length. A conventional
[monopole](/reference/monopole-antenna/) is resonant only near one frequency because its
length is a fixed fraction of a [wavelength](/reference/wavelength/). The discone instead
presents a continuously changing radius from the feedpoint outward: at any frequency in
band, some portion of the cone-and-disc structure is the right size to radiate. This keeps
the feedpoint impedance roughly constant — near 50 Ω — across a wide range, so
[SWR](/reference/standing-wave-ratio/) stays low without retuning.

The cone's slant height sets the **low-frequency limit** (roughly a quarter wavelength at
the lowest usable frequency), while the disc diameter and the disc-to-cone spacing set the
**high-frequency** behaviour, giving a usable range of about 10:1. The pattern is
omnidirectional in azimuth and, like a vertical monopole, tilts toward the horizon with a
null overhead; at the upper end of the band the main lobe rises somewhat and gain stays
modest — a discone trades gain for sheer bandwidth, delivering roughly unity gain rather
than the concentrated gain of a beam.

Practical discones use a skeleton of rods instead of solid metal for the disc and cone,
which behaves almost identically while cutting weight and wind load.

## Relevance to SDR

For **general SDR scanning**, a discone is often the single best all-round antenna: one
feedline covers VHF, UHF, and beyond, so a listener can sweep aviation, marine,
public-safety trunking, and business bands without swapping hardware. Its
omnidirectionality means it hears every [trunking site](/reference/trunking-site/) at once,
and its vertical [polarization](/reference/polarization/) matches land-mobile traffic.

GopherTrunk decodes whatever the front end delivers and imposes no antenna requirement,
but a discone's breadth suits GT's multi-protocol, multi-band nature — you can leave it
connected while monitoring different systems across the spectrum. The cost is modest gain:
for a weak, distant single site, a directional [Yagi](/reference/yagi-uda-antenna/) or a
band-optimized [ground-plane](/reference/ground-plane-antenna/) will hear better, but for
"one antenna, everything," the discone is hard to beat.

## Sources

[^wiki]: [Discone antenna](https://en.wikipedia.org/wiki/Discone_antenna) — Wikipedia, for the disc-over-cone geometry, ~10:1 bandwidth, near-constant impedance, and omnidirectional vertical pattern.
