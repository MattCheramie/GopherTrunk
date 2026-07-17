---
slug: rack-server
title: Rack server
entry_type: hardware
category: hw-servers
description: A rack server is a server built in a flat, standardized chassis that bolts into a 19-inch equipment rack, letting many machines be stacked densely in a data center.
keywords: rack server, rack-mount, 19-inch rack, rack unit, 1U, 2U, 4U, server chassis, cold aisle
aka: [Rack-mount server]
infobox:
  - { label: Type, value: Server form factor }
  - { label: Mounts in, value: 19-inch rack }
  - { label: Height unit, value: "U (1.75 in / 44.45 mm)" }
  - { label: Common sizes, value: 1U, 2U, 4U }
  - { label: Airflow, value: Front-to-back }
see_also: [server, blade-server, data-center, dedicated-server, network-attached-storage, colocation]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Rack_unit
  - https://en.wikipedia.org/wiki/19-inch_rack
---

A **rack server** is a [server](/reference/server/) built in a flat, standardized chassis that bolts into a 19-inch equipment rack, so many machines can be stacked densely in a cabinet.[^rack]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A 19-inch equipment rack shown from the front. Rack unit numbers run up the left side. The rack holds a network switch at top, several 1U slim servers, a 2U server that occupies two units, a 4U server, and network-attached storage near the bottom, illustrating how different chassis heights stack in the same frame." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <rect x="150" y="16" width="200" height="184" rx="3" fill-opacity="0.03" stroke-width="1.5"/>
    <g font-size="7" stroke="none" text-anchor="end" fill-opacity="0.75">
      <text x="144" y="30">12U</text><text x="144" y="46">10U</text><text x="144" y="70">8U</text>
      <text x="144" y="110">6U</text><text x="144" y="150">3U</text><text x="144" y="182">1U</text>
    </g>
    <g stroke-width="1.1">
      <rect x="158" y="22" width="184" height="14" rx="1" fill-opacity="0.10"/><text x="250" y="32" text-anchor="middle" font-size="7" stroke="none">Network switch &#183; 1U</text>
      <rect x="158" y="38" width="184" height="14" rx="1" fill-opacity="0.16"/><text x="250" y="48" text-anchor="middle" font-size="7" stroke="none">1U server</text>
      <rect x="158" y="54" width="184" height="14" rx="1" fill-opacity="0.16"/><text x="250" y="64" text-anchor="middle" font-size="7" stroke="none">1U server</text>
      <rect x="158" y="70" width="184" height="30" rx="1" fill-opacity="0.20"/><text x="250" y="88" text-anchor="middle" font-size="7" stroke="none">2U server</text>
      <rect x="158" y="102" width="184" height="58" rx="1" fill-opacity="0.24"/><text x="250" y="134" text-anchor="middle" font-size="7" stroke="none">4U server &#183; more bays &amp; cooling</text>
      <rect x="158" y="162" width="184" height="32" rx="1" fill-opacity="0.12"/><text x="250" y="181" text-anchor="middle" font-size="7" stroke="none">NAS storage</text>
    </g>
    <g stroke-width="1.3" fill="none">
      <line x1="380" y1="120" x2="356" y2="120"/><path d="M356 120 l8 -3 v6 z" stroke-width="1" fill="currentColor"/>
    </g>
    <text x="384" y="116" font-size="7.5" stroke="none">cold-aisle</text>
    <text x="384" y="126" font-size="7.5" stroke="none">air in</text>
    <text x="250" y="208" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">1U = 1.75 in (44.45 mm) &#183; taller chassis = more drives &amp; slots</text>
  </g>
</svg>
<figcaption>A 19-inch rack stacks chassis measured in units (U): slim 1U "pizza boxes", taller 2U and 4U servers with more bays and cooling, plus switches and storage — all sharing the same rails, power, and front-to-back airflow.</figcaption>
</figure>

## Overview

Height is measured in *rack units* (**U**), each 1.75 in (44.45 mm) tall.[^ru] A 1U "pizza box" is the slimmest common server; 2U and 4U chassis trade density for more drive bays, expansion slots, and cooling. The 19-inch rack standard means servers, switches, and storage from different vendors share the same rails, power distribution, and cable management.

Rack servers fill the cabinets of a [data center](/reference/data-center/), where airflow runs front-to-back through cold and hot aisles. The standardization is the quiet superpower here: because every vendor builds to the same 19-inch width and U increment, an operator can mix machines, switches, and storage in one frame and cable them predictably.

## Form factors

Chassis height is the main dial, trading density against expandability:

| Size | Height | Trade-off | Typical role |
|------|--------|-----------|--------------|
| 1U | 1.75 in | Densest, limited bays/cooling | Web & app servers |
| 2U | 3.5 in | Balance of density and expansion | General workhorse |
| 4U | 7 in | Most drives, slots, and cooling | Storage, GPU, database |

The taller the chassis, the fewer fit per rack — so operators pick the smallest U count that still holds the drives, cards, and cooling a workload needs.

## Where it fits

The rack server is the workhorse form factor for a [dedicated server](/reference/dedicated-server/) or a machine you place via [colocation](/reference/colocation/). When you need even higher density and shared power, a [blade server](/reference/blade-server/) packs more compute into the same space. Storage often rides alongside as [network-attached storage](/reference/network-attached-storage/). For most GopherTrunk users a rack server is overkill — a small box near the antenna is enough — but a rack machine is a natural home for long-term storage and serving of decoded data.

## Sources

[^rack]: [19-inch rack](https://en.wikipedia.org/wiki/19-inch_rack) — Wikipedia, on the rack standard and mounting.
[^ru]: [Rack unit](https://en.wikipedia.org/wiki/Rack_unit) — Wikipedia, on the U height unit.
