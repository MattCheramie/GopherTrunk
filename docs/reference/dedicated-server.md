---
slug: dedicated-server
title: Dedicated server
entry_type: hardware
category: hw-servers
description: A dedicated server is a whole physical server rented for your exclusive use, with no other customers sharing the hardware.
keywords: dedicated server, bare metal, exclusive hardware, single tenant, uncontended, dedicated hosting
aka: [Dedicated server, Bare metal server]
infobox:
  - { label: Type, value: Physical server }
  - { label: Tenancy, value: Single (exclusive) }
  - { label: Performance, value: Uncontended }
  - { label: You manage, value: The whole machine }
  - { label: Contrast, value: Shared / VPS hosting }
see_also: [server, virtual-private-server, bare-metal-server, web-hosting, cloud-computing, home-server]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Dedicated_hosting_service
---

A **dedicated server** is a whole physical [server](/reference/server/) rented for your exclusive use — no other customers share the hardware.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A tenancy spectrum across three machines. Shared hosting packs many customers onto one operating system. A VPS splits one machine into a few isolated virtual servers. A dedicated server holds a single tenant who has the whole box. Contention falls and control rises from left to right." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <text x="70" y="20" text-anchor="middle" font-size="8.5" font-weight="600" stroke="none">Shared hosting</text>
    <rect x="24" y="30" width="92" height="86" rx="3" fill-opacity="0.05" stroke-width="1.3"/>
    <g stroke-width="0.9" fill-opacity="0.16">
      <rect x="30" y="38" width="24" height="16" rx="1"/><rect x="58" y="38" width="24" height="16" rx="1"/><rect x="86" y="38" width="24" height="16" rx="1"/>
      <rect x="30" y="58" width="24" height="16" rx="1"/><rect x="58" y="58" width="24" height="16" rx="1"/><rect x="86" y="58" width="24" height="16" rx="1"/>
      <rect x="30" y="78" width="24" height="16" rx="1"/><rect x="58" y="78" width="24" height="16" rx="1"/><rect x="86" y="78" width="24" height="16" rx="1"/>
    </g>
    <text x="70" y="110" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">many tenants, one OS</text>

    <text x="230" y="20" text-anchor="middle" font-size="8.5" font-weight="600" stroke="none">VPS</text>
    <rect x="184" y="30" width="92" height="86" rx="3" fill-opacity="0.05" stroke-width="1.3"/>
    <g stroke-width="1" fill-opacity="0.16">
      <rect x="190" y="40" width="80" height="20" rx="1"/>
      <rect x="190" y="64" width="80" height="20" rx="1"/>
      <rect x="190" y="88" width="80" height="16" rx="1"/>
    </g>
    <text x="230" y="112" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">a few VMs share it</text>

    <text x="390" y="20" text-anchor="middle" font-size="8.5" font-weight="600" stroke="none">Dedicated</text>
    <rect x="344" y="30" width="92" height="86" rx="3" fill-opacity="0.05" stroke-width="1.3"/>
    <rect x="350" y="40" width="80" height="64" rx="2" fill-opacity="0.20" stroke-width="1.1"/>
    <text x="390" y="76" text-anchor="middle" font-size="8" stroke="none">one tenant</text>
    <text x="390" y="112" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">the whole box is yours</text>

    <line x1="24" y1="140" x2="436" y2="140" stroke-width="1.2"/>
    <path d="M436 140 l-8 -3 v6 z" stroke-width="1"/>
    <text x="30" y="156" font-size="7.5" stroke="none" fill-opacity="0.8">more sharing / lower cost</text>
    <text x="430" y="156" text-anchor="end" font-size="7.5" stroke="none" fill-opacity="0.8">less contention / more control</text>
  </g>
</svg>
<figcaption>Along the tenancy spectrum, shared hosting crams many customers onto one operating system and a VPS gives each tenant an isolated slice; a dedicated server hands the entire machine to a single tenant, trading cost for uncontended performance and full control.</figcaption>
</figure>

## Overview

Unlike a [virtual private server](/reference/virtual-private-server/), which carves one machine into many tenants, a dedicated server is yours alone. That means raw, uncontended performance: no neighbors competing for CPU, memory, or disk, and no hypervisor tax on throughput. You (or the provider, under a managed plan) handle the metal — the operating system, patching, and recovery.

In practice a dedicated server is a [bare-metal server](/reference/bare-metal-server/) offered on a rental basis, usually on a monthly lease rather than the hourly billing of bare-metal cloud. Providers rack, power, and network the box; you treat it as your own from the operating system up.

## When to choose it

The decision usually comes down to how steady and heavy the workload is:

| Factor | Dedicated server | VPS |
|--------|------------------|-----|
| Performance | Uncontended, full machine | Shared, can vary |
| Cost | Higher, fixed | Lower, scalable |
| Best for | Sustained heavy load | Light or bursty load |
| Isolation | Complete | Logical (hypervisor) |

Pick a dedicated server when predictable, full-machine performance is worth the higher price compared with a VPS — sustained heavy workloads, strict latency budgets, or licensing that demands a single tenant. For lighter or bursty needs, a VPS or [cloud computing](/reference/cloud-computing/) is usually the better value.

## Where it fits

A dedicated server sits between renting a shared slice and owning a [home server](/reference/home-server/) outright: exclusive hardware without having to house or maintain the physical box. For GopherTrunk it is a solid home for a busy back end that stores and serves decoded calls, though like any remote server it cannot capture RF — that still needs a radio front end near the antenna.

## Sources

[^wiki]: [Dedicated hosting service](https://en.wikipedia.org/wiki/Dedicated_hosting_service) — Wikipedia, on single-tenant dedicated servers.
