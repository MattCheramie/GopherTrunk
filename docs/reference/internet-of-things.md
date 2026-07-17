---
slug: internet-of-things
title: Internet of Things (IoT)
entry_type: concept
category: hw-microcontrollers
description: The Internet of Things is the network of everyday physical objects with embedded computing and connectivity that send and receive data, typically built on cheap wireless microcontrollers reporting through a gateway to a server or the cloud.
keywords: Internet of Things, IoT, smart home, embedded, wireless sensors, ESP32, connected devices, gateway, cloud
aka: [IoT, Internet of Things]
autolink: true
infobox:
  - { label: Type, value: Networked embedded devices }
  - { label: Typical node, value: Cheap wireless microcontroller }
  - { label: Connectivity, value: Wi-Fi, Bluetooth, cellular }
  - { label: Reports to, value: Gateway, server, or cloud }
see_also: [esp32, microcontroller, sensor, cloud-computing, firmware, server]
related_lessons:
  - { title: "ESP32", url: /learn/intro-hardware/esp32/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Internet_of_things
---

**The Internet of Things (IoT)** is the network of everyday physical objects — sensors, appliances, wearables — with embedded computing and connectivity that lets them send and receive data.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 156" role="img" aria-label="Data flow of an Internet of Things system. On the left several small sensor nodes send wireless readings to a local gateway. The gateway forwards the data over the internet to a cloud service in the centre-right, which stores and analyses it. A dashboard on a phone reads results back from the cloud, and control commands flow back down to the nodes." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="24" y="34" width="52" height="22" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="24" y="66" width="52" height="22" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="24" y="98" width="52" height="22" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="150" y="62" width="66" height="30" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <path d="M286 60 q10 -20 32 -14 q6 -18 30 -12 q22 -4 22 16 q18 4 8 22 q4 16 -20 14 l-62 0 q-24 2 -22 -18 q-12 -8 -0 -18 z" fill="currentColor" fill-opacity="0.08"/>
    <rect x="392" y="96" width="44" height="30" rx="4" fill="currentColor" fill-opacity="0.10"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="76" y1="45" x2="150" y2="72"/>
    <line x1="76" y1="77" x2="150" y2="77"/>
    <line x1="76" y1="109" x2="150" y2="82"/>
    <line x1="216" y1="77" x2="300" y2="70"/>
    <path d="M336 100 L336 111 L392 111" stroke-dasharray="3 3"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="50" y="49" font-size="7.5">node</text>
    <text x="50" y="81" font-size="7.5">node</text>
    <text x="50" y="113" font-size="7.5">node</text>
    <text x="10" y="132" font-size="7.5" text-anchor="start">cheap wireless MCUs + sensors</text>
    <text x="183" y="74" font-size="8" font-weight="600">Gateway</text>
    <text x="183" y="85" font-size="7" fill-opacity="0.85">local hub</text>
    <text x="332" y="66" font-size="8.5" font-weight="600">Cloud</text>
    <text x="332" y="78" font-size="7" fill-opacity="0.85">store + analyse</text>
    <text x="414" y="115" font-size="7.5">app</text>
  </g>
</svg>
<figcaption>A typical IoT path: cheap wireless sensor nodes report to a local gateway, which forwards the data over the internet to a cloud service that stores and analyses it; a phone app reads results back and control commands flow back down to the nodes. Each node is tiny, but collectively they number in the billions.</figcaption>
</figure>

## Overview

IoT devices are typically built on cheap wireless [microcontrollers](/reference/microcontroller/) like the [ESP32](/reference/esp32/), each reading the world through [sensors](/reference/sensor/) and reporting to a [server](/reference/server/) or the [cloud](/reference/cloud-computing/). Individually each node is tiny; collectively they number in the billions.

Most systems follow a tiered shape: constrained nodes at the edge, a gateway that aggregates and relays their traffic, and a backend that stores, analyses, and visualises the data. That structure keeps the per-node hardware — and its power draw — as small as possible.

## Connectivity choices

Nodes reach the network in different ways depending on range, power, and data rate:

| Link | Range | Power | Typical use |
|------|-------|-------|-------------|
| Wi-Fi | Home / building | Higher | Smart-home gadgets |
| Bluetooth LE | Room | Low | Wearables, beacons |
| [LoRa](/reference/lora/) / LPWAN | Kilometres | Very low | Rural/field sensors |
| Cellular (NB-IoT/LTE-M) | Wide-area | Medium | Trackers, metering |

## Where it fits

The appeal of IoT is putting just enough computing into ordinary objects to make them measurable and controllable from afar. Those same low-power wireless nodes are part of the radio landscape: their transmissions are among the signals filling the airwaves GopherTrunk listens to, though the devices themselves are far too small to run it. The tiered node-gateway-cloud pattern also mirrors a distributed SDR setup, where small capture nodes feed a central decode server.

## Sources

[^wiki]: [Internet of things](https://en.wikipedia.org/wiki/Internet_of_things) — Wikipedia, on networked physical objects with embedded computing and connectivity.
