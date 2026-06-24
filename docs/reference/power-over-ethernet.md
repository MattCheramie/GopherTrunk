---
slug: power-over-ethernet
title: Power over Ethernet (PoE)
entry_type: concept
category: hw-networking
description: Power over Ethernet delivers electrical power to a device over the same twisted-pair cable that carries its network data, removing the need for a separate power supply at the device.
keywords: Power over Ethernet, PoE, PoE+, 802.3af, 802.3at, 802.3bt, powered device, injector
aka: [PoE]
infobox:
  - { label: Type, value: Power-and-data over twisted pair }
  - { label: Standard, value: IEEE 802.3af / at / bt }
  - { label: Delivers, value: ~13 W up to ~90 W }
  - { label: Source, value: PoE switch or injector }
see_also: [ethernet, network-switch, wireless-access-point, network-interface-card, lan-and-wan, router]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_over_Ethernet
---

**Power over Ethernet** (**PoE**) delivers electrical power to a device over the same twisted-pair cable that carries its [Ethernet](/reference/ethernet/) data, so the device needs no separate power supply.[^wiki]

## Overview

Standardized in IEEE 802.3 (af, at/PoE+, and bt), PoE lets a *power sourcing equipment* device — a PoE [switch](/reference/network-switch/) or an inline injector — feed a *powered device* anywhere from about 13 W up to roughly 90 W, negotiating the level so it never over-drives a device that can't accept it. This is ideal for gear mounted where a power outlet is awkward: [wireless access points](/reference/wireless-access-point/), IP cameras, and small computers all run on a single cable.

## What it's for

PoE collapses two cables into one, which is exactly the appeal for an antenna-side node. A GopherTrunk capture box built on a PoE-capable [Raspberry Pi](/reference/raspberry-pi/) HAT can sit on a mast or in an attic with only a network cable run to it — data down, power up — instead of needing mains wiring at the antenna.

## Sources

[^wiki]: [Power over Ethernet](https://en.wikipedia.org/wiki/Power_over_Ethernet) — Wikipedia, on carrying power alongside data on Ethernet cabling.
