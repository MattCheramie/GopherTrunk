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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 208" role="img" aria-label="Three wireless clients — a laptop, a phone, and a tablet — associate over Wi-Fi shown as dashed radio links with a central access point, which bridges their traffic over a solid wired Ethernet link to the LAN switch." xmlns="http://www.w3.org/2000/svg">
  <text x="52" y="24" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">Wi-Fi clients</text>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="18" y="44" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="53" y="62">laptop</text>
    <rect x="18" y="92" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="53" y="110">phone</text>
    <rect x="18" y="140" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="53" y="158">tablet</text>
  </g>
  <rect x="176" y="82" width="98" height="48" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="225" y="102" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">access point</text>
  <text x="225" y="117" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">SSID · bridge</text>
  <rect x="350" y="82" width="94" height="48" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <text x="397" y="102" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">switch</text>
  <text x="397" y="117" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">wired LAN</text>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.55" fill="none" stroke-dasharray="4 3">
    <line x1="88" y1="58" x2="176" y2="98"/>
    <line x1="88" y1="106" x2="176" y2="106"/>
    <line x1="88" y1="154" x2="176" y2="114"/>
  </g>
  <text x="128" y="150" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.8">Wi-Fi · 802.11</text>
  <line x1="274" y1="106" x2="350" y2="106" stroke="currentColor" stroke-width="1.6" fill="none" marker-end="url(#wap_ar)"/>
  <text x="312" y="98" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">Ethernet</text>
  <text x="230" y="192" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">radio links on one side, a wired uplink on the other</text>
  <defs><marker id="wap_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An access point is a bridge with a foot in each world: wireless clients associate to its SSID over the air (dashed 802.11 links), and it relays their traffic down a wired Ethernet uplink to the switch. That is the whole job — it puts Wi-Fi on the air for a network that is otherwise wired.</figcaption>
</figure>

## Overview

An access point broadcasts one or more named networks (SSIDs) and relays traffic between associated wireless clients and the wired side, usually plugging into a [switch](/reference/network-switch/). Standalone APs are common in larger buildings, where several units share an SSID so devices roam seamlessly, often as a controller-managed or mesh system. Many are powered over the data cable by [Power over Ethernet](/reference/power-over-ethernet/), simplifying mounting. In a home, the AP is typically built into the same box as the [router](/reference/router/) and [modem](/reference/modem/).

## Where it fits

The AP is the radio edge of a wired network — the piece that actually puts Wi-Fi on the air, distinct from the routing and switching behind it. For a GopherTrunk node that must report wirelessly, a well-placed AP gives a cleaner link than a distant home router, though a wired [Ethernet](/reference/ethernet/) drop is still preferable near sensitive RF gear.

## Sources

[^wiki]: [Wireless access point](https://en.wikipedia.org/wiki/Wireless_access_point) — Wikipedia, on the device that bridges Wi-Fi clients onto a wired network.
