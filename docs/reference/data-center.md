---
slug: data-center
title: Data center
entry_type: concept
category: hw-servers
description: A data center is a purpose-built facility that houses computing, storage, and networking equipment with the power, cooling, and connectivity needed to run it reliably around the clock.
keywords: data center, datacenter, server farm, colocation, power and cooling, uptime, hyperscale
aka: [Datacenter, Server farm]
infobox:
  - { label: Type, value: Computing facility }
  - { label: Houses, value: Servers, storage, networking }
  - { label: Provides, value: Power, cooling, connectivity }
  - { label: Measured by, value: Uptime tier, PUE }
see_also: [server, rack-server, colocation, cloud-computing, high-availability, data-storage]
cite_urls:
  - https://en.wikipedia.org/wiki/Data_center
---

A **data center** is a purpose-built facility that houses computing, storage, and networking equipment together with the power, cooling, and connectivity needed to keep it running reliably around the clock.[^wiki]

## Overview

At its core a data center is a building full of [rack servers](/reference/rack-server/) and the supporting systems that keep them alive: redundant utility feeds and backup generators, uninterruptible power supplies, air or liquid cooling, fire suppression, and high-capacity network links to the outside world. Facilities are graded by uptime *tiers* (Tier I–IV) and by energy efficiency, often reported as **PUE** (power usage effectiveness — total facility power divided by IT power). Scale ranges from a single room to *hyperscale* campuses run by cloud providers.

## Where it fits

A data center is where most [servers](/reference/server/) actually live. You can put your own hardware in one through [colocation](/reference/colocation/), rent capacity inside one as [cloud computing](/reference/cloud-computing/), or rely on one indirectly every time you use a web service. The facility's redundancy is the foundation of [high availability](/reference/high-availability/). In GopherTrunk terms a data center is fine for storing and serving decoded calls, but the RF capture still happens at the antenna — the radio front end cannot move into the building.

## Sources

[^wiki]: [Data center](https://en.wikipedia.org/wiki/Data_center) — Wikipedia, on data center facilities, tiers, and infrastructure.
