---
slug: gateway
title: Gateway
entry_type: concept
category: hw-networking
description: A gateway is a network node that joins two networks, acting as the entry and exit point through which traffic leaves a local network for another, such as the wider internet.
keywords: gateway, default gateway, network gateway, edge, protocol translation, egress
infobox:
  - { label: Type, value: Network entry/exit node }
  - { label: Job, value: Join one network to another }
  - { label: Common form, value: Router as default gateway }
  - { label: May also do, value: Protocol translation }
see_also: [router, lan-and-wan, ip-address, modem, network-switch, edge-computing]
cite_urls:
  - https://en.wikipedia.org/wiki/Gateway_(telecommunications)
---

A **gateway** is a network node that joins two networks, acting as the entry and exit point through which traffic leaves a local network for another — most commonly the wider internet.[^wiki]

## Overview

On a home or office network the *default gateway* is usually the [router](/reference/router/): the address a host sends packets to when their destination is not on the local [LAN](/reference/lan-and-wan/). More broadly, a gateway can translate between dissimilar networks or protocols — bridging an IoT radio network onto IP, for instance — doing more than a router that merely forwards [IP](/reference/ip-address/) packets. The term describes a *role* at a network boundary rather than one specific box.

## Where it fits

A gateway marks the edge of a network, which is also where [edge computing](/reference/edge-computing/) and protocol bridging often live. In a distributed GopherTrunk setup a small box can act as a gateway, gathering decoded data from several capture nodes on a private LAN and forwarding it out through one egress point to a remote [server](/reference/server/).

## Sources

[^wiki]: [Gateway (telecommunications)](https://en.wikipedia.org/wiki/Gateway_(telecommunications)) — Wikipedia, on the node that joins two networks.
