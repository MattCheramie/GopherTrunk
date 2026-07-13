---
slug: phased-array-antenna
title: Phased-array antenna
entry_type: term
category: antennas
description: A phased-array antenna steers its beam electronically by adjusting the relative phase of many elements, with no moving parts, used in radar, 5G, and satcom.
keywords: phased array antenna, electronic beam steering, phase shifter, AESA, active electronically scanned array, beam steering, antenna array, grating lobes, 5G beamforming
aka: [phased array, electronically scanned array, AESA]
autolink: true
infobox:
  - { label: Type, value: Steered antenna array }
  - { label: Steering, value: Electronic (phase, no motion) }
  - { label: Uses, value: Radar, 5G, satcom }
see_also: [antenna, beamforming, radiation-pattern, antenna-gain, beamwidth, patch-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Phased_array
  - https://en.wikipedia.org/wiki/Beam_steering
---

A **phased-array antenna** is an array of many small radiating elements whose beam is
aimed **electronically** — by adjusting the relative phase (and often amplitude) fed to
each element — rather than by physically turning the antenna.[^wiki] By delaying some
elements relative to others, the array makes the wavefronts add up constructively in one
chosen direction and cancel elsewhere, so the main lobe can be swept across the sky in
microseconds with no moving parts. This ability to steer, and to form multiple beams at
once, is what underlies modern radar, 5G base stations, and flat satellite terminals; the
underlying signal-processing operation is [beamforming](/reference/beamforming/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A row of antenna elements each fed through a phase shifter with increasing delay produces a tilted combined wavefront, steering the beam off to one side." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="phar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="2" fill="none">
    <line x1="60" y1="150" x2="60" y2="130"/>
    <line x1="110" y1="150" x2="110" y2="130"/>
    <line x1="160" y1="150" x2="160" y2="130"/>
    <line x1="210" y1="150" x2="210" y2="130"/>
    <line x1="260" y1="150" x2="260" y2="130"/>
  </g>
  <g fill="currentColor" font-size="8">
    <text x="50" y="168">0°</text><text x="98" y="168">Δφ</text><text x="146" y="168">2Δφ</text><text x="196" y="168">3Δφ</text><text x="246" y="168">4Δφ</text>
  </g>
  <text x="30" y="182" font-size="9" fill="currentColor">progressive phase per element</text>
  <line x1="60" y1="120" x2="320" y2="60" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="120" y1="128" x2="360" y2="72" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="180" y1="95" x2="330" y2="30" stroke="currentColor" stroke-dasharray="5 4" marker-end="url(#phar)"/>
  <text x="335" y="26" font-size="10" fill="currentColor">steered beam</text>
</svg>
<figcaption>Feeding each element a progressively larger phase tilts the combined wavefront, aiming the beam off-axis without moving the antenna.</figcaption>
</figure>

## How it works

Consider a straight row of identical elements spaced a fixed distance apart. If every
element radiates in phase, the wavefronts stack up parallel to the array and the beam
points straight ahead (broadside). Now add a **progressive phase shift** — element 2 lags
element 1 by Δφ, element 3 by 2Δφ, and so on. The individual wavefronts now line up along
a *tilted* plane, and the beam points off to the side by an angle that depends directly on
Δφ. Change Δφ and the beam moves; because Δφ is just a number in a phase shifter or a
digital multiply, steering is effectively instantaneous.

Several array facts follow from this:

- **More elements, narrower beam and higher gain.** A longer array behaves like a larger
  aperture, so [gain](/reference/antenna-gain/) rises and [beamwidth](/reference/beamwidth/)
  shrinks with element count — the same aperture logic as a dish, but reconfigurable.
- **Scan loss.** As the beam steers away from broadside, the array's projected aperture
  shrinks, so gain falls and the beam broadens toward the edges of coverage.
- **Grating lobes.** If the elements are spaced much more than half a wavelength apart, the
  array forms unwanted extra beams (grating lobes) in other directions. Keeping spacing
  around λ/2 avoids them.
- **Nulls too.** The same phase control that builds the main lobe can place deep **nulls**
  on interferers — adaptive arrays exploit this to reject jammers.

## Variants

- **Passive (PESA):** one transmitter/receiver drives all elements through phase shifters.
- **Active (AESA):** every element has its own tiny transmit/receive module. AESAs are
  more capable and reliable (a few failed modules degrade gracefully) and dominate modern
  radar.
- **Digital / hybrid beamforming:** each element or subarray is digitised separately, and
  the beam is formed in software, allowing many simultaneous independent beams — the
  architecture behind massive-MIMO 5G.

The elements themselves are often [patch antennas](/reference/patch-antenna/) on a board,
which is how a flat panel can hide a steerable array.

## Relevance to SDR

Phased arrays are where antennas and DSP merge, and the SDR world touches them directly
through [beamforming](/reference/beamforming/): with several coherent receivers sampling
several antennas, software can steer, null, and estimate directions of arrival exactly as a
hardware array would, but entirely in the digital domain. Coherent multi-channel SDRs make
small experimental receive arrays practical for direction finding and passive radar. On the
infrastructure side, phased arrays are the enabling antenna for 5G NR beam steering, modern
radar, and electronically steered flat-panel satellite terminals.

GopherTrunk is a single-channel land-mobile trunking **receiver** with no array hardware and
no beamforming, so it does not implement phased-array steering; the systems it decodes (P25,
DMR, NXDN, TETRA) are received on ordinary omnidirectional antennas. This page is included
to relate the array world to the broader RF landscape and to explain how electronic beam
steering works.

## Sources

[^wiki]: [Phased array](https://en.wikipedia.org/wiki/Phased_array) — Wikipedia, for progressive phase steering, grating lobes, scan loss, and PESA/AESA architectures.
