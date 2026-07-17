---
slug: data-center
title: Data center
entry_type: concept
category: hw-servers
description: A data center is a purpose-built facility that houses computing, storage, and networking equipment with the power, cooling, and connectivity needed to run it reliably around the clock.
keywords: data center, datacenter, server farm, colocation, power and cooling, uptime, hyperscale, hot aisle, cold aisle, PUE
aka: [Datacenter, Server farm]
infobox:
  - { label: Type, value: Computing facility }
  - { label: Houses, value: Servers, storage, networking }
  - { label: Provides, value: Power, cooling, connectivity }
  - { label: Measured by, value: Uptime tier, PUE }
  - { label: Scale, value: Room to hyperscale campus }
see_also: [server, rack-server, colocation, cloud-computing, high-availability, data-storage]
cite_urls:
  - https://en.wikipedia.org/wiki/Data_center
---

A **data center** is a purpose-built facility that houses computing, storage, and networking equipment together with the power, cooling, and connectivity needed to keep it running reliably around the clock.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A cross-section of two rows of server racks arranged around a cold aisle and hot aisles. Cool air rises from a raised floor into the cold aisle between the racks, is drawn front to back through the servers, and exits as hot air into the outer aisles for the cooling system to capture." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <g stroke-width="1.2">
      <rect x="110" y="40" width="70" height="90" rx="2" fill-opacity="0.12"/>
      <rect x="280" y="40" width="70" height="90" rx="2" fill-opacity="0.12"/>
    </g>
    <g stroke-width="0.8" fill-opacity="0.22">
      <rect x="116" y="46" width="58" height="10"/><rect x="116" y="60" width="58" height="10"/>
      <rect x="116" y="74" width="58" height="10"/><rect x="116" y="88" width="58" height="10"/>
      <rect x="116" y="102" width="58" height="10"/><rect x="116" y="116" width="58" height="10"/>
      <rect x="286" y="46" width="58" height="10"/><rect x="286" y="60" width="58" height="10"/>
      <rect x="286" y="74" width="58" height="10"/><rect x="286" y="88" width="58" height="10"/>
      <rect x="286" y="102" width="58" height="10"/><rect x="286" y="116" width="58" height="10"/>
    </g>
    <text x="145" y="26" text-anchor="middle" font-size="8" stroke="none">rack row</text>
    <text x="315" y="26" text-anchor="middle" font-size="8" stroke="none">rack row</text>
    <line x1="230" y1="132" x2="230" y2="52" stroke-width="1.4" fill="none"/>
    <path d="M230 52 l-4 8 h8 z" stroke-width="1.2"/>
    <text x="230" y="148" text-anchor="middle" font-size="8" stroke="none" font-weight="600">COLD aisle</text>
    <text x="230" y="160" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.8">cool air up from raised floor</text>
    <g stroke-width="1.3" fill="none">
      <line x1="180" y1="85" x2="228" y2="85"/>
      <line x1="352" y1="85" x2="400" y2="85"/>
    </g>
    <path d="M180 85 l8 -3 v6 z" stroke-width="1" fill="currentColor"/>
    <path d="M400 85 l-8 -3 v6 z" stroke-width="1" fill="currentColor" transform="translate(0,0)"/>
    <text x="70" y="88" text-anchor="middle" font-size="8" stroke="none">HOT aisle</text>
    <text x="415" y="88" text-anchor="middle" font-size="8" stroke="none">HOT aisle</text>
    <text x="230" y="14" text-anchor="middle" font-size="9" font-weight="600" stroke="none">Hot-aisle / cold-aisle airflow</text>
  </g>
</svg>
<figcaption>Racks face each other across a cold aisle fed by cool air; servers pull that air front-to-back and exhaust into the hot aisles, where the cooling plant captures the heat — the airflow discipline that lets a room pack thousands of machines.</figcaption>
</figure>

## Overview

At its core a data center is a building full of [rack servers](/reference/rack-server/) and the supporting systems that keep them alive: redundant utility feeds and backup generators, uninterruptible power supplies, air or liquid cooling, fire suppression, and high-capacity network links to the outside world. Cabinets are arranged into *hot* and *cold* aisles so that cool supply air and hot exhaust never mix, which is the single biggest lever on cooling efficiency.

Facilities are graded by uptime *tiers* (Tier I–IV) and by energy efficiency, often reported as **PUE** (power usage effectiveness — total facility power divided by IT power, where 1.0 is the unreachable ideal). Scale ranges from a single server room to *hyperscale* campuses run by cloud providers and drawing tens of megawatts.

## Tiers

The Uptime Institute tier system pins expected availability to how much of the facility is redundant:

| Tier | Redundancy | Approx. uptime | Annual downtime |
|------|------------|----------------|-----------------|
| I | None (single path) | 99.671% | ~29 hours |
| II | Redundant components | 99.741% | ~22 hours |
| III | Concurrently maintainable | 99.982% | ~1.6 hours |
| IV | Fault tolerant | 99.995% | ~26 minutes |

Higher tiers cost more to build and run, so operators match the tier to how much an outage would actually cost them.

## Where it fits

A data center is where most [servers](/reference/server/) actually live. You can put your own hardware in one through [colocation](/reference/colocation/), rent capacity inside one as [cloud computing](/reference/cloud-computing/), or rely on one indirectly every time you use a web service. The facility's redundancy is the foundation of [high availability](/reference/high-availability/). In GopherTrunk terms a data center is fine for storing and serving decoded calls, but the RF capture still happens at the antenna — the radio front end cannot move into the building.

## Sources

[^wiki]: [Data center](https://en.wikipedia.org/wiki/Data_center) — Wikipedia, on data center facilities, tiers, PUE, and airflow.
