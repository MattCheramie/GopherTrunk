---
slug: sigfox
title: Sigfox
entry_type: protocol
category: wireless-data-iot
description: "Sigfox is an ultra-narrowband LPWAN that sends tiny messages in ~100 Hz sub-GHz channels with heavy time and frequency diversity for very-low-power IoT."
keywords: Sigfox, UNB, ultra narrowband, LPWAN, IoT, DBPSK, 0G network, UnaBiz, 868 MHz, 902 MHz, low power
aka: [Sigfox, "UNB network", "0G network"]
autolink: true
infobox:
  - { label: Type, value: Ultra-narrowband LPWAN }
  - { label: Operator, value: "Sigfox / UnaBiz (single global operator model)" }
  - { label: Introduced, value: "2010" }
  - { label: Access, value: Random time + frequency, unslotted }
  - { label: Channel, value: ~100 Hz ultra-narrowband }
  - { label: Modulation, value: DBPSK uplink, GFSK downlink }
  - { label: Bands, value: "Sub-GHz ISM (RC1 868 MHz, RC2 902 MHz, …)" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [nb-iot, lorawan, lora, internet-of-things, frequency-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Sigfox
  - https://www.sigfox.com/
---

**Sigfox** is an **ultra-narrowband (UNB)** low-power wide-area network for the
[Internet of Things](/reference/internet-of-things/), designed to carry very small
messages over long distances at minimal power and cost.[^wiki] Its defining trick is
extreme spectral thrift: each uplink occupies only about **100 Hz** of a sub-GHz ISM band,
so a receiver's noise bandwidth is tiny and the link budget is generous. Sigfox is often
marketed as a "0G" network, deliberately positioned below cellular data.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A wide ISM band containing many extremely narrow Sigfox uplinks scattered across frequency, illustrating ultra-narrowband transmission with frequency diversity." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="25" width="400" height="90" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="2">
    <line x1="70" y1="112" x2="70" y2="55"/>
    <line x1="130" y1="112" x2="130" y2="40"/>
    <line x1="205" y1="112" x2="205" y2="62"/>
    <line x1="270" y1="112" x2="270" y2="48"/>
    <line x1="340" y1="112" x2="340" y2="70"/>
    <line x1="395" y1="112" x2="395" y2="52"/>
  </g>
  <text x="230" y="135" text-anchor="middle" font-size="9" fill="currentColor">frequency → · each spike ≈100 Hz wide; devices pick random slots</text>
  <text x="20" y="70" font-size="8" fill="currentColor" transform="rotate(-90 20 70)">power</text>
</svg>
<figcaption>Sigfox packs many ~100 Hz uplinks into a wide sub-GHz band; because devices choose random frequencies and repeat, collisions are rare and base stations scan the whole band.</figcaption>
</figure>

## Overview

A Sigfox device has no network to join and no channel to request. It simply transmits a
short message on a randomly chosen frequency, then repeats it (by default three times) on
different frequencies at different times. Base stations continuously scan the whole band,
and the back end reconciles the copies. This unslotted, cooperative-reception model keeps
the endpoint radio extraordinarily simple and cheap.

## Technical characteristics

| Property | Value |
|----------|-------|
| Uplink | DBPSK, ~100 bps, ~100 Hz channel |
| Downlink | GFSK, ~600 bps |
| Payload | Up to 12 bytes uplink, 8 bytes downlink |
| Duty limits | ~140 uplink and 4 downlink messages per device per day |
| Bands | Sub-GHz ISM per region (868 MHz EU, 902 MHz US, …) |
| Diversity | Time + frequency (repeated transmissions) |

The tiny payload and daily message cap suit metering, alarms, and status pings rather than
streaming — a single Sigfox device can run for years on a small battery.

## History

Sigfox was founded in France in 2010 and built out national UNB networks across Europe and
beyond under a single-operator model.[^site] After financial restructuring, the technology
and network assets were acquired by UnaBiz in 2022, which continues to operate and evolve
the standard.

## Deployment

Sigfox targets massive, ultra-low-cost sensor fleets: utility sub-metering, logistics
tracking, environmental monitoring, and simple alarms. It competes with unlicensed
[LoRaWAN](/reference/lorawan/) and licensed cellular LPWANs such as
[NB-IoT](/reference/nb-iot/); the trade is Sigfox's simplicity and battery life against its
strict message limits and dependence on one operator's coverage.

## Decoding it with GopherTrunk

Sigfox is out of scope for GopherTrunk, which decodes trunked land-mobile voice, not IoT
telemetry. Its ~100 Hz uplinks are so narrow they are hard to even spot on a normal
[waterfall](/reference/waterfall-display/), and the network's device provisioning and back
end are proprietary. GopherTrunk implements neither the UNB PHY nor the Sigfox cloud
protocol.

## Sources

[^wiki]: [Sigfox](https://en.wikipedia.org/wiki/Sigfox) — Wikipedia, for the ultra-narrowband air interface, message limits, and single-operator model.
[^site]: [Sigfox](https://www.sigfox.com/) — Sigfox / UnaBiz, for the "0G" positioning, network operation, and technology stewardship.
