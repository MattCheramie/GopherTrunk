---
slug: blade-server
title: Blade server
entry_type: hardware
category: hw-servers
description: A blade server is a stripped-down server module that slides into a shared enclosure, which supplies power, cooling, and networking to many blades at once for high density.
keywords: blade server, blade enclosure, blade chassis, server density, modular server
aka: [Server blade]
infobox:
  - { label: Type, value: Server form factor }
  - { label: Slots into, value: Blade enclosure }
  - { label: Shares, value: Power, cooling, networking }
  - { label: Optimized for, value: Density }
see_also: [rack-server, server, data-center, dedicated-server, virtualization, high-availability]
cite_urls:
  - https://en.wikipedia.org/wiki/Blade_server
---

A **blade server** is a stripped-down [server](/reference/server/) module that slides into a shared enclosure; the enclosure supplies power, cooling, and networking to many blades at once.[^wiki]

## Overview

Where a [rack server](/reference/rack-server/) is a complete, self-contained machine, a blade keeps only the essentials — CPU, memory, and often local storage — and offloads power supplies, fans, and switching to the chassis. A single enclosure can hold a dozen or more blades, cutting cabling and power overhead and packing more compute into each [data center](/reference/data-center/) cabinet. The trade-off is vendor lock-in to that enclosure and a larger up-front cost, so blades pay off mainly at scale.

## Trade-offs

Blades shine when you need many similar servers in a small footprint — for example as the compute pool behind heavy [virtualization](/reference/virtualization/) or a [high-availability](/reference/high-availability/) cluster. For a handful of machines, plain rack servers are simpler and cheaper. A GopherTrunk deployment almost never needs blade density; its bottleneck is RF capture at the antenna, not raw server compute.

## Sources

[^wiki]: [Blade server](https://en.wikipedia.org/wiki/Blade_server) — Wikipedia, on blade form factors and enclosures.
