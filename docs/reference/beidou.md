---
slug: beidou
title: BeiDou (BDS)
entry_type: protocol
category: satellite-gnss
description: "BeiDou is China's satellite navigation system, a global GNSS using a mixed MEO/GEO/IGSO constellation and CDMA signals, with a short-message service alongside positioning."
keywords: BeiDou, BDS, Compass, China satellite navigation, GNSS, CDMA, MEO GEO IGSO, B1 B2 B3, short message service, satnav
aka: [BeiDou, BDS, Compass]
autolink: true
infobox:
  - { label: Type, value: Satellite navigation (GNSS) }
  - { label: Operator, value: China (CSNO / CNSA) }
  - { label: Introduced, value: "2000 regional, 2020 global" }
  - { label: Access, value: "CDMA" }
  - { label: Band, value: "B1 ~1561/1575 MHz, B2, B3" }
  - { label: Modulation, value: BPSK / BOC-spread DSSS }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [gnss, gps-gnss, glonass, galileo, multilateration]
cite_urls:
  - https://en.wikipedia.org/wiki/BeiDou
  - http://en.beidou.gov.cn/
---

**BeiDou** (**BDS**, the BeiDou Navigation Satellite System; formerly "Compass") is
China's satellite navigation system and the fourth global member of the
[GNSS](/reference/gnss/) family.[^wiki] Like [GPS](/reference/gps-gnss/) and
[Galileo](/reference/galileo/) it uses code division ([CDMA](/reference/cdma/)) in the
L-band, and a receiver fixes its position by
[multilaterating](/reference/multilateration/) ranges to four or more satellites — but
BeiDou is distinctive for its **mixed-orbit constellation** and a built-in short-message
service.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="BeiDou combines geostationary, inclined geosynchronous, and medium Earth orbit satellites in one hybrid constellation." xmlns="http://www.w3.org/2000/svg">
  <circle cx="230" cy="90" r="20" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="230" y="94" text-anchor="middle" font-size="9" fill="currentColor">Earth</text>
  <circle cx="230" cy="90" r="130" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <ellipse cx="230" cy="90" rx="80" ry="130" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <circle cx="230" cy="30" r="5" fill="currentColor"/><text x="230" y="20" text-anchor="middle" font-size="8" fill="currentColor">GEO</text>
  <circle cx="150" cy="140" r="5" fill="currentColor"/><text x="150" y="158" text-anchor="middle" font-size="8" fill="currentColor">IGSO</text>
  <circle cx="360" cy="90" r="5" fill="currentColor"/><text x="360" y="80" text-anchor="middle" font-size="8" fill="currentColor">MEO</text>
  <circle cx="100" cy="90" r="5" fill="currentColor"/>
</svg>
<figcaption>BeiDou blends geostationary (GEO), inclined geosynchronous (IGSO), and medium-Earth-orbit (MEO) satellites in one system.</figcaption>
</figure>

## Overview

BeiDou reached full global coverage in 2020 with roughly 30 operational satellites.
Unlike the purely medium-Earth-orbit constellations of GPS, [GLONASS](/reference/glonass/),
and Galileo, BeiDou mixes three orbit types: geostationary (GEO) and inclined
geosynchronous (IGSO) satellites that dwell over the Asia-Pacific region for strong
regional coverage, plus a ring of medium-Earth-orbit (MEO) satellites for global
service. The GEO satellites also relay a two-way short-message service, letting
BeiDou-equipped terminals send brief text reports where no cellular coverage exists.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [CDMA](/reference/cdma/) |
| Carriers | B1I/B1C ≈ 1561/1575 MHz, B2, B3 |
| Constellation | GEO + IGSO + MEO (hybrid) |
| MEO altitude | ~21,500 km |
| Modulation | BPSK and BOC direct-sequence spread spectrum |
| Extra service | Regional short-message (two-way) |

The hybrid constellation is a deliberate design choice: the geostationary and
inclined-geosynchronous satellites give China and its neighbours many always-visible
signals for robust regional accuracy, while the MEO satellites provide the worldwide
coverage expected of a global [GNSS](/reference/gnss/). Modern B1C signals are placed
on 1575.42 MHz to interoperate with GPS L1 and Galileo E1.

## History

BeiDou was built in phases. BeiDou-1 (from 2000) was an experimental regional system
using a small number of geostationary satellites. BeiDou-2/Compass extended regional
service across the Asia-Pacific from around 2012, and BeiDou-3 completed the global
constellation, declared fully operational in July 2020.

## Deployment

BeiDou is tracked by essentially all modern multi-constellation GNSS chips and is
especially strong across Asia. Its short-message capability is used for
disaster-response and maritime reporting where terrestrial networks are absent. As with
the other constellations, everyday receivers blend BeiDou with GPS, GLONASS, and
Galileo for the best fix.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode BeiDou. L-band satellite navigation is out of scope for
a VHF/UHF land-mobile trunking scanner. BeiDou signals can be captured with a
general-purpose software-defined radio and an active L-band antenna, and its
GPS-compatible B1C carrier means one capture can span several constellations, but
computing a fix requires dedicated GNSS receiver software. For GopherTrunk, GNSS
matters only as an external time and frequency reference.

## Sources

[^wiki]: [BeiDou](https://en.wikipedia.org/wiki/BeiDou) — Wikipedia, for BeiDou's phased history, the hybrid GEO/IGSO/MEO constellation, CDMA L-band signals, the short-message service, and the 2020 global completion.
