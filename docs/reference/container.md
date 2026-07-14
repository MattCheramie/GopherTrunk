---
slug: container
title: Container
entry_type: concept
category: hw-servers
description: A container is a lightweight, isolated package that bundles an application with its dependencies and runs as an ordinary process on a shared operating-system kernel, making software portable and reproducible.
keywords: container, containerization, OS-level virtualization, image, namespaces, cgroups, portable app
infobox:
  - { label: Type, value: OS-level isolation }
  - { label: Shares, value: Host kernel }
  - { label: Packages, value: App + dependencies }
  - { label: Contrast, value: Virtual machine }
see_also: [docker, kubernetes, virtualization, hypervisor, operating-system, serverless-computing]
cite_urls:
  - https://en.wikipedia.org/wiki/Containerization_(computing)
---

A **container** is a lightweight, isolated package that bundles an application with its dependencies and runs as an ordinary process on a shared [operating-system](/reference/operating-system/) kernel — making software portable and reproducible across machines.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 236" role="img" aria-label="Two towers side by side. On the left, virtual machines each stack an application on top of its own guest operating system, above a hypervisor and the host hardware. On the right, containers hold only the application and sit directly on one shared host operating system, so the container tower is shorter and lighter because there is no per-container guest OS." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" text-anchor="middle" font-size="8.5">
    <text x="122" y="20" font-size="9" font-weight="600">Virtual machines</text>
    <g stroke="currentColor">
      <rect x="25" y="40" width="93" height="92" rx="4" fill="none" stroke-width="1.1" stroke-dasharray="4 3"/>
      <rect x="32" y="62" width="79" height="26" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="71" y="79">App</text>
      <rect x="32" y="92" width="79" height="34" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="71" y="112">Guest OS</text>
      <rect x="124" y="40" width="93" height="92" rx="4" fill="none" stroke-width="1.1" stroke-dasharray="4 3"/>
      <rect x="131" y="62" width="79" height="26" rx="3" fill="currentColor" fill-opacity="0.2" stroke-width="1"/><text x="170" y="79">App</text>
      <rect x="131" y="92" width="79" height="34" rx="3" fill="currentColor" fill-opacity="0.1" stroke-width="1"/><text x="170" y="112">Guest OS</text>
    </g>
    <rect x="20" y="140" width="200" height="26" rx="4" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="120" y="157" font-weight="600">Hypervisor</text>
    <rect x="20" y="170" width="200" height="26" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.2"/><text x="120" y="187">Host hardware</text>
  </g>
  <g fill="currentColor" text-anchor="middle" font-size="8.5">
    <text x="350" y="20" font-size="9" font-weight="600">Containers</text>
    <g stroke="currentColor">
      <rect x="250" y="86" width="58" height="46" rx="4" fill="currentColor" fill-opacity="0.2" stroke-width="1" stroke-dasharray="4 3"/><text x="279" y="112">App</text>
      <rect x="316" y="86" width="58" height="46" rx="4" fill="currentColor" fill-opacity="0.2" stroke-width="1" stroke-dasharray="4 3"/><text x="345" y="112">App</text>
      <rect x="382" y="86" width="58" height="46" rx="4" fill="currentColor" fill-opacity="0.2" stroke-width="1" stroke-dasharray="4 3"/><text x="411" y="112">App</text>
    </g>
    <rect x="250" y="140" width="190" height="26" rx="4" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="345" y="157" font-weight="600">Container engine</text>
    <rect x="250" y="170" width="190" height="26" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/><text x="345" y="187" font-weight="600">One shared host OS + kernel</text>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.5">
    <line x1="71" y1="132" x2="71" y2="140"/>
    <line x1="170" y1="132" x2="170" y2="140"/>
    <line x1="279" y1="132" x2="279" y2="140"/>
    <line x1="345" y1="132" x2="345" y2="140"/>
    <line x1="411" y1="132" x2="411" y2="140"/>
  </g>
</svg>
<figcaption>Same job, two weights. Each virtual machine carries a full guest OS, so the tower is tall; a container holds only the app and shares the one host OS and kernel underneath. Dropping the per-container OS is what makes containers start in a blink and pack far more densely — at the cost of the harder isolation a separate kernel gives.</figcaption>
</figure>

## Overview

Containers are a form of OS-level [virtualization](/reference/virtualization/). Unlike a virtual machine, a container does not carry its own kernel or boot an operating system; it shares the host's kernel and is isolated using kernel features such as namespaces (separate views of processes, files, and networking) and cgroups (resource limits). A container starts from an *image* — a layered, read-only template — so the same image runs identically on a laptop, a server, or the cloud. The result is fast startup, low overhead, and "it works the same everywhere" packaging.

## Where it fits

Containers are usually built and run with [Docker](/reference/docker/) and orchestrated at scale by [Kubernetes](/reference/kubernetes/). They are lighter than full virtualization under a [hypervisor](/reference/hypervisor/) but offer weaker isolation, since all containers share one kernel. Packaging GopherTrunk in a container makes the decoder easy to deploy reproducibly, though USB radio access (the SDR dongle) must be passed through from the host.

## Sources

[^wiki]: [Containerization (computing)](https://en.wikipedia.org/wiki/Containerization_(computing)) — Wikipedia, on OS-level containers.
