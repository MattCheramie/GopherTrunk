---
slug: radiation-pattern
title: Radiation pattern
entry_type: term
category: antennas
description: A radiation pattern is the graphical map of how an antenna radiates or receives power as a function of direction, showing the main lobe, side lobes, nulls, and its E- and H-plane cuts.
keywords: radiation pattern, antenna pattern, main lobe, side lobes, back lobe, nulls, E-plane, H-plane, azimuth, elevation, polar plot, front-to-back
aka: [radiation pattern, antenna pattern, far-field pattern]
autolink: true
infobox:
  - { label: Type, value: Antenna directional characteristic }
  - { label: Shows, value: Main lobe, side lobes, nulls }
  - { label: Cuts, value: E-plane and H-plane }
see_also: [beamwidth, antenna-gain, front-to-back-ratio, antenna, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Radiation_pattern
---

**A radiation pattern** is a map of how an [antenna](/reference/antenna/) sends or receives
power in each direction, usually drawn as a polar plot of relative field strength versus
angle.[^wiki] Because a passive antenna is reciprocal, the same pattern describes both its
transmit and its receive behaviour — a lobe that radiates strongly toward a bearing also hears
strongly from that bearing. The pattern is the single most informative picture of what an
antenna does, and features like [gain](/reference/antenna-gain/) and
[beamwidth](/reference/beamwidth/) are read directly off it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A polar radiation pattern with a large forward main lobe, two smaller side lobes, a small back lobe, and nulls between them, plotted around a central antenna point." xmlns="http://www.w3.org/2000/svg">
  <circle cx="180" cy="95" r="80" fill="none" stroke="currentColor" stroke-width="0.6" stroke-opacity="0.4"/>
  <circle cx="180" cy="95" r="50" fill="none" stroke="currentColor" stroke-width="0.6" stroke-opacity="0.4"/>
  <circle cx="180" cy="95" r="20" fill="none" stroke="currentColor" stroke-width="0.6" stroke-opacity="0.4"/>
  <path d="M180 95 C 250 20, 400 55, 420 95 C 400 135, 250 170, 180 95 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/>
  <path d="M180 95 C 150 60, 100 62, 95 80 C 100 92, 150 88, 180 95 Z" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/>
  <path d="M180 95 C 150 130, 100 128, 95 110 C 100 98, 150 102, 180 95 Z" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/>
  <circle cx="180" cy="95" r="3" fill="currentColor"/>
  <text x="360" y="90" font-size="9" fill="currentColor">main lobe</text>
  <text x="100" y="55" font-size="9" fill="currentColor">side lobe</text>
  <text x="100" y="150" font-size="9" fill="currentColor">side lobe</text>
</svg>
<figcaption>A directional pattern: a dominant main lobe, weaker side lobes, and nulls where reception falls away — the shape a Yagi or panel antenna produces.</figcaption>
</figure>

## How it works

An antenna's far-field pattern is set by how the currents on its structure add up in phase at
different angles. In directions where the contributions add constructively, the field is strong;
where they cancel, a **null** appears. The result is a set of **lobes**:

- The **main lobe** points where the antenna radiates most strongly. Its width — the
  [beamwidth](/reference/beamwidth/) — is measured between the half-power (−3 dB) points.
- **Side lobes** are the smaller lobes flanking the main lobe. They represent power leaking in
  unwanted directions; a good design keeps them well below the main lobe (often −20 dB or lower).
- The **back lobe** points opposite the main lobe; the main-to-back ratio is the
  [front-to-back ratio](/reference/front-to-back-ratio/).
- **Nulls** are the deep minima between lobes — useful for rejecting an interferer by pointing a
  null at it.

Because a pattern is three-dimensional, it is usually shown as two orthogonal 2-D slices. The
**E-plane** cut is taken in the plane containing the electric-field vector; the **H-plane** cut
is taken in the plane containing the magnetic field, perpendicular to it. For a vertical dipole,
the E-plane is the vertical (elevation) cut — a figure-eight — and the H-plane is the horizontal
(azimuth) cut — a circle. Together the two cuts specify the antenna's directivity to good
approximation.

## In practice

Patterns are quoted for the **far field**, where the shape no longer depends on distance. They
are plotted either in linear field strength or, more commonly, in decibels normalized so the
main lobe peak sits at 0 dB. An **isotropic** radiator (a theoretical point that radiates
equally in all directions) is a perfect sphere and the reference for **dBi** gain; an
**omnidirectional** antenna such as a vertical whip is a doughnut — uniform in azimuth but
shaped in elevation. Real patterns are distorted by nearby metal, the ground, and the mast, so a
modelled pattern and a measured one seldom match exactly.

## Relevance to SDR

The radiation pattern is what you are really choosing when you pick a scanner antenna. A discone
or vertical gives a broad, near-omnidirectional azimuth pattern so it hears
[trunking sites](/reference/trunking-site/) from any bearing — ideal when you do not know where
the transmitter is. A [Yagi](/reference/yagi-uda-antenna/) or log-periodic concentrates its
pattern into a narrow main lobe, adding gain and letting you reject co-channel interference by
aiming a null at it — useful for pulling in one distant [control channel](/reference/control-channel/)
or for direction finding. **GopherTrunk** processes whatever samples reach the SDR and has no
knowledge of the antenna pattern itself, but the pattern determines the signal-to-noise ratio at
the receiver input, which sets whether the decoder locks at all.

## Sources

[^wiki]: [Radiation pattern](https://en.wikipedia.org/wiki/Radiation_pattern) — Wikipedia, for lobe terminology and the E-plane/H-plane definitions.
