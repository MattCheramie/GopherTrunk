---
slug: ip-address
title: IP address
entry_type: concept
category: hw-networking
description: An IP address is a numeric label assigned to each device on a network running the Internet Protocol, identifying the host and letting packets be routed to it.
keywords: IP address, IPv4, IPv6, Internet Protocol, subnet, DHCP, NAT, host address
aka: [Internet Protocol address]
infobox:
  - { label: Type, value: Network address }
  - { label: Identifies, value: A host on an IP network }
  - { label: Versions, value: IPv4 (32-bit), IPv6 (128-bit) }
  - { label: Assigned by, value: DHCP or statically }
see_also: [router, gateway, lan-and-wan, network-interface-card, network-switch, ethernet]
cite_urls:
  - https://en.wikipedia.org/wiki/IP_address
---

An **IP address** is a numeric label assigned to each device on a network that uses the Internet Protocol, identifying the host so that packets can be routed to it.[^wiki]

## Overview

IP addresses come in two versions: **IPv4**, a 32-bit value written as four dotted numbers (like `192.168.1.10`), and **IPv6**, a 128-bit form created because IPv4 addresses ran short. An address is bound to a device's [NIC](/reference/network-interface-card/) and is paired with a *subnet mask* that says which part identifies the network and which the host. Addresses are usually handed out automatically by DHCP, and on home networks private addresses are shared to the internet through a [router](/reference/router/) performing address translation (NAT).

## Where it fits

The IP address is how a [router](/reference/router/) and [gateway](/reference/gateway/) decide where a packet should go, both within a [LAN and across a WAN](/reference/lan-and-wan/). To reach a GopherTrunk capture node — to stream its decoded calls or open its web view — you connect to its IP address on the local network, so giving such nodes stable, known addresses makes a multi-node site far easier to manage.

## Sources

[^wiki]: [IP address](https://en.wikipedia.org/wiki/IP_address) — Wikipedia, on the numeric label that identifies a host on an IP network.
