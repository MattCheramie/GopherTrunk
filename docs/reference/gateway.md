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

<figure class="figure" markdown="0">
<svg viewBox="0 0 452 210" role="img" aria-label="Three hosts on a local network on the left all funnel through a single gateway node in the middle, which is the one entry and exit point at the network boundary, and out to the internet or another network on the right. A dashed line marks the boundary between the local and external sides." xmlns="http://www.w3.org/2000/svg">
  <text x="86" y="24" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85" font-weight="600">local network</text>
  <text x="382" y="24" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85" font-weight="600">external</text>
  <line x1="226" y1="18" x2="226" y2="192" stroke="currentColor" stroke-width="1" stroke-opacity="0.3" stroke-dasharray="5 4"/>
  <g fill="currentColor" font-size="7.5" text-anchor="middle">
    <circle cx="48" cy="55" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="48" y="58">host</text>
    <circle cx="48" cy="103" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="48" y="106">host</text>
    <circle cx="48" cy="151" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="48" y="154">host</text>
  </g>
  <rect x="170" y="76" width="104" height="54" rx="6" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.4"/>
  <text x="222" y="98" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">Gateway</text>
  <text x="222" y="112" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">one entry / exit point</text>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75" fill="none">
    <line x1="61" y1="57" x2="170" y2="96" marker-end="url(#gw_ar)"/>
    <line x1="61" y1="103" x2="170" y2="103" marker-end="url(#gw_ar)"/>
    <line x1="61" y1="149" x2="170" y2="110" marker-end="url(#gw_ar)"/>
  </g>
  <rect x="340" y="76" width="96" height="54" rx="8" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 4"/>
  <text x="388" y="99" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">Internet</text>
  <text x="388" y="113" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">/ another network</text>
  <line x1="274" y1="103" x2="340" y2="103" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.8" fill="none" marker-end="url(#gw_ar)"/>
  <text x="307" y="94" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.7">egress</text>
  <text x="226" y="204" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">all traffic between the two networks passes through the one boundary node</text>
  <defs><marker id="gw_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A gateway is the single node where one network meets another. Hosts on the local side send anything bound off-network to it, and it forwards traffic out the one egress point — often translating between the two networks along the way. It names a role at the boundary, not one specific box; on a home LAN the router fills it.</figcaption>
</figure>

## Overview

On a home or office network the *default gateway* is usually the [router](/reference/router/): the address a host sends packets to when their destination is not on the local [LAN](/reference/lan-and-wan/). More broadly, a gateway can translate between dissimilar networks or protocols — bridging an IoT radio network onto IP, for instance — doing more than a router that merely forwards [IP](/reference/ip-address/) packets. The term describes a *role* at a network boundary rather than one specific box.

## Where it fits

A gateway marks the edge of a network, which is also where [edge computing](/reference/edge-computing/) and protocol bridging often live. In a distributed GopherTrunk setup a small box can act as a gateway, gathering decoded data from several capture nodes on a private LAN and forwarding it out through one egress point to a remote [server](/reference/server/).

## Sources

[^wiki]: [Gateway (telecommunications)](https://en.wikipedia.org/wiki/Gateway_(telecommunications)) — Wikipedia, on the node that joins two networks.
