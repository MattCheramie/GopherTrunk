---
slug: near-field-far-field
title: Near field & far field
entry_type: term
category: antennas
description: "The near field and far field are the two regions around an antenna: a reactive near zone where fields are complex, and the far zone beyond the Fraunhofer distance where the radiation pattern is fixed."
keywords: near field, far field, Fresnel region, Fraunhofer distance, reactive near field, radiating near field, far zone, antenna regions, plane wave, 2D squared over lambda
aka: [near field, far field, Fraunhofer region, Fresnel region]
autolink: true
infobox:
  - { label: Type, value: Antenna field regions }
  - { label: Boundary, value: Fraunhofer distance 2D²/λ }
  - { label: Far field, value: Pattern fixed, wave locally planar }
see_also: [field-strength, radiation-pattern, wavelength, friis-transmission-equation, antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Near_and_far_field
---

**The near field and far field** are the two regions of space around a transmitting
[antenna](/reference/antenna/), distinguished by how the electromagnetic field behaves with
distance.[^wiki] Close to the antenna, in the **near field**, energy is partly stored and sloshes
back and forth between the antenna and the surrounding space, and the field structure is complex
and distance-dependent. Far away, in the **far field**, the wave has settled into a clean,
outward-travelling spherical wave whose [radiation pattern](/reference/radiation-pattern/) no
longer changes shape with distance — only its amplitude falls off. Almost all communication and
scanning happens in the far field.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Concentric zones around an antenna: a small reactive near-field region, a radiating near-field region, and beyond the Fraunhofer distance the far field where wavefronts become nearly planar." xmlns="http://www.w3.org/2000/svg">
  <circle cx="60" cy="75" r="3" fill="currentColor"/>
  <circle cx="60" cy="75" r="26" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/>
  <circle cx="60" cy="75" r="70" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <text x="60" y="120" text-anchor="middle" font-size="8" fill="currentColor">reactive</text>
  <text x="150" y="120" text-anchor="middle" font-size="8" fill="currentColor">radiating near field</text>
  <line x1="200" y1="20" x2="200" y2="130" stroke="currentColor" stroke-width="1.1"/>
  <text x="200" y="14" text-anchor="middle" font-size="8" fill="currentColor">2D²/λ</text>
  <path d="M240 30 Q 255 75 240 120" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M290 35 Q 300 75 290 115" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M345 40 L 345 110" stroke="currentColor" stroke-width="1.2"/>
  <path d="M400 42 L 400 108" stroke="currentColor" stroke-width="1.2"/>
  <text x="330" y="132" text-anchor="middle" font-size="8" fill="currentColor">far field (planar)</text>
</svg>
<figcaption>Field regions from the antenna outward: reactive near field, radiating near field, and — past the Fraunhofer distance — the far field, where wavefronts are essentially planar.</figcaption>
</figure>

## How it works

Every antenna surrounds itself with two kinds of field. **Reactive** (stored) fields dominate very
close in; they represent energy trapped in the antenna's immediate vicinity, falling off steeply
(as 1/*r*² and 1/*r*³) and carrying no net power away. **Radiating** fields fall off more gently
(as 1/*r*) and carry power to infinity. The two coexist near the antenna and compete; only far
away does the radiating field win decisively. This gives three nested regions:

- **Reactive near field** — the innermost zone, extending to roughly 0.62·√(*D*³/λ) for an
  antenna of largest dimension *D*. Stored energy dominates; probing here disturbs the antenna.
- **Radiating near field (Fresnel region)** — beyond that, out to the Fraunhofer distance. Power
  radiates, but the pattern is still evolving: the relative phase of contributions from different
  parts of the antenna changes with distance, so the "beam" is not yet fully formed.
- **Far field (Fraunhofer region)** — beyond the **Fraunhofer distance**, conventionally

  > *d*_F = 2*D*² / λ.

  Here the wavefront is locally planar, the electric and magnetic fields are in phase and
  perpendicular, their ratio is the impedance of free space (377 Ω), and the pattern is fixed.

The 2*D*²/λ boundary comes from requiring that path-length differences across the aperture stay
within λ/16, so the wave looks flat. For a small antenna (*D* comparable to λ) the far field
begins within a wavelength or two; for a large dish it can be hundreds of metres away.

## In practice

The far-field assumption underpins nearly every everyday RF formula. A quoted antenna
[gain](/reference/antenna-gain/) or beamwidth is a far-field property; the
[Friis transmission equation](/reference/friis-transmission-equation/) and
[field-strength](/reference/field-strength/) calculations assume a plane wave and hold only beyond
*d*_F. Antenna ranges must place the test source at least a Fraunhofer distance away (or use a
compact/near-field range that measures the near field and mathematically transforms it to the far
field). Near-field effects also explain **coupling**: two antennas placed within each other's near
field interact strongly, which matters for co-sited scanner antennas and for stacked arrays.

## Relevance to SDR

For virtually all scanning and trunking work the transmitter is many wavelengths away, so the
receiver sits comfortably in the far field and the tidy plane-wave, fixed-pattern picture applies
— which is why a published antenna gain and a simple path-loss estimate are meaningful. The near
field matters in two practical situations: when antennas are mounted close together (a scanner
antenna beside a transmitting antenna can couple strongly, causing desense or damage), and when a
handheld or nearby object is within a wavelength of the antenna and detunes it. **GopherTrunk**
operates purely on far-field signals as delivered by the SDR and has no near-field awareness; the
concept matters at the antenna-installation stage, not in the decode chain.

## Sources

[^wiki]: [Near and far field](https://en.wikipedia.org/wiki/Near_and_far_field) — Wikipedia, for the region definitions and the Fraunhofer distance 2D²/λ.
