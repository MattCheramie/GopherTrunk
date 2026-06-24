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

## Overview

A USB system has one *host* (the computer) that controls a tree of *devices* through hubs — keyboards, mice, drives, cameras, dongles. Ports are hot-pluggable, so devices can be attached and removed while running. Successive versions (USB 2.0, 3.x, 4) raised speeds by orders of magnitude, and the reversible USB-C connector added high power delivery (USB-PD), letting one cable both run and charge a device. It is part of a machine's [I/O](/reference/input-output/), distinct from internal expansion buses like [PCIe](/reference/pci-express/).

## What it's for

USB is how most external hardware reaches the [CPU](/reference/central-processing-unit/): a single cable carries data and often power to a [peripheral](/reference/peripheral/). It is central to SDR — an [RTL-SDR](/reference/rtl-sdr/) dongle plugs into a USB port and streams IQ samples to the host, and many capture nodes are powered over the same USB connection. The trade-off versus internal buses is bandwidth and latency: convenient and universal, but a USB link will not match a dedicated PCIe slot for raw throughput.

## Sources

[^wiki]: [USB](https://en.wikipedia.org/wiki/USB) — Wikipedia, on the Universal Serial Bus standard, versions, and connectors.
