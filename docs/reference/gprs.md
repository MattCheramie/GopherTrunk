---
slug: gprs
title: GPRS (2.5G)
entry_type: protocol
category: cellular
description: "GPRS is the packet-data upgrade to GSM (2.5G) that adds always-on IP connectivity by pooling unused TDMA time slots for bursty data."
keywords: GPRS, General Packet Radio Service, 2.5G, packet data, GSM, TDMA, EDGE, coding schemes, PDP context, cellular
aka: [GPRS, General Packet Radio Service, 2.5G]
autolink: true
infobox:
  - { label: Type, value: "Cellular packet data (2.5G)" }
  - { label: Standards body, value: "ETSI, later 3GPP" }
  - { label: Introduced, value: "2000" }
  - { label: Access, value: "Shared TDMA slots (packet)" }
  - { label: Channel spacing, value: 200 kHz (on GSM carriers) }
  - { label: Modulation, value: GMSK }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [gsm, edge-cellular, gmsk, tdma, 3gpp]
cite_urls:
  - https://en.wikipedia.org/wiki/General_Packet_Radio_Service
  - https://www.etsi.org/technologies/mobile/2g
---

**GPRS** (General Packet Radio Service) is the packet-switched data upgrade to
[GSM](/reference/gsm/), often called 2.5G. Instead of tying up a whole circuit for the
duration of a session, GPRS pools spare [TDMA](/reference/tdma/) time slots and hands
them out packet by packet, giving handsets always-on IP connectivity for web browsing,
email, and WAP at a few tens of kilobits per second.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A GSM TDMA frame where several time slots are pooled and dynamically shared for GPRS packet data among users." xmlns="http://www.w3.org/2000/svg">
  <line x1="24" y1="112" x2="440" y2="112" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#gprsar)"/>
  <text x="232" y="136" text-anchor="middle" font-size="9" fill="currentColor">time → · slots shared packet-by-packet among data users</text>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="34" y="46" width="48" height="52" fill="none"/>
    <rect x="82" y="46" width="48" height="52" fill="currentColor" fill-opacity="0.30"/>
    <rect x="130" y="46" width="48" height="52" fill="currentColor" fill-opacity="0.12"/>
    <rect x="178" y="46" width="48" height="52" fill="currentColor" fill-opacity="0.30"/>
    <rect x="226" y="46" width="48" height="52" fill="currentColor" fill-opacity="0.12"/>
    <rect x="274" y="46" width="48" height="52" fill="none"/>
    <rect x="322" y="46" width="48" height="52" fill="none"/>
    <rect x="370" y="46" width="48" height="52" fill="none"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="58" y="76">V</text><text x="106" y="76">D</text><text x="154" y="76">D</text><text x="202" y="76">D</text><text x="250" y="76">D</text><text x="298" y="76">V</text><text x="346" y="76">·</text><text x="394" y="76">·</text></g>
  <text x="202" y="38" text-anchor="middle" font-size="8" fill="currentColor">D = pooled data slots · V = voice</text>
  <defs><marker id="gprsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>GPRS reuses idle GSM time slots as a shared packet pool, so bursty data need not reserve a full circuit.</figcaption>
</figure>

## Overview

Circuit-switched GSM data (CSD) reserved a whole time slot for an entire call, which
suited voice but wasted capacity on bursty Internet traffic. GPRS instead treats one
or more slots as a shared resource: a device grabs slots only when it has packets to
send, and multiple users are multiplexed onto the same slots. A handset can bond
several downlink slots for higher throughput, and coding schemes trade error
protection against speed as radio conditions vary.

## Technical characteristics

| Property | Value |
|----------|-------|
| Generation | 2.5G |
| Access | Packet-switched over shared GSM TDMA slots |
| Modulation | GMSK (as GSM) |
| Coding schemes | CS-1 to CS-4 (≈8–20 kbit/s per slot) |
| Multislot | Up to several slots bonded per direction |
| Typical rate | 30–80 kbit/s in practice |
| Core network | Adds SGSN and GGSN packet nodes to GSM |

Data sessions are described by a PDP context that maps the device onto an external IP
network; billing shifts from per-minute to per-megabyte.

## History

GPRS was specified by [ETSI](/reference/etsi/) in the late 1990s and reached commercial
networks around 2000, later maintained by [3GPP](/reference/3gpp/). It was the first
mass-market always-on mobile data service and the foundation for the faster
[EDGE](/reference/edge-cellular/) enhancement that followed, which kept the same packet
core but swapped in higher-order modulation.

## Deployment

GPRS rode on existing [GSM](/reference/gsm/) spectrum, so operators could enable it
with software and core-network upgrades rather than new radios. It powered early mobile
Internet, MMS, and machine-to-machine telemetry. As 3G and 4G arrived it became a
low-rate fallback, and it still serves undemanding IoT and metering devices where 2G
networks remain switched on.

## Decoding it with GopherTrunk

GopherTrunk targets land-mobile and utility signals; **cellular packet data such as
GPRS is out of scope and is not decoded.** GPRS traffic is authenticated,
operator-licensed, and typically ciphered, and it rides the same private
[GSM](/reference/gsm/) carriers. It is listed here only for reference within the
cellular family.

## Sources

[^wiki]: [General Packet Radio Service](https://en.wikipedia.org/wiki/General_Packet_Radio_Service) — Wikipedia, for the 2.5G GPRS packet-data service, its shared-slot operation over GSM, and its coding schemes.
