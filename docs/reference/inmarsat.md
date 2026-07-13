---
slug: inmarsat
title: Inmarsat
entry_type: protocol
category: satellite-gnss
description: "Inmarsat is a geostationary L-band satellite network carrying maritime and aeronautical data services such as STD-C and Classic Aero, receivable with an SDR and a patch antenna."
keywords: Inmarsat, L-band, geostationary satellite, STD-C, SafetyNET, EGC, Classic Aero, AERO, FleetBroadband, BGAN, satellite communications, SafetyNET
aka: [Inmarsat, INMARSAT]
autolink: true
infobox:
  - { label: Type, value: GEO satellite comms }
  - { label: Standards body, value: Inmarsat / IMO (STD-C) }
  - { label: Introduced, value: "1982" }
  - { label: Access, value: TDM/TDMA, FDMA }
  - { label: Channel spacing, value: "~1.5–1.6 GHz L-band" }
  - { label: Modulation, value: BPSK / QPSK / OQPSK }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [iridium, globalstar, orbcomm, bpsk, qpsk, patch-antenna, frequency-bands]
cite_urls:
  - https://en.wikipedia.org/wiki/Inmarsat
  - https://en.wikipedia.org/wiki/EGC
---

**Inmarsat** is a network of [geostationary](/reference/frequency-bands/) satellites
providing [L-band](/reference/frequency-bands/) mobile-satellite services — maritime,
aeronautical, and land — including the store-and-forward **STD-C** data service and the
**Classic Aero** aircraft link.[^wiki] Unlike the low-orbit
[Iridium](/reference/iridium/) and [Globalstar](/reference/globalstar/) constellations,
Inmarsat parks a handful of large satellites over the equator so a fixed terminal simply
points at a spot in the sky and stays connected.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A geostationary Inmarsat satellite over the equator relaying between a ship terminal and a coast earth station." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="inmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M20 150 Q230 175 440 150" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="230" y="170" text-anchor="middle" font-size="9" fill="currentColor">Earth (equator)</text>
  <circle cx="230" cy="30" r="10" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="230" y="18" text-anchor="middle" font-size="9" fill="currentColor">GEO satellite (~35 786 km)</text>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="70" y="135">ship / aircraft</text><text x="395" y="135">coast earth station</text></g>
  <line x1="80" y1="128" x2="222" y2="40" stroke="currentColor" marker-end="url(#inmar)"/>
  <line x1="238" y1="40" x2="388" y2="128" stroke="currentColor" marker-end="url(#inmar)"/>
</svg>
<figcaption>Inmarsat relays via geostationary satellites: a mobile terminal and a ground gateway both see the same fixed spacecraft.</figcaption>
</figure>

## Overview

Inmarsat began in 1979 as the intergovernmental International Maritime Satellite
Organization, chartered to give ships reliable distress and safety communications; the
first spacecraft entered service in 1982. It was privatised in 1999 and, after decades as
a standalone operator, was acquired by Viasat in 2023. Its services span several
generations of satellites and air interfaces, from the low-rate STD-C messaging terminal
to the wideband FleetBroadband and BGAN IP services.[^wiki]

## Technical characteristics

| Property | Value |
|----------|-------|
| Orbit | Geostationary (equatorial, ~35 786 km) |
| Band | L-band, ~1.5 GHz downlink / ~1.6 GHz uplink |
| Access | FDMA channels with TDM/TDMA framing |
| Modulation | BPSK (STD-C), QPSK/OQPSK (Aero, BGAN) |
| STD-C rate | 1200 symbols/s, ~600 bps effective |
| Services | STD-C, SafetyNET/EGC, Classic Aero, FleetBroadband, BGAN |

STD-C is a text-and-data teleprinter service that also carries the maritime
**SafetyNET** / Enhanced Group Call (EGC) broadcasts — navigational and meteorological
warnings addressed to ships in a geographic area.[^egc] Because these are open
[BPSK](/reference/bpsk/) broadcasts on a fixed downlink, they are a well-known
[SDR](/reference/software-defined-radio/) reception target.

## History

The organisation was created under an International Maritime Organization convention to
answer the safety-of-life-at-sea need that HF and marine VHF could not fully meet. Each
satellite generation added capacity and higher-rate services, moving from analogue voice
through the digital Aero and Mini-M terminals to today's IP-based FleetBroadband and
Global Xpress Ka-band system.[^wiki]

## Deployment

Inmarsat underpins the Global Maritime Distress and Safety System (GMDSS): ships carry
STD-C terminals to receive SafetyNET warnings and send distress alerts. Classic Aero and
SwiftBroadband serve aviation datalink and cabin connectivity, and land terminals cover
remote-area voice and data. The L-band downlinks are receivable at a fixed site with a
small **[patch antenna](/reference/patch-antenna/)** and a low-noise amplifier; hobbyists
routinely decode STD-C EGC traffic and Aero ACARS-style messages with free software.

## Decoding it with GopherTrunk

GopherTrunk is a terrestrial land-mobile trunking scanner (P25, DMR, TETRA, and
similar), so Inmarsat is out of scope for its decode chain — the framing, forward error
correction, and message formats are entirely different, and the signals sit in L-band
rather than the VHF/UHF land-mobile bands GopherTrunk tunes. Inmarsat is included here as
a reference point: it is the geostationary counterpart to the LEO systems
[Iridium](/reference/iridium/), [Globalstar](/reference/globalstar/), and
[Orbcomm](/reference/orbcomm/), and its open STD-C downlink is a good first
satellite-decoding project with a dedicated tool.

## Sources

[^wiki]: [Inmarsat](https://en.wikipedia.org/wiki/Inmarsat) — Wikipedia, for the operator's history, geostationary L-band services, and the STD-C, Aero, and BGAN air interfaces.
[^egc]: [Enhanced Group Call](https://en.wikipedia.org/wiki/EGC) — Wikipedia, for the SafetyNET/EGC maritime safety broadcast service carried over STD-C.
