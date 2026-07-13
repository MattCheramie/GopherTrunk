---
slug: orbcomm
title: Orbcomm
entry_type: protocol
category: satellite-gnss
description: "Orbcomm is a low-earth-orbit little-LEO satellite network carrying short machine-to-machine data messages on VHF near 137–150 MHz, receivable with an ordinary SDR."
keywords: Orbcomm, little LEO, VHF satellite, M2M, IoT, asset tracking, SDPSK, store and forward, 137 MHz, satellite data, telemetry
aka: [Orbcomm, ORBCOMM]
autolink: true
infobox:
  - { label: Type, value: LEO data satellite (little-LEO) }
  - { label: Standards body, value: Orbcomm Inc. }
  - { label: Introduced, value: "1998" }
  - { label: Access, value: FDMA/TDMA, store-and-forward }
  - { label: Channel spacing, value: "137–138 MHz down / 148–150 MHz up" }
  - { label: Modulation, value: SDPSK (~4800 bps) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [iridium, globalstar, inmarsat, differential-decoding, ais, noaa-apt]
cite_urls:
  - https://en.wikipedia.org/wiki/Orbcomm
  - https://en.wikipedia.org/wiki/Symmetric_differential_phase-shift_keying
---

**Orbcomm** is a low-earth-orbit satellite network built for short **machine-to-machine
(M2M)** data messages — telemetry, asset tracking, and remote monitoring — rather than
voice. It is a *little-LEO* system: its satellites operate in the VHF band, with a
downlink near 137–138 MHz and an uplink near 148–150 MHz, using **store-and-forward**
relay so a satellite can collect a message over a remote area and drop it to a gateway
minutes later.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An Orbcomm VHF satellite stores a telemetry message from a remote terminal and forwards it to a ground gateway on a later pass." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="orbc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="20" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.4"/>
  <circle cx="120" cy="45" r="8" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <circle cx="340" cy="45" r="8" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <path d="M132 45 Q230 25 328 45" fill="none" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#orbc)"/>
  <text x="230" y="20" text-anchor="middle" font-size="9" fill="currentColor">store → forward as orbit moves</text>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="70" y="158">remote terminal (uplink)</text><text x="390" y="158">gateway (137 MHz down)</text></g>
  <line x1="80" y1="128" x2="114" y2="54" stroke="currentColor" marker-end="url(#orbc)"/>
  <line x1="346" y1="54" x2="385" y2="128" stroke="currentColor" marker-end="url(#orbc)"/>
</svg>
<figcaption>Orbcomm satellites store a message picked up over a remote area and forward it to a ground gateway on a later part of the orbit.</figcaption>
</figure>

## Overview

Orbcomm Inc. was founded in 1993 and began commercial service in 1998 with a
constellation of small VHF satellites. The design deliberately chose VHF and tiny
satellites to keep both the space and ground segments cheap: a subscriber terminal is a
low-power modem the size of a paperback, and the low frequency means a short whip antenna
suffices.[^wiki] The trade-off is throughput — the system moves short packets, not
streams.

## Technical characteristics

| Property | Value |
|----------|-------|
| Orbit | LEO, ~715 km, multiple planes |
| Downlink | 137–138 MHz VHF |
| Uplink | 148–150 MHz VHF |
| Modulation | Symmetric differential PSK (SDPSK) |
| Data rate | ~4800 bps downlink, ~2400 bps uplink |
| Access | FDMA channels, TDMA slots, store-and-forward |

The downlink uses **symmetric differential PSK**, a differentially encoded phase
modulation whose [differential decoding](/reference/differential-decoding/) removes the
need to resolve an absolute carrier phase — convenient when the satellite is racing
overhead with a large [Doppler shift](/reference/doppler-shift/).[^sdpsk]

## History

The first operational satellites launched in the mid-1990s, with full commercial service
from 1998. Over time Orbcomm expanded through second-generation (OG2) satellites and
acquisitions of other M2M providers, and some of its satellites carry
[AIS](/reference/ais/) receivers to collect ship position reports from space for maritime
tracking.[^wiki]

## Deployment

Orbcomm serves fleet and container tracking, oil-and-gas and utility telemetry,
heavy-equipment monitoring, and other industrial IoT where cellular coverage is absent.
Because the downlink sits at 137–138 MHz — the same VHF slice used by the
[NOAA APT](/reference/noaa-apt/) weather satellites — it is easily received with an
ordinary VHF [SDR](/reference/software-defined-radio/) and a modest antenna, and its
packets have been demodulated by hobbyists with open-source tools.

## Decoding it with GopherTrunk

GopherTrunk does not decode Orbcomm. Although the 137 MHz downlink falls in a band a VHF
SDR can hear, Orbcomm's SDPSK modulation, satellite framing, and store-and-forward packet
formats are unrelated to the terrestrial land-mobile trunking protocols GopherTrunk
targets. It appears here as the VHF, data-only member of the LEO family alongside
[Iridium](/reference/iridium/) and [Globalstar](/reference/globalstar/).

## Sources

[^wiki]: [Orbcomm](https://en.wikipedia.org/wiki/Orbcomm) — Wikipedia, for the little-LEO constellation, VHF frequency plan, store-and-forward M2M service, and satellite AIS payloads.
[^sdpsk]: [Symmetric differential phase-shift keying](https://en.wikipedia.org/wiki/Symmetric_differential_phase-shift_keying) — Wikipedia, for the differential PSK scheme used on the Orbcomm downlink.
