---
slug: ethernet
title: Ethernet
entry_type: concept
category: hw-networking
description: Ethernet is the dominant family of wired local-area networking standards, defining the cables, connectors, and frame format that carry data between computers on a LAN at speeds from 10 Mb/s to 400 Gb/s.
keywords: Ethernet, IEEE 802.3, twisted pair, RJ45, Gigabit Ethernet, frame, MAC address, wired networking, speed grades
aka: [IEEE 802.3]
infobox:
  - { label: Type, value: Wired LAN standard }
  - { label: Standard, value: IEEE 802.3 }
  - { label: Common cabling, value: Twisted pair (RJ45), fiber }
  - { label: Speeds, value: 10 Mb/s – 400 Gb/s }
  - { label: Addressing, value: 48-bit MAC }
see_also: [network-switch, network-interface-card, fiber-optic, power-over-ethernet, wi-fi, lan-and-wan]
cite_urls:
  - https://en.wikipedia.org/wiki/Ethernet
---

**Ethernet** is the dominant family of wired local-area networking standards, defining the cabling, signaling, and frame format that carry data between computers on a [LAN](/reference/lan-and-wan/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The byte layout of an Ethernet frame drawn as a row of labelled fields: a 7-byte preamble and 1-byte start delimiter, a 6-byte destination MAC, a 6-byte source MAC, a 2-byte type field, the 46 to 1500-byte payload, and a 4-byte frame check sequence at the end." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <g stroke-width="1.3">
      <rect x="14" y="40" width="44" height="30" fill-opacity="0.06"/>
      <rect x="58" y="40" width="26" height="30" fill-opacity="0.06"/>
      <rect x="84" y="40" width="60" height="30" fill-opacity="0.12"/>
      <rect x="144" y="40" width="60" height="30" fill-opacity="0.12"/>
      <rect x="204" y="40" width="40" height="30" fill-opacity="0.06"/>
      <rect x="244" y="40" width="150" height="30" fill-opacity="0.18"/>
      <rect x="394" y="40" width="52" height="30" fill-opacity="0.06"/>
    </g>
    <g stroke="none" text-anchor="middle" font-size="8">
      <text x="36" y="58">Preamble</text>
      <text x="71" y="58">SFD</text>
      <text x="114" y="55" font-weight="600">Dest</text>
      <text x="114" y="65">MAC</text>
      <text x="174" y="55" font-weight="600">Src</text>
      <text x="174" y="65">MAC</text>
      <text x="224" y="58">Type</text>
      <text x="319" y="55" font-weight="600">Payload</text>
      <text x="319" y="65">(data)</text>
      <text x="420" y="58">FCS</text>
    </g>
    <g stroke="none" text-anchor="middle" font-size="7.5" fill-opacity="0.85">
      <text x="36" y="84">7 B</text>
      <text x="71" y="84">1 B</text>
      <text x="114" y="84">6 B</text>
      <text x="174" y="84">6 B</text>
      <text x="224" y="84">2 B</text>
      <text x="319" y="84">46–1500 B</text>
      <text x="420" y="84">4 B</text>
    </g>
  </g>
  <text x="230" y="108" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">MAC addresses steer the frame; the FCS is a CRC that catches corruption in transit</text>
</svg>
<figcaption>An Ethernet frame: a preamble and start delimiter mark its beginning, destination and source MAC addresses say who it is for and from, a type field names the payload's protocol, and a frame check sequence (a CRC) at the tail lets the receiver detect corruption.</figcaption>
</figure>

## Overview

Standardized as **IEEE 802.3**, Ethernet packages data into *frames* tagged with source and destination MAC addresses and sends them over twisted-pair copper (terminated in the familiar 8-pin RJ45 plug) or over [fiber-optic](/reference/fiber-optic/) cable for longer runs and higher speeds. Each 48-bit MAC address is globally unique to a network interface, so a [switch](/reference/network-switch/) can learn which port each device sits behind and forward frames only where they need to go.

Speeds have climbed from the original 10 Mb/s through Fast and Gigabit Ethernet to 10, 100, and 400 Gb/s, each grade reusing the same frame format so old and new gear interoperate. Devices attach through a [NIC](/reference/network-interface-card/) and connect to a switch; the same cabling can also carry power via [Power over Ethernet](/reference/power-over-ethernet/).

A tail *frame check sequence* — a cyclic redundancy check — lets the receiver discard any frame that arrived corrupted, so upper layers see a clean byte stream or nothing at all. That reliability, plus predictable latency, is why wired Ethernet remains the default for anything that must not drop.

## Speed grades

The family has scaled by three orders of magnitude while keeping one frame format:

| Name | Rate | Typical medium | Era |
|------|------|----------------|-----|
| Ethernet | 10 Mb/s | Twisted pair / coax | 1980s |
| Fast Ethernet | 100 Mb/s | Cat 5 twisted pair | 1990s |
| Gigabit | 1 Gb/s | Cat 5e/6 twisted pair | 2000s |
| 10 GigE | 10 Gb/s | Cat 6a copper / fiber | 2000s–10s |
| 100–400 GigE | 100–400 Gb/s | Fiber | Data centers |

## Where it fits

Ethernet is the default for fixed, performance-sensitive connections where a cable is practical — servers, desktops, and infrastructure — while [Wi-Fi](/reference/wi-fi/) handles mobility. A wired Ethernet link gives a GopherTrunk capture node a stable, low-jitter path to stream decoded calls to a [server](/reference/server/), avoiding the contention and interference of a shared radio channel; paired with [PoE](/reference/power-over-ethernet/), a single cable can both feed and power a mast-mounted node.

## Sources

[^wiki]: [Ethernet](https://en.wikipedia.org/wiki/Ethernet) — Wikipedia, on the wired LAN standard family, the 802.3 frame format, and its speed grades.
