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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 210" role="img" aria-label="Clients on the left send requests to a load balancer in the middle, which spreads them across a pool of four identical backend servers on the right. One server has failed its health check and is greyed out and removed from the pool, so no traffic is routed to it." xmlns="http://www.w3.org/2000/svg">
  <text x="52" y="24" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">clients</text>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <circle cx="52" cy="60" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="52" cy="105" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="52" cy="150" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
  </g>
  <rect x="152" y="80" width="96" height="50" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="200" y="100" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">Load balancer</text>
  <text x="200" y="115" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">spread · health-check</text>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75" fill="none">
    <line x1="65" y1="62" x2="152" y2="98" marker-end="url(#lb_ar)"/>
    <line x1="65" y1="105" x2="152" y2="105" marker-end="url(#lb_ar)"/>
    <line x1="65" y1="148" x2="152" y2="112" marker-end="url(#lb_ar)"/>
  </g>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="340" y="20" width="82" height="32" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="381" y="40">server</text>
    <rect x="340" y="62" width="82" height="32" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="381" y="82">server</text>
    <rect x="340" y="104" width="82" height="32" rx="4" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1" stroke-opacity="0.5" stroke-dasharray="4 3"/><text x="381" y="124" fill-opacity="0.5">server ✕</text>
    <rect x="340" y="146" width="82" height="32" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="381" y="166">server</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="248" y1="100" x2="340" y2="36" stroke-opacity="0.75" marker-end="url(#lb_ar)"/>
    <line x1="248" y1="103" x2="340" y2="78" stroke-opacity="0.75" marker-end="url(#lb_ar)"/>
    <line x1="248" y1="112" x2="340" y2="120" stroke-opacity="0.3" stroke-dasharray="4 3"/>
    <line x1="248" y1="115" x2="340" y2="162" stroke-opacity="0.75" marker-end="url(#lb_ar)"/>
  </g>
  <text x="294" y="128" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.6">removed</text>
  <text x="220" y="198" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">a failed health check is pulled from the pool — the rest keep serving</text>
  <defs><marker id="lb_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The balancer fronts a pool of identical backends and spreads requests among them (round-robin, least-connections, or client hashing). Health checks pull a dead server out automatically, so the pool survives losing any one member — the basis of both scaling out and staying available.</figcaption>
</figure>

## Overview

A load balancer sits in front of a pool of identical backends and spreads requests among them using rules such as round-robin, least-connections, or hashing on the client. It also runs *health checks*, automatically removing a server that stops responding so users are not routed to a dead machine. Balancers operate at the transport layer (L4, forwarding TCP/UDP) or the application layer (L7, inspecting HTTP to route by path or host). They may be dedicated hardware appliances or software running on commodity servers.

## Where it fits

Load balancing is fundamental to [scalability](/reference/scalability/) — adding servers behind a balancer increases throughput — and to [high availability](/reference/high-availability/), since the pool survives the loss of any one member. It overlaps with a [reverse proxy](/reference/reverse-proxy/), which also fronts backends, and with a [content delivery network](/reference/content-delivery-network/) for global traffic. Orchestrators like [Kubernetes](/reference/kubernetes/) provide built-in balancing across pods.

## Sources

[^wiki]: [Load balancing (computing)](https://en.wikipedia.org/wiki/Load_balancing_(computing)) — Wikipedia, on load balancing techniques.
