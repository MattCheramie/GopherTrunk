---
slug: lan-and-wan
title: LAN & WAN
entry_type: concept
category: hw-networking
description: A LAN is a network covering a small area such as a home or office, while a WAN spans long distances connecting many networks; the internet is the largest WAN.
keywords: LAN, WAN, local area network, wide area network, internet, network scope, subnet
aka: [local area network, wide area network]
infobox:
  - { label: LAN, value: Local area network (one site) }
  - { label: WAN, value: Wide area network (many sites) }
  - { label: Joined by, value: Router / gateway }
  - { label: Largest WAN, value: The internet }
see_also: [router, gateway, network-switch, ip-address, ethernet, wi-fi]
cite_urls:
  - https://en.wikipedia.org/wiki/Local_area_network
  - https://en.wikipedia.org/wiki/Wide_area_network
---

A **LAN** (local area network) covers a small area such as a home, office, or building, while a **WAN** (wide area network) spans long distances and ties many networks together — the internet being the largest WAN of all.[^lan][^wan]

## Overview

A LAN connects nearby devices over [Ethernet](/reference/ethernet/) and [Wi-Fi](/reference/wi-fi/), usually through a [switch](/reference/network-switch/) and a single [IP](/reference/ip-address/) subnet, giving high speed and low latency within one site. A WAN links such sites across cities or continents over leased lines, fiber, or the public internet, at greater distance but typically lower speed and higher latency. The boundary between them is a [router](/reference/router/) acting as the [gateway](/reference/gateway/): the LAN side faces inward, the WAN side faces out.

## Where it fits

The LAN-versus-WAN distinction is mostly about *scope*, and it shapes how a distributed system is laid out. A GopherTrunk site keeps its capture nodes and storage [server](/reference/server/) on a fast local LAN, then reaches across the WAN only to publish results or pull in remote feeds — keeping the bandwidth-heavy raw data local and sending just the decoded output over the slower wide-area link.

## Sources

[^lan]: [Local area network](https://en.wikipedia.org/wiki/Local_area_network) — Wikipedia, on networks covering a small area.
[^wan]: [Wide area network](https://en.wikipedia.org/wiki/Wide_area_network) — Wikipedia, on networks spanning long distances.
