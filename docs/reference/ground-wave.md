---
slug: ground-wave
title: Ground-wave propagation
entry_type: term
category: propagation
description: Ground-wave propagation is the surface-following mode of LF/MF radio waves that hugs the curved earth, carrying AM broadcast and marine signals beyond line of sight.
keywords: ground wave, surface wave, LF propagation, MF propagation, AM broadcast, Norton surface wave, space wave, vertical polarization
aka: [ground wave, surface wave]
autolink: true
infobox:
  - { label: Type, value: LF/MF surface-following mode }
  - { label: Mechanism, value: Diffraction along a lossy earth }
  - { label: Best band, value: LF and MF (below ~3 MHz) }
see_also: [radio-propagation, sky-wave, radio-horizon, broadcast-am, polarization]
cite_urls:
  - https://en.wikipedia.org/wiki/Surface_wave
  - https://en.wikipedia.org/wiki/Ground_wave
---

**Ground-wave propagation** is the mode in which a low- or medium-frequency
[radio wave](/reference/radio-wave/) follows the curved surface of the earth,
diffracting around the horizon instead of travelling in a straight line.[^wiki] It is
the dominant daytime path for [AM broadcast](/reference/broadcast-am/) and marine
signals below a few megahertz, and it is why a distant AM station can be heard in
daylight far past the geometric [radio horizon](/reference/radio-horizon/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A vertical transmitter on a curved earth radiating a wavefront that bends downward and follows the surface to a distant receiver, with signal strength shrinking with distance." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 150 Q230 105 450 150" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <line x1="70" y1="128" x2="70" y2="60" stroke="currentColor" stroke-width="2"/><text x="52" y="53" font-size="9" fill="currentColor">TX</text>
  <path d="M70 70 Q170 66 250 96" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#gwar)"/>
  <path d="M70 82 Q190 80 300 116" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#gwar)"/>
  <path d="M70 94 Q210 96 350 130" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.5" marker-end="url(#gwar)"/>
  <line x1="380" y1="138" x2="380" y2="120" stroke="currentColor" stroke-width="2"/><text x="366" y="114" font-size="9" fill="currentColor">RX</text>
  <text x="150" y="150" font-size="8" fill="currentColor" fill-opacity="0.7">wave follows the surface, weakening with distance</text>
  <defs><marker id="gwar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The vertically polarised wavefront tilts into the lossy ground and diffracts along the earth's curve, reaching receivers well beyond line of sight.</figcaption>
</figure>

## How it works

A vertically polarised wave launched close to the ground induces currents in the
earth. Because soil and seawater are lossy conductors, the lower part of the wavefront
travels slightly slower than the top, tilting the wavefront forward and bending the
energy downward so it clings to the surface. This forward tilt is what lets the wave
diffract around the curvature of the planet rather than shooting off into space.

Three things make the ground wave a low-band phenomenon:

- **Frequency.** Ground loss rises steeply with frequency, so attenuation is modest at
  LF and MF but severe in the HF range and above. By VHF the surface wave dies within a
  few kilometres and [line-of-sight](/reference/radio-horizon/) dominates instead.
- **Polarization.** The mode requires vertical [polarization](/reference/polarization/);
  a horizontally polarised antenna near the ground is shorted out by the earth's image.
- **Ground conductivity.** Seawater is an excellent conductor and extends ground-wave
  range dramatically, while dry, rocky, or sandy soil absorbs the wave quickly. Coastal
  and maritime coverage is therefore far better than inland coverage at the same power.

Ground-wave field strength falls faster than the free-space inverse-distance law
because the lossy earth continually drains energy from the wave, so range is set by a
combination of transmit power, frequency, and the electrical properties of the terrain
rather than by the geometric horizon.

## In practice

Ground wave is the reliable, always-on daytime path for the AM broadcast band
(roughly 0.5–1.7 MHz): the same station that covers a metropolitan area by ground wave
in daylight can be heard hundreds of kilometres away at night when the
[sky-wave](/reference/sky-wave/) path opens as ionospheric absorption fades. Longwave
navigation and time signals, non-directional beacons, and maritime NAVTEX all lean on
ground-wave stability. Because the mode is steady and predictable, it does not fade in
the fluttering way that ionospheric paths do — its main limit is simply distance and
soil loss.

## Relevance to SDR

Ground-wave signals sit below the tuning floor of a basic
[RTL-SDR](/reference/rtl-sdr/), which cannot reach the LF/MF range directly; an
[upconverter](/reference/upconverter/) or a native HF receiver such as the
[Airspy HF+](/reference/airspy-hf-plus/) is needed to hear them. For a trunking scanner
like **GopherTrunk**, ground wave is background context rather than a decode target:
land-mobile trunked systems (P25, DMR, TETRA) live in VHF and UHF bands where
ground-wave range is negligible and coverage is instead governed by
[line-of-sight](/reference/radio-horizon/) and terrain. Understanding the mode still
matters for spectrum literacy — it explains why the low bands behave so differently
from the VHF/UHF bands the scanner actually works in.

## Sources

[^wiki]: [Surface wave](https://en.wikipedia.org/wiki/Surface_wave) — Wikipedia, on ground-wave/surface-wave propagation along a conducting earth and its frequency and polarization dependence.
