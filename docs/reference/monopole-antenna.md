---
slug: monopole-antenna
title: Monopole antenna
entry_type: term
category: antennas
description: A monopole is a single quarter-wave conductor worked against a ground plane, which mirrors it into an effective half-wave dipole with an omnidirectional pattern.
keywords: monopole antenna, quarter-wave monopole, quarter-wave whip, ground plane, image theory, vertical antenna, marconi antenna
aka: [monopole, quarter-wave monopole, marconi antenna]
autolink: true
infobox:
  - { label: Type, value: Resonant vertical }
  - { label: Length, value: ~λ/4 over a ground plane }
  - { label: Pattern, value: Omnidirectional in azimuth }
see_also: [ground-plane-antenna, dipole-antenna, whip-antenna, radials-counterpoise, polarization]
cite_urls:
  - https://en.wikipedia.org/wiki/Monopole_antenna
  - https://en.wikipedia.org/wiki/Ground_plane
---

A **monopole antenna** is a single straight conductor, classically a **quarter
[wavelength](/reference/wavelength/)** long, driven at its base against a conducting
**[ground plane](/reference/ground-plane-antenna/)**.[^wiki] The ground plane reflects
the element to form the electrical equivalent of a half-wave
[dipole](/reference/dipole-antenna/), so a physically short rod behaves like a much
larger antenna. Because it is vertical and omnidirectional in azimuth, the monopole is
the workhorse form for mobile whips, base verticals, and countless scanner antennas.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A vertical quarter-wave rod above a horizontal ground plane, with its mirror image shown dashed below the plane completing a half-wave dipole." xmlns="http://www.w3.org/2000/svg">
  <line x1="150" y1="120" x2="150" y2="40" stroke="currentColor" stroke-width="3"/>
  <circle cx="150" cy="120" r="3" fill="currentColor"/>
  <line x1="70" y1="122" x2="230" y2="122" stroke="currentColor" stroke-width="1.5"/>
  <text x="235" y="126" font-size="9" fill="currentColor">ground plane</text>
  <line x1="150" y1="124" x2="150" y2="204" stroke="currentColor" stroke-width="2" stroke-dasharray="4 3" stroke-opacity="0.55"/>
  <text x="158" y="80" font-size="10" fill="currentColor">λ/4</text>
  <text x="158" y="170" font-size="9" fill="currentColor" opacity="0.6">image (λ/4)</text>
  <ellipse cx="150" cy="82" rx="95" ry="24" fill="none" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="3 3"/>
  <text x="250" y="82" font-size="9" fill="currentColor">pattern (above plane)</text>
  <line x1="360" y1="40" x2="360" y2="200" stroke="currentColor" stroke-width="1"/>
  <text x="330" y="30" font-size="9" fill="currentColor">≈ half-wave dipole</text>
</svg>
<figcaption>A quarter-wave monopole plus its mirror image in the ground plane forms an effective half-wave dipole; only the upper half physically exists.</figcaption>
</figure>

## How it works

A quarter-wave rod on its own is not resonant, but placed over a large conducting
sheet it is completed by its **electrical image**. Image theory says that a vertical
current above a perfect conductor is indistinguishable, in the region above the plane,
from that current plus a mirror-image current below it. The rod and its image together
look like a centre-fed half-wave dipole, so the monopole resonates at a quarter
wavelength — half the physical size of a dipole for the same band.

Two consequences follow from radiating over only the upper half-space. First, all the
transmitted or received power is concentrated above the plane, so a monopole has about
**3 dB more [gain](/reference/antenna-gain/)** than the equivalent dipole (5.15 dBi
versus 2.15 dBi for the ideal case). Second, the feedpoint impedance is halved: an
ideal quarter-wave monopole presents roughly **37 Ω** at resonance, close enough to
50 Ω coax to give a usable match without a transformer. The
[radiation pattern](/reference/radiation-pattern/) is omnidirectional in the horizontal
plane and vertically [polarized](/reference/polarization/), with a null straight up and
peak response toward the horizon — ideal for hearing distant land-mobile sites.

The catch is that the "ground plane" must actually conduct. A real ground plane is
never perfect, so its size, conductivity, and the return path it offers all shift the
resonance, the pattern, and the take-off angle. On a vehicle the metal roof serves as
the plane; on a base vertical a set of [radials](/reference/radials-counterpoise/) does
the same job electrically.

## Relevance to SDR

The monopole and its variants dominate practical SDR scanning. A quarter-wave whip cut
for the target [band](/reference/frequency-bands/) — a length in centimetres of roughly
7125 divided by the frequency in MHz — is the cheapest antenna that actually works for
VHF/UHF trunking. A [ground-plane antenna](/reference/ground-plane-antenna/) is simply
a monopole given its own artificial ground, and a mobile
[whip](/reference/whip-antenna/) is a monopole using the vehicle body as the plane.

GopherTrunk is a receive-only decoder and imposes no antenna requirement, but because
land-mobile P25, DMR, and NXDN signals are vertically polarized, a vertical monopole
matches their polarization and rejects horizontally polarized interference. Getting the
element high and clear of obstructions, and giving it an adequate ground plane, improves
signal-to-noise more than anything downstream in the radio.

## In practice

- **Size the ground plane.** A plane at least a quarter wavelength in radius approximates
  the ideal; smaller planes raise the take-off angle and shift resonance upward.
- **Elevated feed changes impedance.** Sloping the radials of a
  [ground-plane antenna](/reference/ground-plane-antenna/) downward raises the ~37 Ω
  feedpoint toward 50 Ω for a better [SWR](/reference/standing-wave-ratio/) match.
- **Watch common-mode current.** Without a good ground return, RF flows on the outside
  of the coax and distorts the pattern; a choke or proper radial system prevents it.

## Sources

[^wiki]: [Monopole antenna](https://en.wikipedia.org/wiki/Monopole_antenna) — Wikipedia, for the quarter-wave monopole, image theory, and its ~37 Ω feedpoint and 3 dB gain over a dipole.
