---
slug: ais
title: Automatic Identification System (AIS)
entry_type: protocol
category: protocols
description: AIS (Automatic Identification System) is a maritime VHF data system in which ships broadcast identity, position, course, and speed using GMSK in a self-organising TDMA scheme.
keywords: AIS, Automatic Identification System, marine tracking, GMSK, SOTDMA, 161.975 162.025, vessel position, NMEA
aka: [AIS]
autolink: true
infobox:
  - { label: Type, value: Maritime data broadcast }
  - { label: Standards body, value: ITU / IMO }
  - { label: Band, value: VHF marine (161.975 / 162.025 MHz) }
  - { label: Access, value: Self-organising TDMA (SOTDMA) }
  - { label: Modulation, value: GMSK, 9600 bps }
  - { label: Content, value: MMSI, position, course, speed }
  - { label: GopherTrunk support, value: Decoded }
see_also: [ads-b, gmsk, tdma, itu, dsc]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Automatic identification system (Wikipedia)", url: https://en.wikipedia.org/wiki/Automatic_identification_system }
  - { title: "GopherTrunk AIS decoder", url: /ais.html }
---

**AIS** (**Automatic Identification System**) is a maritime safety system in which
ships and shore stations broadcast their **identity, position, course, and speed** on
VHF marine frequencies. It uses [GMSK](/reference/gmsk/) modulation in a
self-organising [TDMA](/reference/tdma/) scheme so many vessels share two channels
without a central controller.

## Overview

Each station transmits position reports keyed to its MMSI identifier on 161.975 MHz
(AIS 1) and 162.025 MHz (AIS 2). Receiving AIS gives a live map of nearby vessel
traffic, the maritime counterpart to [ADS-B](/reference/ads-b/).

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | VHF marine, ~162 MHz |
| Access | SOTDMA |
| Modulation | GMSK, 9600 bps |
| Framing | HDLC-style with CRC-16 |

## History

Standardised by the [ITU](/reference/itu/) and mandated by the IMO for SOLAS vessels
from the early 2000s to improve collision avoidance and traffic monitoring.

## Deployment

Commercial shipping, port authorities, vessel-traffic services, and hobbyist tracking.

## Decoding it with GopherTrunk

GopherTrunk demodulates the GMSK bursts, frames them, and decodes position reports.
See the [AIS decoder](/ais.html) page.
