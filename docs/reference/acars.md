---
slug: acars
title: ACARS
entry_type: protocol
category: aviation-marine
description: "ACARS (Aircraft Communications Addressing and Reporting System) is a datalink for short text messages between aircraft and ground over VHF, HF, and satcom links."
keywords: ACARS, Aircraft Communications Addressing and Reporting System, VHF datalink, MSK, 131.550 MHz, aircraft messaging, OOOI, ARINC, satcom, HFDL, VDL
aka: [ACARS]
autolink: true
infobox:
  - { label: Type, value: Aircraft datalink messaging }
  - { label: Standards body, value: "ARINC / AEEC" }
  - { label: Introduced, value: 1978 }
  - { label: Access, value: "VHF (131.x MHz), HF, satcom" }
  - { label: Modulation, value: "MSK, 2400 bps (Plain Old ACARS)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [minimum-shift-keying, vdl-mode-2, hfdl, ads-c, cpdlc, inmarsat]
cite_urls:
  - https://en.wikipedia.org/wiki/ACARS
  - https://en.wikipedia.org/wiki/Aircraft_Communications_Addressing_and_Reporting_System
---

**ACARS** (**Aircraft Communications Addressing and Reporting System**) is a datalink
for short, addressed text messages between aircraft and ground stations, originally
over **VHF** and now also over **HF** and **satellite**.[^wiki] It carries operational
traffic — position and status reports, weather requests, gate assignments, maintenance
data, and the automatic OOOI (Out/Off/On/In) timestamps — that would otherwise clog
voice frequencies. The classic VHF form ("Plain Old ACARS", POA) transmits at 2400 bps
using [minimum-shift keying](/reference/minimum-shift-keying/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An aircraft exchanges short addressed ACARS text blocks with a ground station over a VHF link, with parallel HF and satellite bearers shown." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="acarsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 60 l30 8 l-8 -14 m8 14 l14 -10" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="45" y="82" font-size="7.5" fill="currentColor">aircraft</text>
  <path d="M75 58 h150" stroke="currentColor" stroke-width="1.2" marker-end="url(#acarsar)"/>
  <path d="M225 70 h-150" stroke="currentColor" stroke-width="1.2" marker-end="url(#acarsar)"/>
  <text x="150" y="50" text-anchor="middle" font-size="7.5" fill="currentColor">VHF 131.x MHz · MSK 2400 bps</text>
  <rect x="230" y="48" width="80" height="30" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="270" y="67" text-anchor="middle" font-size="8" fill="currentColor">ground / DSP</text>
  <path d="M310 63 h100" stroke="currentColor" stroke-width="1.1" marker-end="url(#acarsar)"/>
  <text x="360" y="56" text-anchor="middle" font-size="7.5" fill="currentColor">airline ops</text>
  <text x="230" y="120" text-anchor="middle" font-size="8" fill="currentColor">bearers: VHF · HFDL · Inmarsat/Iridium satcom</text>
</svg>
<figcaption>ACARS exchanges short addressed text blocks between aircraft and ground, over VHF (MSK), HF, or satellite, then routes them to airline operations.</figcaption>
</figure>

## Overview

ACARS predates the modern IP aviation datalinks and remains ubiquitous. A message is a
short block of text addressed by aircraft registration and a label indicating its type.
The system is character-oriented and human-readable at heart, which is why hobbyists can
decode plain ACARS with a simple VHF receiver and software. Higher-rate successors —
[VDL Mode 2](/reference/vdl-mode-2/) on VHF and satcom links — carry the same
applications when more throughput is needed.

## Technical characteristics

| Property | Value |
|----------|-------|
| Primary VHF channel | 131.550 MHz (plus 130.025, 131.725, others) |
| VHF modulation | MSK, 2400 bps, over an AM carrier |
| Message size | Short text blocks (up to ~220 characters) |
| Addressing | Aircraft registration + label + block ID |
| Integrity | Block check character / CRC |
| Bearers | VHF (POA), VDL Mode 2, HFDL, Inmarsat/Iridium satcom |
| Applications | OOOI, position, weather, free text, maintenance, ADS-C, CPDLC |

Plain Old ACARS keys [minimum-shift keying](/reference/minimum-shift-keying/) onto an
AM carrier at 2400 bps — a constant-envelope scheme that is spectrum-thrifty and easy to
demodulate. Each transmission is a self-contained, acknowledged block; the ground network
routes it to the relevant airline or service provider.

## History

ARINC introduced ACARS in 1978 to automate the flight crew's manual position and status
reporting. It steadily absorbed new applications through the 1980s and 1990s and became
the transport for FANS applications such as [ADS-C](/reference/ads-c/) and
[CPDLC](/reference/cpdlc/) over oceanic satcom and HF, extending its reach far beyond
its VHF origins.

## Deployment

ACARS is carried by virtually every airline aircraft worldwide and is one of the most
popular aviation signals for SDR hobbyists precisely because the VHF form is unencrypted
and easy to receive. Datalink service providers (ARINC/SITA) operate the ground
networks. Modern aircraft use the higher-rate VDL Mode 2 and satcom bearers, but legacy
VHF ACARS remains in daily use.

## Decoding it with GopherTrunk

**Not decoded.** GopherTrunk is a land-mobile trunking scanner plus 1090 MHz
[ADS-B](/reference/ads-b/); it does not demodulate the aviation VHF ACARS channels or
the HF/satcom bearers. VHF ACARS is very approachable with a cheap SDR and dedicated
decoders, and this page is provided as honest context — the underlying modulation,
[MSK](/reference/minimum-shift-keying/), and the successor links
[VDL Mode 2](/reference/vdl-mode-2/) and [HFDL](/reference/hfdl/) have their own pages.

## Sources

[^wiki]: [ACARS](https://en.wikipedia.org/wiki/ACARS) — Wikipedia, for the ACARS addressed-messaging system, its 2400 bps MSK VHF physical layer, message structure, OOOI/operational applications, and VHF/HF/satcom bearers.
