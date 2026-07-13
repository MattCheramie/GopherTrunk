---
slug: parabolic-antenna
title: Parabolic (dish) antenna
entry_type: term
category: antennas
description: A parabolic antenna uses a curved reflector to focus radio waves onto a small feed, giving very high gain and a narrow beam at microwave and satellite frequencies.
keywords: parabolic antenna, dish antenna, reflector antenna, parabolic reflector, satellite dish, prime focus, offset feed, Cassegrain, high gain antenna, microwave antenna
aka: [dish antenna, parabolic reflector, satellite dish, reflector antenna]
autolink: true
infobox:
  - { label: Type, value: Reflector (aperture) antenna }
  - { label: Gain, value: High (tens of dBi) }
  - { label: Beam, value: Narrow pencil beam }
see_also: [antenna, antenna-gain, effective-aperture, beamwidth, radiation-pattern, horn-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Parabolic_antenna
  - https://en.wikipedia.org/wiki/Parabolic_reflector
---

A **parabolic antenna** is a reflector [antenna](/reference/antenna/) that uses a
metal dish shaped as a paraboloid to collect an incoming plane wave and focus all of
its energy onto a small feed placed at the reflector's focus.[^wiki] Because a
paraboloid turns a parallel beam into a single point (and vice versa on transmit), the
dish behaves as a large collecting aperture, so it delivers very high
[gain](/reference/antenna-gain/) and a narrow [beam](/reference/beamwidth/) — the two
defining traits that make it the workhorse of microwave links, radar, and satellite
ground stations.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="Parallel incoming rays strike a parabolic dish and reflect to a common focal point where the feed sits, illustrating how the reflector concentrates energy." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="paar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M120 20 Q 40 100 120 180" fill="none" stroke="currentColor" stroke-width="3"/>
  <circle cx="150" cy="100" r="4" fill="currentColor"/>
  <text x="158" y="96" font-size="10" fill="currentColor">feed (focus)</text>
  <line x1="360" y1="40" x2="150" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="360" y1="100" x2="150" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="360" y1="160" x2="150" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="440" y1="40" x2="112" y2="40" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#paar)"/>
  <line x1="440" y1="100" x2="132" y2="100" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#paar)"/>
  <line x1="440" y1="160" x2="112" y2="160" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#paar)"/>
  <text x="370" y="30" font-size="10" fill="currentColor">plane wave in</text>
</svg>
<figcaption>A paraboloid reflects every parallel ray through a single focus, concentrating a large collecting area onto one small feed.</figcaption>
</figure>

## How it works

The reflecting surface is a section of a paraboloid of revolution. A parabola has the
geometric property that every ray arriving parallel to its axis reflects to a common
focal point, and all such paths have equal length. So a plane wave sweeping in from a
distant transmitter arrives at the feed **in phase** and adds coherently, while noise
and signals from other directions do not. On transmit the process reverses: the feed
illuminates the dish and the reflector collimates the energy into a tight pencil beam.

The dish is an **aperture antenna** — its performance is set by physical size, not by
resonant length. Its gain follows the aperture relation

G ≈ (4π / λ²) · A · η

where A is the physical area, λ the [wavelength](/reference/wavelength/), and η the
aperture efficiency (typically 0.5–0.7 after feed spillover, blockage, and illumination
taper). Equivalently the dish presents an [effective aperture](/reference/effective-aperture/)
of ηA to the incoming field. Two consequences follow directly: gain rises with the
**square** of diameter (double the dish, gain up ~6 dB), and rises with frequency, which
is why dishes dominate at microwave bands where a modest physical size is many
wavelengths across. The half-power [beamwidth](/reference/beamwidth/) shrinks roughly as
θ ≈ 70·λ/D degrees, so a large high-frequency dish can have a beam under a degree wide —
demanding accurate pointing.

## Variants

- **Prime-focus (front-fed):** the feed sits at the focus on the dish axis. Simple, but
  the feed and its support struts block part of the aperture and add spillover noise.
- **Offset-fed:** the reflector is a slice of the paraboloid cut away from the axis, so
  the feed sits below the beam and causes no blockage. Common on home satellite-TV and
  VSAT dishes.
- **Cassegrain / Gregorian:** a small convex (Cassegrain) or concave (Gregorian)
  sub-reflector near the focus folds the path back to a feed at the dish vertex. This
  shortens feed lines and lets the feed "look" at cold sky rather than warm ground,
  lowering noise — standard on large radio-astronomy and deep-space antennas.

The feed itself is usually a [horn antenna](/reference/horn-antenna/) or a dipole with a
small reflector, chosen so its pattern illuminates the dish evenly without spilling
energy past the rim.

## Relevance to SDR

Parabolic dishes appear wherever a link needs high gain and a narrow beam: satellite
downlinks (Ku/Ka-band TV, VSAT data, [Inmarsat](/reference/inmarsat/) and other L-band
terminals), terrestrial microwave backhaul, weather-satellite and deep-space reception,
and radar. For the SDR hobbyist, a dish (or its cheaper mesh/grid cousin) is the usual
front end for receiving weak signals from geostationary and low-earth-orbit satellites,
where the extra tens of dB of gain is what lifts the carrier above the
[noise floor](/reference/noise-floor/). Because the dish is only a passive collector, an
SDR sees exactly what lands at the feed — the receiver's job is unchanged, but the link
budget improves dramatically.

GopherTrunk is a land-mobile trunking decoder (P25, DMR, NXDN, TETRA and similar), and
those systems use omnidirectional or modest gain [vertical antennas](/reference/monopole-antenna/),
not dishes. So a parabolic antenna is not part of a typical GopherTrunk install; it is
relevant here as the canonical high-gain aperture antenna and the reference point for
understanding [effective aperture](/reference/effective-aperture/) and gain in the wider
RF world.

## Sources

[^wiki]: [Parabolic antenna](https://en.wikipedia.org/wiki/Parabolic_antenna) — Wikipedia, for reflector geometry, feed variants, and the aperture gain relation.
