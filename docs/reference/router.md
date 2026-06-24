---
slug: router
title: Router
entry_type: hardware
category: hw-networking
description: A router is a networking device that forwards data packets between networks, choosing the path each packet takes and joining a local network to the wider internet.
keywords: router, packet routing, default gateway, home router, NAT, routing table
infobox:
  - { label: Type, value: Networking device }
  - { label: Operates at, value: Network layer (IP) }
  - { label: Job, value: Forward packets between networks }
  - { label: Common role, value: Home/office internet gateway }
see_also: [network-switch, gateway, ip-address, modem, wireless-access-point, lan-and-wan]
cite_urls:
  - https://en.wikipedia.org/wiki/Router_(computing)
---

A **router** is a networking device that forwards data packets between separate networks, deciding which path each packet should take toward its destination.[^wiki]

## Overview

A router reads the destination [IP address](/reference/ip-address/) on each packet and consults a *routing table* to choose the next hop, moving traffic between networks rather than within one — that distinguishes it from a [switch](/reference/network-switch/), which forwards frames inside a single network. The familiar home "router" is really several devices in one box: a router, a [switch](/reference/network-switch/), a [wireless access point](/reference/wireless-access-point/), and often a [modem](/reference/modem/), bridging a home [LAN](/reference/lan-and-wan/) to the internet and performing address translation (NAT).

## Where it fits

A router is the [gateway](/reference/gateway/) between a [local network and the wider WAN](/reference/lan-and-wan/). On a GopherTrunk network it lets a capture node, a storage [server](/reference/server/), and a viewing laptop reach each other and, where desired, the internet — while keeping the capture node addressable on the LAN.

## Sources

[^wiki]: [Router (computing)](https://en.wikipedia.org/wiki/Router_(computing)) — Wikipedia, on the device that forwards packets between networks.
