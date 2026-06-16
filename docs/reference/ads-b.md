---
slug: ads-b
title: ADS-B
entry_type: protocol
category: protocols
description: ADS-B (Automatic Dependent Surveillance–Broadcast) is an aviation system in which aircraft broadcast identity, position, altitude, and velocity on 1090 MHz using pulse-position modulation.
keywords: ADS-B, Mode S, 1090 MHz, aircraft tracking, pulse position modulation, CPR, Extended Squitter, DO-260
aka: [ADS-B, ADSB]
autolink: true
infobox:
  - { label: Type, value: Aviation surveillance broadcast }
  - { label: Standards body, value: ICAO / RTCA / EUROCAE }
  - { label: Frequency, value: 1090 MHz (also 978 MHz UAT) }
  - { label: Modulation, value: PPM (Mode S Extended Squitter) }
  - { label: Bit rate, value: 1 Mbps }
  - { label: Position coding, value: Compact Position Reporting (CPR) }
  - { label: GopherTrunk support, value: Decoded }
see_also: [ais, compact-position-reporting, cyclic-redundancy-check, icao]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Automatic Dependent Surveillance–Broadcast (Wikipedia)", url: https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast }
  - { title: "GopherTrunk ADS-B decoder", url: /adsb.html }
---

**ADS-B** (**Automatic Dependent Surveillance–Broadcast**) is an aviation
surveillance system in which aircraft continuously broadcast their **identity,
position, altitude, and velocity** on **1090 MHz** (and 978 MHz UAT in the U.S.). It
is the aeronautical counterpart to [AIS](/reference/ais/) and one of the most popular
SDR applications.

## Overview

ADS-B rides on Mode S Extended Squitter messages, modulated with **pulse-position
modulation** at 1 Mbps. Latitude/longitude are encoded with
[Compact Position Reporting](/reference/compact-position-reporting/), which resolves
to a precise position from a pair of even/odd messages.

## Technical characteristics

| Property | Value |
|----------|-------|
| Frequency | 1090 MHz (978 MHz UAT) |
| Modulation | PPM (Mode S) |
| Bit rate | 1 Mbps |
| Integrity | CRC-24 |
| Position | CPR even/odd encoding |

## History

Standardised by [ICAO](/reference/icao/) with MOPS from RTCA (DO-260) and EUROCAE
(ED-102); mandated for most controlled airspace through the 2010s–2020s.

## Deployment

Commercial and general aviation worldwide; widely received by hobbyists feeding
flight-tracking networks.

## Decoding it with GopherTrunk

GopherTrunk detects the 1090 MHz squitters in the magnitude domain, validates the
[CRC-24](/reference/cyclic-redundancy-check/), and decodes aircraft state. See the
[ADS-B decoder](/adsb.html) page.
