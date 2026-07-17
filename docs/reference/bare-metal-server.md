---
slug: bare-metal-server
title: Bare-metal server
entry_type: hardware
category: hw-servers
description: A bare-metal server is a single physical machine rented or owned entirely by one tenant, with no hypervisor or shared virtualization layer between the user and the hardware.
keywords: bare-metal server, single-tenant, no hypervisor, dedicated hardware, bare metal cloud, noisy neighbor, provisioning
aka: [Bare metal]
infobox:
  - { label: Type, value: Single-tenant physical server }
  - { label: Virtualization, value: None imposed }
  - { label: Tenancy, value: One customer }
  - { label: Billing, value: Monthly lease or hourly cloud }
  - { label: Contrast, value: VPS / cloud instance }
see_also: [dedicated-server, virtual-private-server, hypervisor, server, infrastructure-as-a-service, virtualization]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bare-metal_server
---

A **bare-metal server** is a single physical machine used entirely by one tenant, with no [hypervisor](/reference/hypervisor/) or shared virtualization layer sitting between the user and the hardware.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="Two layer stacks compared. On a bare-metal server the operating system and workload sit directly on the hardware with a single tenant and no hypervisor. On a virtualized host a hypervisor sits above the hardware and divides it into several virtual machines shared by many tenants." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <text x="115" y="20" text-anchor="middle" font-size="9" font-weight="600" stroke="none">BARE METAL</text>
    <rect x="30" y="70" width="170" height="34" rx="3" fill-opacity="0.14" stroke-width="1.3"/>
    <text x="115" y="91" text-anchor="middle" font-size="8.5" stroke="none">Your OS + workload</text>
    <rect x="30" y="120" width="170" height="34" rx="3" fill-opacity="0.06" stroke-width="1.3"/>
    <text x="115" y="141" text-anchor="middle" font-size="8.5" stroke="none">Hardware · CPU · disk · NIC</text>
    <text x="115" y="172" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">one tenant &#183; direct access</text>

    <text x="345" y="20" text-anchor="middle" font-size="9" font-weight="600" stroke="none">VIRTUALIZED HOST</text>
    <rect x="260" y="60" width="52" height="28" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <rect x="319" y="60" width="52" height="28" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <rect x="378" y="60" width="52" height="28" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <text x="286" y="78" text-anchor="middle" font-size="8" stroke="none">VM</text>
    <text x="345" y="78" text-anchor="middle" font-size="8" stroke="none">VM</text>
    <text x="404" y="78" text-anchor="middle" font-size="8" stroke="none">VM</text>
    <rect x="260" y="94" width="170" height="22" rx="3" fill-opacity="0.20" stroke-width="1.3"/>
    <text x="345" y="109" text-anchor="middle" font-size="8.5" stroke="none">Hypervisor</text>
    <rect x="260" y="122" width="170" height="32" rx="3" fill-opacity="0.06" stroke-width="1.3"/>
    <text x="345" y="142" text-anchor="middle" font-size="8.5" stroke="none">Hardware</text>
    <text x="345" y="172" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">many tenants share the box</text>
  </g>
</svg>
<figcaption>On bare metal your operating system runs straight on the hardware as the sole tenant; a virtualized host instead inserts a hypervisor that slices one machine among many tenants, adding isolation but also overhead and contention.</figcaption>
</figure>

## Overview

The term contrasts with virtualized cloud instances, where many tenants share one physical box through a hypervisor. On bare metal there is no virtualization overhead and no "noisy neighbor" competing for the same CPU and disk — you get the whole machine's performance and direct access to hardware features such as specific CPU instructions, NUMA layout, or an attached GPU.

Modern "bare-metal cloud" offerings rent such machines by the hour with cloud-style provisioning APIs, blending dedicated hardware with on-demand billing. That gives much of the flexibility of a cloud instance while preserving the predictability of a single-tenant box. Because nothing sits between you and the silicon, bare metal is also the usual foundation for running *your own* [virtualization](/reference/virtualization/) or container stack on top.

The cost is responsibility and lead time: you own the operating system and its upkeep, and physical provisioning is slower than spinning up a virtual machine — though bare-metal cloud has narrowed that gap to minutes.

## Trade-offs

The three common tenancy models trade performance against price and convenience:

| Model | Tenancy | Virtualization | Best for |
|-------|---------|----------------|----------|
| Bare metal | One tenant | None imposed | Predictable performance, hardware access |
| [VPS](/reference/virtual-private-server/) | Shared host | Hypervisor slice | Cost-sensitive, bursty workloads |
| Shared hosting | Many per OS | None (OS-level) | Simple sites, minimal control |

Bare metal is the right call when you need predictable performance, full control of the hardware, or you intend to run your own virtualization on top. A [dedicated server](/reference/dedicated-server/) is essentially a long-lease bare-metal box; a VPS is the shared, cheaper alternative.

## Where it fits

Within [infrastructure as a service](/reference/infrastructure-as-a-service/), bare metal is the lowest-abstraction rung — the closest a rented machine gets to one you own. GopherTrunk runs the same on bare metal as on a VPS; the deciding factor is whether you also need a real radio front end attached, which favors local hardware, or whether the box merely stores and serves decoded calls, where a shared VPS is cheaper.

## Sources

[^wiki]: [Bare-metal server](https://en.wikipedia.org/wiki/Bare-metal_server) — Wikipedia, on single-tenant physical servers and bare-metal cloud.
