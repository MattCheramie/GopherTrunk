---
slug: horn-antenna
title: Horn antenna
entry_type: term
category: antennas
description: A horn antenna is a flared section of waveguide that radiates a directional microwave beam, widely used as a dish feed and as a standard-gain calibration reference.
keywords: horn antenna, waveguide horn, pyramidal horn, conical horn, sectoral horn, standard gain horn, feed horn, microwave antenna, aperture antenna
aka: [waveguide horn, feed horn, pyramidal horn, standard-gain horn]
autolink: true
infobox:
  - { label: Type, value: Aperture (waveguide) antenna }
  - { label: Gain, value: Moderate (10–25 dBi) }
  - { label: Roles, value: Dish feed / gain reference }
see_also: [antenna, antenna-gain, effective-aperture, parabolic-antenna, beamwidth, impedance]
cite_urls:
  - https://en.wikipedia.org/wiki/Horn_antenna
  - https://en.wikipedia.org/wiki/Waveguide
---

A **horn antenna** is a flared, open-ended section of waveguide
that guides a microwave signal out into free space as a directional beam.[^wiki] The
gradual flare acts as a matching transition between the confined field inside the
metal-walled guide and the [impedance](/reference/impedance/) of free space, so energy
radiates cleanly with very little reflected back. Horns are simple, rugged, and
broadband, which is why they serve two everyday jobs: illuminating the reflector of a
[parabolic antenna](/reference/parabolic-antenna/) as its feed, and acting as a
calibrated **standard-gain** reference for measuring other antennas.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A rectangular waveguide flares open into a horn, and the wavefront expands and radiates from the mouth as a directional beam." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="hoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="30" y="70" width="90" height="30" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="40" y="120" font-size="10" fill="currentColor">waveguide</text>
  <path d="M120 70 L 240 30 L 240 140 L 120 100 Z" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="150" y="30" font-size="10" fill="currentColor">flare</text>
  <path d="M255 42 Q 275 85 255 128" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <path d="M275 50 Q 300 85 275 120" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="245" y1="85" x2="430" y2="85" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#hoar)"/>
  <text x="350" y="78" font-size="10" fill="currentColor">beam</text>
</svg>
<figcaption>The flare transforms the guided wave into an expanding wavefront that leaves the horn mouth as a directional beam.</figcaption>
</figure>

## How it works

Inside a rectangular or circular waveguide, energy travels as a confined mode with a
characteristic impedance set by the guide's dimensions. If the guide simply stopped at an
open end, the large impedance mismatch between the guide and free space (~377 Ω) would
reflect much of the power back down the line. The horn's flare fixes this: by opening the
walls **gradually** over several wavelengths, it presents a smooth impedance taper so the
wave "lets go" of the walls and radiates with low reflection.

Like the dish, a horn is an **aperture antenna** — its gain is governed by the area of
its mouth, not by a resonant length. A larger mouth means a larger [effective
aperture](/reference/effective-aperture/), higher [gain](/reference/antenna-gain/), and a
narrower [beamwidth](/reference/beamwidth/). There is a design tension, though: the flare
also introduces a **phase error** across the aperture because the wavefront at the mouth
is spherical, not flat. Making the horn longer for a given mouth size reduces this error
but adds bulk. A well-proportioned "optimum" horn balances the two for maximum gain at a
manageable length, typically yielding 10–25 dBi over a broad band.

## Variants

- **Pyramidal horn:** flared in both the E and H planes from a rectangular guide. The
  most common general-purpose and standard-gain type.
- **Sectoral horn:** flared in only one plane, giving a fan-shaped beam — wide in one
  axis, narrow in the other.
- **Conical horn:** flared from a circular guide; pairs naturally with circular
  waveguide feeds and supports circular [polarization](/reference/polarization/).
- **Corrugated horn:** grooves cut into the inner walls equalise the E- and H-plane
  patterns, producing a symmetric, low-spillover beam. Prized as a reflector feed because
  even illumination raises the dish's aperture efficiency.

## In practice

As a **dish feed**, the horn's pattern is chosen to illuminate the reflector's rim at
roughly −10 dB, trading a little spillover against under-illuminating the aperture. As a
**standard-gain horn**, a precisely machined pyramidal horn has a gain that can be
computed from its dimensions to a fraction of a dB, so antenna and EMC labs use it as a
transfer standard: measure the horn, then measure the antenna under test against it.
Horns are also the feeds inside satcom terminals and the radiating elements of some
radar and radiometer systems.

## Relevance to SDR

For the SDR listener, horns show up mainly as the feed at the focus of a satellite or
microwave dish, and in test-bench work as gain references and probes. They are not used
for the VHF/UHF land-mobile bands that dominate scanning, because at those frequencies a
horn large enough to be useful would be impractically huge — a wavelength at 150 MHz is
two metres. GopherTrunk decodes trunked land-mobile traffic received on modest vertical
antennas, so it neither needs nor models horn antennas; the horn is covered here as the
fundamental waveguide aperture antenna and the practical companion to the
[parabolic reflector](/reference/parabolic-antenna/).

## Sources

[^wiki]: [Horn antenna](https://en.wikipedia.org/wiki/Horn_antenna) — Wikipedia, for horn geometry, the flare-versus-phase-error trade-off, and standard-gain use.
