---
slug: high-availability
title: High availability
entry_type: concept
category: hw-servers
description: High availability is the design of systems to keep running with minimal downtime by removing single points of failure through redundancy, failover, and health monitoring.
keywords: high availability, HA, uptime, redundancy, failover, single point of failure, nines, active-passive
aka: [HA]
infobox:
  - { label: Type, value: System design goal }
  - { label: Achieved by, value: Redundancy & failover }
  - { label: Measured in, value: "Uptime (nines)" }
  - { label: Enemy, value: Single point of failure }
  - { label: Sibling goal, value: Scalability }
see_also: [scalability, load-balancer, raid, data-center, kubernetes, server]
cite_urls:
  - https://en.wikipedia.org/wiki/High_availability
---

**High availability** (HA) is the practice of designing systems to keep running with minimal downtime, by removing single points of failure through redundancy, failover, and health monitoring.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A high-availability pair. Clients connect to a load balancer, which health-checks two servers and routes traffic to the active primary. A dashed standby server waits to take over. If the primary fails, the load balancer redirects to the standby, which is promoted to active." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <text x="42" y="94" text-anchor="middle" font-size="8" stroke="none">clients</text>
    <g stroke-width="1.1" fill-opacity="0.16">
      <circle cx="26" cy="72" r="6"/><circle cx="26" cy="98" r="6"/><circle cx="42" cy="112" r="6"/>
    </g>
    <line x1="52" y1="92" x2="118" y2="92" stroke-width="1.4" fill="none"/>
    <path d="M118 92 l-8 -3 v6 z" stroke-width="1"/>
    <rect x="120" y="70" width="80" height="44" rx="4" fill-opacity="0.18" stroke-width="1.4"/>
    <text x="160" y="88" text-anchor="middle" font-size="8" stroke="none" font-weight="600">Load</text>
    <text x="160" y="100" text-anchor="middle" font-size="8" stroke="none" font-weight="600">balancer</text>
    <text x="160" y="112" text-anchor="middle" font-size="6.5" stroke="none" fill-opacity="0.8">health checks</text>
    <line x1="200" y1="82" x2="320" y2="52" stroke-width="1.6" fill="none"/>
    <path d="M320 52 l-8 -1 3 6 z" stroke-width="1"/>
    <line x1="200" y1="102" x2="320" y2="132" stroke-width="1.2" stroke-dasharray="4 3" fill="none"/>
    <rect x="322" y="34" width="96" height="40" rx="4" fill-opacity="0.16" stroke-width="1.4"/>
    <text x="370" y="52" text-anchor="middle" font-size="8" stroke="none" font-weight="600">Primary</text>
    <text x="370" y="65" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.85">active</text>
    <rect x="322" y="114" width="96" height="40" rx="4" fill-opacity="0.04" stroke-width="1.3" stroke-dasharray="4 3"/>
    <text x="370" y="132" text-anchor="middle" font-size="8" stroke="none">Standby</text>
    <text x="370" y="145" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.85">promoted on failure</text>
    <line x1="370" y1="74" x2="370" y2="114" stroke-width="1" stroke-dasharray="2 2" fill="none"/>
    <text x="384" y="97" font-size="6.5" stroke="none" fill-opacity="0.7">replication</text>
    <text x="230" y="182" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">no single component can take the whole service down</text>
  </g>
</svg>
<figcaption>A load balancer health-checks a redundant pair and sends traffic to the active primary; when the primary fails its health check, traffic fails over to the replicated standby, which is promoted — so one dead component never means an outage.</figcaption>
</figure>

## Overview

Availability is often expressed in "nines": 99.9% uptime allows about nine hours of downtime a year, while 99.999% ("five nines") allows about five minutes. Reaching high numbers means that no single component — server, disk, network link, power feed — can take the whole system down.

The toolkit includes redundant hardware, [load balancers](/reference/load-balancer/) that route around dead servers, [RAID](/reference/raid/) for disks, replicated databases, and automatic *failover* to a standby when the primary fails. Health monitoring ties it together: the system must *detect* a failure quickly, or redundancy sits idle while users see errors. The [data center](/reference/data-center/) itself contributes with redundant power and cooling.

## The nines

Each additional nine cuts allowable downtime roughly tenfold — and gets sharply more expensive to guarantee:

| Availability | Nickname | Downtime / year |
|--------------|----------|-----------------|
| 99% | Two nines | ~3.65 days |
| 99.9% | Three nines | ~8.8 hours |
| 99.99% | Four nines | ~53 minutes |
| 99.999% | Five nines | ~5 minutes |

The right target is an economic choice: chase only as many nines as the cost of downtime justifies, since redundancy, testing, and on-call all scale with the goal.

## Where it fits

High availability is about staying *up*, while [scalability](/reference/scalability/) is about handling *more load*; the two are related but distinct, and many techniques (replication, multiple servers) serve both. Orchestrators like [Kubernetes](/reference/kubernetes/) automate failover for containerized apps. A hobby GopherTrunk node rarely justifies HA, but a monitoring site that must never miss a call would replicate capture nodes and back-end [servers](/reference/server/) so any one can fail without an outage.

## Sources

[^wiki]: [High availability](https://en.wikipedia.org/wiki/High_availability) — Wikipedia, on availability, redundancy, failover, and the nines.
