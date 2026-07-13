---
slug: biconical-antenna
title: Biconical antenna
entry_type: term
category: antennas
description: A biconical antenna is a dipole whose arms flare into cones, giving broadband omnidirectional coverage; its ground-plane relative is the discone.
keywords: biconical antenna, bicone antenna, broadband dipole, EMC antenna, discone antenna, wideband antenna, flared dipole, omnidirectional broadband
aka: [bicone antenna, biconical dipole]
autolink: true
infobox:
  - { label: Type, value: Broadband flared dipole }
  - { label: Bandwidth, value: Wide (multi-octave) }
  - { label: Pattern, value: Omnidirectional (like a dipole) }
see_also: [antenna, dipole-antenna, discone-antenna, impedance, radiation-pattern, frequency-bands]
cite_urls:
  - https://en.wikipedia.org/wiki/Biconical_antenna
  - https://en.wikipedia.org/wiki/Discone_antenna
---

A **biconical antenna** is a [dipole](/reference/dipole-antenna/) whose two straight arms
are replaced by two conductive **cones** meeting point-to-point at the feed.[^wiki] The
flare is the whole trick: a thin-wire dipole is resonant and only works well near one
frequency, but fattening the arms into cones smooths the antenna's
[impedance](/reference/impedance/) so it stays reasonably matched over a very wide band —
often several octaves. The [pattern](/reference/radiation-pattern/) remains
dipole-like — omnidirectional in the plane perpendicular to the axis — so the biconical
gives broadband coverage in every horizontal direction at once, which is why it is a
standard antenna for EMC test labs and wideband monitoring.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Two cones point tip-to-tip at a central feed, forming a vertical biconical antenna with an omnidirectional doughnut pattern around it." xmlns="http://www.w3.org/2000/svg">
  <path d="M230 88 L 180 20 L 280 20 Z" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M230 92 L 180 160 L 280 160 Z" fill="none" stroke="currentColor" stroke-width="2"/>
  <circle cx="230" cy="90" r="3" fill="currentColor"/>
  <text x="238" y="94" font-size="9" fill="currentColor">feed</text>
  <ellipse cx="230" cy="90" rx="130" ry="26" fill="none" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="4 3"/>
  <text x="300" y="30" font-size="9" fill="currentColor">upper cone</text>
  <text x="300" y="158" font-size="9" fill="currentColor">lower cone</text>
  <text x="365" y="90" font-size="9" fill="currentColor">omni pattern</text>
</svg>
<figcaption>Flaring the dipole arms into cones broadens the impedance match, giving multi-octave bandwidth while keeping an omnidirectional pattern.</figcaption>
</figure>

## How it works

An ideal infinite biconical structure has a **constant** characteristic impedance set only
by the cone angle — it behaves like a transmission line that happens to radiate, with no
resonance to tie it to one frequency. Real cones are finite, so reflections from their ends
introduce some frequency dependence, but the antenna still holds a usable match over a huge
range compared with a thin dipole. Widening the cone angle lowers the impedance and further
flattens it; the designer picks an angle that trades bandwidth against a convenient match to
the feed line.

The radiation mechanism is otherwise that of a fat vertical dipole: currents flow along the
cone surfaces, and the antenna radiates broadside with a doughnut pattern and nulls off the
ends. Gain is modest — comparable to a dipole — because bandwidth, not gain, is the point.
Cages of rods or spokes are usually substituted for solid metal cones to cut weight and wind
load without changing the electrical behaviour, since the current mainly hugs the outer edge.

## Variants

The most important relative is the **[discone antenna](/reference/discone-antenna/)**: take
a biconical antenna and replace the upper cone with a flat **disc**, which acts as a ground
plane. The result is an unbalanced, monopole-style radiator with the same multi-octave
bandwidth and vertical omnidirectional pattern, but fed against a disc instead of a matching
cone — mechanically simpler and the form most scanner listeners actually own. A related
broadband form is the **bowtie**, a two-dimensional (flat) biconical often used for TV and
UWB. Where more *gain* rather than pure bandwidth is wanted, the
[log-periodic](/reference/log-periodic-antenna/) is the directional broadband alternative.

## Relevance to SDR

Because a wideband SDR can watch a huge slice of spectrum at once, an antenna that stays
matched across that whole slice is genuinely useful, and the biconical family fills the
role. In practice most SDR users deploy the discone version as a set-and-forget
receive antenna covering roughly VHF through the low microwave range, so a single antenna
serves airband, marine, land-mobile, and utility monitoring without swapping. Calibrated
biconicals are also the reference radiators in EMC emissions testing, where flat broadband
response matters more than gain.

For GopherTrunk specifically, a **discone or biconical is a very practical antenna**: it
covers the VHF and UHF land-mobile bands where P25, DMR, NXDN, and TETRA trunked systems
live, and its omnidirectional pattern suits scanning a multi-site system whose towers lie in
different directions. The trade-off is the usual one — its broadband, low-gain nature means
it hears everything but favours nothing, so a weak distant site may still call for a tuned
or directional antenna.

## Sources

[^wiki]: [Biconical antenna](https://en.wikipedia.org/wiki/Biconical_antenna) — Wikipedia, for the flared-dipole construction, broadband impedance behaviour, and the discone relationship.
