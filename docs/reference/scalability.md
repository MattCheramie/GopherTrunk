---
slug: scalability
title: Scalability
entry_type: concept
category: hw-servers
description: Scalability is a system's ability to handle growing load by adding resources, either by using bigger machines (vertical) or by adding more machines (horizontal).
keywords: scalability, scaling, vertical scaling, horizontal scaling, scale up, scale out, elasticity, stateless
infobox:
  - { label: Type, value: System property }
  - { label: Vertical, value: Bigger machine (scale up) }
  - { label: Horizontal, value: More machines (scale out) }
  - { label: Related, value: Elasticity }
  - { label: Sibling goal, value: High availability }
see_also: [high-availability, load-balancer, cloud-computing, kubernetes, serverless-computing, server]
cite_urls:
  - https://en.wikipedia.org/wiki/Scalability
---

**Scalability** is a system's ability to handle growing load by adding resources — either by using a bigger machine (vertical scaling) or by adding more machines (horizontal scaling).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Two scaling strategies. Scaling up replaces one server with a single larger, more powerful server. Scaling out keeps modest servers but adds more of them side by side behind a load balancer that spreads the work across all of them." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <text x="115" y="18" text-anchor="middle" font-size="9" font-weight="600" stroke="none">SCALE UP (vertical)</text>
    <rect x="60" y="112" width="34" height="30" rx="2" fill-opacity="0.14" stroke-width="1.2"/>
    <text x="77" y="131" text-anchor="middle" font-size="7" stroke="none">1x</text>
    <line x1="104" y1="127" x2="128" y2="127" stroke-width="1.3" fill="none"/>
    <path d="M128 127 l-8 -3 v6 z" stroke-width="1"/>
    <rect x="134" y="60" width="56" height="82" rx="3" fill-opacity="0.22" stroke-width="1.4"/>
    <text x="162" y="104" text-anchor="middle" font-size="8" stroke="none">4x</text>
    <text x="115" y="160" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">one bigger machine</text>
    <text x="115" y="171" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.7">simple, but has a ceiling</text>

    <text x="345" y="18" text-anchor="middle" font-size="9" font-weight="600" stroke="none">SCALE OUT (horizontal)</text>
    <rect x="300" y="34" width="90" height="20" rx="3" fill-opacity="0.18" stroke-width="1.3"/>
    <text x="345" y="48" text-anchor="middle" font-size="7.5" stroke="none">load balancer</text>
    <g stroke-width="1.2" fill-opacity="0.14">
      <rect x="278" y="96" width="30" height="30" rx="2"/>
      <rect x="314" y="96" width="30" height="30" rx="2"/>
      <rect x="350" y="96" width="30" height="30" rx="2"/>
      <rect x="386" y="96" width="30" height="30" rx="2"/>
    </g>
    <g stroke-width="1" fill="none">
      <line x1="330" y1="54" x2="293" y2="96"/>
      <line x1="340" y1="54" x2="329" y2="96"/>
      <line x1="350" y1="54" x2="365" y2="96"/>
      <line x1="360" y1="54" x2="401" y2="96"/>
    </g>
    <text x="345" y="145" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">more modest machines</text>
    <text x="345" y="156" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.7">no hard ceiling; needs stateless design</text>
  </g>
</svg>
<figcaption>Scaling up swaps in a single larger server — simple but bounded by the biggest box you can buy; scaling out adds more modest servers behind a load balancer, which has no hard ceiling but requires software built to spread across many machines.</figcaption>
</figure>

## Overview

*Vertical* scaling, or scaling up, means giving one server more CPU, memory, or faster disks. It is simple but bounded by the largest machine you can buy and leaves a single point of failure. *Horizontal* scaling, or scaling out, means running many servers behind a [load balancer](/reference/load-balancer/) and dividing the work among them.

Scaling out has no hard ceiling and pairs naturally with redundancy, but the software must be designed for it — *stateless* services and shared data stores scale out far more easily than monoliths that keep session state in local memory. *Elasticity* is the related ability to add and remove capacity automatically as demand changes, adding servers under load and releasing them when the spike passes.

## Trade-offs

The two directions have opposite strengths, and real systems often scale up first, then out:

| Aspect | Scale up (vertical) | Scale out (horizontal) |
|--------|--------------------|------------------------|
| How | Bigger single machine | More machines |
| Ceiling | Largest box available | Effectively none |
| Complexity | Low | Higher (distribution) |
| Fault tolerance | Single point of failure | Survives node loss |
| App requirement | Runs as-is | Must be stateless-friendly |

Scaling up is the quick win for a single overloaded server; scaling out is what carries a system past the limits of any one machine — and, done right, improves availability along the way.

## Where it fits

Scalability is about handling *more*, while [high availability](/reference/high-availability/) is about staying *up*; horizontal scaling tends to deliver both. [Cloud computing](/reference/cloud-computing/), [Kubernetes](/reference/kubernetes/), and [serverless computing](/reference/serverless-computing/) all exist largely to make scaling out routine. GopherTrunk scales horizontally in a natural way: cover more sites by adding capture nodes, each decoding its own RF and feeding a shared back end.

## Sources

[^wiki]: [Scalability](https://en.wikipedia.org/wiki/Scalability) — Wikipedia, on vertical and horizontal scaling and elasticity.
