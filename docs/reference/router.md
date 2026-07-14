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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A router sits between two networks — a home LAN on the 192.168.1.0/24 subnet with two hosts on the left, and the wider internet on the right — reading each packet's destination IP and forwarding it from one network to the other." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="40" width="150" height="120" rx="6" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1.1" stroke-dasharray="5 3"/>
  <text x="91" y="34" text-anchor="middle" font-size="8.5" fill="currentColor" font-weight="600">LAN · 192.168.1.0/24</text>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="36" y="66" width="110" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="91" y="85">host .10</text>
    <rect x="36" y="106" width="110" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="91" y="125">host .20</text>
  </g>
  <rect x="198" y="80" width="66" height="46" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="231" y="100" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">router</text>
  <text x="231" y="114" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">routing table</text>
  <rect x="300" y="66" width="144" height="74" rx="6" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.2"/>
  <text x="372" y="99" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">internet / WAN</text>
  <text x="372" y="114" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">other networks</text>
  <g stroke="currentColor" stroke-width="1.5" fill="none">
    <line x1="166" y1="103" x2="198" y2="103" marker-end="url(#rtr_ar)"/>
    <line x1="264" y1="103" x2="300" y2="103" marker-end="url(#rtr_ar)"/>
  </g>
  <text x="231" y="150" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">forwards by destination IP</text>
  <text x="230" y="184" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">joins separate networks — a switch only moves traffic within one</text>
  <defs><marker id="rtr_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A router straddles two networks — here a home LAN and the internet — and reads each packet's destination IP against a routing table to pick the next hop across the boundary. That is what sets it apart from a switch, which only forwards frames inside a single network.</figcaption>
</figure>

## Overview

A router reads the destination [IP address](/reference/ip-address/) on each packet and consults a *routing table* to choose the next hop, moving traffic between networks rather than within one — that distinguishes it from a [switch](/reference/network-switch/), which forwards frames inside a single network. The familiar home "router" is really several devices in one box: a router, a [switch](/reference/network-switch/), a [wireless access point](/reference/wireless-access-point/), and often a [modem](/reference/modem/), bridging a home [LAN](/reference/lan-and-wan/) to the internet and performing address translation (NAT).

## Where it fits

A router is the [gateway](/reference/gateway/) between a [local network and the wider WAN](/reference/lan-and-wan/). On a GopherTrunk network it lets a capture node, a storage [server](/reference/server/), and a viewing laptop reach each other and, where desired, the internet — while keeping the capture node addressable on the LAN.

## Sources

[^wiki]: [Router (computing)](https://en.wikipedia.org/wiki/Router_(computing)) — Wikipedia, on the device that forwards packets between networks.
