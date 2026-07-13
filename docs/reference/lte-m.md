---
slug: lte-m
title: LTE-M (Cat-M1)
entry_type: protocol
category: wireless-data-iot
description: "LTE-M is a 3GPP cellular LPWAN (Cat-M1/eMTC) using a 1.4 MHz band within LTE to support mobility, voice, and higher data rates than NB-IoT for low-power IoT."
keywords: LTE-M, Cat-M1, eMTC, LTE Cat-M, 3GPP, cellular IoT, LPWAN, 1.4 MHz, PSM, eDRX, VoLTE, mobility
aka: [LTE-M, "Cat-M1", "eMTC", "LTE Cat-M"]
autolink: true
infobox:
  - { label: Type, value: Cellular LPWAN }
  - { label: Standards body, value: 3GPP (Release 13, 2016) }
  - { label: Access, value: OFDMA down / SC-FDMA up (LTE numerology) }
  - { label: Bandwidth, value: 1.4 MHz (Cat-M1); up to 5 MHz (Cat-M2) }
  - { label: Peak rate, value: ~1 Mbps (Cat-M1); ~4 Mbps (Cat-M2) }
  - { label: Spectrum, value: Licensed (operator) }
  - { label: Extras, value: Full mobility, VoLTE support }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [lte, nb-iot, 3gpp, lorawan, sigfox, internet-of-things]
cite_urls:
  - https://en.wikipedia.org/wiki/LTE-M
  - https://www.3gpp.org/technologies/mtc-ph2
---

**LTE-M** (LTE Cat-M1, also called **eMTC** — enhanced Machine-Type Communications) is a
[3GPP](/reference/3gpp/) cellular LPWAN that carves a low-power, low-complexity device
class out of the [LTE](/reference/lte/) air interface.[^wiki] It is the higher-capability
cousin of [NB-IoT](/reference/nb-iot/): by using a wider **1.4 MHz** slice of an LTE
carrier it supports real **mobility** (cell handover), meaningfully higher data rates, and
even voice ([VoLTE](/reference/volte/)), while still offering deep-sleep power savings for
the [Internet of Things](/reference/internet-of-things/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A comparison bar chart: NB-IoT occupies a narrow 180 kHz block while LTE-M occupies a wider 1.4 MHz block, both inside a full LTE carrier, showing LTE-M trades bandwidth for higher rate and mobility." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="40" y="90" width="20" height="30" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <text x="50" y="135">NB-IoT</text><text x="50" y="82">180 kHz</text>
    <rect x="140" y="55" width="70" height="65" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <text x="175" y="135">LTE-M</text><text x="175" y="47">1.4 MHz</text>
    <rect x="300" y="30" width="140" height="90" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <text x="370" y="135">LTE carrier</text><text x="370" y="22">up to 20 MHz</text>
  </g>
  <text x="230" y="150" text-anchor="middle" font-size="8.5" fill="currentColor">wider band → higher rate, handover, and voice</text>
</svg>
<figcaption>LTE-M uses a wider slice (1.4 MHz) than NB-IoT, buying mobility, higher throughput, and voice at the cost of a bit more device complexity.</figcaption>
</figure>

## Overview

LTE-M reuses LTE's OFDMA downlink and SC-FDMA uplink and its licensed-band security and
core network, but restricts the device to a simplified, half-duplex-capable, single-antenna
profile to cut cost and power. Crucially, unlike NB-IoT, an LTE-M device can hand over
between cells while moving, which makes it suitable for tracking vehicles, wearables, and
other things that do not sit still.

## Technical characteristics

| Property | Value |
|----------|-------|
| Bandwidth | 1.4 MHz (Cat-M1); up to 5 MHz (Cat-M2) |
| Downlink | OFDMA, 15 kHz subcarriers |
| Uplink | SC-FDMA |
| Peak rate | ~1 Mbps (Cat-M1); ~4 Mbps (Cat-M2) |
| Mobility | Full cell handover |
| Voice | VoLTE supported |
| Power saving | PSM and extended DRX (eDRX) |

The extra bandwidth and mobility make LTE-M feel like "small LTE" rather than a telemetry
trickle, closing the gap between classic cellular data and ultra-narrowband IoT.

## History

3GPP standardized LTE-M in Release 13 (2016) as Cat-M1/eMTC, alongside
[NB-IoT](/reference/nb-iot/).[^3gpp] Release 14 added Cat-M2 with wider bandwidth and higher
rates, and later releases carried LTE-M forward as a 5G-era machine-type technology.

## Deployment

Carriers offer LTE-M for asset and fleet tracking, connected health and wearables, smart
utilities, and alarm panels — anywhere moderate data, mobility, or occasional voice matters.
It sits between low-rate LPWANs like [LoRaWAN](/reference/lorawan/) and
[Sigfox](/reference/sigfox/) and full LTE data plans.

## Decoding it with GopherTrunk

LTE-M is out of scope for GopherTrunk, which decodes trunked land-mobile voice. It is
encrypted, SIM-authenticated cellular traffic on licensed spectrum; an SDR scanner does not
recover its payloads, and GopherTrunk contains no LTE PHY or core-network stack. It would
appear on a [waterfall](/reference/waterfall-display/) only as a narrow carrier inside an
operator's band.

## Sources

[^wiki]: [LTE-M](https://en.wikipedia.org/wiki/LTE-M) — Wikipedia, for the Cat-M1/eMTC definition, 1.4 MHz bandwidth, mobility, and voice support.
[^3gpp]: [MTC / LTE-M](https://www.3gpp.org/technologies/mtc-ph2) — 3GPP, for the Release 13/14 machine-type communications enhancements.
