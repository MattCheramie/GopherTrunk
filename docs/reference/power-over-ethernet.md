---
slug: power-over-ethernet
title: Power over Ethernet (PoE)
entry_type: concept
category: hw-networking
description: Power over Ethernet delivers electrical power to a device over the same twisted-pair cable that carries its network data, removing the need for a separate power supply at the device and easing mast- or ceiling-mounted installs.
keywords: Power over Ethernet, PoE, PoE+, PoE++, 802.3af, 802.3at, 802.3bt, powered device, PSE, injector, power class
aka: [PoE]
infobox:
  - { label: Type, value: Power-and-data over twisted pair }
  - { label: Standard, value: IEEE 802.3af / at / bt }
  - { label: Delivers, value: ~13 W up to ~90 W }
  - { label: Source, value: PoE switch or injector (PSE) }
  - { label: Load, value: Powered device (PD) }
see_also: [ethernet, network-switch, wireless-access-point, network-interface-card, lan-and-wan, raspberry-pi]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_over_Ethernet
---

**Power over Ethernet** (**PoE**) delivers electrical power to a device over the same twisted-pair cable that carries its [Ethernet](/reference/ethernet/) data, so the device needs no separate power supply.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A PoE link: a power sourcing equipment switch on the left injects DC power onto the same Ethernet cable that carries data, and a single cable runs to a powered device on the right — such as an access point on a mast — that draws both data and power from it." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <g stroke-width="1.3">
      <rect x="18" y="50" width="92" height="40" rx="4" fill-opacity="0.14"/>
      <rect x="350" y="50" width="92" height="40" rx="4" fill-opacity="0.14"/>
    </g>
    <g stroke-width="1.5" fill="none">
      <path d="M110 64 H350"/>
      <path d="M110 76 H350"/>
    </g>
    <g stroke-width="1.2">
      <path d="M344 64 l8 -4 v8 Z" fill="currentColor"/>
      <path d="M344 76 l8 -4 v8 Z" fill="currentColor"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8.5">
    <text x="64" y="66" font-weight="600">PoE switch</text>
    <text x="64" y="80" font-size="8">(PSE)</text>
    <text x="396" y="66" font-weight="600">access point</text>
    <text x="396" y="80" font-size="8">(powered device)</text>
    <text x="230" y="58" font-size="8" fill-opacity="0.95">data ⇄</text>
    <text x="230" y="90" font-size="8" fill-opacity="0.95">+ DC power →</text>
    <text x="230" y="120" font-size="8" fill-opacity="0.9">one cable carries both — no outlet needed at the device</text>
  </g>
</svg>
<figcaption>PoE runs power and data down the same cable: a power sourcing equipment switch (the PSE) injects DC alongside the Ethernet signal, and a single run reaches a powered device — here a mast-mounted access point — that needs no mains outlet of its own.</figcaption>
</figure>

## Overview

Standardized in IEEE 802.3 (af, at/PoE+, and bt/PoE++), PoE lets a *power sourcing equipment* (PSE) device — a PoE [switch](/reference/network-switch/) or an inline injector — feed a *powered device* (PD) anywhere from about 13 W up to roughly 90 W over the twisted pairs. The two ends first *negotiate*: the PSE detects a compliant device and its power class before applying full voltage, so it never over-drives gear that cannot accept power or damages a plain Ethernet device.

That single-cable delivery is ideal for equipment mounted where a power outlet is awkward: [wireless access points](/reference/wireless-access-point/) on ceilings, IP cameras on walls, VoIP phones, and small computers all run on one cable that carries both data and power. Because the power rides the same infrastructure, a centralized PoE switch can also power-cycle or monitor its attached devices remotely.

Later standards raised the ceiling — from the original ~13 W to ~90 W — to feed hungrier devices like pan-tilt-zoom cameras and multi-radio access points that early PoE could not support.

## Power classes

Successive standards lifted the wattage available at the device:

| Standard | Name | At the device | Typical loads |
|----------|------|---------------|---------------|
| 802.3af | PoE | ~12.95 W | VoIP phones, basic APs |
| 802.3at | PoE+ | ~25.5 W | PTZ cameras, dual-radio APs |
| 802.3bt Type 3 | PoE++ | ~51 W | Multi-radio APs, small PCs |
| 802.3bt Type 4 | PoE++ | ~71–90 W | Displays, high-power devices |

## Where it fits

PoE collapses two cables into one, which is exactly the appeal for an antenna-side node. A GopherTrunk capture box built on a PoE-capable [Raspberry Pi](/reference/raspberry-pi/) HAT can sit on a mast or in an attic with only a single network cable run to it — data down, power up — instead of needing mains wiring at the antenna. That keeps the install clean, removes a mains supply from near the front end, and lets the whole node be power-cycled from the switch in the rack below.

## Sources

[^wiki]: [Power over Ethernet](https://en.wikipedia.org/wiki/Power_over_Ethernet) — Wikipedia, on carrying power alongside data on Ethernet cabling and the 802.3af/at/bt power classes.
