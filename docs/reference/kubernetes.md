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

## Overview

Kubernetes groups [containers](/reference/container/) into *pods*, schedules them onto a pool of worker nodes, and continuously reconciles the cluster toward a desired state you declare. If a node dies, it reschedules the affected pods elsewhere; if load rises, it can add replicas. It also wires up service discovery and internal [load balancing](/reference/load-balancer/), rolling updates, and self-healing restarts. The containers themselves are usually built with [Docker](/reference/docker/) or another compatible runtime.

## Where it fits

Kubernetes is the standard answer to running many containers reliably at scale, providing [high availability](/reference/high-availability/) and elastic [scalability](/reference/scalability/) on top of plain virtualization. It is powerful but operationally heavy — overkill for a single service. A lone GopherTrunk capture node needs nothing like it, but a fleet of nodes feeding a shared back end could be coordinated by Kubernetes.

## Sources

[^wiki]: [Kubernetes](https://en.wikipedia.org/wiki/Kubernetes) — Wikipedia, on Kubernetes' design and history.
[^home]: [Kubernetes](https://kubernetes.io/) — the project's official site.
