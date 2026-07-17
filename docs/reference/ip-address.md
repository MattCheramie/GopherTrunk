---
slug: ip-address
title: IP address
entry_type: concept
category: hw-networking
description: An IP address is a numeric label assigned to each device on a network running the Internet Protocol, identifying the host and letting packets be routed to it; IPv4 uses 32 bits, IPv6 uses 128.
keywords: IP address, IPv4, IPv6, Internet Protocol, subnet, subnet mask, DHCP, NAT, dotted quad, host address
aka: [Internet Protocol address]
infobox:
  - { label: Type, value: Network address }
  - { label: Identifies, value: A host on an IP network }
  - { label: Versions, value: IPv4 (32-bit), IPv6 (128-bit) }
  - { label: Assigned by, value: DHCP or statically }
  - { label: Split by, value: Subnet mask (network / host) }
see_also: [router, gateway, lan-and-wan, network-interface-card, network-switch, ethernet]
cite_urls:
  - https://en.wikipedia.org/wiki/IP_address
---

An **IP address** is a numeric label assigned to each device on a network that uses the Internet Protocol, identifying the host so that packets can be routed to it.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An IPv4 address 192.168.1.10 broken into its four octets. Each dotted number is shown as an 8-bit binary group, and a subnet mask marks the first three octets as the network portion and the last as the host portion." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <g stroke-width="1.3">
      <rect x="24" y="30" width="92" height="30" fill-opacity="0.12"/>
      <rect x="124" y="30" width="92" height="30" fill-opacity="0.12"/>
      <rect x="224" y="30" width="92" height="30" fill-opacity="0.12"/>
      <rect x="332" y="30" width="92" height="30" fill-opacity="0.20"/>
    </g>
    <g stroke="none" text-anchor="middle" font-family="ui-monospace, monospace">
      <text x="70" y="50" font-size="12" font-weight="600">192</text>
      <text x="170" y="50" font-size="12" font-weight="600">168</text>
      <text x="270" y="50" font-size="12" font-weight="600">1</text>
      <text x="378" y="50" font-size="12" font-weight="600">10</text>
      <text x="70" y="78" font-size="8" fill-opacity="0.85">11000000</text>
      <text x="170" y="78" font-size="8" fill-opacity="0.85">10101000</text>
      <text x="270" y="78" font-size="8" fill-opacity="0.85">00000001</text>
      <text x="378" y="78" font-size="8" fill-opacity="0.85">00001010</text>
    </g>
    <line x1="24" y1="96" x2="316" y2="96" stroke-width="1.4"/>
    <line x1="332" y1="96" x2="424" y2="96" stroke-width="1.4"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8.5">
    <text x="170" y="112" font-weight="600">network (mask 255.255.255.0)</text>
    <text x="378" y="112" font-weight="600">host</text>
    <text x="230" y="138" fill-opacity="0.9">four 8-bit octets = 32 bits · the subnet mask says where network ends and host begins</text>
  </g>
</svg>
<figcaption>An IPv4 address is 32 bits written as four dotted octets. The subnet mask splits it into a network portion — shared by every host on the same LAN — and a host portion that uniquely names the device within that network.</figcaption>
</figure>

## Overview

IP addresses come in two versions: **IPv4**, a 32-bit value written as four dotted numbers (like `192.168.1.10`), and **IPv6**, a 128-bit form created because the pool of IPv4 addresses ran short. An address is bound to a device's [NIC](/reference/network-interface-card/) and is paired with a *subnet mask* that says which leading bits identify the network and which the individual host.

Addresses are usually handed out automatically by *DHCP* when a device joins a network, though servers and infrastructure often get *static* addresses that never change. On home and office networks, a block of *private* addresses is reused behind a [router](/reference/router/) that performs *network address translation* (NAT), sharing one public address among many internal devices.

Routing works entirely on these numbers: every [router](/reference/router/) between source and destination looks only at the destination address to decide the next hop, which is what lets a packet cross the world without any single device knowing the whole path.

## IPv4 vs IPv6

The jump to IPv6 was driven by exhaustion of the older, smaller address space:

| Trait | IPv4 | IPv6 |
|-------|------|------|
| Size | 32 bits | 128 bits |
| Notation | Dotted decimal `192.168.1.10` | Hex groups `2001:db8::1` |
| Address count | ~4.3 billion | ~3.4 × 10³⁸ |
| Typical assignment | DHCP + NAT | DHCP / autoconfiguration |
| Why it exists | Original design | Relieve IPv4 exhaustion |

## Where it fits

The IP address is how a [router](/reference/router/) and [gateway](/reference/gateway/) decide where a packet should go, both within a [LAN and across a WAN](/reference/lan-and-wan/). To reach a GopherTrunk capture node — to stream its decoded calls or open the daemon's web console — you connect to its IP address on the local network, so giving each node a stable, known address (static, or a DHCP reservation) makes a multi-node site far easier to manage and script against.

## Sources

[^wiki]: [IP address](https://en.wikipedia.org/wiki/IP_address) — Wikipedia, on the numeric label that identifies a host, subnet masks, and IPv4 versus IPv6.
