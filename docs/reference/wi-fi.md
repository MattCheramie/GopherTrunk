---
slug: wi-fi
title: Wi-Fi
entry_type: concept
category: hw-networking
description: Wi-Fi is the family of wireless local-area networking standards that let devices connect to a network over radio in the 2.4, 5, and 6 GHz bands instead of a cable.
keywords: Wi-Fi, IEEE 802.11, WLAN, wireless networking, 2.4 GHz, 5 GHz, Wi-Fi 6, access point
aka: [WiFi, IEEE 802.11, WLAN]
infobox:
  - { label: Type, value: Wireless LAN standard }
  - { label: Standard, value: IEEE 802.11 }
  - { label: Bands, value: 2.4, 5, 6 GHz }
  - { label: Reach, value: Tens of metres indoors }
see_also: [wireless-access-point, ethernet, bluetooth, network-interface-card, router, electromagnetic-spectrum]
cite_urls:
  - https://en.wikipedia.org/wiki/Wi-Fi
---

**Wi-Fi** is the family of wireless local-area networking standards that let devices join a network over radio, in the 2.4, 5, and 6 GHz bands, instead of a wired connection.[^wiki]

## Overview

Built on the **IEEE 802.11** standards, Wi-Fi connects client devices to a [wireless access point](/reference/wireless-access-point/), which bridges them onto the wired [LAN](/reference/lan-and-wan/). Successive generations — now labelled Wi-Fi 4, 5, 6, and 7 — have raised throughput and efficiency using wider channels and smarter modulation, though real-world range and speed depend on interference and walls. Because it shares unlicensed [spectrum](/reference/electromagnetic-spectrum/), Wi-Fi competes with [Bluetooth](/reference/bluetooth/), microwaves, and neighbouring networks for airtime.

## What it's for

Wi-Fi trades the raw stability of [Ethernet](/reference/ethernet/) for mobility and easy deployment, making it the default for laptops, phones, and IoT devices. A GopherTrunk node can report over Wi-Fi where running a cable is impractical, but the 2.4 GHz band is busy and can desensitize a nearby receiver, so a wired link or careful placement is wiser when the antenna and the radio share a roof.

## Sources

[^wiki]: [Wi-Fi](https://en.wikipedia.org/wiki/Wi-Fi) — Wikipedia, on the IEEE 802.11 wireless networking family.
