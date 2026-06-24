---
slug: virtualization
title: Virtualization
entry_type: concept
category: hw-servers
description: Virtualization splits one physical computer into several isolated virtual machines, each behaving like a separate computer, managed by a hypervisor.
keywords: virtualization, virtual machine, hypervisor, VM, isolation, host
aka: [Virtualization, Virtualisation]
infobox:
  - { label: Type, value: Resource technology }
  - { label: Splits, value: One host into many VMs }
  - { label: Managed by, value: Hypervisor }
  - { label: Underpins, value: VPS & cloud }
see_also: [virtual-private-server, cloud-computing, server, dedicated-server, operating-system]
related_lessons:
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
  - { title: "Combining tiers", url: /learn/intro-hardware/combining-tiers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Virtualization
---

**Virtualization** is splitting one physical computer into several isolated virtual machines, each behaving like a separate computer with its own operating system.[^wiki]

## Overview

The piece that makes this work is the *hypervisor*: a software layer that creates and runs the virtual machines and divides the host's CPU, memory, and storage between them. Each VM is walled off from its neighbors, so one tenant's crash or workload does not spill into another's.

## Where it fits

Virtualization is the foundation under the rest of this category. A [virtual private server](/reference/virtual-private-server/) is one such VM rented out to you, and [cloud computing](/reference/cloud-computing/) is the same idea run across whole data centers. Without it, every customer would need their own [physical server](/reference/server/) — virtualization is what makes affordable, on-demand servers possible.

## Sources

[^wiki]: [Virtualization](https://en.wikipedia.org/wiki/Virtualization) — Wikipedia, on virtual machines and the hypervisor model.
