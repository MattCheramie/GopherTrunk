---
slug: lan-and-wan
title: LAN & WAN
entry_type: concept
category: hw-networking
description: A LAN is a network covering a small area such as a home or office, while a WAN spans long distances connecting many networks; a router joins the two, and the internet is the largest WAN.
keywords: LAN, WAN, local area network, wide area network, internet, network scope, subnet, router, gateway, backhaul
aka: [local area network, wide area network]
infobox:
  - { label: LAN, value: Local area network (one site) }
  - { label: WAN, value: Wide area network (many sites) }
  - { label: Joined by, value: Router / gateway }
  - { label: LAN reach, value: One building or campus }
  - { label: Largest WAN, value: The internet }
see_also: [router, gateway, network-switch, ip-address, ethernet, wi-fi]
cite_urls:
  - https://en.wikipedia.org/wiki/Local_area_network
  - https://en.wikipedia.org/wiki/Wide_area_network
---

A **LAN** (local area network) covers a small area such as a home, office, or building, while a **WAN** (wide area network) spans long distances and ties many networks together — the internet being the largest WAN of all.[^lan][^wan]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A network topology: three local devices connect to a switch, the switch to a router, and the router reaches out across a wide-area cloud to the internet. A dashed box groups the devices and switch as the local-area network, with the router straddling the boundary to the wide-area side." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <rect x="10" y="18" width="250" height="114" rx="6" fill="none" stroke-width="1" stroke-dasharray="4 3" stroke-opacity="0.7"/>
    <g stroke-width="1.3">
      <rect x="24" y="30" width="34" height="22" rx="3" fill-opacity="0.1"/>
      <rect x="24" y="62" width="34" height="22" rx="3" fill-opacity="0.1"/>
      <rect x="24" y="94" width="34" height="22" rx="3" fill-opacity="0.1"/>
      <rect x="110" y="58" width="46" height="30" rx="3" fill-opacity="0.16"/>
      <rect x="196" y="58" width="46" height="30" rx="3" fill-opacity="0.16"/>
    </g>
    <g stroke-width="1.2" fill="none">
      <line x1="58" y1="41" x2="110" y2="70"/>
      <line x1="58" y1="73" x2="110" y2="73"/>
      <line x1="58" y1="105" x2="110" y2="76"/>
      <line x1="156" y1="73" x2="196" y2="73"/>
      <line x1="242" y1="73" x2="300" y2="73"/>
    </g>
    <path d="M312 62 q-14 0 -14 12 q-14 2 -10 14 q2 10 16 8 h58 q18 2 20 -12 q2 -14 -14 -16 q-2 -12 -18 -8 q-8 -8 -20 -2 q-10 -2 -18 4 Z" fill-opacity="0.1" stroke-width="1.3"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8">
    <text x="41" y="128" font-size="8">devices</text>
    <text x="133" y="77" font-weight="600">switch</text>
    <text x="219" y="77" font-weight="600">router</text>
    <text x="356" y="80" font-weight="600">internet</text>
    <text x="135" y="16" font-size="8.5" font-weight="600">LAN (one site)</text>
    <text x="356" y="112" font-size="8.5" font-weight="600">WAN</text>
  </g>
</svg>
<figcaption>Within one site, devices reach a switch and share a single subnet — that is the LAN. The router at the edge is the gateway: its inside face joins the LAN, its outside face reaches across the WAN to other networks and the internet.</figcaption>
</figure>

## Overview

A LAN connects nearby devices over [Ethernet](/reference/ethernet/) and [Wi-Fi](/reference/wi-fi/), usually through a [switch](/reference/network-switch/) and a single [IP](/reference/ip-address/) subnet, giving high speed and low latency within one site. Because every device is close and under one administration, a LAN can run fast and cheap, and traffic between two local machines never has to leave the building.

A WAN links such sites across cities or continents over leased lines, fiber, or the public internet, at greater distance but typically lower speed and higher latency than a LAN. The boundary between them is a [router](/reference/router/) acting as the [gateway](/reference/gateway/): the LAN side faces inward toward local devices, the WAN side faces out toward the wider network.

The distinction is one of *scope*, not of any single technology — the same Ethernet and IP that run a LAN also carry WAN traffic — and it governs where you put fast local links versus slower, costlier long-haul ones.

## LAN vs WAN

The two differ mainly in reach, and everything else follows from that:

| Trait | LAN | WAN |
|-------|-----|-----|
| Span | One site or building | Cities to continents |
| Speed | High (1–100 Gb/s) | Often lower, varies |
| Latency | Very low | Higher |
| Owned by | You / one organization | Carriers, shared |
| Example | Office network | The internet |

## Where it fits

The LAN-versus-WAN distinction shapes how a distributed system is laid out. A GopherTrunk site keeps its capture nodes and storage [server](/reference/server/) on a fast local LAN, then reaches across the WAN only to publish results or pull in remote feeds — keeping the bandwidth-heavy raw IQ data local and sending just the compact decoded output over the slower, costlier wide-area link. Getting that split right is the difference between a responsive multi-node site and one that saturates its uplink.

## Sources

[^lan]: [Local area network](https://en.wikipedia.org/wiki/Local_area_network) — Wikipedia, on networks covering a small area.
[^wan]: [Wide area network](https://en.wikipedia.org/wiki/Wide_area_network) — Wikipedia, on networks spanning long distances.
