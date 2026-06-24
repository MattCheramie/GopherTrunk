---
slug: bluetooth
title: Bluetooth
entry_type: concept
category: hw-networking
description: Bluetooth is a short-range wireless standard for connecting devices over the 2.4 GHz band, used for audio, peripherals, and low-power sensor links between nearby gadgets.
keywords: Bluetooth, Bluetooth Low Energy, BLE, 2.4 GHz, wireless, PAN, pairing
aka: [BT, Bluetooth Low Energy, BLE]
infobox:
  - { label: Type, value: Short-range wireless }
  - { label: Band, value: 2.4 GHz ISM }
  - { label: Range, value: ~1–100 m (class-dependent) }
  - { label: Variants, value: Classic, Low Energy (BLE) }
see_also: [wi-fi, near-field-communication, wireless-access-point, electromagnetic-spectrum, internet-of-things, ethernet]
cite_urls:
  - https://en.wikipedia.org/wiki/Bluetooth
---

**Bluetooth** is a short-range wireless standard for connecting devices over the 2.4 GHz band, used for audio, peripherals, and low-power links between nearby gadgets.[^wiki]

## Overview

Bluetooth forms small *personal-area networks* between paired devices — headphones to a phone, a keyboard to a laptop, a sensor to a hub — hopping rapidly across the crowded 2.4 GHz ISM band to dodge interference. **Bluetooth Low Energy** (BLE) is a separate, power-frugal profile aimed at battery devices that wake, send a short burst, and sleep, which has made it the backbone of many [IoT](/reference/internet-of-things/) sensors and wearables. Unlike [Wi-Fi](/reference/wi-fi/), it targets close-range, low-bandwidth links rather than network access; for touch-range pairing, [NFC](/reference/near-field-communication/) is closer still.

## What it's for

Bluetooth is built for convenience over short distances at low power, not throughput. It is rarely part of a GopherTrunk data path, but its dense 2.4 GHz traffic — alongside Wi-Fi — is exactly the kind of noisy [spectrum](/reference/electromagnetic-spectrum/) an SDR user learns to recognize and avoid when siting an antenna.

## Sources

[^wiki]: [Bluetooth](https://en.wikipedia.org/wiki/Bluetooth) — Wikipedia, on the short-range wireless standard.
