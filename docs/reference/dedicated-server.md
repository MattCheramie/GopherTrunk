---
slug: dedicated-server
title: Dedicated server
entry_type: hardware
category: hw-servers
description: A dedicated server is a whole physical server rented for your exclusive use, with no other customers sharing the hardware.
keywords: dedicated server, bare metal, exclusive hardware, single tenant, uncontended
aka: [Dedicated server, Bare metal server]
infobox:
  - { label: Type, value: Physical server }
  - { label: Tenancy, value: Single (exclusive) }
  - { label: Performance, value: Uncontended }
  - { label: You manage, value: The whole machine }
see_also: [server, virtual-private-server, web-hosting, cloud-computing, home-server]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Dedicated_hosting_service
---

A **dedicated server** is a whole physical [server](/reference/server/) rented for your exclusive use — no other customers share the hardware.[^wiki]

## Overview

Unlike a [virtual private server](/reference/virtual-private-server/), which carves one machine into many tenants, a dedicated server is yours alone. That means raw, uncontended performance: no neighbors competing for CPU, memory, or disk. You (or the provider, under a managed plan) handle the metal — the operating system, patching, and recovery.

## When to choose it

Pick a dedicated server when predictable, full-machine performance is worth the higher price compared with a VPS — sustained heavy workloads, strict latency budgets, or licensing that demands a single tenant. For lighter or bursty needs, a VPS or [cloud computing](/reference/cloud-computing/) is usually the better value.

## Sources

[^wiki]: [Dedicated hosting service](https://en.wikipedia.org/wiki/Dedicated_hosting_service) — Wikipedia, on single-tenant dedicated servers.
