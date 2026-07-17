---
slug: colocation
title: Colocation
entry_type: concept
category: hw-servers
description: Colocation is renting space, power, cooling, and network connectivity in a data center for hardware you own, so you keep your own servers while outsourcing the facility.
keywords: colocation, colo, data center space, rack space, cage, hosting your own hardware, cross-connect
aka: [Colo]
infobox:
  - { label: Type, value: Hosting model }
  - { label: You provide, value: The hardware }
  - { label: Facility provides, value: Space, power, cooling, network }
  - { label: Sold by, value: Rack, cabinet, or cage }
  - { label: Contrast, value: Cloud / managed hosting }
see_also: [data-center, dedicated-server, rack-server, managed-hosting, cloud-computing, server]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Colocation_centre
---

**Colocation** (colo) is renting space, power, cooling, and network connectivity in a [data center](/reference/data-center/) for hardware you own — you keep your own [servers](/reference/server/) while outsourcing the building around them.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A data-center building outline contains a locked cage holding a customer-owned server rack. Arrows from the facility feed power, cooling, and a network uplink into the cage, while a label shows the customer owns and maintains the servers inside." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <rect x="120" y="24" width="320" height="150" rx="4" fill-opacity="0.04" stroke-width="1.4"/>
    <text x="280" y="18" text-anchor="middle" font-size="9" font-weight="600" stroke="none">Provider's data center</text>
    <rect x="250" y="52" width="120" height="104" rx="3" fill-opacity="0.10" stroke-width="1.3" stroke-dasharray="4 3"/>
    <text x="310" y="66" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.9">your locked cage</text>
    <g stroke-width="1.1">
      <rect x="272" y="74" width="76" height="74" rx="2" fill-opacity="0.06"/>
      <rect x="278" y="80" width="64" height="12" rx="1" fill-opacity="0.16"/>
      <rect x="278" y="96" width="64" height="12" rx="1" fill-opacity="0.16"/>
      <rect x="278" y="112" width="64" height="12" rx="1" fill-opacity="0.16"/>
      <rect x="278" y="128" width="64" height="12" rx="1" fill-opacity="0.16"/>
    </g>
    <text x="310" y="164" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">your servers</text>
    <g stroke-width="1.3" fill="none">
      <line x1="150" y1="70" x2="248" y2="70"/>
      <line x1="150" y1="104" x2="248" y2="104"/>
      <line x1="150" y1="138" x2="248" y2="138"/>
    </g>
    <g stroke-width="1.3" fill="currentColor">
      <path d="M248 70 l-8 -3 v6 z"/>
      <path d="M248 104 l-8 -3 v6 z"/>
      <path d="M248 138 l-8 -3 v6 z"/>
    </g>
    <g font-size="8" stroke="none" text-anchor="end">
      <text x="146" y="67">Power</text>
      <text x="146" y="101">Cooling</text>
      <text x="146" y="135">Network uplink</text>
    </g>
    <text x="146" y="150" text-anchor="end" font-size="7.5" stroke="none" fill-opacity="0.7">(facility provides)</text>
  </g>
</svg>
<figcaption>You buy and install the servers; the colocation provider supplies the secured space plus metered power, cooling, and network uplink around them — a split between owning the machines and outsourcing the facility.</figcaption>
</figure>

## Overview

A colocation provider sells you a slice of the facility — a few [rack-server](/reference/rack-server/) units, a full cabinet, or a locked cage — plus metered power, cooling, and uplink bandwidth. You buy, install, and maintain the machines; they guarantee the environment and uptime, along with physical security and often a fast *cross-connect* to carriers or cloud on-ramps.

This sits between running a [home server](/reference/home-server/) (you own everything, including the room) and renting compute outright. The appeal is keeping full ownership and control of the hardware while gaining a facility you could never economically build: redundant power, industrial cooling, and carrier-grade connectivity.

## Tiers

Colocation is usually sold at one of three granularities, trading price for isolation and control:

| Unit | What you get | Typical tenant |
|------|--------------|----------------|
| Rack units | A few U in a shared cabinet | One or two servers |
| Cabinet | A full lockable rack | Small business |
| Cage | Fenced private floor space | Compliance-heavy or large fleets |

Power is metered (per kilowatt or per circuit), and providers bill separately for cross-connects and extra bandwidth, so the real cost depends as much on power draw as on floor space.

## Where it fits

Colocation suits organizations that have already invested in hardware, need physical control of their machines, or have compliance reasons to keep ownership, but lack a reliable facility of their own. The alternatives are a fully provider-owned [dedicated server](/reference/dedicated-server/), a [managed hosting](/reference/managed-hosting/) arrangement where the provider also runs the software, or [cloud computing](/reference/cloud-computing/) where you rent capacity with no hardware at all. A GopherTrunk back end could be colocated, but the RF capture still belongs at the antenna, not in the colo cage.

## Sources

[^wiki]: [Colocation centre](https://en.wikipedia.org/wiki/Colocation_centre) — Wikipedia, on renting data center space for customer-owned equipment.
