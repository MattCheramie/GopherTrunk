---
slug: cospas-sarsat
title: COSPAS-SARSAT
entry_type: protocol
category: satellite-gnss
description: "COSPAS-SARSAT is the international satellite system that detects and locates 406 MHz emergency distress beacons, relaying alerts to search-and-rescue authorities worldwide."
keywords: COSPAS-SARSAT, 406 MHz beacon, distress beacon, EPIRB, PLB, ELT, search and rescue, SARSAT, MEOSAR, LEOSAR, GEOSAR, multilateration, Doppler location
aka: [COSPAS-SARSAT, Cospas-Sarsat]
autolink: true
infobox:
  - { label: Type, value: Satellite distress-alerting system }
  - { label: Operator, value: "International (US, Russia, Canada, France, +)" }
  - { label: Introduced, value: "1982 (first rescue)" }
  - { label: Access, value: "406.0–406.1 MHz uplink" }
  - { label: Modulation, value: "Phase-modulated, biphase-L (BPSK)" }
  - { label: Locating, value: "Doppler + GNSS-encoded position" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [epirb-406, multilateration, doppler-shift, gnss]
cite_urls:
  - https://en.wikipedia.org/wiki/Cospas-Sarsat
  - https://cospas-sarsat.int/en/
---

**COSPAS-SARSAT** is the international, humanitarian satellite system that detects and
locates activated emergency distress beacons and relays the alert to search-and-rescue
authorities.[^wiki] It listens on the worldwide distress frequency of **406 MHz** for
the short digital bursts sent by an [EPIRB, PLB, or ELT](/reference/epirb-406/), then
pinpoints the beacon using a combination of satellite
[multilateration](/reference/multilateration/), [Doppler](/reference/doppler-shift/)
processing, and any position the beacon itself encodes from
[GNSS](/reference/gnss/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A 406 megahertz distress beacon is heard by satellites which relay the burst to a ground station and mission control that dispatches rescue." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="csar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="120" cy="30" r="7" fill="currentColor"/><text x="120" y="18" text-anchor="middle" font-size="8" fill="currentColor">LEO/MEO/GEO</text>
  <path d="M60 165 L114 40" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#csar)"/>
  <text x="55" y="180" font-size="9" fill="currentColor">406 MHz beacon</text>
  <path d="M126 38 L300 150" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#csar)"/>
  <rect x="298" y="150" width="30" height="18" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="313" y="184" text-anchor="middle" font-size="8" fill="currentColor">LUT ground station</text>
  <path d="M330 158 L400 150" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#csar)"/>
  <text x="415" y="150" font-size="9" fill="currentColor">SAR</text>
</svg>
<figcaption>A beacon burst is relayed by satellite to a Local User Terminal, which forwards a located alert to rescue services.</figcaption>
</figure>

## Overview

COSPAS-SARSAT is not a navigation constellation but a distress-alerting overlay that
rides on several satellite systems. A beacon in an emergency transmits a five-watt
burst on 406 MHz roughly every 50 seconds. Instruments aboard low-Earth-orbit
(LEOSAR), geostationary (GEOSAR), and — since the 2010s — GNSS medium-Earth-orbit
satellites (MEOSAR) receive the burst and forward it to ground stations called Local
User Terminals, which decode the beacon's identity and compute its location before
passing the alert to national rescue coordination centres.

## Technical characteristics

| Property | Value |
|----------|-------|
| Uplink | 406.0–406.1 MHz |
| Burst | ~0.5 s digital message, ~5 W, every ~50 s |
| Modulation | Phase-modulated biphase-L (BPSK-family) |
| Space segment | LEOSAR + GEOSAR + MEOSAR |
| Location methods | Doppler, [multilateration](/reference/multilateration/) (MEOSAR), encoded GNSS position |
| Legacy analog | 121.5 MHz homing (no longer satellite-monitored) |

The digital 406 MHz message carries a unique 15-hex-character beacon identity, the
country of registration, and often a GNSS position embedded by the beacon. LEOSAR
satellites locate a beacon by measuring the [Doppler shift](/reference/doppler-shift/)
of the 406 MHz carrier as they pass overhead; the modern MEOSAR payloads on GPS,
Galileo, and GLONASS satellites instead locate it almost instantly by
[multilateration](/reference/multilateration/) from time-of-arrival differences across
many satellites. The older analog 121.5 MHz beacons are no longer satellite-monitored
and remain useful only for short-range homing.

## History

COSPAS-SARSAT was created by the United States, Soviet Union, Canada, and France, with
the first satellite launched and the first lives saved in 1982. It has since been
credited with tens of thousands of rescues. The transition from analog 121.5 MHz to
digital 406 MHz beacons (completed for satellite monitoring in 2009) dramatically cut
false alerts and improved location accuracy.

## Deployment

The system is operated by an intergovernmental organization with dozens of member
states and covers the entire globe. Its beacons — [EPIRBs](/reference/epirb-406/) on
ships, ELTs in aircraft, and PLBs carried by individuals — are mandated for many
maritime and aviation uses.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode COSPAS-SARSAT. Although the 406 MHz uplink sits in the
UHF range a scanner can tune, decoding distress beacons is a specialized, regulated
task and is out of scope for GopherTrunk's land-mobile trunking focus. Because the
beacon signal is a terrestrial 406 MHz burst rather than a weak L-band satellite
signal, it is technically far more accessible to a general software-defined radio than
[GNSS](/reference/gnss/) is — hobby decoders for the beacon message format exist — but
deliberately transmitting or interfering with 406 MHz is illegal, and GopherTrunk
provides no support for it.

## Sources

[^wiki]: [Cospas-Sarsat](https://en.wikipedia.org/wiki/Cospas-Sarsat) — Wikipedia, for the system's founding, the 406 MHz beacon burst, the LEOSAR/GEOSAR/MEOSAR segments, Doppler and multilateration location, and the retirement of 121.5 MHz satellite monitoring.
