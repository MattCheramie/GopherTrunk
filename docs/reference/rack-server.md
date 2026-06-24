---
slug: rack-server
title: Rack server
entry_type: hardware
category: hw-servers
description: A rack server is a server built in a flat, standardized chassis that bolts into a 19-inch equipment rack, letting many machines be stacked densely in a data center.
keywords: rack server, rack-mount, 19-inch rack, rack unit, 1U, 2U, server chassis
aka: [Rack-mount server]
infobox:
  - { label: Type, value: Server form factor }
  - { label: Mounts in, value: 19-inch rack }
  - { label: Height unit, value: "U (1.75 in / 44.45 mm)" }
  - { label: Common sizes, value: 1U, 2U, 4U }
see_also: [server, blade-server, data-center, dedicated-server, network-attached-storage, colocation]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Rack_unit
  - https://en.wikipedia.org/wiki/19-inch_rack
---

A **rack server** is a [server](/reference/server/) built in a flat, standardized chassis that bolts into a 19-inch equipment rack, so many machines can be stacked densely in a cabinet.[^rack]

## Overview

Height is measured in *rack units* (**U**), each 1.75 in (44.45 mm) tall.[^ru] A 1U "pizza box" is the slimmest common server; 2U and 4U chassis trade density for more drive bays, expansion slots, and cooling. The 19-inch rack standard means servers, switches, and storage from different vendors share the same rails, power distribution, and cable management. Rack servers fill the cabinets of a [data center](/reference/data-center/), where airflow runs front-to-back through cold and hot aisles.

## Where it fits

The rack server is the workhorse form factor for a [dedicated server](/reference/dedicated-server/) or a machine you place via [colocation](/reference/colocation/). When you need even higher density and shared power, a [blade server](/reference/blade-server/) packs more compute into the same space. Storage often rides alongside as [network-attached storage](/reference/network-attached-storage/). For most GopherTrunk users a rack server is overkill — a small box near the antenna is enough — but a rack machine is a natural home for long-term storage and serving of decoded data.

## Sources

[^rack]: [19-inch rack](https://en.wikipedia.org/wiki/19-inch_rack) — Wikipedia, on the rack standard and mounting.
[^ru]: [Rack unit](https://en.wikipedia.org/wiki/Rack_unit) — Wikipedia, on the U height unit.
