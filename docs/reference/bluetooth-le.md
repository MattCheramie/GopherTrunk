---
slug: bluetooth-le
title: Bluetooth Low Energy (BLE)
entry_type: protocol
category: wireless-data-iot
description: "Bluetooth Low Energy (BLE) is a 2.4 GHz low-power air interface using GFSK across 40 channels, with three dedicated advertising channels for discovery and beacons."
keywords: Bluetooth Low Energy, BLE, Bluetooth Smart, GFSK, advertising channels, 2.4 GHz, beacon, IoT, low power, LE Coded PHY, Bluetooth 4.0, Bluetooth 5
aka: [BLE, Bluetooth LE, Bluetooth Smart, Bluetooth Low Energy]
autolink: true
infobox:
  - { label: Type, value: Low-power wireless PHY }
  - { label: Standards body, value: Bluetooth SIG }
  - { label: Introduced, value: "2010 (Bluetooth 4.0)" }
  - { label: Access, value: FHSS (37 data) + 3 advertising }
  - { label: Channel spacing, value: 2 MHz (40 channels) }
  - { label: Modulation, value: "GFSK (1M, 2M, Coded PHY)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [bluetooth-rf, bluetooth, gfsk, bluetooth-sig, internet-of-things, ant-plus]
cite_urls:
  - https://en.wikipedia.org/wiki/Bluetooth_Low_Energy
  - https://en.wikipedia.org/wiki/Gaussian_frequency-shift_keying
---

**Bluetooth Low Energy** (**BLE**) is the low-power personal-area radio introduced in
Bluetooth 4.0, using [GFSK](/reference/gfsk/) modulation across 40 channels in the
2.4 GHz band and reserving three channels for advertising and discovery.[^wiki] It is a
distinct air interface from [Bluetooth Classic](/reference/bluetooth-rf/), optimised for
devices that wake, send a short burst, and sleep — the backbone of many
[IoT](/reference/internet-of-things/) sensors, beacons, and wearables.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The BLE channel plan: 40 channels of 2 MHz each across 2.4 GHz, with three advertising channels placed at the band edges and centre to avoid Wi-Fi, and 37 data channels in between." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="blear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.5" marker-end="url(#blear)"/>
  <text x="435" y="118" text-anchor="end" font-size="9" fill="currentColor">2.402 → 2.480 GHz</text>
  <g stroke="currentColor" stroke-width="0.8">
    <rect x="40" y="55" width="10" height="45" fill="currentColor" fill-opacity="0.5"/>
    <g fill="currentColor" fill-opacity="0.18">
      <rect x="54" y="70" width="8" height="30"/><rect x="64" y="70" width="8" height="30"/><rect x="74" y="70" width="8" height="30"/><rect x="84" y="70" width="8" height="30"/><rect x="94" y="70" width="8" height="30"/><rect x="104" y="70" width="8" height="30"/><rect x="114" y="70" width="8" height="30"/><rect x="124" y="70" width="8" height="30"/><rect x="134" y="70" width="8" height="30"/><rect x="144" y="70" width="8" height="30"/><rect x="154" y="70" width="8" height="30"/><rect x="164" y="70" width="8" height="30"/><rect x="174" y="70" width="8" height="30"/><rect x="184" y="70" width="8" height="30"/><rect x="194" y="70" width="8" height="30"/><rect x="204" y="70" width="8" height="30"/><rect x="214" y="70" width="8" height="30"/></g>
    <rect x="226" y="55" width="10" height="45" fill="currentColor" fill-opacity="0.5"/>
    <g fill="currentColor" fill-opacity="0.18">
      <rect x="240" y="70" width="8" height="30"/><rect x="250" y="70" width="8" height="30"/><rect x="260" y="70" width="8" height="30"/><rect x="270" y="70" width="8" height="30"/><rect x="280" y="70" width="8" height="30"/><rect x="290" y="70" width="8" height="30"/><rect x="300" y="70" width="8" height="30"/><rect x="310" y="70" width="8" height="30"/><rect x="320" y="70" width="8" height="30"/><rect x="330" y="70" width="8" height="30"/><rect x="340" y="70" width="8" height="30"/><rect x="350" y="70" width="8" height="30"/><rect x="360" y="70" width="8" height="30"/><rect x="370" y="70" width="8" height="30"/><rect x="380" y="70" width="8" height="30"/><rect x="390" y="70" width="8" height="30"/><rect x="400" y="70" width="8" height="30"/><rect x="410" y="70" width="8" height="30"/></g>
    <rect x="422" y="55" width="10" height="45" fill="currentColor" fill-opacity="0.5"/>
  </g>
  <text x="230" y="30" text-anchor="middle" font-size="10" fill="currentColor">3 advertising channels (solid) + 37 data channels (light)</text>
</svg>
<figcaption>BLE reserves three advertising channels, spaced to dodge the common Wi-Fi channels, with 37 hopping data channels in between.</figcaption>
</figure>

## Overview

BLE splits the 2.4 GHz band into 40 channels of 2 MHz each. Three of them — numbers 37,
38, and 39, deliberately placed at the band edges and centre to sidestep the busiest
Wi-Fi channels — carry *advertising* packets that let devices announce themselves,
broadcast beacons, or begin a connection. The remaining 37 *data* channels are used
inside an established connection, hopping between them for interference resilience. The
base PHY is GFSK at 1 Mbit/s; Bluetooth 5 added a 2 Mbit/s mode and a long-range
"Coded PHY" that trades rate for sensitivity.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | 2.402–2.480 GHz ISM |
| Channels | 40 × 2 MHz (3 advertising + 37 data) |
| Modulation | GFSK |
| Bit rate | 1 Mbit/s (LE 1M); 2 Mbit/s (LE 2M) |
| Long range | LE Coded PHY (S=2, S=8) |
| Connection hop | Adaptive over 37 data channels |

The advertising channels make BLE traffic unusually observable: a device that only ever
advertises transmits short, unencrypted packets on three known frequencies, which is why
BLE beacons and proximity trackers are straightforward to survey.

## History

The Low Energy layer was designed as "Wibree" by Nokia, then folded into Bluetooth 4.0
(2010) by the [Bluetooth SIG](/reference/bluetooth-sig/). Bluetooth 5 (2016) doubled the
data rate and added long-range and higher-throughput advertising options; later releases
added direction-finding and LE Audio.

## Deployment

BLE is pervasive in fitness trackers, smart-home sensors, medical devices, retail
beacons, and phone accessories. It coexists with, and often complements,
[ANT+](/reference/ant-plus/) in sports sensors and competes with proprietary sub-GHz
links elsewhere in the IoT space.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode BLE. Although the three advertising channels are fixed
and their packets are short, following BLE is a wideband 2.4 GHz task handled by
purpose-built tools (dedicated sniffers, or SDRs with GFSK demodulators), not by
GopherTrunk's narrowband land-mobile trunking decoders. BLE is relevant here only as
ambient 2.4 GHz activity.

## Sources

[^wiki]: [Bluetooth Low Energy](https://en.wikipedia.org/wiki/Bluetooth_Low_Energy) — Wikipedia, on the BLE air interface, its 40-channel plan, advertising channels, GFSK PHYs, and history.
