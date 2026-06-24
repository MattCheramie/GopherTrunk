---
slug: high-availability
title: High availability
entry_type: concept
category: hw-servers
description: High availability is the design of systems to keep running with minimal downtime by removing single points of failure through redundancy, failover, and health monitoring.
keywords: high availability, HA, uptime, redundancy, failover, single point of failure, nines
aka: [HA]
infobox:
  - { label: Type, value: System design goal }
  - { label: Achieved by, value: Redundancy & failover }
  - { label: Measured in, value: "Uptime (nines)" }
  - { label: Enemy, value: Single point of failure }
see_also: [scalability, load-balancer, raid, data-center, kubernetes, server]
cite_urls:
  - https://en.wikipedia.org/wiki/High_availability
---

**High availability** (HA) is the practice of designing systems to keep running with minimal downtime, by removing single points of failure through redundancy, failover, and health monitoring.[^wiki]

## Overview

Availability is often expressed in "nines": 99.9% uptime allows about nine hours of downtime a year, while 99.999% ("five nines") allows about five minutes. Reaching high numbers means that no single component — server, disk, network link, power feed — can take the whole system down. The toolkit includes redundant hardware, [load balancers](/reference/load-balancer/) that route around dead servers, [RAID](/reference/raid/) for disks, replicated databases, and automatic *failover* to a standby when the primary fails. The [data center](/reference/data-center/) itself contributes with redundant power and cooling.

## Where it fits

High availability is about staying *up*, while [scalability](/reference/scalability/) is about handling *more load*; the two are related but distinct, and many techniques (replication, multiple servers) serve both. Orchestrators like [Kubernetes](/reference/kubernetes/) automate failover for containerized apps. A hobby GopherTrunk node rarely justifies HA, but a monitoring site that must never miss a call would replicate capture nodes and back-end servers so any one can fail without an outage.

## Sources

[^wiki]: [High availability](https://en.wikipedia.org/wiki/High_availability) — Wikipedia, on availability, redundancy, and failover.
