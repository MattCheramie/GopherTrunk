---
slug: helical-antenna
title: Helical antenna
entry_type: term
category: antennas
description: A helical antenna is a wire coil over a ground plane; in axial mode it radiates a circularly polarized end-fire beam ideal for satellite links.
keywords: helical antenna, helix antenna, axial mode helix, normal mode helix, circular polarization, end-fire antenna, satellite antenna, quadrifilar helix
aka: [helix antenna, axial-mode helix]
autolink: true
infobox:
  - { label: Type, value: Wire coil over ground plane }
  - { label: Axial mode, value: Circular polarization, end-fire }
  - { label: Uses, value: Satellite up/downlinks }
see_also: [antenna, polarization, antenna-gain, radiation-pattern, beamwidth, patch-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Helical_antenna
  - https://en.wikipedia.org/wiki/Circular_polarization
---

A **helical antenna** is a conductor wound into a helix, usually mounted over a ground
plane and fed at one end.[^wiki] Its most useful form, the **axial mode** helix, radiates
a beam along the axis of the coil that is **circularly [polarized](/reference/polarization/)** —
the electric field rotates once per turn, tracing a corkscrew through space. Circular
polarization is exactly what satellite links need, because a spinning or tumbling
spacecraft, and the Faraday rotation the ionosphere imposes, would fade a linearly
polarized signal in and out; a circularly polarized helix stays connected regardless of
orientation. That, plus its useful [gain](/reference/antenna-gain/) and broad bandwidth,
makes the helix a staple of VHF/UHF satellite ground stations.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A wire wound into a helix rises from a circular ground plane, radiating an end-fire beam along its axis to the right, with a rotating field indicated." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="hlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <ellipse cx="70" cy="140" rx="45" ry="12" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="30" y="168" font-size="9" fill="currentColor">ground plane</text>
  <path d="M70 138 C 100 120 100 110 70 108 C 40 106 40 96 70 92 C 100 90 100 80 70 76 C 40 74 40 64 70 60 C 100 58 100 48 70 44 C 40 42 40 34 70 30" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="105" y="60" font-size="10" fill="currentColor">helix</text>
  <line x1="120" y1="90" x2="430" y2="90" stroke="currentColor" stroke-dasharray="5 4" marker-end="url(#hlar)"/>
  <path d="M200 90 q 15 -14 30 0 q 15 14 30 0" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="300" y="80" font-size="10" fill="currentColor">circularly polarized beam</text>
</svg>
<figcaption>An axial-mode helix launches an end-fire beam along its axis whose field rotates once per turn, giving circular polarization.</figcaption>
</figure>

## How it works

The behaviour of a helix depends on the size of one turn compared to the wavelength.

- **Axial (beam) mode.** When the circumference of a turn is about one wavelength, the
  current travels around each loop and, turn to turn, sets up a wave that reinforces
  **along the axis** — an end-fire beam. Because the current is always a quarter-turn
  ahead of itself as it climbs the helix, the radiated field rotates, producing circular
  polarization whose handedness follows the winding direction. Gain grows with the number
  of turns (a long helix of many turns gives 10–15 dBi and a tightening
  [beamwidth](/reference/beamwidth/)), and the match to the feed line stays good over a
  wide band — roughly a 1.7:1 frequency range — which is unusual for a resonant antenna.
- **Normal (broadside) mode.** When the whole helix is small compared to a wavelength, it
  radiates broadside like a shortened, loaded vertical. This is the "rubber duck" you see
  on handhelds: coiling the wire fits a resonant length into a stubby package, at the
  cost of efficiency and bandwidth.

The axial mode is the one people mean by "helical antenna" as a directional element; the
normal mode is a packaging trick for compact whips.

## Variants

The **quadrifilar helix (QFH)** winds four helical elements together and feeds them in
phase quadrature. It produces a broad, near-hemispherical circularly polarized pattern
rather than a narrow beam, so it does not need to be pointed — the reason it is the
favourite antenna for receiving low-earth-orbit weather satellites (NOAA APT, Meteor)
and other passes that cross the whole sky. Ordinary single-wire axial helices, by
contrast, are pointed at geostationary satellites or used in arrays.

## Relevance to SDR

For SDR satellite work the helix is a go-to build: a few turns of wire over a reflector
disk gives a cheap, high-gain, circularly polarized antenna for uplinks and downlinks in
the 137 MHz, 400 MHz, and higher amateur/weather-satellite bands, and the QFH variant is
the classic homebrew antenna for [NOAA](/reference/noaa-apt/) and Meteor image reception.
The end-fire helix also complements the [patch antenna](/reference/patch-antenna/) as the
other common way to get circular polarization, trading the patch's flat profile for more
gain.

GopherTrunk decodes terrestrial land-mobile trunking (P25, DMR, NXDN, TETRA), which uses
vertically polarized omnidirectional antennas, not helices, so a helical antenna is not
part of a GopherTrunk setup. It is included here as the standard circularly polarized
antenna and a useful contrast to linear radiators.

## Sources

[^wiki]: [Helical antenna](https://en.wikipedia.org/wiki/Helical_antenna) — Wikipedia, for axial versus normal mode, circular polarization, and gain versus turns.
