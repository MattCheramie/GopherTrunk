---
slug: bare-metal-server
title: Bare-metal server
entry_type: hardware
category: hw-servers
description: A bare-metal server is a single physical machine rented or owned entirely by one tenant, with no hypervisor or shared virtualization layer between the user and the hardware.
keywords: bare-metal server, single-tenant, no hypervisor, dedicated hardware, bare metal cloud
aka: [Bare metal]
infobox:
  - { label: Type, value: Single-tenant physical server }
  - { label: Virtualization, value: None imposed }
  - { label: Tenancy, value: One customer }
  - { label: Contrast, value: VPS / cloud instance }
see_also: [dedicated-server, virtual-private-server, hypervisor, server, infrastructure-as-a-service, virtualization]
related_lessons:
  - { title: "Dedicated servers", url: /learn/intro-hardware/dedicated-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bare-metal_server
---

A **bare-metal server** is a single physical machine used entirely by one tenant, with no [hypervisor](/reference/hypervisor/) or shared virtualization layer sitting between the user and the hardware.[^wiki]

## Overview

The term contrasts with virtualized cloud instances, where many tenants share one physical box through a hypervisor. On bare metal there is no virtualization overhead and no "noisy neighbor" competing for the same CPU and disk — you get the whole machine's performance and direct access to hardware features. Modern "bare-metal cloud" offerings rent such machines by the hour with cloud-style provisioning, blending dedicated hardware with on-demand billing.

## Trade-offs

Bare metal is the right call when you need predictable performance, full control of the hardware, or you intend to run your own [virtualization](/reference/virtualization/) on top. A [dedicated server](/reference/dedicated-server/) is essentially a long-lease bare-metal box; a [virtual private server](/reference/virtual-private-server/) is the shared, cheaper alternative. Within [infrastructure as a service](/reference/infrastructure-as-a-service/), bare metal is the lowest-abstraction rung. GopherTrunk runs the same on bare metal as on a VPS; the deciding factor is whether you also need a real radio front end attached, which favors local hardware.

## Sources

[^wiki]: [Bare-metal server](https://en.wikipedia.org/wiki/Bare-metal_server) — Wikipedia, on single-tenant physical servers.
