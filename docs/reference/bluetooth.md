---
slug: bluetooth
title: Bluetooth
entry_type: concept
category: hw-networking
description: Bluetooth is a short-range wireless standard for connecting devices over the 2.4 GHz band, used for audio, peripherals, and low-power sensor links; its Low Energy profile (BLE) powers many battery devices and IoT sensors.
keywords: Bluetooth, Bluetooth Low Energy, BLE, 2.4 GHz, wireless, PAN, piconet, pairing, frequency hopping, ISM band
aka: [BT, Bluetooth Low Energy, BLE]
infobox:
  - { label: Type, value: Short-range wireless }
  - { label: Band, value: 2.4 GHz ISM }
  - { label: Range, value: ~1–100 m (class-dependent) }
  - { label: Variants, value: Classic, Low Energy (BLE) }
  - { label: Access, value: Frequency-hopping spread spectrum }
see_also: [wi-fi, bluetooth-le, near-field-communication, wireless-access-point, electromagnetic-spectrum, internet-of-things, ethernet]
cite_urls:
  - https://en.wikipedia.org/wiki/Bluetooth
---

**Bluetooth** is a short-range wireless standard for connecting devices over the 2.4 GHz band, used for audio, peripherals, and low-power links between nearby gadgets.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A Bluetooth piconet drawn as a star: a central phone in the middle is linked by short radio hops to a headset, a keyboard, and a battery sensor arranged around it, all sharing the 2.4 GHz band by frequency hopping." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <g stroke-width="1.2" stroke-dasharray="3 3" fill="none">
      <line x1="230" y1="85" x2="110" y2="45"/>
      <line x1="230" y1="85" x2="110" y2="125"/>
      <line x1="230" y1="85" x2="350" y2="45"/>
      <line x1="230" y1="85" x2="350" y2="125"/>
    </g>
    <rect x="214" y="62" width="32" height="46" rx="4" fill-opacity="0.12" stroke-width="1.4"/>
    <rect x="86" y="30" width="48" height="30" rx="4" fill-opacity="0.12" stroke-width="1.4"/>
    <rect x="86" y="110" width="48" height="30" rx="4" fill-opacity="0.12" stroke-width="1.4"/>
    <rect x="326" y="30" width="48" height="30" rx="4" fill-opacity="0.12" stroke-width="1.4"/>
    <rect x="326" y="110" width="48" height="30" rx="4" fill-opacity="0.12" stroke-width="1.4"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8.5">
    <text x="230" y="89" font-weight="600">phone</text>
    <text x="230" y="126" font-size="8">central</text>
    <text x="110" y="49">headset</text>
    <text x="110" y="129">keyboard</text>
    <text x="350" y="49">sensor</text>
    <text x="350" y="129">wearable</text>
    <text x="230" y="150" font-size="8" fill-opacity="0.9">one piconet, 2.4 GHz ISM band, ~1600 frequency hops per second</text>
  </g>
</svg>
<figcaption>A Bluetooth piconet: one central device (here a phone) holds short radio links to a handful of nearby peripherals, all hopping together across the 2.4 GHz band to dodge interference from Wi-Fi and other users of the crowded ISM spectrum.</figcaption>
</figure>

## Overview

Bluetooth forms small *personal-area networks* between paired devices — headphones to a phone, a keyboard to a laptop, a sensor to a hub — hopping rapidly across the crowded 2.4 GHz ISM band to dodge interference. The devices in one link form a *piconet*, with one central node coordinating the others, and the hopping pattern spreads their traffic thinly over the band so neighbours collide only briefly.

**Bluetooth Low Energy** (BLE) is a separate, power-frugal profile aimed at battery devices that wake, send a short burst, and sleep. That duty-cycled design has made it the backbone of many [IoT](/reference/internet-of-things/) sensors, beacons, and wearables, where a coin cell must last months or years. Classic Bluetooth, by contrast, keeps a continuous stream open for audio and file transfer.

Unlike [Wi-Fi](/reference/wi-fi/), Bluetooth targets close-range, low-bandwidth links rather than network access; for touch-range pairing, [NFC](/reference/near-field-communication/) is closer still. All three share the same congested unlicensed spectrum, so they are engineered to coexist rather than to reach far.

## Classic vs Low Energy

The two flavours share a band and a name but are built for different jobs:

| Trait | Classic Bluetooth | Low Energy (BLE) |
|-------|-------------------|------------------|
| Aimed at | Audio, continuous streams | Sensors, bursts, beacons |
| Power draw | Higher | Very low (sleeps between bursts) |
| Data rate | Up to ~2–3 Mb/s | ~0.1–2 Mb/s |
| Connection | Held open | Brief, on demand |
| Typical use | Headphones, speakers | Wearables, IoT, tags |

Modern chips implement both, so one radio can stream music and also talk to a low-power fitness band.

## Where it fits

Bluetooth is built for convenience over short distances at low power, not throughput, so it is rarely part of a GopherTrunk data path. Its relevance to an SDR user is as *interference*: the dense, hopping 2.4 GHz traffic from Bluetooth and [Wi-Fi](/reference/wi-fi/) is exactly the kind of noisy [spectrum](/reference/electromagnetic-spectrum/) one learns to recognize and avoid when siting an antenna, and a nearby BLE device can raise the noise floor a wideband receiver sees.

## Sources

[^wiki]: [Bluetooth](https://en.wikipedia.org/wiki/Bluetooth) — Wikipedia, on the short-range wireless standard, piconets, and Bluetooth Low Energy.
