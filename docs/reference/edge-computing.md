---
slug: edge-computing
title: Edge computing
entry_type: concept
category: hw-servers
description: Edge computing processes data near where it is produced — at or close to the device or sensor — instead of sending everything to a distant data center, reducing latency and bandwidth use.
keywords: edge computing, edge, latency, bandwidth, local processing, IoT, near the source, fog computing
infobox:
  - { label: Type, value: Computing model }
  - { label: Runs, value: Near the data source }
  - { label: Reduces, value: Latency & bandwidth }
  - { label: Contrast, value: Centralized cloud }
  - { label: Ally, value: Cloud (for heavy lifting) }
see_also: [cloud-computing, content-delivery-network, internet-of-things, raspberry-pi, data-center, scalability]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Edge_computing
---

**Edge computing** processes data near where it is produced — at or close to the device or sensor — instead of shipping everything to a distant [data center](/reference/data-center/), reducing latency and bandwidth use.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A topology showing sensors on the left feeding a nearby edge node that filters and acts on data locally, then sends only small summaries upstream over a thin link to a central cloud data center on the right. A dashed baseline contrasts a purely centralized model that would send all raw data to the cloud." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <g stroke-width="1.1" fill-opacity="0.16">
      <circle cx="34" cy="46" r="9"/><circle cx="34" cy="78" r="9"/><circle cx="34" cy="110" r="9"/>
    </g>
    <text x="34" y="132" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">sensors</text>
    <g stroke-width="1.3" fill="none">
      <line x1="44" y1="46" x2="120" y2="72"/>
      <line x1="44" y1="78" x2="120" y2="78"/>
      <line x1="44" y1="110" x2="120" y2="84"/>
    </g>
    <rect x="122" y="52" width="96" height="56" rx="4" fill-opacity="0.14" stroke-width="1.4"/>
    <text x="170" y="76" text-anchor="middle" font-size="8.5" stroke="none" font-weight="600">Edge node</text>
    <text x="170" y="90" text-anchor="middle" font-size="7.5" stroke="none">filter &#183; aggregate &#183; act</text>
    <text x="170" y="126" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">close to the source</text>
    <line x1="218" y1="80" x2="330" y2="80" stroke-width="1.6" fill="none"/>
    <path d="M330 80 l-8 -3 v6 z" stroke-width="1"/>
    <text x="274" y="72" text-anchor="middle" font-size="7.5" stroke="none">only summaries</text>
    <text x="274" y="94" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.7">thin uplink</text>
    <rect x="332" y="50" width="104" height="60" rx="4" fill-opacity="0.06" stroke-width="1.4"/>
    <text x="384" y="76" text-anchor="middle" font-size="8.5" stroke="none" font-weight="600">Cloud</text>
    <text x="384" y="90" text-anchor="middle" font-size="7.5" stroke="none">storage &#183; analytics</text>
    <text x="384" y="126" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">distant data center</text>
    <text x="230" y="160" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">a purely central model would push ALL raw data across the same link</text>
  </g>
</svg>
<figcaption>An edge node near the sensors does the time-sensitive and high-volume work locally and forwards only compact summaries upstream, so the central cloud handles heavy storage and analytics without drowning in raw data.</figcaption>
</figure>

## Overview

In a purely centralized model, devices send raw data to the [cloud](/reference/cloud-computing/) and wait for a response. Edge computing moves some of that work outward: a small computer near the source filters, aggregates, or acts on data locally, sending only summaries or alerts upstream. This cuts round-trip latency, saves bandwidth, and keeps working when the network link is slow or down.

It is closely tied to the [internet of things](/reference/internet-of-things/), where many sensors generate more data than is practical to forward whole. The edge node may be an industrial gateway, a cell-tower micro data center, or a hobbyist single-board computer — the defining trait is proximity to where data is born, not any particular hardware.

## Trade-offs

Where computation happens is a balance, and edge and cloud each win on different axes:

| Property | Edge | Central cloud |
|----------|------|---------------|
| Latency | Low (local) | Higher (round trip) |
| Bandwidth use | Low (summaries) | High (raw data) |
| Compute power | Modest | Effectively unlimited |
| Works offline | Yes | No |
| Storage & analytics | Limited | Deep |

The two are complementary, not rivals: keep the reflex-fast, high-volume work at the edge and let the cloud do the heavy, long-horizon lifting.

## Where it fits

Edge computing complements the cloud rather than replacing it — heavy storage and analytics stay central while time-sensitive or high-volume processing happens locally. A [content delivery network](/reference/content-delivery-network/) is edge computing for content. GopherTrunk is naturally an edge workload: a [Raspberry Pi](/reference/raspberry-pi/) by the antenna decodes RF on the spot and forwards only the decoded calls, rather than streaming raw IQ across the network.

## Sources

[^wiki]: [Edge computing](https://en.wikipedia.org/wiki/Edge_computing) — Wikipedia, on the edge computing model.
