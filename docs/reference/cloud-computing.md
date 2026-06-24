---
slug: cloud-computing
title: Cloud computing
entry_type: concept
category: hw-servers
description: Cloud computing is computing power and storage provided over the internet on demand, scaling up and down with near-zero upfront cost and ongoing fees.
keywords: cloud computing, on demand, data center, IaaS, scalable, pay as you go
aka: [Cloud computing, The cloud]
infobox:
  - { label: Type, value: On-demand computing }
  - { label: Cost model, value: Pay as you go }
  - { label: Scales, value: Up and down }
  - { label: Built on, value: Virtualization }
see_also: [virtualization, virtual-private-server, server, dedicated-server, home-server]
related_lessons:
  - { title: "Combining tiers", url: /learn/intro-hardware/combining-tiers/ }
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Cloud_computing
---

**Cloud computing** is computing power and storage provided over the internet on demand, instead of from machines you own and run.[^wiki]

## Overview

The defining trait is elasticity: you can scale resources up when traffic spikes and back down when it fades, with near-zero upfront cost and ongoing fees for what you use. Under the hood it is [virtualization](/reference/virtualization/) at scale, run across large data centers — the same technology behind a single [virtual private server](/reference/virtual-private-server/), multiplied.

## Trade-offs

The cloud trades capital cost for recurring fees and adds network latency compared with a local device, so it is not always the right home for low-latency or always-on tasks. It often pairs with on-site hardware in combined-tier systems: a [home server](/reference/home-server/) captures and pre-processes locally while the cloud stores and serves the results — the model GopherTrunk fits, since the cloud cannot touch RF directly.

## Sources

[^wiki]: [Cloud computing](https://en.wikipedia.org/wiki/Cloud_computing) — Wikipedia, on on-demand computing over the internet.
