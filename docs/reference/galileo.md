---
slug: galileo
title: Galileo
entry_type: protocol
category: satellite-gnss
description: "Galileo is the European Union's civil-controlled satellite navigation system, a CDMA GNSS in the L-band offering an open service and an encrypted high-accuracy service."
keywords: Galileo, European GNSS, EU satellite navigation, EUSPA, ESA, E1, E5, CDMA, GNSS, open service, high accuracy service, satnav
aka: [Galileo]
autolink: true
infobox:
  - { label: Type, value: Satellite navigation (GNSS) }
  - { label: Operator, value: EUSPA / ESA (civil-controlled) }
  - { label: Introduced, value: "2016 (initial services)" }
  - { label: Access, value: "CDMA" }
  - { label: Band, value: "E1 1575.42 MHz, E5, E6" }
  - { label: Modulation, value: BOC / CBOC-spread DSSS }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [gnss, gps-gnss, glonass, beidou, multilateration]
cite_urls:
  - https://en.wikipedia.org/wiki/Galileo_(satellite_navigation)
  - https://www.euspa.europa.eu/eu-space-programme/galileo
---

**Galileo** is the European Union's global satellite navigation system and a member
of the [GNSS](/reference/gnss/) family, notable for being the first such system built
under **civilian rather than military control**.[^wiki] Like
[GPS](/reference/gps-gnss/) and [BeiDou](/reference/beidou/) it shares one L-band
frequency across all satellites using code division ([CDMA](/reference/cdma/)), and a
receiver fixes its position by [multilaterating](/reference/multilateration/) ranges
to four or more satellites.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Galileo satellites orbit in three planes and share the E1 frequency, interoperable with GPS on the same 1575.42 megahertz carrier." xmlns="http://www.w3.org/2000/svg">
  <ellipse cx="230" cy="80" rx="150" ry="55" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <ellipse cx="230" cy="80" rx="55" ry="150" fill="none" stroke="currentColor" stroke-opacity="0.5" transform="rotate(60 230 80)"/>
  <ellipse cx="230" cy="80" rx="55" ry="150" fill="none" stroke="currentColor" stroke-opacity="0.5" transform="rotate(-60 230 80)"/>
  <circle cx="230" cy="80" r="18" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <g fill="currentColor"><circle cx="80" cy="80" r="4"/><circle cx="380" cy="80" r="4"/><circle cx="160" cy="12" r="4"/><circle cx="300" cy="148" r="4"/></g>
  <text x="230" y="84" text-anchor="middle" font-size="9" fill="currentColor">Earth</text>
  <text x="230" y="156" text-anchor="middle" font-size="9" fill="currentColor">3 orbital planes, ~23,222 km, E1 shared by CDMA</text>
</svg>
<figcaption>Galileo's satellites occupy three inclined MEO planes and transmit on E1, co-located with GPS L1 for interoperability.</figcaption>
</figure>

## Overview

Galileo is a constellation designed for a full complement of 24 operational
satellites (plus spares) in three medium-Earth-orbit planes at about 23,222 km. It
provides several services: a free **Open Service** for mass-market positioning, a
**High Accuracy Service** delivering decimetre-level corrections, and an encrypted
**Public Regulated Service** for government and safety-of-life users. Its E1 Open
Service is deliberately placed on the same 1575.42 MHz carrier as GPS L1 so a single
antenna and front end can receive both.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [CDMA](/reference/cdma/) |
| Carriers | E1 = 1575.42 MHz, E5a/E5b ≈ 1176–1207 MHz, E6 |
| Modulation | CBOC (E1), AltBOC (E5) — offshoots of BPSK spreading |
| Satellites | 24 operational + spares, three MEO planes |
| Altitude | ~23,222 km |
| Distinctive | First civil-controlled GNSS; SAR relay payload |

Galileo's binary-offset-carrier (BOC/CBOC) modulation splits signal energy away from
the band centre, which sharpens the correlation peak and improves multipath rejection
compared with the simple BPSK spreading of legacy GPS C/A — one reason
multi-constellation receivers value it. Galileo satellites also carry a
[COSPAS-SARSAT](/reference/cospas-sarsat/) search-and-rescue relay payload and can even
send a return-link acknowledgement to a distress beacon.

## History

Galileo grew out of European ambitions in the 1990s for a navigation system
independent of the US military-run GPS. The first test satellites flew in 2005–2008,
initial services began in December 2016, and the constellation reached near-completion
in the early 2020s, with a second generation of satellites in development.

## Deployment

Galileo is used everywhere modern GNSS chips are used — its signals are tracked
alongside GPS, [GLONASS](/reference/glonass/), and [BeiDou](/reference/beidou/) in
virtually every current smartphone and survey receiver. Its High Accuracy Service and
tight time reference make it attractive for precision agriculture, surveying, and
timing.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode Galileo. Satellite navigation in the L-band is outside
the scope of a VHF/UHF land-mobile trunking scanner. Galileo can be received by a
general-purpose software-defined radio with an active antenna and low-noise amplifier —
and because its E1 signal overlaps GPS L1, the same capture can hold both — but turning
that into a position fix needs dedicated GNSS software, not GopherTrunk. Any relevance
to GopherTrunk is limited to GNSS as an external timing reference.

## Sources

[^wiki]: [Galileo (satellite navigation)](https://en.wikipedia.org/wiki/Galileo_(satellite_navigation)) — Wikipedia, for Galileo's civil governance, service tiers, E1/E5 signals and BOC modulation, orbit, and the search-and-rescue payload.
