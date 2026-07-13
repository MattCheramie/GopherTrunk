---
slug: fis-b
title: FIS-B
entry_type: protocol
category: aviation-marine
description: "FIS-B (Flight Information Service–Broadcast) is a free ground uplink on the 978 MHz UAT link that broadcasts weather, NOTAMs, and aeronautical information to aircraft."
keywords: FIS-B, Flight Information Service Broadcast, 978 MHz, UAT, weather uplink, NEXRAD, METAR, TAF, NOTAM, ADS-B In, cockpit weather
aka: [FIS-B, Flight Information Service-Broadcast]
autolink: true
infobox:
  - { label: Type, value: Ground-to-air weather/info uplink }
  - { label: Standards body, value: "RTCA DO-358 / FAA" }
  - { label: Frequency, value: 978 MHz (UAT) }
  - { label: Products, value: "NEXRAD, METAR, TAF, NOTAM, TFR" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [uat-978, tis-b, ads-b, frequency-shift-keying, reed-solomon-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Flight_Information_Services-Broadcast
  - https://en.wikipedia.org/wiki/Universal_Access_Transceiver
---

**FIS-B** (**Flight Information Service–Broadcast**) is a free ground-to-air uplink,
carried on the [UAT 978 MHz](/reference/uat-978/) link, that broadcasts graphical and
textual **weather and aeronautical information** — radar mosaics, METARs, TAFs, NOTAMs,
and airspace notices — to any aircraft with an [ADS-B](/reference/ads-b/) In
receiver.[^wiki] It is the weather counterpart to the traffic service
[TIS-B](/reference/tis-b/), and the pair are the FAA's incentive for equipping with UAT.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A ground station broadcasts layered FIS-B weather and notice products on 978 megahertz up to an aircraft cockpit display." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fisar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="40" y="60" width="90" height="30" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="85" y="79" text-anchor="middle" font-size="8" fill="currentColor">ground station</text>
  <g font-size="7.5" fill="currentColor">
    <rect x="160" y="40" width="70" height="14" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.9"/><text x="195" y="50" text-anchor="middle">NEXRAD</text>
    <rect x="160" y="60" width="70" height="14" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.9"/><text x="195" y="70" text-anchor="middle">METAR/TAF</text>
    <rect x="160" y="80" width="70" height="14" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.9"/><text x="195" y="90" text-anchor="middle">NOTAM/TFR</text>
  </g>
  <path d="M130 75 h28" stroke="currentColor" stroke-width="1.1" marker-end="url(#fisar)"/>
  <path d="M230 67 h150" stroke="currentColor" stroke-width="1.2" marker-end="url(#fisar)"/>
  <text x="305" y="60" text-anchor="middle" font-size="7.5" fill="currentColor">978 MHz FIS-B uplink</text>
  <path d="M395 63 l20 6 l-6 -12 m6 12 l10 -8" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="410" y="92" text-anchor="middle" font-size="7.5" fill="currentColor">cockpit display</text>
  <text x="230" y="122" text-anchor="middle" font-size="8" fill="currentColor">no subscription · US NAS</text>
</svg>
<figcaption>FIS-B broadcasts layered weather and notice products (NEXRAD, METAR/TAF, NOTAM/TFR) on 978 MHz to any listening cockpit.</figcaption>
</figure>

## Overview

FIS-B turns the ground-uplink half of each [UAT](/reference/uat-978/) frame into a
rolling weather broadcast. Because it is a *broadcast*, one transmission serves every
aircraft in a station's coverage, and there is no request/response and no per-user cost.
Products are transmitted on a cycle, with time-critical items (NEXRAD regional radar,
special-use airspace) repeated frequently and slowly changing bulk products less often.

## Technical characteristics

| Property | Value |
|----------|-------|
| Direction | Ground-to-air (broadcast) |
| Frequency | 978 MHz (UAT link only) |
| Carrier products | NEXRAD (regional + CONUS), METAR, TAF, PIREP, winds/temps aloft |
| Notices | NOTAM, TFR, SUA status, AIRMET/SIGMET |
| Transport | UAT ground-uplink segments, Reed–Solomon coded |
| Update cadence | Product-dependent (minutes) |
| Standard | RTCA DO-358 |

Each product is segmented into UAT ground frames, protected by a
[Reed–Solomon code](/reference/reed-solomon-code/), and reassembled by the receiver.
NEXRAD imagery is delivered as a coarse national mosaic plus a finer regional block
around the aircraft, trading resolution against the limited link bandwidth.

## History

FIS-B was specified by the FAA to accompany the UAT rollout, with the message and
product set standardised in RTCA DO-358. It replaced earlier subscription-based
satellite weather services for many general-aviation pilots by making a broad product
set free at the point of use.

## Deployment

FIS-B is a **U.S.-only** service tied to the 978 MHz UAT infrastructure; it is not
available on the worldwide 1090 MHz link. Pilots receive it through portable or panel
ADS-B In units, typically alongside [TIS-B](/reference/tis-b/) traffic. Because
products are broadcast on a cycle and can be several minutes old, FIS-B weather is
advisory — suitable for strategic planning, not tactical thunderstorm penetration.

## Decoding it with GopherTrunk

**Not decoded.** FIS-B lives entirely on the UAT 978 MHz link, which GopherTrunk does
not tune or demodulate — GopherTrunk's aviation decoding is limited to 1090 MHz
[ADS-B](/reference/ads-b/). FIS-B is receivable with a wideband SDR and open UAT
decoders, but reconstructing its weather products is outside GopherTrunk's scope. See
[UAT (978 MHz)](/reference/uat-978/) for the underlying link.

## Sources

[^wiki]: [Flight Information Services-Broadcast](https://en.wikipedia.org/wiki/Flight_Information_Services-Broadcast) — Wikipedia, for the FIS-B free weather/aeronautical uplink, its NEXRAD/METAR/TAF/NOTAM product set, and delivery on the 978 MHz UAT ground-uplink link.
