---
slug: wireless-access-point
title: Wireless access point
entry_type: hardware
category: hw-networking
description: A wireless access point is a device that lets Wi-Fi clients join a wired network, bridging radio links onto the LAN and extending wireless coverage across a site.
keywords: wireless access point, WAP, access point, Wi-Fi AP, SSID, mesh, roaming
aka: [WAP, access point, AP]
infobox:
  - { label: Type, value: Networking device }
  - { label: Job, value: Bridge Wi-Fi clients to wired LAN }
  - { label: Standard, value: IEEE 802.11 (Wi-Fi) }
  - { label: Powered by, value: Mains or PoE }
see_also: [wi-fi, network-switch, router, power-over-ethernet, ethernet, lan-and-wan]
cite_urls:
  - https://en.wikipedia.org/wiki/Wireless_access_point
---

A **wireless access point** (WAP, or just AP) is a device that lets [Wi-Fi](/reference/wi-fi/) clients join a wired network, bridging their radio links onto the [LAN](/reference/lan-and-wan/).[^wiki]

## Overview

An access point broadcasts one or more named networks (SSIDs) and relays traffic between associated wireless clients and the wired side, usually plugging into a [switch](/reference/network-switch/). Standalone APs are common in larger buildings, where several units share an SSID so devices roam seamlessly, often as a controller-managed or mesh system. Many are powered over the data cable by [Power over Ethernet](/reference/power-over-ethernet/), simplifying mounting. In a home, the AP is typically built into the same box as the [router](/reference/router/) and [modem](/reference/modem/).

## Where it fits

The AP is the radio edge of a wired network — the piece that actually puts Wi-Fi on the air, distinct from the routing and switching behind it. For a GopherTrunk node that must report wirelessly, a well-placed AP gives a cleaner link than a distant home router, though a wired [Ethernet](/reference/ethernet/) drop is still preferable near sensitive RF gear.

## Sources

[^wiki]: [Wireless access point](https://en.wikipedia.org/wiki/Wireless_access_point) — Wikipedia, on the device that bridges Wi-Fi clients onto a wired network.
