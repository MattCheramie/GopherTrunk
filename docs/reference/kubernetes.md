---
slug: kubernetes
title: Kubernetes
entry_type: concept
category: hw-servers
description: Kubernetes is an open-source system for automating the deployment, scaling, and management of containerized applications across a cluster of machines, originally developed at Google.
keywords: Kubernetes, k8s, container orchestration, cluster, pods, scaling, Google, CNCF
autolink: true
aka: [K8s]
infobox:
  - { label: Type, value: Container orchestrator }
  - { label: Origin, value: Google, 2014 }
  - { label: Manages, value: Containers across a cluster }
  - { label: Unit, value: Pod }
see_also: [container, docker, load-balancer, high-availability, scalability, virtualization]
cite_urls:
  - https://en.wikipedia.org/wiki/Kubernetes
  - https://kubernetes.io/
---

**Kubernetes** (often "K8s") is an open-source system for automating the deployment, scaling, and management of containerized applications across a cluster of machines.[^wiki] It was created at Google and released in 2014, and is now stewarded by the Cloud Native Computing Foundation.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 224" role="img" aria-label="Incoming traffic reaches a single Kubernetes Service, which load-balances requests across four identical replica Pods. The pods are grouped two-per-box into two worker nodes, showing that the Service spreads load evenly over replicas scheduled across the cluster." xmlns="http://www.w3.org/2000/svg">
  <circle cx="40" cy="110" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
  <text x="40" y="90" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">traffic</text>
  <rect x="92" y="86" width="92" height="48" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="138" y="106" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">Service</text>
  <text x="138" y="120" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">load-balances</text>
  <line x1="53" y1="110" x2="92" y2="110" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75" fill="none" marker-end="url(#k8s_ar)"/>
  <g stroke="currentColor">
    <rect x="316" y="16" width="124" height="94" rx="6" fill="none" stroke-width="1.1" stroke-dasharray="5 4"/>
    <rect x="316" y="116" width="124" height="94" rx="6" fill="none" stroke-width="1.1" stroke-dasharray="5 4"/>
  </g>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="328" y="24" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="378" y="45">Pod</text>
    <rect x="328" y="62" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="378" y="83">Pod</text>
    <rect x="328" y="124" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="378" y="145">Pod</text>
    <rect x="328" y="162" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="378" y="183">Pod</text>
  </g>
  <g fill="currentColor" font-size="7" text-anchor="middle" fill-opacity="0.65">
    <text x="378" y="106">worker node</text>
    <text x="378" y="206">worker node</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75" fill="none">
    <line x1="184" y1="106" x2="328" y2="41" marker-end="url(#k8s_ar)"/>
    <line x1="184" y1="108" x2="328" y2="79" marker-end="url(#k8s_ar)"/>
    <line x1="184" y1="112" x2="328" y2="141" marker-end="url(#k8s_ar)"/>
    <line x1="184" y1="114" x2="328" y2="179" marker-end="url(#k8s_ar)"/>
  </g>
  <defs><marker id="k8s_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A Service is one stable address that spreads traffic across a set of identical replica Pods, which Kubernetes schedules onto whatever worker nodes have room. Add replicas and the Service balances over more of them; lose a node and its pods are rescheduled elsewhere — scaling and self-healing without changing the front door.</figcaption>
</figure>

## Overview

Kubernetes groups [containers](/reference/container/) into *pods*, schedules them onto a pool of worker nodes, and continuously reconciles the cluster toward a desired state you declare. If a node dies, it reschedules the affected pods elsewhere; if load rises, it can add replicas. It also wires up service discovery and internal [load balancing](/reference/load-balancer/), rolling updates, and self-healing restarts. The containers themselves are usually built with [Docker](/reference/docker/) or another compatible runtime.

## Where it fits

Kubernetes is the standard answer to running many containers reliably at scale, providing [high availability](/reference/high-availability/) and elastic [scalability](/reference/scalability/) on top of plain virtualization. It is powerful but operationally heavy — overkill for a single service. A lone GopherTrunk capture node needs nothing like it, but a fleet of nodes feeding a shared back end could be coordinated by Kubernetes.

## Sources

[^wiki]: [Kubernetes](https://en.wikipedia.org/wiki/Kubernetes) — Wikipedia, on Kubernetes' design and history.
[^home]: [Kubernetes](https://kubernetes.io/) — the project's official site.
