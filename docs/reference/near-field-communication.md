---
slug: near-field-communication
title: Near-field communication (NFC)
entry_type: concept
category: hw-mobile
description: Near-field communication (NFC) is a short-range wireless technology that lets devices exchange small amounts of data over a few centimeters, used for contactless payments, transit cards, and tap-to-pair.
keywords: NFC, near-field communication, contactless, tap to pay, RFID, 13.56 MHz, contactless payment, tag, tap to pair, inductive coupling
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 155" role="img" aria-label="An NFC tap shown as inductive coupling. A phone on the left carries an antenna coil driven by its NFC controller, generating a 13.56 megahertz magnetic field. A passive tag or card on the right has its own coil. When the two coils are within a few centimeters, the phone's field induces current in the tag's coil, powering it and carrying data both ways." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <rect x="40" y="34" width="70" height="86" rx="8"/>
    <ellipse cx="94" cy="77" rx="9" ry="26"/>
    <ellipse cx="88" cy="77" rx="9" ry="20"/>
    <rect x="350" y="46" width="70" height="62" rx="6"/>
    <ellipse cx="366" cy="77" rx="9" ry="22"/>
    <ellipse cx="372" cy="77" rx="9" ry="16"/>
    <g stroke-width="0.9">
      <path d="M120 60 C170 48 210 48 250 60" stroke-dasharray="4 3"/>
      <path d="M120 77 C180 70 260 70 340 77" stroke-dasharray="4 3"/>
      <path d="M120 94 C170 106 210 106 250 94" stroke-dasharray="4 3"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="75" y="136">phone &#183; NFC coil</text>
    <text x="385" y="124">passive tag / card</text>
    <text x="230" y="42" font-size="8">13.56 MHz magnetic field</text>
    <text x="230" y="128" font-size="8">&#8804; ~4 cm &#183; field powers the tag</text>
  </g>
</svg>
<figcaption>NFC works by inductive coupling: the phone's coil radiates a 13.56 MHz magnetic field that, within a few centimeters, induces current in a passive tag's coil — powering it and carrying data both ways without the tag needing its own battery.</figcaption>
</figure>

## Overview

NFC operates at 13.56 MHz and grew out of [RFID](/reference/radio-wave/). Its very short range is a feature, not a limit: a tap is deliberate and hard to eavesdrop on. A device can act as a reader (scanning a tag or card), a tag (presenting credentials), or in peer-to-peer mode.

An NFC controller in the [SoC](/reference/system-on-a-chip/), wired to a small loop antenna, can even power a passive tag through its magnetic field, so unpowered stickers and cards work with no battery of their own. The [mobile OS](/reference/mobile-operating-system/) exposes the controller to apps for payments and pairing, usually behind a secure element that holds payment credentials.

## NFC vs other short-range radios

NFC deliberately trades range and speed for intimacy and simplicity:

| Property | NFC | Bluetooth | Wi-Fi |
|----------|-----|-----------|-------|
| Range | ~4 cm | ~10 m | ~30 m+ |
| Data rate | ~424 kbit/s | 1–3 Mbit/s | 100s Mbit/s |
| Pairing | Instant tap | Discovery/pair | Join network |
| Powers passive device | Yes | No | No |
| Typical use | Pay, ticket, tap-pair | Audio, peripherals | Internet, streaming |

Its slowness barely matters, because NFC carries only tiny payloads — a token, an ID, or a handoff that then continues over a faster radio.

## Where it fits

NFC is the technology behind tap-to-pay on a [smartphone](/reference/smartphone/) or [smartwatch](/reference/smartwatch/), transit cards, and instant Bluetooth pairing. It complements an [eSIM](/reference/esim/) and the longer-range radios in a device: where the cellular and Wi-Fi radios reach across networks, NFC handles the intimate, intentional "touch here" interactions of everyday mobile use.

## Sources

[^wiki]: [Near-field communication](https://en.wikipedia.org/wiki/Near-field_communication) — Wikipedia, on NFC technology and uses.
