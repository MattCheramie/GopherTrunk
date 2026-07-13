---
slug: bluetooth-sig
title: Bluetooth SIG
entry_type: organization
category: organizations
description: "The Bluetooth SIG is the standards body that develops and licenses the Bluetooth wireless specification, including Classic and Low Energy radio."
keywords: Bluetooth SIG, Bluetooth Special Interest Group, Bluetooth standard, BLE, Bluetooth Low Energy, GFSK, 2.4 GHz, wireless
aka: [Bluetooth SIG, Bluetooth Special Interest Group]
autolink: true
infobox:
  - { label: Type, value: Standards body (SIG) }
  - { label: Founded, value: "1998" }
  - { label: Standards, value: Bluetooth Classic and LE }
see_also: [bluetooth, bluetooth-rf, bluetooth-le, gfsk, wi-fi-alliance]
cite_urls:
  - https://www.bluetooth.com/
  - https://en.wikipedia.org/wiki/Bluetooth_Special_Interest_Group
---

**The Bluetooth SIG** (Special Interest Group) is the non-profit membership organization
that owns, develops, and licenses the [Bluetooth](/reference/bluetooth/) wireless
specification, spanning both classic Bluetooth and [Bluetooth Low Energy](/reference/bluetooth-le/).[^home]
It holds the Bluetooth trademark, publishes the Core Specification, and runs the
qualification program every product must pass before it may carry the Bluetooth name.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="The Bluetooth SIG publishes the Core Specification and qualification program that Classic and Low Energy devices implement." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bsig_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="104" height="34" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="72" y="55">Bluetooth SIG</text><text x="72" y="67" font-size="7.5">Core Spec</text>
    <rect x="200" y="18" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="260" y="34">Bluetooth Classic</text>
    <rect x="200" y="70" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="260" y="86">Bluetooth LE</text>
    <rect x="360" y="44" width="84" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="402" y="60">qualified</text>
    <g stroke="currentColor" stroke-width="1"><line x1="124" y1="50" x2="198" y2="34" marker-end="url(#bsig_ar)"/><line x1="124" y1="64" x2="198" y2="80" marker-end="url(#bsig_ar)"/><line x1="320" y1="30" x2="360" y2="52" marker-end="url(#bsig_ar)"/><line x1="320" y1="82" x2="360" y2="60" marker-end="url(#bsig_ar)"/></g>
  </g>
</svg>
<figcaption>The Bluetooth SIG maintains one Core Specification covering both Classic and Low Energy radios.</figcaption>
</figure>

## Overview

The Bluetooth SIG was formally established in 1998 by a group of companies — Ericsson, IBM,
Intel, Nokia, and Toshiba among the founders — to promote a single short-range radio
standard and prevent the market from fracturing into incompatible variants. It has since
grown to tens of thousands of member companies, organized into promoter, associate, and
adopter tiers, with the promoters steering the roadmap.

Its central product is the **Bluetooth Core Specification**, a periodically revised document
that defines the radio, baseband, link layer, and host protocols. A pivotal moment came
with **Bluetooth 4.0** in 2010, which folded in [Bluetooth Low Energy](/reference/bluetooth-le/) —
a redesigned, ultra-low-power variant aimed at coin-cell sensors and wearables rather than
the audio-streaming use cases of classic Bluetooth. Later releases added features such as
long-range and high-throughput LE PHYs, LE Audio, and the direction-finding extensions used
for indoor positioning. Beyond the core radio, the SIG also standardizes higher-layer
**profiles** — the interoperable behaviors for headsets, keyboards, health devices, and mesh
networking — and enforces branding through its qualification and listing process. The
organization is headquartered in Kirkland, Washington.

## Relevance to SDR

Bluetooth is an attractive and challenging target for software-defined radio. It operates
in the crowded 2.4 GHz ISM band, uses [GFSK](/reference/gfsk/) (and, in higher data-rate
modes, DPSK) modulation, and — in the classic profile — hops across 79 channels many times
per second, which makes it hard to follow with a single narrowband receiver. Bluetooth LE
uses 40 wider channels and a simpler advertising structure, so LE advertising packets are a
common first sniffing target for SDR and dedicated tools alike. The SIG's published
specifications are what make such reception possible at all: they define the access
addresses, whitening, [CRC](/reference/cyclic-redundancy-check/), and channel maps a
decoder must reproduce.

GopherTrunk does not decode Bluetooth. It is a trunked land-mobile scanner focused on
narrowband voice systems in the VHF/UHF public-safety bands, and Bluetooth's frequency
hopping in the microwave ISM band is outside both its RF front-end assumptions and its
protocol scope. The Bluetooth SIG appears in this guide as part of the broader landscape of
wireless standards bodies, alongside the [Wi-Fi Alliance](/reference/wi-fi-alliance/) that
shares the same 2.4 GHz band.

## Sources

[^home]: [Bluetooth SIG](https://www.bluetooth.com/) — the group's official site, for the Core Specification, profiles, and qualification program.
[^wiki]: [Bluetooth Special Interest Group](https://en.wikipedia.org/wiki/Bluetooth_Special_Interest_Group) — Wikipedia, for the SIG's history, membership tiers, and role.
