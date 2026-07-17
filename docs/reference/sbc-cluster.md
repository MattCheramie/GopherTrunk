---
slug: sbc-cluster
title: SBC cluster
entry_type: concept
category: hw-sbc
description: An SBC cluster is several single-board computers networked together to share work, used to learn distributed computing, build cheap low-power compute, or run lightweight container orchestration at home.
keywords: SBC cluster, Raspberry Pi cluster, Pi cluster, Kubernetes cluster, distributed computing, parallel computing, home lab, low-power cluster, edge nodes
aka: [Pi cluster]
infobox:
  - { label: Type, value: Computing arrangement }
  - { label: Made of, value: Several networked SBCs }
  - { label: Linked by, value: Ethernet / a switch }
  - { label: Used for, value: Learning, light compute, redundancy }
  - { label: Often runs, value: Kubernetes / container orchestration }
see_also: [single-board-computer, raspberry-pi, odroid, rock-pi, edge-computing, home-server]
related_lessons:
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_cluster
---

**An SBC cluster** is several [single-board computers](/reference/single-board-computer/) networked together so they share work — a small computer cluster built from boards like the [Raspberry Pi](/reference/raspberry-pi/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="A cluster topology. A central network switch connects up to four single-board computer nodes over Ethernet. One node is marked as the controller that schedules work, and the other three are workers that run it; a job can be spread across the workers and a failed board replaced without taking the cluster down." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="176" y="66" width="108" height="30" rx="4" fill-opacity="0.14" fill="currentColor"/>
    <path d="M110 66 L200 42 M230 96 L230 130 M320 66 L260 42 M150 96 L150 130 M310 96 L310 130"/>
    <rect x="70" y="20" width="80" height="26" rx="4" fill-opacity="0.16" fill="currentColor"/>
    <rect x="210" y="20" width="80" height="26" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="110" y="130" width="80" height="26" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="270" y="130" width="80" height="26" rx="4" fill-opacity="0.06" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="230" y="85" font-size="9" font-weight="600">network switch</text>
    <text x="110" y="37" font-size="7.5" font-weight="600">controller</text>
    <text x="250" y="37" font-size="7.5">worker</text>
    <text x="150" y="147" font-size="7.5">worker</text>
    <text x="310" y="147" font-size="7.5">worker</text>
    <text x="230" y="164" font-size="7.5" fill-opacity="0.9">controller schedules · workers run · a dead node is just replaced</text>
  </g>
</svg>
<figcaption>An SBC cluster wires several boards to a switch: a controller node schedules work across the worker nodes, and because the work is spread rather than pinned, a failed board can be swapped out without bringing the whole system down.</figcaption>
</figure>

## Overview

The boards are linked over Ethernet through a switch and coordinated by software, frequently Kubernetes or another container orchestrator, so jobs can be spread across nodes and a failed board can be replaced without taking the whole system down. One board usually acts as the controller that decides what runs where, and the rest are workers that carry it out; the orchestrator watches for a node dropping out and reschedules its work elsewhere.

People build them to learn distributed computing hands-on, to assemble cheap and low-power compute, or to run a resilient home lab. A rack of small boards is a tangible way to see scheduling, networking, and failover work — everything a cloud does invisibly is here on a shelf, blinking, and cheap enough that a mistake costs a board, not a bill.

## Why build one

| Motive | What the cluster gives | Honest limit |
|--------|------------------------|--------------|
| Learning | Real distributed-systems practice | A single PC is faster |
| Redundancy | Survives one node failing | More parts to fail overall |
| Low-power compute | Many watts-scale nodes | Poor throughput per dollar |
| Physical distribution | Nodes near different locations | Coordination overhead |

## Trade-offs

A cluster of small boards rarely beats one capable machine on raw throughput per dollar or watt, so the payoff is usually learning, redundancy, or physical distribution rather than peak performance. Splitting a single heavy job across weak nodes and a slow network often runs slower than the same job on one strong box, once the coordination cost is counted.

In GopherTrunk terms, the natural reason to spread work across boards is geography, not horsepower: separate capture nodes near different antennas, each decoding locally and feeding a shared collector — closer to [edge computing](/reference/edge-computing/) than to a number-crunching cluster. Each node does the radio work where the signal is strong, and only compact decoded results travel back over the network, which is exactly the case where distribution earns its keep.

## Sources

[^wiki]: [Computer cluster](https://en.wikipedia.org/wiki/Computer_cluster) — Wikipedia, on networking computers to work as one system.
