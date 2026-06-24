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

## Overview

A switch learns which MAC address lives on which port and builds a forwarding table, so traffic between two hosts does not flood every other port — this is what makes a modern wired LAN fast and quiet compared with the old shared hubs. Switches range from cheap unmanaged boxes with a handful of ports to managed units offering VLANs, monitoring, and [Power over Ethernet](/reference/power-over-ethernet/). A switch works within one network; moving traffic *between* networks is the job of a [router](/reference/router/).

## Where it fits

The switch is the wiring hub of a wired LAN, where each device's [NIC](/reference/network-interface-card/) plugs in. For a GopherTrunk site with several wired nodes — capture boxes, a storage [server](/reference/server/), a workstation — a small switch ties them together, and a PoE switch can even power a [Raspberry Pi](/reference/raspberry-pi/) node over the same cable.

## Sources

[^wiki]: [Network switch](https://en.wikipedia.org/wiki/Network_switch) — Wikipedia, on the device that forwards frames within a local network.
