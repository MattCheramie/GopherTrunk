---
slug: scalability
title: Scalability
entry_type: concept
category: hw-servers
description: Scalability is a system's ability to handle growing load by adding resources, either by using bigger machines (vertical) or by adding more machines (horizontal).
keywords: scalability, scaling, vertical scaling, horizontal scaling, scale up, scale out, elasticity
infobox:
  - { label: Type, value: System property }
  - { label: Vertical, value: Bigger machine (scale up) }
  - { label: Horizontal, value: More machines (scale out) }
  - { label: Related, value: Elasticity }
see_also: [high-availability, load-balancer, cloud-computing, kubernetes, serverless-computing, server]
cite_urls:
  - https://en.wikipedia.org/wiki/Scalability
---

**Scalability** is a system's ability to handle growing load by adding resources — either by using a bigger machine (vertical scaling) or by adding more machines (horizontal scaling).[^wiki]

## Overview

*Vertical* scaling, or scaling up, means giving one server more CPU, memory, or faster disks. It is simple but bounded by the largest machine you can buy and leaves a single point of failure. *Horizontal* scaling, or scaling out, means running many servers behind a [load balancer](/reference/load-balancer/) and dividing the work among them. Scaling out has no hard ceiling and pairs naturally with redundancy, but the software must be designed for it — stateless services and shared data stores scale out far more easily than monoliths. *Elasticity* is the related ability to add and remove capacity automatically as demand changes.

## Where it fits

Scalability is about handling *more*, while [high availability](/reference/high-availability/) is about staying *up*; horizontal scaling tends to deliver both. [Cloud computing](/reference/cloud-computing/), [Kubernetes](/reference/kubernetes/), and [serverless computing](/reference/serverless-computing/) all exist largely to make scaling out routine. GopherTrunk scales horizontally in a natural way: cover more sites by adding capture nodes, each decoding its own RF and feeding a shared back end.

## Sources

[^wiki]: [Scalability](https://en.wikipedia.org/wiki/Scalability) — Wikipedia, on vertical and horizontal scaling.
