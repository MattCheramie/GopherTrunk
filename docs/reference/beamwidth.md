---
slug: beamwidth
title: Beamwidth
entry_type: term
category: antennas
description: Beamwidth is the angular width of an antenna's main lobe, most often the half-power (−3 dB) beamwidth, and it trades off inversely against antenna gain.
keywords: beamwidth, half-power beamwidth, HPBW, 3 dB beamwidth, main lobe, first-null beamwidth, directivity, gain, azimuth, elevation
aka: [beamwidth, half-power beamwidth, HPBW]
autolink: true
infobox:
  - { label: Type, value: Antenna directional metric }
  - { label: Usual measure, value: −3 dB (half-power) angle }
  - { label: Relation, value: Narrower beam = higher gain }
see_also: [radiation-pattern, antenna-gain, front-to-back-ratio, yagi-uda-antenna, parabolic-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Beamwidth
---

**Beamwidth** is the angular width of an antenna's main lobe — how wide, in degrees, the
"beam" of strongest radiation is.[^wiki] The standard measure is the **half-power beamwidth
(HPBW)**: the angle between the two directions on either side of the peak where the radiated
power has fallen to half (−3 dB) of its maximum. Beamwidth is read directly off the
[radiation pattern](/reference/radiation-pattern/) and is inversely related to
[antenna gain](/reference/antenna-gain/) — a narrower beam concentrates energy into a smaller
solid angle, so it delivers more gain.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A main lobe emanating from an antenna with two dashed lines marking the half-power points on each side, and the angle between them labelled as the half-power beamwidth." xmlns="http://www.w3.org/2000/svg">
  <path d="M60 90 C 140 30, 380 45, 420 90 C 380 135, 140 150, 60 90 Z" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <circle cx="60" cy="90" r="3" fill="currentColor"/>
  <line x1="60" y1="90" x2="410" y2="90" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.6"/>
  <line x1="60" y1="90" x2="400" y2="52" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <line x1="60" y1="90" x2="400" y2="128" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <path d="M150 74 A 90 90 0 0 1 150 106" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="175" y="94" font-size="10" fill="currentColor">HPBW</text>
  <text x="360" y="48" font-size="8.5" fill="currentColor">−3 dB point</text>
  <text x="360" y="140" font-size="8.5" fill="currentColor">−3 dB point</text>
  <text x="330" y="86" font-size="8.5" fill="currentColor">peak</text>
</svg>
<figcaption>The half-power beamwidth is the angle between the two −3 dB points that straddle the main-lobe peak.</figcaption>
</figure>

## How it works

Radiation from an antenna is strongest along the main-lobe axis and falls away to either side.
If you sweep a receiver around the antenna and note the two angles where the received power drops
to one-half (3 dB below peak), the angle between them is the HPBW. Antennas have two beamwidths
that generally differ — one in the **azimuth** (horizontal) plane and one in the **elevation**
(vertical) plane — because the main lobe is rarely circular in cross-section. A parabolic dish,
for instance, might have a 4° azimuth beamwidth and a 6° elevation beamwidth.

Two other measures appear in datasheets. The **first-null beamwidth (FNBW)** is the wider angle
between the nulls that bracket the main lobe; it is always larger than the HPBW. Occasionally a
−10 dB beamwidth is quoted for feed antennas illuminating a reflector.

## In practice

Beamwidth and gain are two views of the same thing: focusing. A rough and widely used estimate
ties the product of the two principal-plane HPBWs to directivity,

> *G* (dBi) ≈ 10 · log₁₀( 41 000 / (θ_az · θ_el) ),

with the beamwidths in degrees. A pencil-beam dish with 2° in each plane predicts roughly 40 dBi;
a 60°-by-60° panel predicts about 10 dBi. The constant (near 41 000) folds in typical aperture
efficiency and side-lobe losses, so it is an approximation, not an identity — but it captures the
essential trade-off: **halving the beamwidth in both planes adds about 6 dB of gain.** The cost
is aiming: a 2° beam must be pointed within a fraction of a degree, while a 60° beam is
forgiving.

## Relevance to SDR

Beamwidth decides how carefully you must aim a directional scanner antenna. A modest
[Yagi](/reference/yagi-uda-antenna/) with a 40°–50° beamwidth adds useful gain toward a distant
[trunking site](/reference/trunking-site/) while still being easy to point by hand. A narrow
[parabolic](/reference/parabolic-antenna/) or long-boom array, with a beamwidth of only a few
degrees, gives much more gain and rejection of off-axis interference but demands a rotator and
accurate bearings. For omnidirectional scanning you deliberately want a *wide* azimuth beamwidth
(effectively 360°) so no bearing is missed, accepting the lower gain that implies. **GopherTrunk**
does not measure or use beamwidth — it is a property of the antenna hardware — but the beamwidth
you choose sets the signal-to-noise ratio the decoder ultimately sees.

## Sources

[^wiki]: [Beamwidth](https://en.wikipedia.org/wiki/Beamwidth) — Wikipedia, for the half-power and first-null beamwidth definitions and the gain relationship.
