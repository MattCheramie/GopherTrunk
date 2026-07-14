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

<figure class="figure" markdown="0">
<svg viewBox="0 0 420 214" role="img" aria-label="One physical computer at the bottom, a hypervisor layer above it, and three isolated virtual machines stacked on top, each with its own guest operating system and application. The hypervisor divides the host's resources and walls the virtual machines off from one another." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <g stroke="currentColor">
      <rect x="48" y="44" width="104" height="92" rx="5" fill="none" stroke-width="1.2" stroke-dasharray="4 3"/>
      <text x="100" y="59" font-size="8.5" font-weight="600">VM 1</text>
      <rect x="55" y="66" width="90" height="28" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="100" y="84">App</text>
      <rect x="55" y="98" width="90" height="32" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="100" y="118" font-size="8.5">Guest OS</text>
      <rect x="162" y="44" width="104" height="92" rx="5" fill="none" stroke-width="1.2" stroke-dasharray="4 3"/>
      <text x="214" y="59" font-size="8.5" font-weight="600">VM 2</text>
      <rect x="169" y="66" width="90" height="28" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="214" y="84">App</text>
      <rect x="169" y="98" width="90" height="32" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="214" y="118" font-size="8.5">Guest OS</text>
      <rect x="276" y="44" width="96" height="92" rx="5" fill="none" stroke-width="1.2" stroke-dasharray="4 3"/>
      <text x="324" y="59" font-size="8.5" font-weight="600">VM 3</text>
      <rect x="283" y="66" width="82" height="28" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="324" y="84">App</text>
      <rect x="283" y="98" width="82" height="32" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="324" y="118" font-size="8.5">Guest OS</text>
    </g>
    <rect x="40" y="142" width="340" height="28" rx="4" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="210" y="160" font-weight="600">Hypervisor — divides &amp; isolates</text>
    <rect x="40" y="174" width="340" height="28" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.2"/><text x="210" y="192">One physical computer · CPU · memory · storage</text>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.5">
    <line x1="100" y1="136" x2="100" y2="142"/>
    <line x1="214" y1="136" x2="214" y2="142"/>
    <line x1="324" y1="136" x2="324" y2="142"/>
  </g>
</svg>
<figcaption>The hypervisor is the layer that makes one machine act like many: it carves the host's CPU, memory, and storage into virtual machines and walls each off from its neighbours, so one tenant's crash or load can't spill into another's. A rented VPS is one such VM; a cloud is this run across whole data centers.</figcaption>
</figure>

## Overview

The piece that makes this work is the *hypervisor*: a software layer that creates and runs the virtual machines and divides the host's CPU, memory, and storage between them. Each VM is walled off from its neighbors, so one tenant's crash or workload does not spill into another's.

## Where it fits

Virtualization is the foundation under the rest of this category. A [virtual private server](/reference/virtual-private-server/) is one such VM rented out to you, and [cloud computing](/reference/cloud-computing/) is the same idea run across whole data centers. Without it, every customer would need their own [physical server](/reference/server/) — virtualization is what makes affordable, on-demand servers possible.

## Sources

[^wiki]: [Virtualization](https://en.wikipedia.org/wiki/Virtualization) — Wikipedia, on virtual machines and the hypervisor model.
