---
slug: patch-antenna
title: Patch (microstrip) antenna
entry_type: term
category: antennas
description: A patch antenna is a flat conductive patch over a ground plane on a dielectric board, a low-profile resonant antenna used in GPS receivers, phones, and Wi-Fi.
keywords: patch antenna, microstrip antenna, printed antenna, planar antenna, GPS antenna, rectangular patch, ceramic patch, low profile antenna, PCB antenna
aka: [microstrip antenna, printed patch antenna, planar patch]
autolink: true
infobox:
  - { label: Type, value: Resonant planar antenna }
  - { label: Profile, value: Low (printed on board) }
  - { label: Pattern, value: Broadside hemisphere }
see_also: [antenna, gps-gnss, polarization, antenna-gain, radiation-pattern, wavelength]
cite_urls:
  - https://en.wikipedia.org/wiki/Patch_antenna
  - https://en.wikipedia.org/wiki/Microstrip_antenna
---

A **patch antenna** (or microstrip antenna) is a flat sheet of conductor mounted a small
fraction of a [wavelength](/reference/wavelength/) above a ground plane, separated by a
thin dielectric board.[^wiki] It is a **resonant** [antenna](/reference/antenna/): the
patch is sized so that its length is about half a wavelength in the dielectric, and the
fringing fields at its two radiating edges launch a beam broadside — straight up, away
from the board. Because the whole structure can be etched onto a circuit board or moulded
as a ceramic block, the patch is cheap, flat, and mechanically robust, which is why it is
the antenna hidden inside GPS units, phones, Wi-Fi cards, and satellite terminals.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A rectangular conducting patch sits above a ground plane separated by a dielectric substrate, with a feed point and a broadside radiation lobe pointing upward." xmlns="http://www.w3.org/2000/svg">
  <rect x="120" y="120" width="220" height="10" fill="currentColor" fill-opacity="0.6"/>
  <text x="345" y="128" font-size="9" fill="currentColor">ground plane</text>
  <rect x="150" y="95" width="160" height="25" fill="none" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="315" y="112" font-size="9" fill="currentColor">substrate</text>
  <rect x="170" y="88" width="120" height="8" fill="currentColor"/>
  <text x="200" y="82" font-size="10" fill="currentColor">patch (~λ/2)</text>
  <circle cx="205" cy="92" r="3" fill="currentColor"/>
  <line x1="205" y1="92" x2="205" y2="130" stroke="currentColor" stroke-width="1.5"/>
  <text x="210" y="150" font-size="9" fill="currentColor">feed</text>
  <path d="M230 88 Q 230 30 230 20" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <ellipse cx="230" cy="48" rx="40" ry="34" fill="none" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="4 3"/>
  <text x="278" y="40" font-size="9" fill="currentColor">broadside lobe</text>
</svg>
<figcaption>A patch resonates at roughly half a wavelength in the substrate; its edge fields radiate a hemispherical beam broadside to the board.</figcaption>
</figure>

## How it works

Think of the patch and the ground plane as the two plates of a very leaky, resonant
cavity. When the patch length is close to a half wavelength (shortened by the substrate's
dielectric constant, so a high-permittivity ceramic makes the patch physically small),
the voltage swings to opposite polarity at the two ends. The fields **fringe** out past
each radiating edge, and because the two edges are half a wavelength apart their
contributions add in the broadside direction. The result is a single main lobe pointing
away from the ground plane, covering roughly the upper hemisphere.

Key consequences of that cavity behaviour:

- **Narrow bandwidth.** A thin, high-Q cavity is only well matched over a small band —
  often a few percent. Thicker or foam substrates widen it at the cost of profile.
- **Low, forgiving gain.** A single patch gives roughly 5–8 dBi with a broad beam,
  ideal when you need coverage of a whole hemisphere (a GPS satellite can be anywhere
  overhead) rather than a pencil beam.
- **Feed sets the match.** Moving the feed point in from the edge finds the spot where
  the patch's impedance matches the 50 Ω line; feeds can be a probe through the ground
  plane, an inset microstrip line, or aperture coupling.

## Variants

Feeding two adjacent edges with a 90° phase offset (or trimming opposite corners) makes
the patch radiate **circular** [polarization](/reference/polarization/) — essential for
GPS, whose satellites transmit right-hand circular so the receiver keeps a steady signal
regardless of orientation. Stacking two patches, or arraying many of them on one board,
widens bandwidth or raises [gain](/reference/antenna-gain/) and narrows the beam, forming
the building block of printed [phased arrays](/reference/phased-array-antenna/).

## Relevance to SDR

The patch is one of the most widely deployed antennas on earth precisely because it
disappears into a device. In SDR terms it is the natural front end for anything at
L-band and up where you want a flat, cheap, hemispheric antenna:
[GPS/GNSS](/reference/gps-gnss/) reception (an SDR fed by a ceramic active patch is the
standard way to sample the L1 band), [Inmarsat](/reference/inmarsat/) and other L-band
satellite downlinks, ADS-B ground stations, and Wi-Fi. Its circular-polarization option
is what makes it the default for satellite work.

GopherTrunk targets VHF/UHF land-mobile trunking (P25, DMR, NXDN, TETRA), where
wavelengths are long and a resonant patch would be large and narrowband, so scanners use
verticals rather than patches. The patch is documented here as the canonical low-profile
printed antenna and the reason a fingernail-sized ceramic block can pull a GPS satellite
out of the noise.

## Sources

[^wiki]: [Patch antenna](https://en.wikipedia.org/wiki/Patch_antenna) — Wikipedia, for the microstrip cavity model, resonant sizing, and circular-polarization feeds.
