---
slug: iridium
title: Iridium
entry_type: protocol
category: satellite-gnss
description: "Iridium is a low-Earth-orbit satellite constellation providing global voice and data via cross-linked satellites, using QPSK bursts in L-band with large Doppler shifts."
keywords: Iridium, LEO constellation, satellite phone, satphone, cross-link, L-band, QPSK, Doppler shift, Iridium NEXT, TDMA FDMA, SBD, satellite communications
aka: [Iridium, Iridium NEXT]
autolink: true
infobox:
  - { label: Type, value: LEO satellite comms constellation }
  - { label: Operator, value: Iridium Communications }
  - { label: Introduced, value: "1998 (NEXT from 2017)" }
  - { label: Access, value: "TDMA/FDMA, 1616–1626.5 MHz" }
  - { label: Modulation, value: "DE-QPSK (burst)" }
  - { label: Orbit, value: "780 km LEO, 66 satellites" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [qpsk, doppler-shift, gnss, multilateration]
cite_urls:
  - https://en.wikipedia.org/wiki/Iridium_satellite_constellation
  - https://www.iridium.com/
---

**Iridium** is a satellite communications constellation that provides voice and data
coverage over the entire surface of the Earth, including the poles, using 66 active
satellites in low Earth orbit.[^wiki] Unlike a navigation [GNSS](/reference/gnss/),
Iridium is a two-way communications network: its satellites relay
[QPSK](/reference/qpsk/) bursts to and from handsets in the L-band and pass traffic
between one another over inter-satellite cross-links, so a call can transit the whole
constellation without touching the ground.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="Iridium satellites in low orbit relay traffic to a handset and to each other over inter-satellite cross-links, with fast motion causing Doppler shift." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="irar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M20 130 Q230 20 440 130" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <circle cx="130" cy="70" r="6" fill="currentColor"/><circle cx="230" cy="45" r="6" fill="currentColor"/><circle cx="330" cy="70" r="6" fill="currentColor"/>
  <line x1="136" y1="70" x2="224" y2="47" stroke="currentColor" stroke-dasharray="3 3"/>
  <line x1="236" y1="47" x2="324" y2="70" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="230" y="30" text-anchor="middle" font-size="8" fill="currentColor">cross-links</text>
  <path d="M200 158 L128 78" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#irar)"/>
  <text x="185" y="172" font-size="9" fill="currentColor">handset (L-band)</text>
  <text x="360" y="60" font-size="9" fill="currentColor">→ moving fast</text>
</svg>
<figcaption>Iridium's 66 LEO satellites relay handset traffic and hand it between satellites over inter-satellite cross-links.</figcaption>
</figure>

## Overview

Iridium's 66 operational satellites (plus in-orbit spares) fly in six polar orbital
planes at about 780 km — far lower than the ~20,000 km medium-Earth orbits of the
navigation constellations. That low altitude means each satellite races across the sky
in minutes, so the network continuously hands a call from one satellite to the next,
and the ground segment needs only a few gateways because most switching happens in
space. Services range from satellite telephony to low-rate Short Burst Data used by
IoT trackers and maritime devices.

## Technical characteristics

| Property | Value |
|----------|-------|
| Orbit | 780 km LEO, six planes, 66 active satellites |
| Band | L-band 1616–1626.5 MHz (user links) |
| Access | Combined [FDMA](/reference/fdma/) and TDMA, ~50 ms frames |
| Modulation | Differentially-encoded [QPSK](/reference/qpsk/), bursty |
| Cross-links | Ka-band inter-satellite links |
| Doppler | Tens of kHz swing per pass |

The low orbit produces a large [Doppler shift](/reference/doppler-shift/) — the
carrier can swing by tens of kilohertz as a satellite rises and sets — which both the
handset and any receiver must track. Iridium's frames combine time- and
frequency-division access, and its downlink bursts (including the periodic ring-alert
and broadcast channels) use differential QPSK so a receiver need not perfectly recover
absolute carrier phase.

## History

The original Iridium system, conceived at Motorola in the late 1980s, launched
its constellation in 1997–1998. The operating company went bankrupt in 2000 but the
network was rescued and continued in service. A full replacement fleet, **Iridium
NEXT**, was launched between 2017 and 2019, adding higher data rates and hosted
payloads (including an aircraft-tracking ADS-B receiver).

## Deployment

Iridium is used for satellite phones, maritime and aviation communications, remote
IoT telemetry, and government services, valued for its true pole-to-pole coverage that
geostationary systems cannot match. Its Short Burst Data service underpins many remote
asset-tracking and safety devices.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode Iridium. It is a land-mobile trunking scanner, and
satellite communications are out of scope. Iridium is, however, a notable
software-defined-radio subject: its ~1.6 GHz downlink is strong enough that hobbyists
with an L-band antenna and tools such as gr-iridium and iridium-toolkit can capture
and analyze the unencrypted signalling bursts, contending mainly with the fast pace of
satellites and the large [Doppler shift](/reference/doppler-shift/). None of that is
part of GopherTrunk; its only connection to satellites is using a
[GNSS](/reference/gnss/) receiver as an external timing reference.

## Sources

[^wiki]: [Iridium satellite constellation](https://en.wikipedia.org/wiki/Iridium_satellite_constellation) — Wikipedia, for the 66-satellite LEO constellation, the polar orbits and cross-links, the L-band QPSK air interface, and the Iridium NEXT replacement fleet.
