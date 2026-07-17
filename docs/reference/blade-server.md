---
slug: blade-server
title: Blade server
entry_type: hardware
category: hw-servers
description: A blade server is a stripped-down server module that slides into a shared enclosure, which supplies power, cooling, and networking to many blades at once for high density.
keywords: blade server, blade enclosure, blade chassis, server density, modular server, midplane, shared backplane
aka: [Server blade]
infobox:
  - { label: Type, value: Server form factor }
  - { label: Slots into, value: Blade enclosure }
  - { label: Shares, value: Power, cooling, networking }
  - { label: Optimized for, value: Density }
  - { label: Contrast, value: Self-contained rack server }
see_also: [rack-server, server, data-center, dedicated-server, virtualization, high-availability]
cite_urls:
  - https://en.wikipedia.org/wiki/Blade_server
---

A **blade server** is a stripped-down [server](/reference/server/) module that slides into a shared enclosure; the enclosure supplies power, cooling, and networking to many blades at once.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A blade enclosure drawn as a tall cabinet holding eight vertical blade modules side by side. A shared backplane at the rear connects every blade to common power supplies, cooling fans, and network switch modules at the bottom of the chassis." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <rect x="40" y="20" width="300" height="120" rx="4" fill-opacity="0.04" stroke-width="1.4"/>
    <g stroke-width="1.1">
      <rect x="52" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="86" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="120" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="154" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="188" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="222" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="256" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
      <rect x="290" y="30" width="30" height="88" rx="2" fill-opacity="0.14"/>
    </g>
    <text x="67" y="132" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.9">blade</text>
    <text x="190" y="14" text-anchor="middle" font-size="9" font-weight="600" stroke="none">Blade enclosure &#183; 8 slots</text>
    <line x1="340" y1="30" x2="340" y2="118" stroke-width="1.3" stroke-dasharray="3 2"/>
    <text x="392" y="60" text-anchor="middle" font-size="8" stroke="none">shared</text>
    <text x="392" y="72" text-anchor="middle" font-size="8" stroke="none">backplane</text>
    <text x="392" y="84" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">(power+data)</text>
    <rect x="40" y="150" width="94" height="30" rx="3" fill-opacity="0.10" stroke-width="1.2"/>
    <text x="87" y="169" text-anchor="middle" font-size="8" stroke="none">Power</text>
    <rect x="143" y="150" width="94" height="30" rx="3" fill-opacity="0.10" stroke-width="1.2"/>
    <text x="190" y="169" text-anchor="middle" font-size="8" stroke="none">Cooling fans</text>
    <rect x="246" y="150" width="94" height="30" rx="3" fill-opacity="0.10" stroke-width="1.2"/>
    <text x="293" y="169" text-anchor="middle" font-size="8" stroke="none">Network</text>
    <text x="190" y="194" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">shared by every blade in the chassis</text>
  </g>
</svg>
<figcaption>Each blade carries only CPU, memory, and local storage; power supplies, fans, and network switching live once in the chassis and are shared across all the slots, which is what makes blades so dense.</figcaption>
</figure>

## Overview

Where a [rack server](/reference/rack-server/) is a complete, self-contained machine, a blade keeps only the essentials — CPU, memory, and often local storage — and offloads power supplies, fans, and switching to the chassis. The blades plug into a common *midplane* or backplane that carries both power and data, so inserting a blade wires it into the enclosure's shared switches and power feeds in one motion.

A single enclosure can hold a dozen or more blades, cutting cabling and power overhead and packing more compute into each [data center](/reference/data-center/) cabinet. The trade-off is vendor lock-in to that enclosure and a larger up-front cost, so blades pay off mainly at scale.

## Anatomy

Blades and rack servers split the same components differently — the blade pushes shared infrastructure out into the chassis:

| Component | Rack server | Blade server |
|-----------|-------------|--------------|
| Power supply | Per server | Shared in enclosure |
| Cooling fans | Per server | Shared in enclosure |
| Network switch | External / top-of-rack | Module in enclosure |
| CPU, RAM, disk | On the server | On each blade |
| Density | Moderate | Highest |

That consolidation is the whole point: fewer cables, fewer power supplies, and more servers per cabinet — at the price of committing to one vendor's enclosure.

## Where it fits

Blades shine when you need many similar servers in a small footprint — for example as the compute pool behind heavy [virtualization](/reference/virtualization/) or a [high-availability](/reference/high-availability/) cluster. For a handful of machines, plain rack servers are simpler and cheaper. A GopherTrunk deployment almost never needs blade density; its bottleneck is RF capture at the antenna, not raw server compute.

## Sources

[^wiki]: [Blade server](https://en.wikipedia.org/wiki/Blade_server) — Wikipedia, on blade form factors, enclosures, and shared backplanes.
