---
slug: nb-iot
title: NB-IoT (Narrowband IoT)
entry_type: protocol
category: wireless-data-iot
description: "NB-IoT is a 3GPP cellular LPWAN using a single 180 kHz carrier for deep-coverage, low-power IoT, deployable in-band, in an LTE guard band, or standalone."
keywords: NB-IoT, Narrowband IoT, LTE Cat-NB1, Cat-NB2, 3GPP, LPWAN, cellular IoT, 180 kHz, coverage enhancement, PSM, eDRX
aka: [NB-IoT, "Narrowband IoT", "Cat-NB1", "Cat-NB2"]
autolink: true
infobox:
  - { label: Type, value: Cellular LPWAN }
  - { label: Standards body, value: 3GPP (Release 13, 2016) }
  - { label: Access, value: OFDMA down / SC-FDMA up (single- or multi-tone) }
  - { label: Bandwidth, value: 180 kHz (one LTE resource block) }
  - { label: Deployment, value: In-band, guard-band, or standalone }
  - { label: Spectrum, value: Licensed (operator) }
  - { label: Coverage, value: "up to ~164 dB maximum coupling loss" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [lte, 3gpp, lte-m, sigfox, lorawan, internet-of-things]
cite_urls:
  - https://en.wikipedia.org/wiki/Narrowband_IoT
  - https://www.3gpp.org/technologies/nb-iot
---

**NB-IoT** (**Narrowband IoT**) is a licensed-spectrum, cellular low-power wide-area
network standardized by [3GPP](/reference/3gpp/) for the
[Internet of Things](/reference/internet-of-things/).[^wiki] It squeezes a complete
cellular link into a single **180 kHz** carrier — the width of one
[LTE](/reference/lte/) resource block — and layers on deep coverage extension and
aggressive sleep modes so a sensor can reach a basement meter and still run for years on a
battery. Unlike unlicensed IoT radios, NB-IoT runs inside operators' licensed bands.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Three NB-IoT deployment modes shown across an LTE carrier: a 180 kHz block inside the LTE band (in-band), one in the guard band at the edge, and one standalone on a refarmed carrier." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="40" width="180" height="50" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <text x="120" y="30">LTE carrier</text>
    <rect x="120" y="45" width="16" height="40" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <text x="128" y="105">in-band</text>
    <rect x="212" y="45" width="16" height="40" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <text x="222" y="105">guard</text>
    <rect x="360" y="45" width="16" height="40" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <text x="368" y="105">standalone</text>
    <line x1="330" y1="65" x2="390" y2="65" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
    <text x="360" y="30">refarmed</text>
  </g>
  <text x="230" y="135" text-anchor="middle" font-size="9" fill="currentColor">each NB-IoT block is 180 kHz wide — one LTE resource block</text>
</svg>
<figcaption>NB-IoT fits in one 180 kHz resource block: inside an LTE carrier, in its guard band, or standalone on a refarmed (e.g. GSM) channel.</figcaption>
</figure>

## Overview

NB-IoT trades throughput and latency for reach and battery life. It uses
[OFDMA](/reference/ofdma/) on the downlink and single-carrier FDMA on the uplink, where a
device may transmit on a single 3.75 or 15 kHz tone to concentrate all its power for the
best link budget. Coverage enhancement repeats transmissions many times so a signal
survives roughly 20 dB deeper into buildings than plain LTE.

## Technical characteristics

| Property | Value |
|----------|-------|
| Bandwidth | 180 kHz carrier |
| Downlink | OFDMA, 15 kHz subcarriers |
| Uplink | SC-FDMA, single-tone (3.75/15 kHz) or multi-tone |
| Duplex | Half-duplex FDD (typical) |
| Peak rate | tens to ~250 kbps (Cat-NB1); higher for NB2 |
| Power saving | PSM and extended DRX (eDRX) |
| Coverage | up to ~164 dB maximum coupling loss |

Because it reuses LTE numerology and the operator's core network, NB-IoT inherits SIM-based
security, authentication, and global roaming — advantages the unlicensed LPWANs lack.

## History

3GPP introduced NB-IoT in Release 13 (frozen in 2016) as Cat-NB1, alongside the related
[LTE-M](/reference/lte-m/).[^3gpp] Release 14 added Cat-NB2 with higher data rates and
positioning, and later releases folded NB-IoT into the 5G ecosystem as a recognized
mMTC (massive machine-type communications) technology.

## Deployment

Operators worldwide have deployed NB-IoT for smart metering, environmental sensing, smart
parking, and asset tracking — applications that send small, infrequent reports and value
deep indoor coverage. It competes with unlicensed [LoRaWAN](/reference/lorawan/) and
[Sigfox](/reference/sigfox/) on one side and, for higher-rate or mobile use, with its
sibling LTE-M on the other.

## Decoding it with GopherTrunk

NB-IoT is out of scope for GopherTrunk, a trunked land-mobile *voice* scanner. NB-IoT is
encrypted, SIM-authenticated cellular traffic on licensed spectrum; receiving user payloads
is not something an SDR scanner does. GopherTrunk implements no LTE/NB-IoT PHY or core-network
stack. On a [waterfall](/reference/waterfall-display/) you would simply see it as a narrow
carrier within an operator's band.

## Sources

[^wiki]: [Narrowband IoT](https://en.wikipedia.org/wiki/Narrowband_IoT) — Wikipedia, for the 180 kHz carrier, deployment modes, and coverage-enhancement design.
[^3gpp]: [NB-IoT](https://www.3gpp.org/technologies/nb-iot) — 3GPP, for the Release 13 origin and the standard's positioning in cellular IoT.
