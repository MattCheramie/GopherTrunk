---
slug: thread-protocol
title: Thread
entry_type: protocol
category: wireless-data-iot
description: "Thread is a low-power IPv6 mesh networking protocol built on the IEEE 802.15.4 radio, giving self-healing, router-less home IoT networks that carry 6LoWPAN-compressed IPv6."
keywords: Thread, Thread Group, IEEE 802.15.4, IPv6 mesh, 6LoWPAN, low-power mesh, border router, Matter, home automation, self-healing network
aka: [Thread, Thread protocol]
autolink: true
infobox:
  - { label: Type, value: Low-power IPv6 mesh }
  - { label: Standards body, value: Thread Group }
  - { label: Introduced, value: "2014" }
  - { label: Access, value: "802.15.4 CSMA/CA, mesh" }
  - { label: Channel spacing, value: "5 MHz (2.4 GHz, 802.15.4)" }
  - { label: Modulation, value: OQPSK with DSSS (802.15.4) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [zigbee-802154, 6lowpan, matter-protocol, internet-of-things, connectivity-standards-alliance, home-automation]
cite_urls:
  - https://en.wikipedia.org/wiki/Thread_(network_protocol)
  - https://en.wikipedia.org/wiki/IEEE_802.15.4
---

**Thread** is a low-power wireless mesh networking protocol that carries native IPv6 to
[IoT](/reference/internet-of-things/) devices, running on the same
[IEEE 802.15.4](/reference/zigbee-802154/) 2.4 GHz radio as Zigbee but replacing its
proprietary networking stack with internet-standard IPv6 over
[6LoWPAN](/reference/6lowpan/).[^wiki] The result is a self-healing, router-less mesh in
which every full device is IP-addressable and can route for its neighbours.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A Thread mesh where routers interconnect redundantly and a border router bridges the mesh to a home IPv6 network and the internet, with no single point of failure." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="thrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.6">
    <line x1="130" y1="45" x2="230" y2="40"/><line x1="130" y1="45" x2="150" y2="110"/><line x1="230" y1="40" x2="150" y2="110"/><line x1="230" y1="40" x2="290" y2="100"/><line x1="150" y1="110" x2="290" y2="100"/><line x1="290" y1="100" x2="230" y2="40"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <circle cx="130" cy="45" r="12" fill="none" stroke="currentColor"/><text x="130" y="48">R</text>
    <circle cx="230" cy="40" r="12" fill="none" stroke="currentColor"/><text x="230" y="43">R</text>
    <circle cx="150" cy="110" r="12" fill="none" stroke="currentColor"/><text x="150" y="113">R</text>
    <circle cx="290" cy="100" r="12" fill="none" stroke="currentColor"/><text x="290" y="103">R</text>
  </g>
  <line x1="242" y1="40" x2="360" y2="60" stroke="currentColor" stroke-width="1" marker-end="url(#thrar)"/>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="360" y="48" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/><text x="395" y="58">border</text><text x="395" y="68">router</text>
  </g>
  <text x="230" y="140" text-anchor="middle" font-size="9" fill="currentColor">redundant router mesh · border router bridges to home IPv6 / internet</text>
</svg>
<figcaption>Thread routers interconnect redundantly with no single point of failure; a border router bridges the mesh to the home IPv6 network.</figcaption>
</figure>

## Overview

Thread reuses the proven 802.15.4 radio — OQPSK with DSSS at 250 kbit/s on 2.4 GHz — but
runs a modern IP stack above it. Devices are either *routers* (always-on, relaying
traffic) or *end devices* (often battery-powered, attaching to a parent router). The mesh
elects and re-elects routers automatically, so there is no single coordinator whose loss
breaks the network; this is Thread's headline reliability feature. A **border router**
connects the mesh to the home Wi-Fi/Ethernet LAN and the wider internet, translating
[6LoWPAN](/reference/6lowpan/)-compressed IPv6 to ordinary IPv6.

## Technical characteristics

| Property | Value |
|----------|-------|
| Radio | IEEE 802.15.4 (2.4 GHz OQPSK/DSSS) |
| Network layer | IPv6 over 6LoWPAN |
| Topology | Self-healing mesh, no single coordinator |
| Roles | Router, REED, end device, border router |
| Security | AES-based link and network keys |
| Data rate | 250 kbit/s (802.15.4 PHY) |

Because it is IP-native, a Thread device can be addressed like any host on the network,
which simplifies application protocols layered on top — most notably
[Matter](/reference/matter-protocol/).

## History

The Thread Group, founded in 2014 by Nest, ARM, Silicon Labs, and others, published
Thread 1.0 in 2015. Adoption accelerated once Matter chose Thread as one of its two
transport networks, and border-router support arrived in mainstream smart-home hubs and
phones.

## Deployment

Thread underpins a growing share of smart-home sensors, locks, and lighting, especially
Matter-certified devices. It coexists on the 802.15.4 radio with Zigbee (a device's chip
often supports either) and complements Wi-Fi for higher-bandwidth endpoints.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode Thread. Its 802.15.4 OQPSK/DSSS radio, AES-secured mesh,
and IPv6 stack are entirely outside GopherTrunk's land-mobile decode chain, and the
traffic is encrypted by default. Thread is relevant to an SDR operator only as 2.4 GHz
band occupancy.

## Sources

[^wiki]: [Thread (network protocol)](https://en.wikipedia.org/wiki/Thread_(network_protocol)) — Wikipedia, on the Thread IPv6 mesh, its 802.15.4 radio, border routers, self-healing mesh, and relationship to Matter.
