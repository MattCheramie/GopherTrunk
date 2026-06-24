---
slug: load-balancer
title: Load balancer
entry_type: concept
category: hw-servers
description: A load balancer distributes incoming network traffic across multiple servers so that no single machine is overwhelmed, improving capacity, responsiveness, and availability.
keywords: load balancer, load balancing, traffic distribution, health check, round robin, L4, L7
infobox:
  - { label: Type, value: Traffic distributor }
  - { label: Spreads, value: Requests across servers }
  - { label: Layers, value: "L4 (transport), L7 (application)" }
  - { label: Improves, value: Capacity & availability }
see_also: [reverse-proxy, high-availability, scalability, server, content-delivery-network, kubernetes]
cite_urls:
  - https://en.wikipedia.org/wiki/Load_balancing_(computing)
---

A **load balancer** distributes incoming network traffic across multiple [servers](/reference/server/) so that no single machine is overwhelmed, improving capacity, responsiveness, and availability.[^wiki]

## Overview

A load balancer sits in front of a pool of identical backends and spreads requests among them using rules such as round-robin, least-connections, or hashing on the client. It also runs *health checks*, automatically removing a server that stops responding so users are not routed to a dead machine. Balancers operate at the transport layer (L4, forwarding TCP/UDP) or the application layer (L7, inspecting HTTP to route by path or host). They may be dedicated hardware appliances or software running on commodity servers.

## Where it fits

Load balancing is fundamental to [scalability](/reference/scalability/) — adding servers behind a balancer increases throughput — and to [high availability](/reference/high-availability/), since the pool survives the loss of any one member. It overlaps with a [reverse proxy](/reference/reverse-proxy/), which also fronts backends, and with a [content delivery network](/reference/content-delivery-network/) for global traffic. Orchestrators like [Kubernetes](/reference/kubernetes/) provide built-in balancing across pods.

## Sources

[^wiki]: [Load balancing (computing)](https://en.wikipedia.org/wiki/Load_balancing_(computing)) — Wikipedia, on load balancing techniques.
