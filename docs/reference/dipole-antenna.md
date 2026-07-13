---
slug: dipole-antenna
title: Dipole antenna
entry_type: term
category: antennas
description: A dipole is a simple resonant antenna of two conductors fed at the centre, typically a half wavelength long, with an omnidirectional pattern broadside to its axis.
keywords: dipole, half-wave dipole, resonant antenna, radiation pattern, balun, feedpoint impedance, dBd, folded dipole
aka: [dipole antenna, dipole]
autolink: true
infobox:
  - { label: Type, value: Resonant antenna }
  - { label: Length, value: ~λ/2 (half-wave) }
  - { label: Pattern, value: Omnidirectional broadside }
see_also: [antenna, monopole-antenna, yagi-uda-antenna, antenna-gain, radiation-pattern, feedpoint-impedance, polarization, wavelength]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Dipole_antenna
  - https://en.wikipedia.org/wiki/Balun
---

A **dipole antenna** is a simple resonant [antenna](/reference/antenna/) made of two
conductors fed at the centre, classically a **half [wavelength](/reference/wavelength/)**
long.[^wiki] It is the reference against which other antennas' [gain](/reference/antenna-gain/)
is often compared — the unit **dBd** means "decibels relative to a half-wave dipole,"
which itself has about 2.15 dBi of gain over an isotropic radiator.

<figure class="figure" markdown="0">
<svg viewBox="0 0 320 150" role="img" aria-label="A centre-fed dipole with two quarter-wave rods and a doughnut-shaped radiation pattern around it." xmlns="http://www.w3.org/2000/svg">
  <line x1="160" y1="20" x2="160" y2="68" stroke="currentColor" stroke-width="3"/>
  <line x1="160" y1="82" x2="160" y2="130" stroke="currentColor" stroke-width="3"/>
  <circle cx="160" cy="75" r="3" fill="currentColor"/>
  <text x="172" y="48" font-size="10" fill="currentColor">λ/4</text><text x="172" y="112" font-size="10" fill="currentColor">λ/4</text>
  <ellipse cx="160" cy="75" rx="110" ry="28" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="4 3"/>
  <text x="250" y="75" font-size="9" fill="currentColor">pattern</text>
</svg>
<figcaption>A half-wave dipole is two quarter-wave rods fed in the middle, most sensitive broadside to its length.</figcaption>
</figure>

## How it works

At resonance a half-wave dipole carries a standing wave of current that is maximum at the
centre feedpoint and falls to zero at the two ends. That current distribution radiates
most strongly **broadside** — perpendicular to the wire — and hardly at all off the ends,
producing the familiar doughnut-shaped [radiation pattern](/reference/radiation-pattern/)
wrapped around the axis. A vertical dipole is therefore omnidirectional in the horizontal
plane, which is exactly what a scanner wants when signals can come from any bearing. The
wave's [polarization](/reference/polarization/) follows the wire's orientation: a vertical
dipole is vertically polarized, a horizontal one horizontally polarized.

The centre [feedpoint impedance](/reference/feedpoint-impedance/) of a thin half-wave
dipole in free space is close to **73 Ω** resistive, a good match to common 50–75 Ω coax
and a big reason the design is so practical. Because the two halves are balanced with
respect to ground while coax is unbalanced, a purist adds a **balun** (balance-to-unbalance
transformer) at the feedpoint to keep current off the outside of the coax shield, which
would otherwise distort the pattern and raise noise pickup.[^balun] For receive-only
scanning the effect is usually mild, but a balun or a few [ferrite](/reference/ferrite-choke/)
chokes on the feedline can noticeably clean up a marginal install.

## Variants

- **Half-wave dipole** — the canonical λ/2 form described above.
- **Folded dipole** — two parallel conductors joined at the ends, raising the feedpoint
  impedance to about 300 Ω (useful with twin-lead) and widening the bandwidth.
- **Quarter-wave [monopole](/reference/monopole-antenna/)** — effectively half a dipole
  worked against a ground plane; a ground-plane whip is the everyday scanner antenna.
- **Driven element of a [Yagi](/reference/yagi-uda-antenna/)** — a dipole surrounded by
  parasitic reflector and director rods to concentrate the pattern into a beam.

## In practice

A dipole is easy to build to size: total length in metres is roughly 143 / f(MHz) for the
half-wave, trimmed a few percent for the wire's thickness and end effects. Cut for the
centre of a target band, it stays usable across a fair bandwidth on either side. Mounted
vertically and up high with a clear path, it is one of the best value receive antennas for
land-mobile and trunked monitoring.

## Relevance to SDR

A dipole (or its grounded cousin, the quarter-wave whip) cut for the target band is a
cheap, effective receive antenna for scanning. Its omnidirectional broadside pattern suits
a wideband SDR that watches many sites at once, and its predictable ~73 Ω feedpoint keeps
[SWR](/reference/standing-wave-ratio/) low without tuning. GopherTrunk decodes whatever the
front end delivers; a resonant dipole simply hands the [ADC](/reference/analog-to-digital-converter/)
a stronger, cleaner signal than a random wire, improving lock on weak
[control channels](/reference/control-channel/).

## Sources

[^wiki]: [Dipole antenna](https://en.wikipedia.org/wiki/Dipole_antenna) — Wikipedia, for the construction, feedpoint impedance, and radiation pattern of the half-wave dipole.
[^balun]: [Balun](https://en.wikipedia.org/wiki/Balun) — Wikipedia, on balanced-to-unbalanced transformers and why they are used at a dipole's feedpoint.
