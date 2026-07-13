---
slug: whip-antenna
title: Whip antenna
entry_type: term
category: antennas
description: A whip antenna is a flexible single-rod monopole, usually a quarter or half wavelength, used on vehicles and handhelds for omnidirectional vertical coverage.
keywords: whip antenna, flexible whip, telescopic antenna, monopole, mobile antenna, rubber duck, quarter-wave whip, vehicle antenna
aka: [whip, flexible whip, telescopic whip, rubber duck]
autolink: true
infobox:
  - { label: Type, value: Flexible monopole }
  - { label: Length, value: ~λ/4 (often shortened/loaded) }
  - { label: Pattern, value: Omnidirectional, vertical }
see_also: [monopole-antenna, ground-plane-antenna, dipole-antenna, rtl-sdr, polarization]
cite_urls:
  - https://en.wikipedia.org/wiki/Whip_antenna
  - https://en.wikipedia.org/wiki/Monopole_antenna
---

A **whip antenna** is a [monopole](/reference/monopole-antenna/) built from a single
flexible or telescopic rod, most often a quarter [wavelength](/reference/wavelength/)
long.[^wiki] The name captures its behaviour: a thin, springy conductor that bends and
whips back when disturbed, which is exactly what a vehicle or handheld antenna needs to
survive motion, wind, and knocks. Whips are the most widely deployed antennas in the
world, found on cars, HTs, portable scanners, and every USB SDR dongle.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A flexible vertical whip rod curving slightly at the top, mounted on a base that uses a car roof or dongle body as its ground plane." xmlns="http://www.w3.org/2000/svg">
  <path d="M170 150 C 170 100, 172 70, 182 38" fill="none" stroke="currentColor" stroke-width="3"/>
  <text x="190" y="70" font-size="10" fill="currentColor">flexible λ/4 rod</text>
  <rect x="160" y="150" width="20" height="12" fill="currentColor" opacity="0.7"/>
  <line x1="120" y1="164" x2="220" y2="164" stroke="currentColor" stroke-width="2"/>
  <text x="225" y="168" font-size="9" fill="currentColor">vehicle body / dongle = ground</text>
  <line x1="170" y1="162" x2="170" y2="182" stroke="currentColor" stroke-width="1.5"/>
  <text x="176" y="180" font-size="8" fill="currentColor">feed</text>
  <ellipse cx="176" cy="95" rx="70" ry="18" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
</svg>
<figcaption>A whip is a flexible quarter-wave monopole; the mount surface — a car roof or the SDR's own body — acts as its ground plane.</figcaption>
</figure>

## How it works

Electrically a whip is a [monopole](/reference/monopole-antenna/): a quarter-wave
element that works against a ground plane, which is usually the metal surface it is
mounted on. On a car the roof or trunk lid is the plane; on a handheld or dongle the
body, the operator's hand, and the coax shield form an imperfect but usable ground. Like
any monopole it is vertically [polarized](/reference/polarization/), omnidirectional in
azimuth, and radiates toward the horizon with a null overhead.

Many whips are physically shorter than a true quarter wave. A **loaded whip** inserts an
inductor — a base coil, a centre coil, or a continuous helical winding (the "rubber
duck") — to make an electrically short rod resonate. Loading buys size at a cost:
shortened, loaded whips have lower radiation resistance, narrower
[bandwidth](/reference/bandwidth/), and less [gain](/reference/antenna-gain/) than a
full-length element. A **half-wave whip** goes the other way, using a longer rod with an
end-matching network; it needs no ground plane and keeps a stable pattern on a poor
mount, which is why many aftermarket mobile antennas are 5/8-wave or collinear whips for
extra gain toward the horizon.

Because the ground on a handheld is small and unpredictable, a rubber-duck whip is a
compromise antenna — convenient but lossy. Swapping it for a full quarter-wave element
with a real ground plane routinely adds several dB of usable signal.

## Relevance to SDR

The telescopic or "rubber duck" whip bundled with an [RTL-SDR](/reference/rtl-sdr/) is a
whip antenna, and it is almost always the weakest link in a scanning setup. Its short,
poorly grounded element makes it broad but insensitive. For VHF/UHF trunking, extending a
telescopic whip to a true quarter wave for the band, or replacing it with a
[ground-plane antenna](/reference/ground-plane-antenna/), gives the biggest single
improvement in signal-to-noise available to a GopherTrunk user.

GopherTrunk itself is a receive-only decoder and works with any antenna; the whip's
vertical [polarization](/reference/polarization/) happens to match land-mobile P25, DMR,
and NXDN signals, so a well-mounted vertical whip already suits the traffic GT targets.

## In practice

- **Extend telescopics to λ/4.** Length in cm ≈ 7125 / f(MHz); a collapsed whip is far
  off resonance.
- **Rubber ducks trade size for loss.** Fine for close-in monitoring, poor for weak
  distant sites.
- **Give it a ground.** A magnetic-mount ground plane or a couple of counterpoise wires
  dramatically improves a handheld whip.

## Sources

[^wiki]: [Whip antenna](https://en.wikipedia.org/wiki/Whip_antenna) — Wikipedia, for the flexible monopole construction, loaded/rubber-duck variants, and mobile mounting as ground plane.
