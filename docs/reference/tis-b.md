---
slug: tis-b
title: TIS-B
entry_type: protocol
category: aviation-marine
description: "TIS-B (Traffic Information Service–Broadcast) is a ground-based uplink that rebroadcasts radar and ADS-B traffic to aircraft so ADS-B In receivers see non-equipped targets."
keywords: TIS-B, Traffic Information Service Broadcast, ADS-B In, traffic uplink, 978 MHz, 1090 MHz, ground station, SSR rebroadcast, ADS-R
aka: [TIS-B, Traffic Information Service-Broadcast]
autolink: true
infobox:
  - { label: Type, value: Ground-to-air traffic uplink }
  - { label: Standards body, value: "RTCA / FAA" }
  - { label: Frequency, value: 978 MHz (UAT) and 1090 MHz }
  - { label: Source, value: SSR, multilateration, ADS-B }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ads-b, uat-978, fis-b, mode-s, mode-ac, multilateration]
cite_urls:
  - https://en.wikipedia.org/wiki/Traffic_information_service_%E2%80%93_broadcast
  - https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast
---

**TIS-B** (**Traffic Information Service–Broadcast**) is a ground-based service that
uplinks a synthesized traffic picture to aircraft, so that an
[ADS-B](/reference/ads-b/) In receiver can see targets that are **not** broadcasting
ADS-B themselves.[^wiki] Ground stations fuse secondary-radar,
[multilateration](/reference/multilateration/), and ADS-B tracks and rebroadcast them
on both the [UAT 978 MHz](/reference/uat-978/) and 1090 MHz links, closing the gap
during the transition to universal ADS-B equippage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A ground station fuses radar and ADS-B tracks and rebroadcasts them as TIS-B so an equipped aircraft can see a non-equipped aircraft." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tisar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="70" y="30" text-anchor="middle" font-size="8" fill="currentColor">radar / MLAT</text>
  <path d="M70 34 v12 m-6 0 l6 -8 l6 8" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <rect x="180" y="55" width="100" height="30" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="74" text-anchor="middle" font-size="8" fill="currentColor">ground station</text>
  <path d="M100 46 l70 20" stroke="currentColor" stroke-width="1.1" marker-end="url(#tisar)"/>
  <text x="120" y="52" font-size="7" fill="currentColor">non-equipped track</text>
  <path d="M280 70 h100" stroke="currentColor" stroke-width="1.2" marker-end="url(#tisar)"/>
  <text x="330" y="63" text-anchor="middle" font-size="7.5" fill="currentColor">TIS-B uplink</text>
  <path d="M395 66 l20 6 l-6 -12 m6 12 l10 -8" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="410" y="95" text-anchor="middle" font-size="7.5" fill="currentColor">ADS-B In aircraft</text>
  <text x="230" y="120" text-anchor="middle" font-size="8" fill="currentColor">978 MHz + 1090 MHz</text>
</svg>
<figcaption>TIS-B: a ground station rebroadcasts radar/ADS-B tracks so an ADS-B In aircraft sees traffic that is not itself ADS-B equipped.</figcaption>
</figure>

## Overview

The value of ADS-B In depends on other aircraft broadcasting ADS-B Out. During the
long equippage transition, and permanently for aircraft that never equip, many targets
are invisible to a purely airborne picture. TIS-B fills that hole: FAA ground stations
watch their surveillance sources, and for each aircraft that is receiving ADS-B, they
uplink the positions of nearby traffic the aircraft would otherwise miss.

## Technical characteristics

| Property | Value |
|----------|-------|
| Direction | Ground-to-air (uplink) |
| Frequencies | 978 MHz (UAT) and 1090 MHz |
| Source data | SSR, MLAT, ADS-B fusion |
| Client-tailored | Uplink keyed to a client aircraft's position |
| Related service | ADS-R (rebroadcasts across the two links) |
| Standard | RTCA MOPS (DO-260/DO-282 families) |

TIS-B is *client-based*: the ground station only advertises traffic within a service
volume around an aircraft it knows is listening, which conserves link bandwidth. A
closely related function, ADS-R (ADS-B Rebroadcast), forwards genuine ADS-B reports
between the 978 and 1090 MHz links so a UAT-only aircraft and a 1090ES-only aircraft
can still see each other.

## History

TIS-B grew out of the earlier addressed Traffic Information Service (TIS) delivered over
[Mode S](/reference/mode-s/), reworked as a broadcast service for the ADS-B era and
standardised by RTCA alongside ADS-B. It is a core element of the FAA's NextGen
surveillance architecture.

## Deployment

TIS-B is operational across the U.S. ground-station network and is a primary reason
general-aviation pilots value ADS-B In: combined with [FIS-B](/reference/fis-b/)
weather on the same [UAT](/reference/uat-978/) link, it delivers a near-complete
traffic-and-weather picture at no subscription cost. Note that TIS-B tracks can lag or
drop targets in radar-only coverage, so it supplements rather than replaces see-and-avoid.

## Decoding it with GopherTrunk

**Not decoded.** TIS-B is carried inside the UAT and 1090ES data streams as
ground-originated traffic reports. GopherTrunk decodes 1090 MHz airborne ADS-B
squitters but does not parse the ground-uplink TIS-B service, which on 978 MHz rides
the [UAT](/reference/uat-978/) link GopherTrunk does not tune. It is receivable with a
wideband SDR and dedicated UAT tooling, but sits outside GopherTrunk's land-mobile and
airborne-ADS-B focus.

## Sources

[^wiki]: [Traffic information service – broadcast](https://en.wikipedia.org/wiki/Traffic_information_service_%E2%80%93_broadcast) — Wikipedia, for the TIS-B ground-uplink service, its radar/ADS-B source fusion, client-based traffic advertisement, and delivery on the 978 and 1090 MHz links.
