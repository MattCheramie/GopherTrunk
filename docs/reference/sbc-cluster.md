---
slug: sbc-cluster
title: SBC cluster
entry_type: concept
category: hw-sbc
description: An SBC cluster is several single-board computers networked together to share work, used to learn distributed computing, build cheap low-power compute, or run lightweight container orchestration at home.
keywords: SBC cluster, Raspberry Pi cluster, Pi cluster, Kubernetes cluster, distributed computing, parallel computing, home lab, low-power cluster
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

## Overview

The boards are linked over Ethernet through a switch and coordinated by software, frequently Kubernetes or another container orchestrator, so jobs can be spread across nodes and a failed board can be replaced without taking the whole system down. People build them to learn distributed computing hands-on, to assemble cheap and low-power compute, or to run a resilient home lab.

## Trade-offs

A cluster of small boards rarely beats one capable machine on raw throughput per dollar or watt, so the payoff is usually learning, redundancy, or physical distribution rather than peak performance. In GopherTrunk terms, the natural reason to spread work across boards is geography, not horsepower: separate capture nodes near different antennas, each decoding locally and feeding a shared collector — closer to [edge computing](/reference/edge-computing/) than to a number-crunching cluster.

## Sources

[^wiki]: [Computer cluster](https://en.wikipedia.org/wiki/Computer_cluster) — Wikipedia, on networking computers to work as one system.
