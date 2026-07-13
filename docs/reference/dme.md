---
slug: dme
title: Distance Measuring Equipment (DME)
entry_type: technology
category: aviation-marine
description: DME (Distance Measuring Equipment) is an aviation navaid that gives slant-range distance by timing the round trip of interrogation and reply pulse pairs between an aircraft and a ground beacon.
keywords: DME, Distance Measuring Equipment, slant range, pulse pair, interrogation, reply, transponder, 50 microsecond delay, 960-1215 MHz, VOR/DME, TACAN
aka: [DME]
autolink: true
infobox:
  - { label: Type, value: Distance navaid (secondary radar principle) }
  - { label: Idea, value: Round-trip pulse timing = slant range }
  - { label: Band, value: L-band 960–1215 MHz }
see_also: [tacan, vor, pulse-position-modulation, ils]
cite_urls:
  - https://en.wikipedia.org/wiki/Distance_measuring_equipment
  - https://www.icao.int/
---

**DME** (**Distance Measuring Equipment**) is a radio navigation aid that tells an
aircraft its **slant-range distance to a ground beacon** by timing a radio round trip. The
airborne set transmits pairs of pulses; the ground station waits a fixed delay and
replies with its own pulse pairs; the aircraft measures the total elapsed time, subtracts
the known delay, and converts to distance.[^wiki] Operating in the L-band
(960–1215 MHz), DME is almost always paired with a [VOR](/reference/vor/) or
[ILS](/reference/ils/) so a crew gets both bearing (or course) and distance from one
tuned facility.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An aircraft sending interrogation pulse pairs to a ground beacon, which replies after a fixed delay; the round-trip time gives the distance." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dmear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="55" y="40" text-anchor="middle" font-size="9" fill="currentColor">aircraft</text>
  <text x="400" y="40" text-anchor="middle" font-size="9" fill="currentColor">ground beacon</text>
  <g stroke="currentColor"><line x1="45" y1="45" x2="45" y2="115"/><line x1="65" y1="45" x2="65" y2="115"/><line x1="395" y1="45" x2="395" y2="115"/><line x1="415" y1="45" x2="415" y2="115"/></g>
  <line x1="70" y1="65" x2="388" y2="65" stroke="currentColor" marker-end="url(#dmear)"/>
  <text x="230" y="60" text-anchor="middle" font-size="8" fill="currentColor">interrogation pulse pair</text>
  <line x1="390" y1="95" x2="72" y2="95" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#dmear)"/>
  <text x="230" y="90" text-anchor="middle" font-size="8" fill="currentColor">reply after fixed 50 µs delay</text>
  <text x="230" y="128" text-anchor="middle" font-size="8.5" fill="currentColor">round-trip time − delay → slant range</text>
</svg>
<figcaption>DME times the round trip of pulse pairs — interrogation out, reply back after a fixed delay — to compute slant-range distance.</figcaption>
</figure>

## How it works

DME is a form of secondary radar. The aircraft's interrogator sends closely spaced
**pulse pairs** (the pairing and spacing distinguish DME from other L-band traffic and
help reject noise). The ground transponder receives them, waits a standardised **50 µs**
fixed delay, and re-transmits pulse pairs on a paired reply frequency. The airborne
receiver correlates its own replies out of the stream — every beacon serves many aircraft
at once, so each interrogator jitters its pulse timing and looks only for replies that
track its own pattern. Elapsed round-trip time, less the 50 µs delay and processing
allowances, converts to distance at the speed of light.

The result is **slant range** — the straight-line distance to the antenna, which exceeds
ground distance when the aircraft is high and close. Because the encoding lives in pulse
position and spacing, DME is conceptually related to
[pulse-position modulation](/reference/pulse-position-modulation/). The military
[TACAN](/reference/tacan/) system reuses the DME distance function and adds bearing, which
is why civil "VOR/DME" and military "VORTAC" facilities interoperate.

## Relevance to SDR

DME is harder to receive casually than the AM navaids because it is pulsed L-band traffic
shared with many users, but its principle — round-trip timing of coded pulse pairs — is
the same idea behind [Mode S](/reference/mode-s/) and other secondary-radar signals a
[software-defined radio](/reference/software-defined-radio/) hobbyist may sample.
**GopherTrunk** does not decode DME; it is a land-mobile trunking scanner, and DME is
documented here to complete the aviation-navaid picture alongside VOR, ILS, and TACAN.

## Sources

[^wiki]: [Distance measuring equipment](https://en.wikipedia.org/wiki/Distance_measuring_equipment) — Wikipedia, for the DME pulse-pair interrogation/reply scheme, the fixed 50 µs transponder delay, slant-range measurement, and pairing with VOR/ILS.
