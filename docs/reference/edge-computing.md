---
slug: edge-computing
title: Edge computing
entry_type: concept
category: hw-servers
description: Edge computing processes data near where it is produced — at or close to the device or sensor — instead of sending everything to a distant data center, reducing latency and bandwidth use.
keywords: edge computing, edge, latency, bandwidth, local processing, IoT, near the source
infobox:
  - { label: Type, value: Computing model }
  - { label: Runs, value: Near the data source }
  - { label: Reduces, value: Latency & bandwidth }
  - { label: Contrast, value: Centralized cloud }
see_also: [cloud-computing, content-delivery-network, internet-of-things, raspberry-pi, data-center, scalability]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Edge_computing
---

**Edge computing** processes data near where it is produced — at or close to the device or sensor — instead of shipping everything to a distant [data center](/reference/data-center/), reducing latency and bandwidth use.[^wiki]

## Overview

In a purely centralized model, devices send raw data to the [cloud](/reference/cloud-computing/) and wait for a response. Edge computing moves some of that work outward: a small computer near the source filters, aggregates, or acts on data locally, sending only summaries or alerts upstream. This cuts round-trip latency, saves bandwidth, and keeps working when the network link is slow or down. It is closely tied to the [internet of things](/reference/internet-of-things/), where many sensors generate more data than is practical to forward whole.

## Where it fits

Edge computing complements the cloud rather than replacing it — heavy storage and analytics stay central while time-sensitive or high-volume processing happens locally. A [content delivery network](/reference/content-delivery-network/) is edge computing for content. GopherTrunk is naturally an edge workload: a [Raspberry Pi](/reference/raspberry-pi/) by the antenna decodes RF on the spot and forwards only the decoded calls, rather than streaming raw IQ across the network.

## Sources

[^wiki]: [Edge computing](https://en.wikipedia.org/wiki/Edge_computing) — Wikipedia, on the edge computing model.
