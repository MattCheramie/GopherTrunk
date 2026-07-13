---
slug: ads-c
title: ADS-C
entry_type: protocol
category: aviation-marine
description: "ADS-C (Automatic Dependent Surveillance–Contract) is an addressed aviation surveillance service in which aircraft report position to ATC over satcom or HF datalink under a contract."
keywords: ADS-C, Automatic Dependent Surveillance Contract, FANS, oceanic surveillance, CPDLC, satcom, ACARS, position report, contract surveillance, Inmarsat
aka: [ADS-C, ADS-Contract]
autolink: true
infobox:
  - { label: Type, value: Addressed surveillance (contract) }
  - { label: Standards body, value: "ICAO / ARINC / RTCA" }
  - { label: Bearer, value: "ACARS over satcom / HF / VHF" }
  - { label: Access, value: Point-to-point (addressed) }
  - { label: Partner service, value: CPDLC }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ads-b, acars, cpdlc, hfdl, inmarsat, icao]
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Contract
  - https://en.wikipedia.org/wiki/Future_Air_Navigation_System
---

**ADS-C** (**Automatic Dependent Surveillance–Contract**) is an addressed,
point-to-point aviation surveillance service in which an aircraft sends position and
intent reports to an air-traffic control centre under a negotiated **contract**,
carried over [ACARS](/reference/acars/) via satellite, HF, or VHF datalink.[^wiki]
Unlike the omnidirectional broadcast of [ADS-B](/reference/ads-b/), ADS-C is a private
conversation between one aircraft and one ATC unit, which makes it the workhorse for
**oceanic and remote airspace** beyond radar and ADS-B ground coverage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An oceanic aircraft sends addressed ADS-C position reports up through a satellite down to an air traffic control center under a reporting contract." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="adscar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 100 l30 8 l-8 -14 m8 14 l14 -10" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="45" y="122" font-size="7.5" fill="currentColor">aircraft</text>
  <circle cx="230" cy="30" r="10" fill="none" stroke="currentColor" stroke-width="1.4"/><path d="M215 20 l-8 -6 m38 6 l8 -6" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="16" text-anchor="middle" font-size="7.5" fill="currentColor">satcom</text>
  <path d="M70 96 L218 38" stroke="currentColor" stroke-width="1.1" marker-end="url(#adscar)"/>
  <path d="M242 38 L390 96" stroke="currentColor" stroke-width="1.1" marker-end="url(#adscar)"/>
  <rect x="360" y="98" width="70" height="26" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="395" y="115" text-anchor="middle" font-size="8" fill="currentColor">ATC center</text>
  <text x="230" y="140" text-anchor="middle" font-size="8" fill="currentColor">periodic / event / demand contract reports</text>
</svg>
<figcaption>ADS-C: an oceanic aircraft delivers addressed position reports to an ATC centre over satcom under a periodic, event, or on-demand contract.</figcaption>
</figure>

## Overview

ADS-C is one half of the FANS (Future Air Navigation System) toolset; its companion is
[CPDLC](/reference/cpdlc/) controller–pilot text messaging. A ground ATC system
establishes a *contract* with the aircraft's avionics specifying what to report and
when. The aircraft then autonomously sends reports drawn from its own navigation system
(GNSS/INS) — hence "dependent" — without further prompting, until the contract is
modified or cancelled.

## Technical characteristics

| Property | Value |
|----------|-------|
| Model | Addressed request/contract, not broadcast |
| Contract types | Periodic, event, and on-demand |
| Report content | Position, altitude, time, ground vector, intent/waypoints |
| Bearer | ACARS over Inmarsat/Iridium satcom, HFDL, or VDL |
| Position source | Aircraft FMS (GNSS/INS) |
| Companion | CPDLC (clearances and text) |
| Standards | ICAO FANS-1/A, ARINC 745/622 |

Three contract types coexist: **periodic** (report every N minutes), **event**
(report when crossing a waypoint, changing level, or deviating), and **on-demand**
(a single report on request). Because reports are addressed and acknowledged over
[ACARS](/reference/acars/), ADS-C tolerates the long latency and low throughput of
oceanic satcom and HF links far better than the once-per-second broadcast of ADS-B.

## History

ADS-C emerged from the FANS-1/A programme in the 1990s, led by Boeing and Airbus with
ICAO, to bring datalink surveillance to oceanic regions that radar could never cover.
It let controllers reduce the large procedural separation minima previously required
over the oceans, increasing capacity on busy tracks such as the North Atlantic.

## Deployment

ADS-C is standard equipment on long-haul transport aircraft and is used by oceanic and
remote-area control centres worldwide, typically over [Inmarsat](/reference/inmarsat/)
or Iridium satcom, with [HFDL](/reference/hfdl/) as an HF fallback in polar regions.
It complements rather than replaces ADS-B: ADS-B for surveillance where ground
receivers exist, ADS-C for the airspace where they do not.

## Decoding it with GopherTrunk

**Not decoded.** ADS-C is an addressed application riding inside
[ACARS](/reference/acars/) over satellite and HF bearers, not a broadcast VHF/UHF
signal in GopherTrunk's land-mobile or 1090 MHz ADS-B scope. Enthusiasts can observe
ACARS-borne ADS-C with satcom or [HFDL](/reference/hfdl/) SDR setups and dedicated
decoders, but reconstructing the contract application is outside GopherTrunk's remit.

## Sources

[^wiki]: [Automatic Dependent Surveillance–Contract](https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Contract) — Wikipedia, for the ADS-C contract model, periodic/event/on-demand reports, ACARS/satcom/HF bearers, and its FANS oceanic-surveillance role alongside CPDLC.
