---
slug: ground-plane-antenna
title: Ground-plane antenna
entry_type: term
category: antennas
description: A ground-plane antenna is a quarter-wave monopole given an artificial ground of radials, forming a self-contained omnidirectional vertical for VHF/UHF work.
keywords: ground plane antenna, ground-plane, quarter-wave vertical, radials, monopole, counterpoise, GPA, omnidirectional vertical
aka: [ground plane antenna, ground-plane vertical, GPA]
autolink: true
infobox:
  - { label: Type, value: Monopole with artificial ground }
  - { label: Element, value: λ/4 vertical + 3–4 radials }
  - { label: Pattern, value: Omnidirectional, vertically polarized }
see_also: [monopole-antenna, radials-counterpoise, whip-antenna, dipole-antenna, antenna-gain]
cite_urls:
  - https://en.wikipedia.org/wiki/Ground_plane
  - https://en.wikipedia.org/wiki/Monopole_antenna
---

A **ground-plane antenna** is a [monopole](/reference/monopole-antenna/) — a
quarter-[wavelength](/reference/wavelength/) vertical element — mounted over an
**artificial ground** made of a few horizontal or sloping
[radials](/reference/radials-counterpoise/) instead of a solid metal sheet.[^wiki] The
radials supply the electrical image the monopole needs, so the whole assembly is a
self-contained, omnidirectional vertical that does not depend on being near real earth
or a vehicle body. It is one of the most common fixed-station antennas for scanning and
amateur VHF/UHF.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A vertical quarter-wave element at a central feedpoint with four radial wires sloping downward from the base, forming an artificial ground plane." xmlns="http://www.w3.org/2000/svg">
  <line x1="230" y1="110" x2="230" y2="30" stroke="currentColor" stroke-width="3"/>
  <text x="238" y="70" font-size="10" fill="currentColor">λ/4 element</text>
  <circle cx="230" cy="112" r="3.5" fill="currentColor"/>
  <text x="240" y="128" font-size="9" fill="currentColor">feedpoint</text>
  <line x1="230" y1="112" x2="120" y2="160" stroke="currentColor" stroke-width="2"/>
  <line x1="230" y1="112" x2="340" y2="160" stroke="currentColor" stroke-width="2"/>
  <line x1="230" y1="112" x2="170" y2="175" stroke="currentColor" stroke-width="2" stroke-opacity="0.7"/>
  <line x1="230" y1="112" x2="300" y2="175" stroke="currentColor" stroke-width="2" stroke-opacity="0.7"/>
  <text x="95" y="172" font-size="9" fill="currentColor">radials (≈λ/4)</text>
  <line x1="230" y1="112" x2="230" y2="135" stroke="currentColor" stroke-width="1.5"/>
  <text x="238" y="150" font-size="8" fill="currentColor">coax</text>
</svg>
<figcaption>A ground-plane antenna is a quarter-wave vertical fed against three or four radials that stand in for a solid ground plane.</figcaption>
</figure>

## How it works

The driven element is an ordinary quarter-wave [monopole](/reference/monopole-antenna/):
it needs a conducting surface beneath it to mirror the element into an effective
half-wave [dipole](/reference/dipole-antenna/). A solid sheet works, but so does a
"skeleton" of a few [radial](/reference/radials-counterpoise/) wires, each about a
quarter wavelength long. Three or four radials are enough to approximate the current
distribution of a continuous plane, keeping the antenna light and wind-transparent while
still supplying the return path.

Radial geometry sets the feedpoint impedance. With the radials **horizontal**, an ideal
ground-plane presents roughly **37 Ω** — the monopole value — a poor match to 50 Ω coax.
Sloping the radials **downward at about 45°** raises the feedpoint impedance to near
**50 Ω**, giving a low [SWR](/reference/standing-wave-ratio/) directly on standard coax
without a matching network. Drooping the radials also lifts the main lobe slightly and
lowers the take-off angle, favouring distant signals near the horizon.

The result is an omnidirectional azimuth [pattern](/reference/radiation-pattern/), a null
overhead, and vertical [polarization](/reference/polarization/) — matched to the vertical
land-mobile signals a scanner listens to. Modest [gain](/reference/antenna-gain/) over a
dipole comes from concentrating radiation into the upper half-space.

## Relevance to SDR

For fixed SDR scanning of VHF/UHF trunked systems, a ground-plane antenna is often the
best value: it is broadband enough to cover a whole band segment, omnidirectional so it
hears all sites at once, and vertically polarized to match the target signals. Many
commercial "discone-lite" and dedicated scanner verticals are ground-plane designs.

GopherTrunk decodes whatever the front end delivers and cares only about
signal-to-noise. Because a ground-plane antenna can be built for pennies from wire and a
coax connector, and mounted high with a clear horizon, it is a common first upgrade over
the stock telescopic [whip](/reference/whip-antenna/) that ships with an
[RTL-SDR](/reference/rtl-sdr/) dongle.

## In practice

- **Cut for the band centre.** Element ≈ 7125 / f(MHz) in cm; radials a touch longer than
  the element.
- **Slope radials for match.** Horizontal ≈ 37 Ω; ~45° droop ≈ 50 Ω for a clean match.
- **More radials, diminishing returns.** Four beats three; beyond about four the gain is
  marginal for an elevated ground-plane.

## Sources

[^wiki]: [Ground plane](https://en.wikipedia.org/wiki/Ground_plane) — Wikipedia, for the radial ground-plane antenna, its ~37 Ω horizontal feedpoint, and the 45° droop that raises it toward 50 Ω.
