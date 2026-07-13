---
slug: gps-gnss
title: Global Positioning System (GPS)
entry_type: protocol
category: satellite-gnss
description: "GPS is the US satellite navigation system whose L1 C/A signal spreads a 1.023 Mcps Gold code by CDMA so receivers trilaterate position from four or more satellites."
keywords: GPS, Global Positioning System, Navstar, L1 C/A, Gold code, CDMA, DSSS, trilateration, pseudorange, GNSS, ephemeris, coarse acquisition
aka: [GPS, Navstar, Navstar GPS, Global Positioning System]
autolink: true
infobox:
  - { label: Type, value: Satellite navigation (GNSS) }
  - { label: Operator, value: US Space Force }
  - { label: Introduced, value: "1978 (full service 1995)" }
  - { label: Access, value: "CDMA (Gold codes)" }
  - { label: Band, value: "L1 1575.42 MHz (+ L2, L5)" }
  - { label: Modulation, value: BPSK-spread DSSS }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [gnss, gold-code, cdma, multilateration, doppler-shift, gps-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Global_Positioning_System
  - https://www.gps.gov/technical/icwg/IS-GPS-200N.pdf
---

**GPS** (**Global Positioning System**, originally Navstar) is the United States'
satellite navigation system and the most widely used member of the
[GNSS](/reference/gnss/) family.[^wiki] Its civilian **L1 C/A** signal, centred at
1575.42 MHz, spreads a navigation message with a satellite-unique
[Gold code](/reference/gold-code/) so that all satellites share one frequency by
[CDMA](/reference/cdma/), and a receiver recovers position by
[multilaterating](/reference/multilateration/) ranges to four or more of them.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A slow 50 bit-per-second navigation message is spread by a fast 1.023 megachip-per-second Gold code before BPSK modulation." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="20" y="30" font-size="10" fill="currentColor">nav data (50 bps)</text>
  <path d="M20 45 h120 v20 h60 v-20 h60" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="20" y="95" font-size="10" fill="currentColor">Gold code (1.023 Mcps)</text>
  <path d="M20 110 h12 v14 h12 v-14 h12 v14 h12 v-14 h12 v14 h12 v-14 h12 v14 h12 v-14 h12 v14 h12 v-14 h12 v14 h12 v-14 h12" fill="none" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7"/>
  <line x1="360" y1="60" x2="430" y2="60" stroke="currentColor" marker-end="url(#gpar)"/>
  <text x="395" y="52" text-anchor="middle" font-size="9" fill="currentColor">BPSK</text>
  <text x="395" y="78" text-anchor="middle" font-size="9" fill="currentColor">on L1</text>
</svg>
<figcaption>Each GPS satellite XORs a slow navigation message with its own fast Gold code, spreading the signal below the noise floor.</figcaption>
</figure>

## Overview

GPS is a constellation of at least 24 satellites in medium Earth orbit, arranged so
that four or more are visible from almost any point on Earth at any time. Each
satellite carries atomic clocks and broadcasts its precise time and orbit. A receiver
correlates the incoming signal against the known Gold code for each satellite,
measures the code phase to derive a *pseudorange*, and solves for its own three
coordinates plus its clock error — four unknowns needing four satellites.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [CDMA](/reference/cdma/) via 1023-chip Gold codes |
| Carrier (civil) | L1 = 1575.42 MHz |
| Spreading rate | 1.023 Mcps (C/A code) |
| Nav data rate | 50 bit/s |
| Modulation | [BPSK](/reference/cdma/)-modulated direct-sequence spread spectrum |
| Satellites | ≥24 in six MEO planes, ~20,200 km |
| Doppler range | ±5 kHz at L1 from satellite motion |

The 1023-chip C/A [Gold code](/reference/gold-code/) repeats every millisecond and
has low cross-correlation between satellites, which is what lets many satellites share
L1 without jamming each other. Because the received power is far below the thermal
[noise floor](/reference/multilateration/), the receiver's correlation gain (about
43 dB) is what pulls the signal up to a usable level.

## History

GPS was initiated by the US Department of Defense in the 1970s; the first Block I
satellite launched in 1978 and full operational capability was declared in 1995.
Selective Availability, which deliberately degraded civilian accuracy, was switched
off in 2000, and modernized signals (L2C, L5, and L1C) have since been added for
better civilian and safety-of-life performance.

## Deployment

GPS underpins navigation, surveying, timing, and synchronization worldwide. Beyond
the obvious phone and vehicle navigation, it disciplines telecom networks, power
grids, and financial timestamps, and provides the reference in a
[GPSDO](/reference/gpsdo/) used to stabilize radio equipment. A commodity
[GPS receiver](/reference/gps-receiver/) module performs the acquisition and tracking
and outputs position plus a one-pulse-per-second tick.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode GPS. It is a land-mobile trunking scanner focused on
VHF/UHF voice protocols, and satellite navigation is outside that scope. GPS is,
however, a favourite software-defined-radio target for other tools: the signal sits
below the noise floor, so a general-purpose SDR needs an active L-band antenna, a
low-noise amplifier, and software that performs a two-dimensional search over code
phase and [Doppler shift](/reference/doppler-shift/) before it can lock. Open-source
receivers such as GNSS-SDR demonstrate the full chain. For GopherTrunk, GPS is
relevant only as an external timing source, not as traffic on the air it decodes.

## Sources

[^wiki]: [Global Positioning System](https://en.wikipedia.org/wiki/Global_Positioning_System) — Wikipedia, for GPS history, the L1 C/A signal structure, Gold-code CDMA, the 50 bit/s navigation message, and the constellation geometry.
