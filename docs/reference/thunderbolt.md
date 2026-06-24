---
slug: thunderbolt
title: Thunderbolt
entry_type: hardware
category: hw-networking
description: Thunderbolt is a high-speed connection standard that carries data, video, and power over a single cable, combining PCI Express and DisplayPort and now sharing the USB-C connector.
keywords: Thunderbolt, Thunderbolt 3, Thunderbolt 4, USB-C, PCI Express, DisplayPort, docking
aka: [Thunderbolt 3, Thunderbolt 4]
autolink: true
infobox:
  - { label: Type, value: High-speed I/O interface }
  - { label: Carries, value: Data, video, power }
  - { label: Connector, value: USB-C }
  - { label: Speed, value: 40 Gb/s (TB3/4), 80 Gb/s (TB5) }
see_also: [usb, pci-express, peripheral, network-interface-card, ethernet]
cite_urls:
  - https://en.wikipedia.org/wiki/Thunderbolt_(interface)
---

**Thunderbolt** is a high-speed connection standard that carries data, video, and power over a single cable, tunnelling [PCI Express](/reference/pci-express/) and DisplayPort and sharing the [USB-C](/reference/usb/) connector.[^wiki]

## Overview

Developed by Intel and Apple, Thunderbolt 3 and 4 run at 40 Gb/s over a USB-C plug, with Thunderbolt 5 reaching 80 Gb/s; a single cable can drive displays, attach external SSDs and GPUs, deliver power, and even act as a peer-to-peer network link. Because it tunnels PCI Express, a Thunderbolt port behaves almost like an external expansion slot, which is why one cable to a dock can fan out to Ethernet, displays, and storage at once. Thunderbolt 3 onward is compatible with USB-C but offers far more bandwidth than plain [USB](/reference/usb/).

## Where it fits

Thunderbolt is the fast end of the [peripheral](/reference/peripheral/) connection spectrum, suited to bandwidth-hungry external devices on laptops and workstations. For an SDR workflow it gives a single-cable path to a dock or a fast external SSD — handy when a GopherTrunk session is recording raw I/Q at high sample rates and writing many megabytes per second to disk.

## Sources

[^wiki]: [Thunderbolt (interface)](https://en.wikipedia.org/wiki/Thunderbolt_(interface)) — Wikipedia, on the high-speed Thunderbolt interface.
