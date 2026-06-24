---
slug: virtual-private-server
title: Virtual private server (VPS)
entry_type: concept
category: hw-servers
description: A virtual private server is a slice of a physical server that behaves like your own machine with root access, created by virtualization via a hypervisor.
keywords: VPS, virtual private server, hypervisor, root access, DigitalOcean, Linode, Hetzner
aka: [VPS]
autolink: true
infobox:
  - { label: Type, value: Virtual server }
  - { label: Access, value: Root / full control }
  - { label: Created by, value: Virtualization }
  - { label: Providers, value: DigitalOcean, Linode, Hetzner }
see_also: [server, web-hosting, dedicated-server, virtualization, cloud-computing]
related_lessons:
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Virtual_private_server
---

A **virtual private server (VPS)** is a slice of a physical [server](/reference/server/) that behaves like your own machine, complete with root access.[^wiki]

## Overview

A VPS sits between shared [web hosting](/reference/web-hosting/) and a whole [dedicated server](/reference/dedicated-server/): more control than the former, far cheaper than the latter. It is created by [virtualization](/reference/virtualization/) — a hypervisor divides one physical host into several isolated virtual machines, each with its own operating system. Providers like DigitalOcean, Linode, and Hetzner rent these by the month.

## When to choose it

Reach for a VPS when shared hosting stops being enough: you need root access, custom services, or a specific software stack. The trade-off is responsibility — you manage the operating system, security updates, and backups yourself. For GopherTrunk this is a fine place to store and serve decoded data, though it still cannot capture RF.

## Sources

[^wiki]: [Virtual private server](https://en.wikipedia.org/wiki/Virtual_private_server) — Wikipedia, on VPS hosting and the hypervisor model.
