---
slug: virtual-private-server
title: Virtual private server (VPS)
entry_type: concept
category: hw-servers
description: A virtual private server is a slice of a physical server that behaves like your own machine with root access, created by virtualization via a hypervisor.
keywords: VPS, virtual private server, hypervisor, root access, DigitalOcean, Linode, Hetzner, virtual machine
aka: [VPS]
autolink: true
infobox:
  - { label: Type, value: Virtual server }
  - { label: Access, value: Root / full control }
  - { label: Created by, value: Virtualization }
  - { label: Neighbors, value: Isolated by hypervisor }
  - { label: Providers, value: DigitalOcean, Linode, Hetzner }
see_also: [server, web-hosting, dedicated-server, bare-metal-server, virtualization, cloud-computing]
related_lessons:
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Virtual_private_server
---

A **virtual private server (VPS)** is a slice of a physical [server](/reference/server/) that behaves like your own machine, complete with root access.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A client on the left connects over the network to one virtual private server, which is a highlighted virtual machine among three sharing a single physical host through a hypervisor. That VPS reads and writes an attached storage volume on the right. The other virtual machines are isolated neighbors on the same host." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <rect x="20" y="74" width="46" height="30" rx="2" fill-opacity="0.14" stroke-width="1.1"/>
    <text x="43" y="93" text-anchor="middle" font-size="7.5" stroke="none">client</text>
    <line x1="66" y1="89" x2="126" y2="89" stroke-width="1.4" fill="none"/>
    <path d="M126 89 l-8 -3 v6 z" stroke-width="1"/>
    <rect x="128" y="34" width="180" height="112" rx="4" fill-opacity="0.04" stroke-width="1.5"/>
    <text x="218" y="28" text-anchor="middle" font-size="8.5" font-weight="600" stroke="none">one physical host</text>
    <rect x="138" y="44" width="160" height="26" rx="2" fill-opacity="0.06" stroke-width="1"/>
    <text x="218" y="60" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">neighbor VM (isolated)</text>
    <rect x="138" y="74" width="160" height="32" rx="2" fill-opacity="0.24" stroke-width="1.5"/>
    <text x="218" y="90" text-anchor="middle" font-size="8" stroke="none" font-weight="600">your VPS</text>
    <text x="218" y="101" text-anchor="middle" font-size="6.5" stroke="none">own OS &#183; root access</text>
    <rect x="138" y="110" width="160" height="18" rx="2" fill-opacity="0.20" stroke-width="1.1"/>
    <text x="218" y="122" text-anchor="middle" font-size="7.5" stroke="none">Hypervisor</text>
    <rect x="138" y="130" width="160" height="10" rx="1" fill-opacity="0.06" stroke-width="1"/>
    <line x1="298" y1="90" x2="358" y2="90" stroke-width="1.4" fill="none"/>
    <path d="M358 90 l-8 -3 v6 z" stroke-width="1"/>
    <path d="M366 76 a20 6 0 0 1 40 0 v28 a20 6 0 0 1 -40 0 z" fill-opacity="0.12" stroke-width="1.3"/>
    <path d="M366 76 a20 6 0 0 0 40 0" fill="none" stroke-width="1.1"/>
    <text x="386" y="118" text-anchor="middle" font-size="7.5" stroke="none">storage</text>
    <text x="230" y="166" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">a hypervisor divides one host into isolated virtual machines</text>
  </g>
</svg>
<figcaption>A hypervisor splits one physical host into several isolated virtual machines; your VPS is one such slice with its own operating system and root access, reachable by clients and backed by its own storage — cheaper than a whole machine, more capable than shared hosting.</figcaption>
</figure>

## Overview

A VPS sits between shared [web hosting](/reference/web-hosting/) and a whole [dedicated server](/reference/dedicated-server/): more control than the former, far cheaper than the latter. It is created by [virtualization](/reference/virtualization/) — a hypervisor divides one physical host into several isolated virtual machines, each with its own operating system, kernel, and root account.

The isolation is logical rather than physical: neighbors share the same silicon, so a very busy tenant can in principle contend for CPU or disk (the "noisy neighbor" effect), though providers cap and balance resources to limit it. Providers like DigitalOcean, Linode, and Hetzner rent these by the month, and can resize or snapshot them far faster than any physical box.

## When to choose it

The VPS is the middle rung of the hosting ladder, and the choice is usually against its neighbors:

| Option | Control | Cost | Performance |
|--------|---------|------|-------------|
| Shared hosting | Low (no root) | Lowest | Shared, variable |
| **VPS** | Root, own OS | Moderate | Good, mostly isolated |
| [Dedicated](/reference/dedicated-server/) / [bare metal](/reference/bare-metal-server/) | Full machine | Highest | Uncontended |

Reach for a VPS when shared hosting stops being enough: you need root access, custom services, or a specific software stack. The trade-off is responsibility — you manage the operating system, security updates, and backups yourself.

## Where it fits

A VPS is the default first "real server" for most projects — enough control to run anything, cheap enough to leave on 24/7. For GopherTrunk this is a fine place to host the web console and store and serve decoded data, though like any virtual or remote server it still cannot capture RF, which needs a radio front end near the antenna.

## Sources

[^wiki]: [Virtual private server](https://en.wikipedia.org/wiki/Virtual_private_server) — Wikipedia, on VPS hosting and the hypervisor model.
