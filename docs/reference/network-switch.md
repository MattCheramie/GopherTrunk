---
slug: network-switch
title: Network switch
entry_type: hardware
category: hw-networking
description: A network switch is a device that connects machines within a single local network, forwarding each Ethernet frame only to the port leading to its destination.
keywords: network switch, Ethernet switch, switched network, MAC address table, managed switch, ports
aka: [Ethernet switch, switch]
infobox:
  - { label: Type, value: Networking device }
  - { label: Operates at, value: Link layer (Ethernet) }
  - { label: Job, value: Forward frames within a LAN }
  - { label: Forwards by, value: MAC address }
see_also: [router, ethernet, network-interface-card, power-over-ethernet, lan-and-wan, gateway]
cite_urls:
  - https://en.wikipedia.org/wiki/Network_switch
---

A **network switch** is a device that connects multiple machines on the same [local network](/reference/lan-and-wan/), forwarding each [Ethernet](/reference/ethernet/) frame only out the port that leads to its destination.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 214" role="img" aria-label="Four hosts plug into a central switch; a frame from host A addressed to host C's MAC is forwarded only out the port leading to C, while the ports to B and D stay quiet — unlike an old hub, which would flood the frame to every port." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <rect x="24" y="26" width="74" height="32" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="61" y="46">host A</text>
    <rect x="134" y="26" width="74" height="32" rx="4" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/><text x="171" y="46">host B</text>
    <rect x="244" y="26" width="74" height="32" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/><text x="281" y="42">host C</text><text x="281" y="53" font-size="7">dest MAC</text>
    <rect x="354" y="26" width="74" height="32" rx="4" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/><text x="391" y="46">host D</text>
  </g>
  <rect x="150" y="120" width="140" height="46" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="220" y="140" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">switch</text>
  <text x="220" y="155" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">MAC → port table</text>
  <g stroke="currentColor" fill="none">
    <line x1="61" y1="58" x2="200" y2="120" stroke-width="1.6" marker-end="url(#nsw_ar)"/>
    <line x1="230" y1="120" x2="281" y2="58" stroke-width="1.6" marker-end="url(#nsw_ar)"/>
    <line x1="210" y1="120" x2="171" y2="58" stroke-width="1" stroke-opacity="0.3" stroke-dasharray="4 3"/>
    <line x1="240" y1="120" x2="391" y2="58" stroke-width="1" stroke-opacity="0.3" stroke-dasharray="4 3"/>
  </g>
  <text x="104" y="86" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">frame for C</text>
  <text x="332" y="100" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.5">no flood</text>
  <text x="220" y="196" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">forwarded to the one matching port — a hub copies it to all</text>
  <defs><marker id="nsw_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A switch learns which MAC sits on each port, so a frame goes only out the port for its destination — here just host C, while B and D stay idle. An older hub had no table and copied every frame to every port; that switched forwarding is why modern wired LANs are fast and quiet.</figcaption>
</figure>

## Overview

A switch learns which MAC address lives on which port and builds a forwarding table, so traffic between two hosts does not flood every other port — this is what makes a modern wired LAN fast and quiet compared with the old shared hubs. Switches range from cheap unmanaged boxes with a handful of ports to managed units offering VLANs, monitoring, and [Power over Ethernet](/reference/power-over-ethernet/). A switch works within one network; moving traffic *between* networks is the job of a [router](/reference/router/).

## Where it fits

The switch is the wiring hub of a wired LAN, where each device's [NIC](/reference/network-interface-card/) plugs in. For a GopherTrunk site with several wired nodes — capture boxes, a storage [server](/reference/server/), a workstation — a small switch ties them together, and a PoE switch can even power a [Raspberry Pi](/reference/raspberry-pi/) node over the same cable.

## Sources

[^wiki]: [Network switch](https://en.wikipedia.org/wiki/Network_switch) — Wikipedia, on the device that forwards frames within a local network.
