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

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 168" role="img" aria-label="A Thunderbolt daisy-chain runs from the host to a dock, on to a display, and then to an external SSD, each hop a bidirectional 40 gigabit-per-second link that tunnels PCI Express, DisplayPort, and power along the same cable." xmlns="http://www.w3.org/2000/svg">
  <text x="235" y="36" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">one cable carries PCIe + DisplayPort + power</text>
  <g font-size="9" text-anchor="middle">
    <rect x="16" y="58" width="86" height="48" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/><text x="59" y="86" fill="currentColor" font-weight="600">host</text>
    <rect x="132" y="58" width="86" height="48" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="175" y="86" fill="currentColor">dock</text>
    <rect x="248" y="58" width="86" height="48" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="291" y="86" fill="currentColor">display</text>
    <rect x="364" y="58" width="90" height="48" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="409" y="86" fill="currentColor">SSD</text>
  </g>
  <g stroke="currentColor" stroke-width="1.5" fill="none">
    <line x1="102" y1="82" x2="132" y2="82" marker-start="url(#tb_ar)" marker-end="url(#tb_ar)"/>
    <line x1="218" y1="82" x2="248" y2="82" marker-start="url(#tb_ar)" marker-end="url(#tb_ar)"/>
    <line x1="334" y1="82" x2="364" y2="82" marker-start="url(#tb_ar)" marker-end="url(#tb_ar)"/>
  </g>
  <text x="235" y="132" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">40 Gb/s per hop — each device passes the link along to the next</text>
  <defs><marker id="tb_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Thunderbolt lets devices daisy-chain: one cable from the host to the first device, another on to the next, and so on. Because it tunnels PCI Express and DisplayPort together at 40 Gb/s, a single chain can carry storage, displays, and power at once — closer to an external expansion slot than an ordinary cable.</figcaption>
</figure>

## Overview

Developed by Intel and Apple, Thunderbolt 3 and 4 run at 40 Gb/s over a USB-C plug, with Thunderbolt 5 reaching 80 Gb/s; a single cable can drive displays, attach external SSDs and GPUs, deliver power, and even act as a peer-to-peer network link. Because it tunnels PCI Express, a Thunderbolt port behaves almost like an external expansion slot, which is why one cable to a dock can fan out to Ethernet, displays, and storage at once. Thunderbolt 3 onward is compatible with USB-C but offers far more bandwidth than plain [USB](/reference/usb/).

## Where it fits

Thunderbolt is the fast end of the [peripheral](/reference/peripheral/) connection spectrum, suited to bandwidth-hungry external devices on laptops and workstations. For an SDR workflow it gives a single-cable path to a dock or a fast external SSD — handy when a GopherTrunk session is recording raw I/Q at high sample rates and writing many megabytes per second to disk.

## Sources

[^wiki]: [Thunderbolt (interface)](https://en.wikipedia.org/wiki/Thunderbolt_(interface)) — Wikipedia, on the high-speed Thunderbolt interface.
