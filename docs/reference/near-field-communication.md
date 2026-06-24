---
slug: near-field-communication
title: Near-field communication (NFC)
entry_type: concept
category: hw-mobile
description: Near-field communication (NFC) is a short-range wireless technology that lets devices exchange small amounts of data over a few centimeters, used for contactless payments, transit cards, and tap-to-pair.
keywords: NFC, near-field communication, contactless, tap to pay, RFID, 13.56 MHz, contactless payment, tag, tap to pair
autolink: true
aka: [NFC]
infobox:
  - { label: Type, value: Short-range wireless }
  - { label: Range, value: ~4 cm }
  - { label: Frequency, value: 13.56 MHz }
  - { label: Uses, value: Payments, transit, pairing }
see_also: [smartphone, esim, mobile-operating-system, smartwatch, wearable-computer, system-on-a-chip]
cite_urls:
  - https://en.wikipedia.org/wiki/Near-field_communication
---

**Near-field communication (NFC)** is a short-range wireless technology that lets two devices exchange small amounts of data when held within a few centimeters of each other.[^wiki]

## Overview

NFC operates at 13.56 MHz and grew out of [RFID](/reference/radio-wave/). Its very short range is a feature: a tap is deliberate and hard to eavesdrop on. A device can act as a reader (scanning a tag or card), a tag (presenting credentials), or in peer-to-peer mode. An NFC controller in the [SoC](/reference/system-on-a-chip/), wired to a small loop antenna, can even power a passive tag through its field, so unpowered stickers and cards work. The [mobile OS](/reference/mobile-operating-system/) exposes it to apps for payments and pairing.

## Where it fits

NFC is the technology behind tap-to-pay on a [smartphone](/reference/smartphone/) or [smartwatch](/reference/smartwatch/), transit cards, and instant Bluetooth pairing. It complements an [eSIM](/reference/esim/) and the longer-range radios in a device: where the cellular and Wi-Fi radios reach across networks, NFC handles the intimate, intentional "touch here" interactions of everyday mobile use.

## Sources

[^wiki]: [Near-field communication](https://en.wikipedia.org/wiki/Near-field_communication) — Wikipedia, on NFC technology and uses.
