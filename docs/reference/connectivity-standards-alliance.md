---
slug: connectivity-standards-alliance
title: Connectivity Standards Alliance (CSA)
entry_type: organization
category: organizations
description: "The Connectivity Standards Alliance is the industry group behind Zigbee and Matter, developing open standards for interoperable smart-home and IoT devices."
keywords: Connectivity Standards Alliance, CSA, Zigbee Alliance, Matter, smart home, IoT, Thread, interoperability, home automation
aka: [Connectivity Standards Alliance, CSA, Zigbee Alliance]
autolink: true
infobox:
  - { label: Type, value: Industry standards alliance }
  - { label: Founded, value: "2002 (as the Zigbee Alliance)" }
  - { label: Standards, value: "Zigbee, Matter" }
see_also: [zigbee-802154, matter-protocol, thread-protocol, internet-of-things, home-automation]
cite_urls:
  - https://csa-iot.org/
  - https://en.wikipedia.org/wiki/Connectivity_Standards_Alliance
---

The **Connectivity Standards Alliance** (**CSA**) is the **industry organisation that
develops open standards for smart-home and Internet-of-Things devices**, best known for the
[Zigbee](/reference/zigbee-802154/) mesh-networking standard and the newer
[Matter](/reference/matter-protocol/) application layer.[^wiki] Formed in 2002 as the Zigbee
Alliance and renamed in 2021, it brings together hundreds of hardware makers, chip vendors,
and platform companies to define specifications and run the certification programmes that let
devices from different brands work together.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 116" role="img" aria-label="The Connectivity Standards Alliance publishing the Zigbee and Matter standards and certifying interoperable smart-home devices." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="165" y="10" width="130" height="28" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="28">CSA</text>
    <rect x="60" y="58" width="120" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="120" y="75">Zigbee</text>
    <rect x="280" y="58" width="120" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="340" y="75">Matter</text>
    <rect x="150" y="94" width="160" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="230" y="108" font-size="8">certified smart-home devices</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="205" y1="38" x2="130" y2="57" marker-end="url(#ar_csa)"/>
      <line x1="255" y1="38" x2="330" y2="57" marker-end="url(#ar_csa)"/>
      <line x1="130" y1="84" x2="215" y2="94" marker-end="url(#ar_csa)"/>
      <line x1="330" y1="84" x2="250" y2="94" marker-end="url(#ar_csa)"/>
    </g>
  </g>
  <defs><marker id="ar_csa" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The CSA publishes the Zigbee and Matter standards and certifies devices to guarantee interoperability.</figcaption>
</figure>

## Overview

The alliance's original work was [Zigbee](/reference/zigbee-802154/), a low-power mesh
networking standard built on the IEEE 802.15.4 radio in the 2.4 GHz ISM band. Zigbee became
one of the dominant technologies for smart-home sensors, lights, and locks, but the market
fragmented into competing ecosystems that often did not interoperate. To fix that, the
alliance renamed itself and led development of **Matter**, an application-layer standard
launched in 2022 that runs over existing IP transports — Wi-Fi, Ethernet, and the
[Thread](/reference/thread-protocol/) mesh — so that a device certified under Matter can be
controlled by any compatible platform regardless of vendor.

The CSA operates the way most modern standards bodies do: member companies contribute to
working groups that write the specifications, and the alliance runs the trademark and
certification programme that lets a product carry the Zigbee or Matter logo only after it has
passed interoperability testing. That certification is the real value it provides — it is the
mechanism that turns a paper standard into a guarantee that two independently built devices
will actually talk to each other.

## Relevance to SDR

The radios the CSA's standards use are squarely in SDR territory. Zigbee occupies the
crowded 2.4 GHz ISM band alongside Wi-Fi and Bluetooth, and its 802.15.4 physical layer —
O-QPSK with direct-sequence spread spectrum — is a common target for hobbyist analysis and
security research with wideband SDRs. Thread, which carries much Matter traffic, uses the same
802.15.4 radio. For anyone surveying the 2.4 GHz band, recognising a CSA-defined signal is
part of making sense of the [IoT](/reference/internet-of-things/) and
[home-automation](/reference/home-automation/) traffic that fills it.

GopherTrunk is a land-mobile trunking scanner and does not decode Zigbee, Thread, or Matter,
so the CSA plays no part in its decode chain. It appears here as context for the wider RF
landscape, where these short-range mesh standards are an increasingly large share of what an
SDR sees in the unlicensed bands.

## Sources

[^home]: [Connectivity Standards Alliance](https://csa-iot.org/) — the alliance's official site, for the Zigbee and Matter specifications and certification programmes.
[^wiki]: [Connectivity Standards Alliance](https://en.wikipedia.org/wiki/Connectivity_Standards_Alliance) — Wikipedia, for the organisation's history, renaming, and standards.
