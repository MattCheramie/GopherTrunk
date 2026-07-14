---
slug: usb
title: USB
entry_type: hardware
category: hw-foundations
description: USB (Universal Serial Bus) is the dominant standard for connecting peripherals and delivering power to computers, using a tiered host-controlled bus with hot-pluggable ports.
keywords: USB, Universal Serial Bus, peripheral, hot-plug, USB-C, USB 3, USB power delivery, host controller
aka: [Universal Serial Bus, USB-C]
autolink: true
infobox:
  - { label: Type, value: Serial peripheral bus }
  - { label: Topology, value: Host-controlled, tiered }
  - { label: Connectors, value: "Type-A, Type-C, micro" }
  - { label: Also carries, value: Power (USB-PD) }
see_also: [input-output, system-bus, pci-express, peripheral, central-processing-unit, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/USB
---

**USB** (Universal Serial Bus) is the most common standard for connecting peripherals to a computer and for delivering power, using a hot-pluggable, host-controlled serial bus.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 236" role="img" aria-label="A tiered USB tree: one host controller at the top branches to two hubs, and each hub in turn branches to two devices — a mouse and a drive under one, a camera and a dongle under the other — the nesting that lets the bus reach up to 127 devices." xmlns="http://www.w3.org/2000/svg">
  <rect x="178" y="18" width="104" height="36" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="35" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">host controller</text>
  <text x="230" y="48" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">root</text>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="70" y="96" width="80" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="110" y="115">hub</text>
    <rect x="310" y="96" width="80" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="350" y="115">hub</text>
  </g>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <rect x="24" y="176" width="64" height="28" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/><text x="56" y="194">mouse</text>
    <rect x="100" y="176" width="64" height="28" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/><text x="132" y="194">drive</text>
    <rect x="270" y="176" width="64" height="28" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/><text x="302" y="194">camera</text>
    <rect x="346" y="176" width="64" height="28" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/><text x="378" y="194">dongle</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" fill="none">
    <line x1="205" y1="54" x2="110" y2="96"/><line x1="255" y1="54" x2="350" y2="96"/>
    <line x1="95" y1="126" x2="56" y2="176"/><line x1="125" y1="126" x2="132" y2="176"/>
    <line x1="335" y1="126" x2="302" y2="176"/><line x1="365" y1="126" x2="378" y2="176"/>
  </g>
  <text x="230" y="224" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">a host-controlled tree of tiers — up to 127 devices, all hot-pluggable</text>
</svg>
<figcaption>USB is a strict tree: one host controller at the root, hubs branching below it, and devices at the leaves. Every transfer is directed by the host, and the tiers can nest until the bus reaches its limit of 127 devices — any of which can be plugged or unplugged while the system runs.</figcaption>
</figure>

## Overview

A USB system has one *host* (the computer) that controls a tree of *devices* through hubs — keyboards, mice, drives, cameras, dongles. Ports are hot-pluggable, so devices can be attached and removed while running. Successive versions (USB 2.0, 3.x, 4) raised speeds by orders of magnitude, and the reversible USB-C connector added high power delivery (USB-PD), letting one cable both run and charge a device. It is part of a machine's [I/O](/reference/input-output/), distinct from internal expansion buses like [PCIe](/reference/pci-express/).

## What it's for

USB is how most external hardware reaches the [CPU](/reference/central-processing-unit/): a single cable carries data and often power to a [peripheral](/reference/peripheral/). It is central to SDR — an [RTL-SDR](/reference/rtl-sdr/) dongle plugs into a USB port and streams IQ samples to the host, and many capture nodes are powered over the same USB connection. The trade-off versus internal buses is bandwidth and latency: convenient and universal, but a USB link will not match a dedicated PCIe slot for raw throughput.

## Sources

[^wiki]: [USB](https://en.wikipedia.org/wiki/USB) — Wikipedia, on the Universal Serial Bus standard, versions, and connectors.
